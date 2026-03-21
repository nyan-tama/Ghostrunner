package handler

import (
	"bytes"
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

func TestProjectsHandler_HandleDestroy(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (homeDir string, body string)
		wantStatus int
		wantJSON   map[string]interface{}
		verify     func(t *testing.T, homeDir string) // 削除後の検証
	}{
		{
			name: "正常系_ディレクトリが削除される",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				home := t.TempDir()
				target := filepath.Join(home, "my-project")
				if err := os.Mkdir(target, 0o755); err != nil {
					t.Fatalf("failed to create target directory: %v", err)
				}
				// ファイルも作成して中身ごと削除されることを確認
				if err := os.WriteFile(filepath.Join(target, "file.txt"), []byte("test"), 0o644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				return home, `{"path":"` + target + `"}`
			},
			wantStatus: http.StatusOK,
			wantJSON:   map[string]interface{}{"success": true},
			verify: func(t *testing.T, homeDir string) {
				t.Helper()
				target := filepath.Join(homeDir, "my-project")
				if _, err := os.Stat(target); !os.IsNotExist(err) {
					t.Errorf("directory should be deleted, but still exists: %s", target)
				}
			},
		},
		{
			name: "正常系_docker-compose.ymlがあってもdocker失敗で削除は続行される",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				home := t.TempDir()
				target := filepath.Join(home, "docker-project")
				if err := os.Mkdir(target, 0o755); err != nil {
					t.Fatalf("failed to create target directory: %v", err)
				}
				// docker-compose.yml を配置（docker compose down は失敗するがログのみで続行）
				if err := os.WriteFile(filepath.Join(target, "docker-compose.yml"), []byte("version: '3'"), 0o644); err != nil {
					t.Fatalf("failed to create docker-compose.yml: %v", err)
				}
				return home, `{"path":"` + target + `"}`
			},
			wantStatus: http.StatusOK,
			wantJSON:   map[string]interface{}{"success": true},
			verify: func(t *testing.T, homeDir string) {
				t.Helper()
				target := filepath.Join(homeDir, "docker-project")
				if _, err := os.Stat(target); !os.IsNotExist(err) {
					t.Errorf("directory should be deleted, but still exists: %s", target)
				}
			},
		},
		{
			name: "異常系_パスが空の場合400エラー",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				return t.TempDir(), `{"path":""}`
			},
			wantStatus: http.StatusBadRequest,
			wantJSON:   map[string]interface{}{"success": false, "error": "パスが指定されていません"},
		},
		{
			name: "異常系_パストラバーサル検出で400エラー",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				home := t.TempDir()
				// ホームディレクトリ直下ではなくサブディレクトリ配下を指定
				nested := filepath.Join(home, "parent", "child")
				return home, `{"path":"` + nested + `"}`
			},
			wantStatus: http.StatusBadRequest,
			wantJSON:   map[string]interface{}{"success": false, "error": "ホームディレクトリ直下のプロジェクトのみ削除できます"},
		},
		{
			name: "異常系_相対パストラバーサルで400エラー",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				home := t.TempDir()
				traversal := filepath.Join(home, "project", "..", "..", "etc")
				return home, `{"path":"` + traversal + `"}`
			},
			wantStatus: http.StatusBadRequest,
			wantJSON:   map[string]interface{}{"success": false, "error": "ホームディレクトリ直下のプロジェクトのみ削除できます"},
		},
		{
			name: "異常系_存在しないディレクトリで404エラー",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				home := t.TempDir()
				nonexistent := filepath.Join(home, "nonexistent")
				return home, `{"path":"` + nonexistent + `"}`
			},
			wantStatus: http.StatusNotFound,
			wantJSON:   map[string]interface{}{"success": false, "error": "指定されたディレクトリが見つかりません"},
		},
		{
			name: "境界値_ホームディレクトリ自体を指定すると400エラー",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				home := t.TempDir()
				// ホームディレクトリ自体を削除対象に指定
				return home, `{"path":"` + home + `"}`
			},
			wantStatus: http.StatusBadRequest,
			wantJSON:   map[string]interface{}{"success": false, "error": "ホームディレクトリ直下のプロジェクトのみ削除できます"},
		},
		{
			name: "異常系_ファイルを指定した場合400エラー",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				home := t.TempDir()
				filePath := filepath.Join(home, "file.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				return home, `{"path":"` + filePath + `"}`
			},
			wantStatus: http.StatusBadRequest,
			wantJSON:   map[string]interface{}{"success": false, "error": "指定されたパスはディレクトリではありません"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeDir, body := tt.setup(t)

			h := &ProjectsHandler{HomeDir: homeDir}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/projects/destroy", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")

			h.HandleDestroy(c)

			// ステータスコード検証
			if w.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			// レスポンスJSON検証
			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response body: %v\nbody: %s", err, w.Body.String())
			}

			for key, wantVal := range tt.wantJSON {
				gotVal, ok := resp[key]
				if !ok {
					t.Errorf("response missing key %q", key)
					continue
				}
				// JSON の bool は float64 にならないが、念のため文字列比較
				if gotVal != wantVal {
					t.Errorf("response[%q]: got %v (%T), want %v (%T)", key, gotVal, gotVal, wantVal, wantVal)
				}
			}

			// 追加検証
			if tt.verify != nil {
				tt.verify(t, homeDir)
			}
		})
	}
}
