package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// mockCreateProjectService は CreateProjectService のモック実装です
type mockCreateProjectService struct {
	validateResult *service.ValidateResult
	openError      error
	projectBaseDir string
}

func (m *mockCreateProjectService) ValidateProjectName(name string) *service.ValidateResult {
	return m.validateResult
}

func (m *mockCreateProjectService) CreateProject(ctx context.Context, req *service.CreateRequest, eventCh chan<- service.CreateEvent) {
	defer close(eventCh)
}

func (m *mockCreateProjectService) OpenInVSCode(path string) error {
	return m.openError
}

func (m *mockCreateProjectService) ProjectBaseDir() string {
	return m.projectBaseDir
}

func TestCreateHandler_HandleValidate(t *testing.T) {
	tests := []struct {
		name           string
		queryName      string
		validateResult *service.ValidateResult
		wantStatus     int
		wantValid      bool
	}{
		{
			name:      "正常なプロジェクト名でvalid=trueが返る",
			queryName: "my-project",
			validateResult: &service.ValidateResult{
				Valid: true,
				Path:  "/tmp/projects/my-project",
			},
			wantStatus: http.StatusOK,
			wantValid:  true,
		},
		{
			name:      "空文字列でvalid=falseが返る",
			queryName: "",
			validateResult: &service.ValidateResult{
				Valid: false,
				Error: "プロジェクト名を入力してください",
			},
			wantStatus: http.StatusOK,
			wantValid:  false,
		},
		{
			name:      "不正な名前でvalid=falseが返る",
			queryName: "My-Project",
			validateResult: &service.ValidateResult{
				Valid: false,
				Error: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）",
			},
			wantStatus: http.StatusOK,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCreateProjectService{
				validateResult: tt.validateResult,
			}
			h := NewCreateHandler(mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/projects/validate?name="+tt.queryName, nil)

			h.HandleValidate(c)

			if w.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", w.Code, tt.wantStatus)
			}

			var resp service.ValidateResult
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v\nbody: %s", err, w.Body.String())
			}

			if resp.Valid != tt.wantValid {
				t.Errorf("valid: got %v, want %v", resp.Valid, tt.wantValid)
			}
		})
	}
}

func TestCreateHandler_HandleOpen(t *testing.T) {
	// テスト用のホームディレクトリ配下に一時ディレクトリを作成
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	// ホームディレクトリ配下にテスト用ディレクトリを作成
	existingDir := filepath.Join(homeDir, ".ghostrunner-test-temp")
	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(existingDir)
	})

	tests := []struct {
		name       string
		body       interface{}
		openError  error
		wantStatus int
		wantError  string
	}{
		{
			name:       "空パスで400が返る",
			body:       OpenRequest{Path: ""},
			wantStatus: http.StatusBadRequest,
			wantError:  "pathは必須です",
		},
		{
			name:       "存在しないパスで404が返る",
			body:       OpenRequest{Path: filepath.Join(homeDir, "nonexistent-ghostrunner-test-dir-12345")},
			wantStatus: http.StatusNotFound,
			wantError:  "指定されたパスが見つかりません",
		},
		{
			name:       "パストラバーサル攻撃_etcpasswdで400が返る",
			body:       OpenRequest{Path: "/etc/passwd"},
			wantStatus: http.StatusBadRequest,
			wantError:  "許可されていないパスです",
		},
		{
			name:       "パストラバーサル攻撃_ルートパスで400が返る",
			body:       OpenRequest{Path: "/"},
			wantStatus: http.StatusBadRequest,
			wantError:  "許可されていないパスです",
		},
		{
			name:       "パストラバーサル攻撃_相対パス混入で400が返る",
			body:       OpenRequest{Path: filepath.Join(homeDir, "..", "etc", "passwd")},
			wantStatus: http.StatusBadRequest,
			wantError:  "許可されていないパスです",
		},
		{
			name:       "不正なJSONで400が返る",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "リクエストが不正です",
		},
		{
			name:       "正常なパスでVS Codeが起動される",
			body:       OpenRequest{Path: existingDir},
			openError:  nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCreateProjectService{
				openError: tt.openError,
			}
			h := NewCreateHandler(mock)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				var marshalErr error
				bodyBytes, marshalErr = json.Marshal(v)
				if marshalErr != nil {
					t.Fatalf("failed to marshal request body: %v", marshalErr)
				}
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/projects/open", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.HandleOpen(c)

			if w.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d\nbody: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantError != "" {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v\nbody: %s", err, w.Body.String())
				}
				gotError, _ := resp["error"].(string)
				if gotError != tt.wantError {
					t.Errorf("error: got %q, want %q", gotError, tt.wantError)
				}
			}
		})
	}
}

func TestCreateHandler_HandleCreateStream_InvalidJSON(t *testing.T) {
	mock := &mockCreateProjectService{
		validateResult: &service.ValidateResult{Valid: true, Path: "/tmp/test"},
	}
	h := NewCreateHandler(mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/projects/create/stream", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleCreateStream(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v\nbody: %s", err, w.Body.String())
	}

	gotError, _ := resp["error"].(string)
	if gotError != "リクエストが不正です" {
		t.Errorf("error: got %q, want %q", gotError, "リクエストが不正です")
	}
}

func TestCreateHandler_HandleCreateStream_InvalidService(t *testing.T) {
	mock := &mockCreateProjectService{
		validateResult: &service.ValidateResult{Valid: true, Path: "/tmp/test"},
	}
	h := NewCreateHandler(mock)

	reqBody := CreateStreamRequest{
		Name:     "my-project",
		Services: []string{"invalid-service"},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/projects/create/stream", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleCreateStream(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if jsonErr := json.Unmarshal(w.Body.Bytes(), &resp); jsonErr != nil {
		t.Fatalf("failed to unmarshal response: %v\nbody: %s", jsonErr, w.Body.String())
	}

	gotError, _ := resp["error"].(string)
	wantContain := "invalid-service"
	if gotError == "" {
		t.Error("error should not be empty")
	}
	if !bytes.Contains([]byte(gotError), []byte(wantContain)) {
		t.Errorf("error %q should contain %q", gotError, wantContain)
	}
}

func TestCreateHandler_HandleOpen_VSCodeError(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	existingDir := filepath.Join(homeDir, ".ghostrunner-test-vscode-err")
	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(existingDir)
	})

	mock := &mockCreateProjectService{
		openError: fmt.Errorf("code command not found"),
	}
	h := NewCreateHandler(mock)

	bodyBytes, err := json.Marshal(OpenRequest{Path: existingDir})
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/projects/open", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleOpen(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code: got %d, want %d\nbody: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	var resp map[string]interface{}
	if jsonErr := json.Unmarshal(w.Body.Bytes(), &resp); jsonErr != nil {
		t.Fatalf("failed to unmarshal response: %v\nbody: %s", jsonErr, w.Body.String())
	}

	gotError, _ := resp["error"].(string)
	if gotError != "VS Codeの起動に失敗しました" {
		t.Errorf("error: got %q, want %q", gotError, "VS Codeの起動に失敗しました")
	}
}

func TestValidateServices(t *testing.T) {
	tests := []struct {
		name     string
		services []string
		wantErr  bool
	}{
		{
			name:     "有効なサービスのみ",
			services: []string{"database", "storage", "cache"},
			wantErr:  false,
		},
		{
			name:     "空スライス",
			services: []string{},
			wantErr:  false,
		},
		{
			name:     "nil",
			services: nil,
			wantErr:  false,
		},
		{
			name:     "不明なサービスを含む",
			services: []string{"database", "unknown"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServices(tt.services)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateServices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
