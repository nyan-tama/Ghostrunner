package dashboard

import (
	"context"
	"sync"
	"testing"
	"time"

	"ghostrunner/backend/internal/idle"
)

// fakeIdleWriter は idle.Writer を満たすテスト用スタブです
type fakeIdleWriter struct {
	mu    sync.Mutex
	calls []writeSummaryCall
	err   error
}

type writeSummaryCall struct {
	sessionID         string
	expectedTimestamp int64
	summary           string
}

func (w *fakeIdleWriter) WriteSummary(sessionID string, expectedTimestamp int64, summary string, at time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, writeSummaryCall{sessionID, expectedTimestamp, summary})
	return w.err
}

func (w *fakeIdleWriter) callCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.calls)
}

// fakeSummarizeService は service.SummarizeService を満たすテスト用スタブです
type fakeSummarizeService struct {
	mu     sync.Mutex
	calls  int
	result string
	err    error
}

func (s *fakeSummarizeService) SummarizeIdle(ctx context.Context, lastAssistant, lastPrompt string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.result, s.err
}

func (s *fakeSummarizeService) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestNeedsSummary(t *testing.T) {
	// timestamp より後の要約時刻（要約済みで最新）
	afterTs := time.Unix(9900, 0).UTC().Format(time.RFC3339)
	// timestamp より前の要約時刻（要約後にマーカーが再書き込みされた＝rawTail変化）
	beforeTs := time.Unix(9700, 0).UTC().Format(time.RFC3339)

	tests := []struct {
		name string
		m    idle.Marker
		want bool
	}{
		{
			name: "Summary空は要約必要",
			m:    idle.Marker{Timestamp: 9820, Summary: "", SummarizedAt: afterTs},
			want: true,
		},
		{
			name: "SummarizedAt空は要約必要",
			m:    idle.Marker{Timestamp: 9820, Summary: "済", SummarizedAt: ""},
			want: true,
		},
		{
			name: "SummarizedAtパース失敗は要約必要",
			m:    idle.Marker{Timestamp: 9820, Summary: "済", SummarizedAt: "not-a-time"},
			want: true,
		},
		{
			name: "要約済みかつ最新(summarizedAtがtimestampより後)は不要",
			m:    idle.Marker{Timestamp: 9820, Summary: "済", SummarizedAt: afterTs},
			want: false,
		},
		{
			name: "要約後にマーカー再書き込み(timestampがsummarizedAtより後)は要約必要",
			m:    idle.Marker{Timestamp: 9820, Summary: "旧要約", SummarizedAt: beforeTs},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsSummary(tt.m); got != tt.want {
				t.Errorf("needsSummary: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectSummarizeTargets(t *testing.T) {
	now := time.Unix(10000, 0).UTC()

	// クロック固定。idleSummarizeDelay=2分 を境界に判定。
	stale := now.Add(-3 * time.Minute).Unix()                         // 滞留（3分）
	fresh := now.Add(-30 * time.Second).Unix()                        // 滞留前（30秒 < idleSummarizeDelay 45秒）
	summarizedAt := time.Unix(stale+30, 0).UTC().Format(time.RFC3339) // stale timestamp より後

	markers := []idle.Marker{
		{SessionID: "fresh", Timestamp: fresh, Summary: ""},                                         // 滞留前 → 対象外
		{SessionID: "stale-unsummarized", Timestamp: stale, Summary: ""},                            // 滞留かつ未要約 → 対象
		{SessionID: "stale-summarized", Timestamp: stale, Summary: "済", SummarizedAt: summarizedAt}, // 滞留だが要約済み → 対象外
	}

	got := selectSummarizeTargets(markers, now)

	if len(got) != 1 {
		t.Fatalf("targets: got %d, want 1 (%+v)", len(got), got)
	}
	if got[0].SessionID != "stale-unsummarized" {
		t.Errorf("target session: got %q, want stale-unsummarized", got[0].SessionID)
	}
}

// W-a: 要約が空/失敗を返すケースで、claimAttempts/lastAttempt により
// 5分以内は再試行されないこと（runOnceを複数回叩き、cooldown内は exec が呼ばれない）。
func TestSummarizer_クールダウンで5分以内は再試行しない(t *testing.T) {
	base := time.Unix(20000, 0).UTC()
	clock := base
	now := func() time.Time { return clock }

	reader := &fakeIdleReader{markers: []idle.Marker{
		{SessionID: "s1", Timestamp: base.Add(-3 * time.Minute).Unix(), Summary: ""},
	}}
	writer := &fakeIdleWriter{}
	svc := &fakeSummarizeService{result: ""} // 空要約（要約できず summary は "" のまま）

	s := NewSummarizer(reader, writer, svc, now)

	// 1回目: 滞留かつ未要約 → 試行される（svc呼び出し1回）
	s.runOnce(context.Background())
	if svc.callCount() != 1 {
		t.Fatalf("1st runOnce: svc calls got %d, want 1", svc.callCount())
	}

	// 2回目: cooldown(5分)内 → スキップされ svc は呼ばれない
	clock = base.Add(1 * time.Minute)
	s.runOnce(context.Background())
	if svc.callCount() != 1 {
		t.Fatalf("2nd runOnce within cooldown: svc calls got %d, want 1 (skipped)", svc.callCount())
	}

	// 3回目: cooldown経過 → 再試行される
	clock = base.Add(6 * time.Minute)
	s.runOnce(context.Background())
	if svc.callCount() != 2 {
		t.Fatalf("3rd runOnce after cooldown: svc calls got %d, want 2", svc.callCount())
	}

	// 空要約のため書き戻しは一度も発生しない
	if writer.callCount() != 0 {
		t.Errorf("writer calls: got %d, want 0 (empty summary is not written)", writer.callCount())
	}
}

// 要約成功時は List 時点(T0)の timestamp を基準に書き戻される。
func TestSummarizer_要約成功でtimestampを基準に書き戻す(t *testing.T) {
	base := time.Unix(30000, 0).UTC()
	now := func() time.Time { return base }
	ts := base.Add(-3 * time.Minute).Unix()

	reader := &fakeIdleReader{markers: []idle.Marker{
		{SessionID: "s1", Timestamp: ts, Summary: "", RawTail: idle.RawTail{LastAssistant: "選んで"}},
	}}
	writer := &fakeIdleWriter{}
	svc := &fakeSummarizeService{result: "A案かB案の選択を待っている"}

	s := NewSummarizer(reader, writer, svc, now)
	s.runOnce(context.Background())

	if writer.callCount() != 1 {
		t.Fatalf("writer calls: got %d, want 1", writer.callCount())
	}
	call := writer.calls[0]
	if call.sessionID != "s1" {
		t.Errorf("sessionID: got %q, want s1", call.sessionID)
	}
	if call.expectedTimestamp != ts {
		t.Errorf("expectedTimestamp: got %d, want %d (List時点T0基準)", call.expectedTimestamp, ts)
	}
	if call.summary != "A案かB案の選択を待っている" {
		t.Errorf("summary: got %q", call.summary)
	}
}
