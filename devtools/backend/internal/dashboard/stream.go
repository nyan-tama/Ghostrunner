package dashboard

import (
	"context"
	"log"
	"reflect"
	"sync"
	"time"
)

const (
	// streamScanInterval はダッシュボード状態の差分検出スキャン間隔です
	streamScanInterval = 2 * time.Second
	// streamBufferSize はsubscriberチャネルのバッファサイズ（最新優先・小さめ）です
	streamBufferSize = 1
)

// StreamService はダッシュボード状態のSSE配信を提供します。
// 内部tickerで短間隔スキャンし、前回Stateと実変化があった場合のみ
// State スナップショット全体をsubscriberへbroadcastします。
type StreamService interface {
	// Subscribe は状態更新を受け取るチャネルと購読解除関数を返します
	Subscribe() (<-chan State, func())
	// Start は差分検出のバックグラウンドスキャンを開始します
	Start(ctx context.Context)
	// Stop はスキャンを停止し全subscriberチャネルを閉じます
	Stop()
}

type streamServiceImpl struct {
	svc Service

	mu          sync.Mutex
	subscribers map[int]chan State
	nextID      int
	lastState   *State

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewStreamService は新しいStreamServiceを生成します
func NewStreamService(svc Service) StreamService {
	return &streamServiceImpl{
		svc:         svc,
		subscribers: make(map[int]chan State),
	}
}

// Subscribe は状態更新チャネルと購読解除関数を返します。
// 購読直後に最新Stateが存在すれば初期値として1件送ります。
func (s *streamServiceImpl) Subscribe() (<-chan State, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan State, streamBufferSize)
	id := s.nextID
	s.nextID++
	s.subscribers[id] = ch

	// 購読直後に現状のStateを反映（バッファに空きがあるため非ブロッキング）
	if s.lastState != nil {
		ch <- *s.lastState
	}

	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := s.subscribers[id]; ok {
			delete(s.subscribers, id)
			close(ch)
		}
	}

	return ch, unsubscribe
}

// Start は差分検出のバックグラウンドスキャンを開始します
func (s *streamServiceImpl) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(streamScanInterval)
		defer ticker.Stop()

		log.Printf("[DashboardStream] started: interval=%s", streamScanInterval)
		for {
			select {
			case <-ctx.Done():
				log.Printf("[DashboardStream] stopped")
				return
			case <-ticker.C:
				s.scanAndBroadcast(ctx)
			}
		}
	}()
}

// Stop はスキャンを停止し全subscriberチャネルを閉じます
func (s *streamServiceImpl) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()
	for id, ch := range s.subscribers {
		delete(s.subscribers, id)
		close(ch)
	}
}

// scanAndBroadcast は状態を取得し、前回と実変化があればbroadcastします
func (s *streamServiceImpl) scanAndBroadcast(ctx context.Context) {
	state, err := s.svc.GetState(ctx)
	if err != nil {
		log.Printf("[DashboardStream] GetState failed: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	changed := s.lastState == nil || statesDiffer(*s.lastState, state)
	stateCopy := state
	s.lastState = &stateCopy
	if !changed {
		return
	}

	for _, ch := range s.subscribers {
		sendCoalesce(ch, state)
	}
}

// sendCoalesce はsubscriberチャネルへ最新Stateを送ります。
// バッファ満杯時は古い値を捨てて最新を入れます（coalesce）。
// 送信元はscanAndBroadcastのみ（mu保持下）のため2回以内に収束します。
func sendCoalesce(ch chan State, state State) {
	for {
		select {
		case ch <- state:
			return
		default:
			select {
			case <-ch:
			default:
			}
		}
	}
}

// statesDiffer は2つのStateに表示上の実変化があるかを判定します。
// GeneratedAt は毎スキャン更新されるため比較対象から除外し、Projects のみを比較します。
// 経過時間はStateに露出していないため差分判定に含まれません（W1）。
func statesDiffer(a, b State) bool {
	return !reflect.DeepEqual(a.Projects, b.Projects)
}
