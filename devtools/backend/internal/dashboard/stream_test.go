package dashboard

import (
	"context"
	"testing"
	"time"
)

// fakeDashboardService は Service を満たすテスト用スタブです
type fakeDashboardService struct {
	state State
	err   error
}

func (f *fakeDashboardService) GetState(ctx context.Context) (State, error) {
	return f.state, f.err
}

func (f *fakeDashboardService) Answer(ctx context.Context, req AnswerRequest) error {
	return nil
}

func TestStatesDiffer(t *testing.T) {
	baseProjects := []ProjectState{
		{
			Name:      "proj",
			Path:      "/a/b",
			Attention: AttentionWatching,
			Kanban:    KanbanCounts{Running: 1},
		},
	}
	base := State{Projects: baseProjects, GeneratedAt: "2026-07-20T12:00:00Z"}

	// 各種実変化を作るヘルパー
	withIdle := func(idle *IdleState) State {
		p := ProjectState{
			Name: "proj", Path: "/a/b", Attention: AttentionWatching,
			Kanban: KanbanCounts{Running: 1}, Idle: idle,
		}
		return State{Projects: []ProjectState{p}, GeneratedAt: "2026-07-20T12:00:00Z"}
	}

	tests := []struct {
		name string
		a    State
		b    State
		want bool
	}{
		{
			name: "GeneratedAtのみ違いは差分なし(W1・経過で再送しない)",
			a:    base,
			b:    State{Projects: baseProjects, GeneratedAt: "2026-07-20T12:00:05Z"},
			want: false,
		},
		{
			name: "完全一致は差分なし",
			a:    base,
			b:    base,
			want: false,
		},
		{
			name: "idle出現は差分あり",
			a:    base,
			b:    withIdle(&IdleState{Timestamp: "2026-07-20T11:58:00Z", Preview: "選んで"}),
			want: true,
		},
		{
			name: "attention変化は差分あり",
			a:    base,
			b: State{Projects: []ProjectState{{
				Name: "proj", Path: "/a/b", Attention: AttentionRequired,
				Kanban: KanbanCounts{Running: 1},
			}}, GeneratedAt: "2026-07-20T12:00:00Z"},
			want: true,
		},
		{
			name: "summary変化は差分あり",
			a:    withIdle(&IdleState{Timestamp: "2026-07-20T11:58:00Z", Summary: ""}),
			b:    withIdle(&IdleState{Timestamp: "2026-07-20T11:58:00Z", Summary: "認証を確認中"}),
			want: true,
		},
		{
			name: "preview変化は差分あり",
			a:    withIdle(&IdleState{Timestamp: "2026-07-20T11:58:00Z", Preview: "旧"}),
			b:    withIdle(&IdleState{Timestamp: "2026-07-20T11:58:00Z", Preview: "新"}),
			want: true,
		},
		{
			name: "kanban変化は差分あり",
			a:    base,
			b: State{Projects: []ProjectState{{
				Name: "proj", Path: "/a/b", Attention: AttentionWatching,
				Kanban: KanbanCounts{Running: 2},
			}}, GeneratedAt: "2026-07-20T12:00:00Z"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statesDiffer(tt.a, tt.b); got != tt.want {
				t.Errorf("statesDiffer: got %v, want %v", got, tt.want)
			}
		})
	}
}

// sendCoalesce は満杯チャネルで古い値を捨てて最新を入れ、ブロックしないこと。
func TestSendCoalesce_満杯チャネルで最新に置き換わる(t *testing.T) {
	ch := make(chan State, streamBufferSize) // streamBufferSize=1

	old := State{GeneratedAt: "old"}
	newer := State{GeneratedAt: "new"}

	ch <- old // バッファを満杯にする

	done := make(chan struct{})
	go func() {
		sendCoalesce(ch, newer) // ブロックしてはいけない
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("sendCoalesce blocked on full channel")
	}

	got := <-ch
	if got.GeneratedAt != "new" {
		t.Errorf("coalesced value: got %q, want new (古い値は捨てられ最新が入る)", got.GeneratedAt)
	}
}

// Subscribe/broadcast の基本動作:
// - 購読後 scanAndBroadcast で最新Stateが購読者に届く
// - 既に lastState があれば購読直後に初期値が届く
// - unsubscribe でチャネルが閉じられる（ライフサイクル正常）
func TestStream_SubscribeBroadcastAndUnsubscribe(t *testing.T) {
	stateA := State{
		Projects:    []ProjectState{{Name: "p", Path: "/a", Attention: AttentionWatching}},
		GeneratedAt: "2026-07-20T12:00:00Z",
	}
	svc := &fakeDashboardService{state: stateA}
	s := &streamServiceImpl{svc: svc, subscribers: make(map[int]chan State)}

	// 1人目購読（lastState nil のため初期値は届かない）
	ch1, unsub1 := s.Subscribe()
	select {
	case v := <-ch1:
		t.Fatalf("unexpected initial value on empty lastState: %+v", v)
	default:
	}

	// スキャンで初回配信 → ch1 が受信
	s.scanAndBroadcast(context.Background())
	got := <-ch1
	if got.GeneratedAt != stateA.GeneratedAt {
		t.Errorf("broadcast: got %q, want %q", got.GeneratedAt, stateA.GeneratedAt)
	}

	// 2人目購読 → lastState があるので購読直後に初期値が届く
	ch2, unsub2 := s.Subscribe()
	got2 := <-ch2
	if got2.GeneratedAt != stateA.GeneratedAt {
		t.Errorf("initial snapshot: got %q, want %q", got2.GeneratedAt, stateA.GeneratedAt)
	}

	// unsubscribe でチャネルが閉じる
	unsub2()
	if _, ok := <-ch2; ok {
		t.Errorf("channel should be closed after unsubscribe")
	}

	// 二重 unsubscribe しても安全
	unsub2()
	unsub1()
	if _, ok := <-ch1; ok {
		t.Errorf("ch1 should be closed after unsubscribe")
	}
}
