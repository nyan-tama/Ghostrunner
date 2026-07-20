package idle

import (
	"testing"
	"time"

	"ghostrunner/backend/internal/projects"
)

func TestMatchProject(t *testing.T) {
	tests := []struct {
		name        string
		cwd         string
		projects    []projects.Project
		wantMatched string
		wantOK      bool
	}{
		{
			name:        "完全一致",
			cwd:         "/a/b",
			projects:    []projects.Project{{Path: "/a/b"}},
			wantMatched: "/a/b",
			wantOK:      true,
		},
		{
			name:        "前方一致(サブディレクトリ)",
			cwd:         "/a/b/sub",
			projects:    []projects.Project{{Path: "/a/b"}},
			wantMatched: "/a/b",
			wantOK:      true,
		},
		{
			name:        "最長一致優先",
			cwd:         "/a/b/c/x",
			projects:    []projects.Project{{Path: "/a/b"}, {Path: "/a/b/c"}},
			wantMatched: "/a/b/c",
			wantOK:      true,
		},
		{
			name:        "パス境界(誤マッチ防止・W7)",
			cwd:         "/a/bc",
			projects:    []projects.Project{{Path: "/a/b"}},
			wantMatched: "",
			wantOK:      false,
		},
		{
			name:        "末尾スラッシュ/未正規化はClean両辺で吸収",
			cwd:         "/a/b/",
			projects:    []projects.Project{{Path: "/a/b"}},
			wantMatched: "/a/b",
			wantOK:      true,
		},
		{
			name:        "不一致",
			cwd:         "/x/y",
			projects:    []projects.Project{{Path: "/a/b"}},
			wantMatched: "",
			wantOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, ok := MatchProject(tt.cwd, tt.projects)
			if ok != tt.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tt.wantOK)
			}
			if matched != tt.wantMatched {
				t.Errorf("matched: got %q, want %q", matched, tt.wantMatched)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	// nanosecondを0にしてtime.Unixの秒切り捨てによる境界ブレを避ける
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	ttl := 6 * time.Hour

	tests := []struct {
		name    string
		elapsed time.Duration
		want    bool
	}{
		{name: "期限内(5h59m)", elapsed: 5*time.Hour + 59*time.Minute, want: false},
		{name: "期限切れ(6h1m)", elapsed: 6*time.Hour + 1*time.Minute, want: true},
		{name: "境界ちょうど(6h)はfalse(等号は失効しない)", elapsed: 6 * time.Hour, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Marker{Timestamp: now.Add(-tt.elapsed).Unix()}
			got := IsExpired(m, now, ttl)
			if got != tt.want {
				t.Errorf("IsExpired: got %v, want %v", got, tt.want)
			}
		})
	}
}
