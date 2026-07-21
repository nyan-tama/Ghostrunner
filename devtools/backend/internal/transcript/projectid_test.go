package transcript

import (
	"os"
	"path/filepath"
	"testing"

	"ghostrunner/backend/internal/projects"
)

// TestDeriveProjectID は非英数字→"-"置換の変換規則を検証します（section4 case1）。
func TestDeriveProjectID(t *testing.T) {
	tests := []struct {
		name string
		cwd  string
		want string
	}{
		{name: "標準パス", cwd: "/Users/user/j-board", want: "-Users-user-j-board"},
		{name: "ドット含む", cwd: "/Users/x/my.app", want: "-Users-x-my-app"},
		{name: "数字保持", cwd: "/srv/app2", want: "-srv-app2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveProjectID(tt.cwd); got != tt.want {
				t.Errorf("deriveProjectID(%q) = %q, want %q", tt.cwd, got, tt.want)
			}
		})
	}
}

// TestDiscoverSessions_Enumerate は projid dir 内の *.jsonl が mtime 付きで列挙されることを検証します（section4 case2）。
func TestDiscoverSessions_Enumerate(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	writeSession(t, home, "-Users-x-app", "s1", asstText("2026-07-20T10:00:00Z", appPath, "a"))
	writeSession(t, home, "-Users-x-app", "s2", asstText("2026-07-20T10:00:00Z", appPath, "b"))

	got, err := discoverSessions(home, []projects.Project{{Path: appPath, Name: "app"}})
	if err != nil {
		t.Fatalf("discoverSessions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("sessions = %d, want 2", len(got))
	}
	ids := map[string]bool{}
	for _, sf := range got {
		ids[sf.sessionID] = true
		if sf.modTime.IsZero() {
			t.Errorf("session %q has zero modTime", sf.sessionID)
		}
	}
	if !ids["s1"] || !ids["s2"] {
		t.Errorf("missing sessions; got %v", ids)
	}
}

// TestDiscoverSessions_GlobFallback は glob が外れても全走査 fallback で拾うことを検証します（W6・section4 case2）。
func TestDiscoverSessions_GlobFallback(t *testing.T) {
	home := t.TempDir()
	appPath := "/Users/x/app"
	// deriveProjectID と一致しない dir 名 → glob 空 → 全走査 fallback
	writeSession(t, home, "unmatched-dir", "s1", asstText("2026-07-20T10:00:00Z", appPath, "a"))

	got, err := discoverSessions(home, []projects.Project{{Path: appPath, Name: "app"}})
	if err != nil {
		t.Fatalf("discoverSessions: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("sessions = %d, want 1 (fallbackで拾う)", len(got))
	}
	if got[0].sessionID != "s1" {
		t.Errorf("sessionID = %q, want s1", got[0].sessionID)
	}
}

// TestDiscoverSessions_MissingProjectsDir は projects ディレクトリ不在で空・非エラーを返すことを検証します。
func TestDiscoverSessions_MissingProjectsDir(t *testing.T) {
	home := t.TempDir() // .claude/projects を作らない
	got, err := discoverSessions(home, []projects.Project{{Path: "/Users/x/app", Name: "app"}})
	if err != nil {
		t.Fatalf("discoverSessions should not error on missing dir: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("sessions = %d, want 0", len(got))
	}
	// projects dir が実際に存在しないことの確認（前提保証）
	if _, statErr := os.Stat(filepath.Join(home, ".claude", "projects")); !os.IsNotExist(statErr) {
		t.Fatalf("precondition: projects dir should not exist")
	}
}
