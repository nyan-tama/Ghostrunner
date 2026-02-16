package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestProjectsHandler_Handle(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) string // ベースディレクトリのパスを返す
		wantStatus     int
		wantSuccess    bool
		wantProjects   []ProjectInfo // nil の場合はレスポンスの projects が空（nil or 空スライス）であることを検証
		wantError      string        // 空文字の場合はエラーなしを期待
		wantErrContain bool          // true の場合、wantError を部分一致で検証
	}{
		{
			name: "正常系_複数ディレクトリがアルファベット順で返される",
			setup: func(t *testing.T) string {
				t.Helper()
				base := t.TempDir()
				for _, d := range []string{"dir1", "dir2", "dir3"} {
					if err := os.Mkdir(filepath.Join(base, d), 0o755); err != nil {
						t.Fatalf("failed to create directory %s: %v", d, err)
					}
				}
				return base
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
			wantProjects: []ProjectInfo{
				{Name: "dir1"},
				{Name: "dir2"},
				{Name: "dir3"},
			},
		},
		{
			name: "隠しディレクトリが除外される",
			setup: func(t *testing.T) string {
				t.Helper()
				base := t.TempDir()
				for _, d := range []string{".hidden", ".config", "visible"} {
					if err := os.Mkdir(filepath.Join(base, d), 0o755); err != nil {
						t.Fatalf("failed to create directory %s: %v", d, err)
					}
				}
				return base
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
			wantProjects: []ProjectInfo{
				{Name: "visible"},
			},
		},
		{
			name: "ファイルが除外されディレクトリのみ返される",
			setup: func(t *testing.T) string {
				t.Helper()
				base := t.TempDir()
				if err := os.Mkdir(filepath.Join(base, "dir1"), 0o755); err != nil {
					t.Fatalf("failed to create directory dir1: %v", err)
				}
				if err := os.WriteFile(filepath.Join(base, "file.txt"), []byte("test"), 0o644); err != nil {
					t.Fatalf("failed to create file.txt: %v", err)
				}
				return base
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
			wantProjects: []ProjectInfo{
				{Name: "dir1"},
			},
		},
		{
			name: "空ディレクトリの場合に空配列が返される",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
			// wantProjects が nil => 0件であることを検証
		},
		{
			name: "ベースディレクトリが存在しない場合に500エラー",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantStatus:     http.StatusInternalServerError,
			wantSuccess:    false,
			wantError:      "ディレクトリ一覧の取得に失敗しました",
			wantErrContain: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := tt.setup(t)

			h := &ProjectsHandler{BaseDir: baseDir}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/projects", nil)

			h.Handle(c)

			// ステータスコード検証
			if w.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", w.Code, tt.wantStatus)
			}

			// レスポンスボディをパース
			var resp ProjectsResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response body: %v\nbody: %s", err, w.Body.String())
			}

			// success フラグ検証
			if resp.Success != tt.wantSuccess {
				t.Errorf("success: got %v, want %v", resp.Success, tt.wantSuccess)
			}

			// エラーメッセージ検証
			if tt.wantError != "" {
				if tt.wantErrContain {
					if resp.Error == "" {
						t.Errorf("error: got empty, want containing %q", tt.wantError)
					}
				} else {
					if resp.Error != tt.wantError {
						t.Errorf("error: got %q, want %q", resp.Error, tt.wantError)
					}
				}
			}

			// プロジェクト一覧検証
			if tt.wantProjects == nil {
				// 空配列を期待するケース
				if len(resp.Projects) != 0 {
					t.Errorf("projects count: got %d, want 0", len(resp.Projects))
				}
			} else {
				if len(resp.Projects) != len(tt.wantProjects) {
					t.Fatalf("projects count: got %d, want %d\nprojects: %+v",
						len(resp.Projects), len(tt.wantProjects), resp.Projects)
				}
				for i, want := range tt.wantProjects {
					got := resp.Projects[i]
					if got.Name != want.Name {
						t.Errorf("projects[%d].Name: got %q, want %q", i, got.Name, want.Name)
					}
					// Path はベースディレクトリ + ディレクトリ名のフルパスであることを検証
					expectedPath := filepath.Join(baseDir, want.Name)
					if got.Path != expectedPath {
						t.Errorf("projects[%d].Path: got %q, want %q", i, got.Path, expectedPath)
					}
				}
			}
		})
	}
}
