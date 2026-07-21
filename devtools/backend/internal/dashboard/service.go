package dashboard

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"time"

	"ghostrunner/backend/internal/idle"
	"ghostrunner/backend/internal/projects"
)

// 失効・ノイズ抑制のしきい値（idle.TTL / idle.MinAge）は idle パッケージに集約しています（SSOT）。
// transcript の分類しきい値と重複させないため、dashboard も idle の定数を参照します。

// Service はダッシュボードの状態集約と回答書き戻しを提供します
type Service interface {
	// GetState は全プロジェクトの集約状態を返します
	GetState(ctx context.Context) (State, error)
	// Answer は確認事項に回答を書き戻します
	Answer(ctx context.Context, req AnswerRequest) error
}

type serviceImpl struct {
	configPath      string
	ghostrunnerRoot string
	idleReader      idle.Reader
	now             func() time.Time
}

// NewService は新しいServiceを生成します。
// idleReader は nil 許容で、nil の場合は質問待ちの付与をスキップします。
func NewService(configPath, ghostrunnerRoot string, idleReader idle.Reader) Service {
	return &serviceImpl{
		configPath:      configPath,
		ghostrunnerRoot: ghostrunnerRoot,
		idleReader:      idleReader,
		now:             time.Now,
	}
}

// NewServiceWithClock はclock注入付きのServiceを生成します（テスト用）。
// idleReader は nil 許容で、nil の場合は質問待ちの付与をスキップします。
func NewServiceWithClock(configPath, ghostrunnerRoot string, idleReader idle.Reader, now func() time.Time) Service {
	return &serviceImpl{
		configPath:      configPath,
		ghostrunnerRoot: ghostrunnerRoot,
		idleReader:      idleReader,
		now:             now,
	}
}

// GetState は全プロジェクトの集約状態を返します
func (s *serviceImpl) GetState(ctx context.Context) (State, error) {
	log.Printf("[DashboardService] GetState started")

	projs, err := projects.LoadProjects(s.configPath)
	if err != nil {
		return State{}, fmt.Errorf("failed to load projects: %w", err)
	}

	if projs == nil {
		log.Printf("[DashboardService] No projects config found, returning empty state")
		return State{
			Projects:    []ProjectState{},
			GeneratedAt: s.now().Format(time.RFC3339),
		}, nil
	}

	var states []ProjectState
	for _, p := range projs {
		select {
		case <-ctx.Done():
			return State{}, ctx.Err()
		default:
		}

		ps, err := ScanProject(p.Path, s.ghostrunnerRoot, s.now())
		if err != nil {
			log.Printf("[DashboardService] ScanProject failed: path=%s, error=%v", p.Path, err)
			// エラーでもwarning付きで含める
			ps = ProjectState{
				Name:       p.Name,
				Path:       p.Path,
				Attention:  AttentionWatching,
				Unanswered: []UnansweredQuestion{},
				Ops:        []OpsEntry{},
				Warnings:   []string{fmt.Sprintf("scan failed: %v", err)},
			}
		}
		states = append(states, ps)
	}

	// 質問待ちマーカーを各プロジェクトへ付与（idleReaderがnilの時はスキップ）
	now := s.now()
	if s.idleReader != nil {
		markers, err := s.idleReader.List(ctx)
		if err != nil {
			log.Printf("[DashboardService] idle marker list failed: %v", err)
		} else {
			attachIdleState(states, markers, now)
		}
	}

	// ソート: Idle存在DESC, attention優先度ASC, 経過時間DESC, isSelf ASC, name ASC（安定ソート）
	sort.SliceStable(states, func(i, j int) bool {
		// 第1キー: 質問待ち(Idle!=nil)を最優先（未回答由来requiredと分離・C2）
		ii := states[i].Idle != nil
		ij := states[j].Idle != nil
		if ii != ij {
			return ii // Idleありが先
		}
		// 第2キー: 動作中(Running!=nil)を次に優先（質問待ちの次・既存attentionより上）
		ri := states[i].Running != nil
		rj := states[j].Running != nil
		if ri != rj {
			return ri // Runningありが先
		}
		// 第3キー: attention優先度ASC
		pi := attentionPriority(states[i].Attention)
		pj := attentionPriority(states[j].Attention)
		if pi != pj {
			return pi < pj
		}
		// 第4キー: 経過時間DESC（長く待たせているものを先に・露出しない内部計算）
		ei := idleElapsed(states[i], now)
		ej := idleElapsed(states[j], now)
		if ei != ej {
			return ei > ej
		}
		// 第5キー: isSelf ASC（isSelf=falseが先）
		if states[i].IsSelf != states[j].IsSelf {
			return !states[i].IsSelf
		}
		// 第6キー: name ASC
		return states[i].Name < states[j].Name
	})

	result := State{
		Projects:    states,
		GeneratedAt: s.now().Format(time.RFC3339),
	}

	log.Printf("[DashboardService] GetState completed: projects=%d", len(states))
	return result, nil
}

// Answer は確認事項に回答を書き戻します
func (s *serviceImpl) Answer(ctx context.Context, req AnswerRequest) error {
	log.Printf("[DashboardService] Answer started: project=%s, plan=%s, line=%d", req.ProjectPath, req.PlanPath, req.LineStart)

	projs, err := projects.LoadProjects(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	if projs == nil {
		return fmt.Errorf("%w: no projects configured", ErrValidation)
	}

	if err := AnswerQuestion(req, projs); err != nil {
		log.Printf("[DashboardService] Answer failed: error=%v", err)
		return err
	}

	log.Printf("[DashboardService] Answer completed: project=%s, plan=%s", req.ProjectPath, req.PlanPath)
	return nil
}

// attachIdleState は reader が返す代表マーカー（プロジェクト毎1件・Status/SessionCount 込み）を
// 各プロジェクトへ付与します。reader が既に per-project 代表へ collapse 済みのため、ここでは
// MatchProject で割り当て → Marker.Status で Idle(waiting)/Running(running) へディスパッチするのみです（C-1）。
// TTL(idle.TTL)を超えた失効マーカーは除外します（削除はしない・読み取り専用）。
// idleMinAge(idle.MinAge)ゲートは waiting のみに適用し、running には適用しません
// （適用すると age<60s の fresh な running が落ち、動作中が一切表示されなくなる・C-1）。
// 付与後は Attention を再評価します（C1）。
func attachIdleState(states []ProjectState, markers []idle.Marker, now time.Time) {
	// MatchProject 用にプロジェクト一覧を組み立てる
	projs := make([]projects.Project, len(states))
	for i, s := range states {
		projs[i] = projects.Project{Path: s.Path, Name: s.Name}
	}

	// マッチしたプロジェクトパス（Clean済み）ごとに代表マーカーを1件割り当てる。
	// reader が既に代表1件に collapse しているため通常は1件。防御的に先着優先で1件に保つ。
	matched := make(map[string]idle.Marker)
	for _, m := range markers {
		if idle.IsExpired(m, now, idle.TTL) {
			continue
		}
		proj, ok := idle.MatchProject(m.Cwd, projs)
		if !ok {
			continue
		}
		if _, exists := matched[proj]; !exists {
			matched[proj] = m
		}
	}

	for i := range states {
		m, ok := matched[filepath.Clean(states[i].Path)]
		if !ok {
			continue
		}

		switch m.Status {
		case idle.StatusWaiting:
			// waiting のみ idleMinAge ゲートを適用（応答直後のノイズ抑制）
			if now.Sub(time.Unix(m.Timestamp, 0)) < idle.MinAge {
				continue
			}
			states[i].Idle = &IdleState{
				Timestamp:    time.Unix(m.Timestamp, 0).Format(time.RFC3339),
				Preview:      truncateRunes(m.RawTail.LastAssistant, 80),
				SessionCount: m.SessionCount,
				Summary:      m.Summary,
				SummarizedAt: m.SummarizedAt,
			}
		case idle.StatusRunning:
			// running は idleMinAge ゲートを通さない（fresh running を落とさない・C-1）
			states[i].Running = &RunningState{
				Preview:      truncateRunes(m.RawTail.LastAssistant, 80),
				SessionCount: m.SessionCount,
			}
		default:
			continue
		}

		// 付与後に Attention を再評価（C1）
		states[i].Attention = determineAttention(states[i])
	}
}

// idleElapsed は質問待ちの経過時間を返します（Idleなしは0）。ソートの内部計算専用で外部には露出しません。
func idleElapsed(s ProjectState, now time.Time) time.Duration {
	if s.Idle == nil {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s.Idle.Timestamp)
	if err != nil {
		return 0
	}
	return now.Sub(t)
}

// truncateRunes は文字列をrune境界を保って先頭n文字に切り詰めます
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// attentionPriority はAttentionのソート優先度を返します（小さいほど優先）
func attentionPriority(a Attention) int {
	switch a {
	case AttentionRequired:
		return 0
	case AttentionProgress:
		return 1
	case AttentionWatching:
		return 2
	default:
		return 3
	}
}
