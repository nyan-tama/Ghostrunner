package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// mockPatrolService はテスト用のPatrolServiceモックです
type mockPatrolService struct {
	registerProjectFunc   func(path string) error
	unregisterProjectFunc func(path string) error
	listProjectsFunc      func() []service.PatrolProject
	scanProjectsFunc      func() []service.ScanResult
	startPatrolFunc       func() error
	stopPatrolFunc        func()
	resumeProjectFunc     func(projectPath, answer string) error
	getStatesFunc         func() map[string]*service.ProjectState
	startPollingFunc      func()
	stopPollingFunc       func()
	subscribeFunc         func() (<-chan service.PatrolEvent, func())
}

func (m *mockPatrolService) RegisterProject(path string) error {
	if m.registerProjectFunc != nil {
		return m.registerProjectFunc(path)
	}
	return nil
}

func (m *mockPatrolService) UnregisterProject(path string) error {
	if m.unregisterProjectFunc != nil {
		return m.unregisterProjectFunc(path)
	}
	return nil
}

func (m *mockPatrolService) ListProjects() []service.PatrolProject {
	if m.listProjectsFunc != nil {
		return m.listProjectsFunc()
	}
	return nil
}

func (m *mockPatrolService) ScanProjects() []service.ScanResult {
	if m.scanProjectsFunc != nil {
		return m.scanProjectsFunc()
	}
	return nil
}

func (m *mockPatrolService) StartPatrol() error {
	if m.startPatrolFunc != nil {
		return m.startPatrolFunc()
	}
	return nil
}

func (m *mockPatrolService) StopPatrol() {
	if m.stopPatrolFunc != nil {
		m.stopPatrolFunc()
	}
}

func (m *mockPatrolService) ResumeProject(projectPath, answer string) error {
	if m.resumeProjectFunc != nil {
		return m.resumeProjectFunc(projectPath, answer)
	}
	return nil
}

func (m *mockPatrolService) GetStates() map[string]*service.ProjectState {
	if m.getStatesFunc != nil {
		return m.getStatesFunc()
	}
	return nil
}

func (m *mockPatrolService) StartPolling() {
	if m.startPollingFunc != nil {
		m.startPollingFunc()
	}
}

func (m *mockPatrolService) StopPolling() {
	if m.stopPollingFunc != nil {
		m.stopPollingFunc()
	}
}

func (m *mockPatrolService) Subscribe() (<-chan service.PatrolEvent, func()) {
	if m.subscribeFunc != nil {
		return m.subscribeFunc()
	}
	ch := make(chan service.PatrolEvent)
	close(ch)
	return ch, func() {}
}

// --- HandleRegister テスト ---

func TestPatrolHandler_HandleRegister(t *testing.T) {
	tests := []struct {
		name        string
		body        interface{}
		mockSetup   func(m *mockPatrolService)
		wantStatus  int
		wantSuccess bool
		wantError   string
	}{
		{
			name: "正常登録",
			body: PatrolRegisterRequest{Path: "/tmp/test-project"},
			mockSetup: func(m *mockPatrolService) {
				m.registerProjectFunc = func(_ string) error { return nil }
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "エラー_パス未指定",
			body:        PatrolRegisterRequest{Path: ""},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			wantError:   "pathは必須です",
		},
		{
			name:        "エラー_不正なJSON",
			body:        "invalid",
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			wantError:   "リクエストが不正です",
		},
		{
			name: "エラー_サービスエラー",
			body: PatrolRegisterRequest{Path: "/nonexistent"},
			mockSetup: func(m *mockPatrolService) {
				m.registerProjectFunc = func(_ string) error {
					return errTest("project already registered")
				}
			},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			wantError:   "project already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPatrolService{}
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}
			h := NewPatrolHandler(mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatalf("failed to marshal body: %v", err)
			}
			c.Request = httptest.NewRequest(http.MethodPost, "/api/patrol/projects", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.HandleRegister(c)

			assertPatrolResponse(t, w, tt.wantStatus, tt.wantSuccess, tt.wantError)
		})
	}
}

// --- HandleRemove テスト ---

func TestPatrolHandler_HandleRemove(t *testing.T) {
	tests := []struct {
		name        string
		body        interface{}
		mockSetup   func(m *mockPatrolService)
		wantStatus  int
		wantSuccess bool
		wantError   string
	}{
		{
			name: "正常解除",
			body: PatrolRemoveRequest{Path: "/tmp/test-project"},
			mockSetup: func(m *mockPatrolService) {
				m.unregisterProjectFunc = func(_ string) error { return nil }
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "エラー_パス未指定",
			body:        PatrolRemoveRequest{Path: ""},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			wantError:   "pathは必須です",
		},
		{
			name: "エラー_未登録プロジェクト",
			body: PatrolRemoveRequest{Path: "/nonexistent"},
			mockSetup: func(m *mockPatrolService) {
				m.unregisterProjectFunc = func(_ string) error {
					return errTest("project not registered")
				}
			},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			wantError:   "project not registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPatrolService{}
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}
			h := NewPatrolHandler(mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/patrol/projects/remove", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.HandleRemove(c)

			assertPatrolResponse(t, w, tt.wantStatus, tt.wantSuccess, tt.wantError)
		})
	}
}

// --- HandleListProjects テスト ---

func TestPatrolHandler_HandleListProjects(t *testing.T) {
	tests := []struct {
		name         string
		mockProjects []service.PatrolProject
		wantStatus   int
		wantCount    int
	}{
		{
			name: "プロジェクト一覧を返す",
			mockProjects: []service.PatrolProject{
				{Path: "/project/a", Name: "a"},
				{Path: "/project/b", Name: "b"},
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:         "空のプロジェクト一覧",
			mockProjects: nil,
			wantStatus:   http.StatusOK,
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPatrolService{
				listProjectsFunc: func() []service.PatrolProject {
					return tt.mockProjects
				},
			}
			h := NewPatrolHandler(mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/patrol/projects", nil)

			h.HandleListProjects(c)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tt.wantStatus)
			}

			var resp PatrolProjectsResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if !resp.Success {
				t.Error("expected success=true")
			}
			if len(resp.Projects) != tt.wantCount {
				t.Errorf("projects count: got %d, want %d", len(resp.Projects), tt.wantCount)
			}
		})
	}
}

// --- HandleStart テスト ---

func TestPatrolHandler_HandleStart(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(m *mockPatrolService)
		wantStatus  int
		wantSuccess bool
		wantError   string
	}{
		{
			name: "正常開始",
			mockSetup: func(m *mockPatrolService) {
				m.startPatrolFunc = func() error { return nil }
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "エラー_既に実行中",
			mockSetup: func(m *mockPatrolService) {
				m.startPatrolFunc = func() error {
					return errTest("patrol is already running")
				}
			},
			wantStatus:  http.StatusConflict,
			wantSuccess: false,
			wantError:   "patrol is already running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPatrolService{}
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}
			h := NewPatrolHandler(mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/patrol/start", nil)

			h.HandleStart(c)

			assertPatrolResponse(t, w, tt.wantStatus, tt.wantSuccess, tt.wantError)
		})
	}
}

// --- HandleStop テスト ---

func TestPatrolHandler_HandleStop(t *testing.T) {
	t.Run("正常停止", func(t *testing.T) {
		stopCalled := false
		mock := &mockPatrolService{
			stopPatrolFunc: func() { stopCalled = true },
		}
		h := NewPatrolHandler(mock)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/patrol/stop", nil)

		h.HandleStop(c)

		if w.Code != http.StatusOK {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
		}
		if !stopCalled {
			t.Error("StopPatrol was not called")
		}
	})
}

// --- HandleResume テスト ---

func TestPatrolHandler_HandleResume(t *testing.T) {
	tests := []struct {
		name        string
		body        interface{}
		mockSetup   func(m *mockPatrolService)
		wantStatus  int
		wantSuccess bool
		wantError   string
	}{
		{
			name: "正常再開",
			body: PatrolResumeRequest{ProjectPath: "/project/a", Answer: "yes"},
			mockSetup: func(m *mockPatrolService) {
				m.resumeProjectFunc = func(_, _ string) error { return nil }
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "エラー_projectPath未指定",
			body:        PatrolResumeRequest{ProjectPath: "", Answer: "yes"},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			wantError:   "projectPathは必須です",
		},
		{
			name:        "エラー_answer未指定",
			body:        PatrolResumeRequest{ProjectPath: "/project/a", Answer: ""},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			wantError:   "answerは必須です",
		},
		{
			name: "エラー_サービスエラー",
			body: PatrolResumeRequest{ProjectPath: "/project/a", Answer: "yes"},
			mockSetup: func(m *mockPatrolService) {
				m.resumeProjectFunc = func(_, _ string) error {
					return errTest("project is not waiting for approval")
				}
			},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			wantError:   "project is not waiting for approval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockPatrolService{}
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}
			h := NewPatrolHandler(mock)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/patrol/resume", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.HandleResume(c)

			assertPatrolResponse(t, w, tt.wantStatus, tt.wantSuccess, tt.wantError)
		})
	}
}

// --- HandleStates テスト ---

func TestPatrolHandler_HandleStates(t *testing.T) {
	t.Run("プロジェクト状態一覧を返す", func(t *testing.T) {
		mock := &mockPatrolService{
			getStatesFunc: func() map[string]*service.ProjectState {
				return map[string]*service.ProjectState{
					"/project/a": {
						Project: service.PatrolProject{Path: "/project/a", Name: "a"},
						Status:  service.StatusRunning,
					},
				}
			},
		}
		h := NewPatrolHandler(mock)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/patrol/states", nil)

		h.HandleStates(c)

		if w.Code != http.StatusOK {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
		}

		var resp PatrolStatesResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if !resp.Success {
			t.Error("expected success=true")
		}
		if len(resp.States) != 1 {
			t.Errorf("states count: got %d, want 1", len(resp.States))
		}
		if state, ok := resp.States["/project/a"]; !ok {
			t.Error("expected state for /project/a")
		} else if state.Status != service.StatusRunning {
			t.Errorf("status: got %q, want %q", state.Status, service.StatusRunning)
		}
	})
}

// --- HandleScan テスト ---

func TestPatrolHandler_HandleScan(t *testing.T) {
	t.Run("スキャン結果を返す", func(t *testing.T) {
		mock := &mockPatrolService{
			scanProjectsFunc: func() []service.ScanResult {
				return []service.ScanResult{
					{
						Project:      service.PatrolProject{Path: "/project/a", Name: "a"},
						PendingTasks: []string{"task1.md"},
					},
				}
			},
		}
		h := NewPatrolHandler(mock)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/patrol/scan", nil)

		h.HandleScan(c)

		if w.Code != http.StatusOK {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
		}

		var resp PatrolScanResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if !resp.Success {
			t.Error("expected success=true")
		}
		if len(resp.Results) != 1 {
			t.Fatalf("results count: got %d, want 1", len(resp.Results))
		}
		if len(resp.Results[0].PendingTasks) != 1 {
			t.Errorf("pending tasks: got %d, want 1", len(resp.Results[0].PendingTasks))
		}
	})
}

// --- HandlePollingStart / HandlePollingStop テスト ---

func TestPatrolHandler_HandlePollingStartStop(t *testing.T) {
	t.Run("ポーリング開始", func(t *testing.T) {
		startCalled := false
		mock := &mockPatrolService{
			startPollingFunc: func() { startCalled = true },
		}
		h := NewPatrolHandler(mock)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/patrol/polling/start", nil)

		h.HandlePollingStart(c)

		if w.Code != http.StatusOK {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
		}
		if !startCalled {
			t.Error("StartPolling was not called")
		}
	})

	t.Run("ポーリング停止", func(t *testing.T) {
		stopCalled := false
		mock := &mockPatrolService{
			stopPollingFunc: func() { stopCalled = true },
		}
		h := NewPatrolHandler(mock)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/patrol/polling/stop", nil)

		h.HandlePollingStop(c)

		if w.Code != http.StatusOK {
			t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
		}
		if !stopCalled {
			t.Error("StopPolling was not called")
		}
	})
}

// --- ヘルパー関数 ---

// errTest はテスト用の簡易エラー型です
type errTest string

func (e errTest) Error() string { return string(e) }

// assertPatrolResponse はPatrolResponseの共通検証を行います
func assertPatrolResponse(t *testing.T, w *httptest.ResponseRecorder, wantStatus int, wantSuccess bool, wantError string) {
	t.Helper()

	if w.Code != wantStatus {
		t.Errorf("status: got %d, want %d", w.Code, wantStatus)
	}

	var resp PatrolResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v\nbody: %s", err, w.Body.String())
	}

	if resp.Success != wantSuccess {
		t.Errorf("success: got %v, want %v", resp.Success, wantSuccess)
	}

	if wantError != "" && resp.Error != wantError {
		t.Errorf("error: got %q, want %q", resp.Error, wantError)
	}
}
