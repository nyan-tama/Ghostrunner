package dashboard

import (
	"context"
	"log"
	"sync"
	"time"

	"ghostrunner/backend/internal/idle"
	"ghostrunner/backend/internal/service"
)

const (
	// summarizerInterval は滞留マーカーの要約ジョブの実行間隔です
	summarizerInterval = 30 * time.Second
	// idleSummarizeDelay はこの時間以上滞留したマーカーのみ要約対象とします。
	// 表示閾値(idleMinAge=60秒)の少し手前に設定し、質問待ちカードが表示される頃には
	// 要約が済んでいる状態を狙います（短時間で解消する質問への無駄打ちは避ける）。
	idleSummarizeDelay = 45 * time.Second
	// summarizeConcurrency は要約の並列実行数（CLIコストを抑えるため小さくします）
	summarizeConcurrency = 2
	// summarizeCooldown は同一sessionへの再要約の最小間隔です。
	// 要約が空/失敗（summary が "" のまま）でも needsSummary が繰り返し true になり
	// 30秒毎に永久再要約されるコスト暴走を防ぎます（W-a）。
	summarizeCooldown = 5 * time.Minute
)

// Summarizer は滞留した質問待ちマーカーを検出し、SummarizeServiceで要約して
// マーカーへ書き戻すバックグラウンドジョブです。
type Summarizer struct {
	reader idle.Reader
	writer idle.Writer
	svc    service.SummarizeService
	now    func() time.Time

	mu          sync.Mutex
	lastAttempt map[string]time.Time // key: sessionID。要約試行時刻（結果によらず記録）

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewSummarizer は新しいSummarizerを生成します。now が nil の場合は time.Now を使います。
func NewSummarizer(reader idle.Reader, writer idle.Writer, svc service.SummarizeService, now func() time.Time) *Summarizer {
	if now == nil {
		now = time.Now
	}
	return &Summarizer{
		reader:      reader,
		writer:      writer,
		svc:         svc,
		now:         now,
		lastAttempt: make(map[string]time.Time),
	}
}

// Start は要約ジョブを開始します。ctx のキャンセルまたは Stop で終了します。
func (s *Summarizer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(summarizerInterval)
		defer ticker.Stop()

		log.Printf("[Summarizer] started: interval=%s, delay=%s", summarizerInterval, idleSummarizeDelay)
		for {
			select {
			case <-ctx.Done():
				log.Printf("[Summarizer] stopped")
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

// Stop は要約ジョブを停止し、実行中のtickが終わるまで待機します
func (s *Summarizer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// runOnce は1回分の要約サイクルを実行します
func (s *Summarizer) runOnce(ctx context.Context) {
	markers, err := s.reader.List(ctx)
	if err != nil {
		log.Printf("[Summarizer] list markers failed: %v", err)
		return
	}

	now := s.now()
	s.pruneLastAttempt(markers)

	candidates := selectSummarizeTargets(markers, now)

	// クールダウン: 前回試行から summarizeCooldown 未満の session はスキップし、
	// 試行するものは結果によらず試行時刻を記録する（W-a）。
	targets := s.claimAttempts(candidates, now)
	if len(targets) == 0 {
		return
	}
	log.Printf("[Summarizer] runOnce: targets=%d", len(targets))

	sem := make(chan struct{}, summarizeConcurrency)
	var wg sync.WaitGroup
	for _, m := range targets {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
		}

		wg.Add(1)
		go func(mk idle.Marker) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			s.summarizeOne(ctx, mk)
		}(m)
	}
	wg.Wait()
}

// summarizeOne は1つのマーカーを要約して書き戻します
func (s *Summarizer) summarizeOne(ctx context.Context, m idle.Marker) {
	summary, err := s.svc.SummarizeIdle(ctx, m.RawTail.LastAssistant, m.RawTail.LastPrompt)
	if err != nil {
		log.Printf("[Summarizer] summarize failed: session=%s, error=%v", m.SessionID, err)
		return
	}
	if summary == "" {
		return
	}
	// 基準は List 時点（T0）の timestamp。要約中にマーカーが削除/再生成されても
	// 新マーカーへ旧要約を上書きしない（C3）。
	if err := s.writer.WriteSummary(m.SessionID, m.Timestamp, summary, s.now()); err != nil {
		log.Printf("[Summarizer] write summary failed: session=%s, error=%v", m.SessionID, err)
	}
}

// claimAttempts はクールダウン内でない候補のみを返し、返した候補の試行時刻を記録します。
func (s *Summarizer) claimAttempts(candidates []idle.Marker, now time.Time) []idle.Marker {
	s.mu.Lock()
	defer s.mu.Unlock()

	targets := make([]idle.Marker, 0, len(candidates))
	for _, m := range candidates {
		if last, ok := s.lastAttempt[m.SessionID]; ok && now.Sub(last) < summarizeCooldown {
			continue
		}
		s.lastAttempt[m.SessionID] = now
		targets = append(targets, m)
	}
	return targets
}

// pruneLastAttempt は現存しないマーカーの試行記録を掃除しメモリ肥大を防ぎます。
func (s *Summarizer) pruneLastAttempt(markers []idle.Marker) {
	alive := make(map[string]struct{}, len(markers))
	for _, m := range markers {
		alive[m.SessionID] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for sessionID := range s.lastAttempt {
		if _, ok := alive[sessionID]; !ok {
			delete(s.lastAttempt, sessionID)
		}
	}
}

// selectSummarizeTargets は滞留かつ未要約の質問待ちマーカーを抽出します（純粋関数）。
// 対象は Status==waiting のみ。動作中(running)は内容が刻々変わるため要約せず、haiku を無駄打ちしません。
// 滞留: now - timestamp >= idleSummarizeDelay
// 未要約: Summary が空、または RawTail が SummarizedAt 以降に更新された
func selectSummarizeTargets(markers []idle.Marker, now time.Time) []idle.Marker {
	targets := make([]idle.Marker, 0, len(markers))
	for _, m := range markers {
		if m.Status != idle.StatusWaiting {
			continue
		}
		elapsed := now.Sub(time.Unix(m.Timestamp, 0))
		if elapsed < idleSummarizeDelay {
			continue
		}
		if !needsSummary(m) {
			continue
		}
		targets = append(targets, m)
	}
	return targets
}

// needsSummary はマーカーが未要約かを判定します。
// Summary が空、SummarizedAt が空、または SummarizedAt のパース失敗、
// あるいはマーカー書き込み時刻（timestamp）が SummarizedAt より後の場合に要約が必要です。
func needsSummary(m idle.Marker) bool {
	if m.Summary == "" {
		return true
	}
	if m.SummarizedAt == "" {
		return true
	}
	summarizedAt, err := time.Parse(time.RFC3339, m.SummarizedAt)
	if err != nil {
		return true
	}
	return time.Unix(m.Timestamp, 0).After(summarizedAt)
}
