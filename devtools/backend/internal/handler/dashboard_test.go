package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ghostrunner/backend/internal/dashboard"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockDashboardService はテスト用のdashboard.Serviceモックです
type mockDashboardService struct {
	getStateFunc func(ctx context.Context) (dashboard.State, error)
	answerFunc   func(ctx context.Context, req dashboard.AnswerRequest) error
}

func (m *mockDashboardService) GetState(ctx context.Context) (dashboard.State, error) {
	if m.getStateFunc != nil {
		return m.getStateFunc(ctx)
	}
	return dashboard.State{}, nil
}

func (m *mockDashboardService) Answer(ctx context.Context, req dashboard.AnswerRequest) error {
	if m.answerFunc != nil {
		return m.answerFunc(ctx, req)
	}
	return nil
}

func setupDashboardRouter(svc dashboard.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewDashboardHandler(svc)
	r.GET("/api/dashboard/state", h.HandleState)
	r.POST("/api/dashboard/answer", h.HandleAnswer)
	return r
}

func TestDashboardHandler_HandleState_200(t *testing.T) {
	mock := &mockDashboardService{
		getStateFunc: func(ctx context.Context) (dashboard.State, error) {
			return dashboard.State{
				Projects: []dashboard.ProjectState{
					{
						Name:       "project-a",
						Path:       "/path/to/a",
						Attention:  dashboard.AttentionProgress,
						Unanswered: []dashboard.UnansweredQuestion{},
						Ops:        []dashboard.OpsEntry{},
						Warnings:   []string{},
					},
				},
				GeneratedAt: "2026-05-26T12:00:00+09:00",
			}, nil
		},
	}

	r := setupDashboardRouter(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/dashboard/state", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dashboard.State
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Projects))
	assert.Equal(t, "project-a", resp.Projects[0].Name)
	assert.Equal(t, "2026-05-26T12:00:00+09:00", resp.GeneratedAt)
}

func TestDashboardHandler_HandleState_500(t *testing.T) {
	mock := &mockDashboardService{
		getStateFunc: func(ctx context.Context) (dashboard.State, error) {
			return dashboard.State{}, fmt.Errorf("internal error")
		},
	}

	r := setupDashboardRouter(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/dashboard/state", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
	assert.NotEmpty(t, resp["error"])
}

func TestDashboardHandler_HandleAnswer_200(t *testing.T) {
	mock := &mockDashboardService{
		answerFunc: func(ctx context.Context, req dashboard.AnswerRequest) error {
			return nil
		},
	}

	body, _ := json.Marshal(dashboard.AnswerRequest{
		ProjectPath: "/path/to/project",
		PlanPath:    "plan.md",
		LineStart:   10,
		Answer:      "A案で",
	})

	r := setupDashboardRouter(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/dashboard/answer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
}

func TestDashboardHandler_HandleAnswer_400_Validation(t *testing.T) {
	mock := &mockDashboardService{
		answerFunc: func(ctx context.Context, req dashboard.AnswerRequest) error {
			return fmt.Errorf("%w: answer is empty", dashboard.ErrValidation)
		},
	}

	body, _ := json.Marshal(dashboard.AnswerRequest{
		ProjectPath: "/path/to/project",
		PlanPath:    "plan.md",
		LineStart:   10,
		Answer:      "",
	})

	r := setupDashboardRouter(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/dashboard/answer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}

func TestDashboardHandler_HandleAnswer_409_AlreadyAnswered(t *testing.T) {
	mock := &mockDashboardService{
		answerFunc: func(ctx context.Context, req dashboard.AnswerRequest) error {
			return fmt.Errorf("%w: no unanswered status in window", dashboard.ErrAlreadyAnswered)
		},
	}

	body, _ := json.Marshal(dashboard.AnswerRequest{
		ProjectPath: "/path/to/project",
		PlanPath:    "plan.md",
		LineStart:   10,
		Answer:      "B案で",
	})

	r := setupDashboardRouter(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/dashboard/answer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}

func TestDashboardHandler_HandleAnswer_400_InvalidJSON(t *testing.T) {
	r := setupDashboardRouter(&mockDashboardService{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/dashboard/answer", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}
