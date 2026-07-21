// Package handler はHTTPハンドラーを提供します
package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// patrolConfig は patrol_projects.json のトップレベル構造
type patrolConfig struct {
	Projects []patrolProject `json:"projects"`
}

// patrolProject は patrol_projects.json 内の各プロジェクト定義
type patrolProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ProjectInfo はプロジェクトディレクトリの情報を表します
type ProjectInfo struct {
	Name string `json:"name"` // ディレクトリ名
	Path string `json:"path"` // 絶対パス
}

// ProjectsResponse は/api/projectsレスポンスの構造体です
type ProjectsResponse struct {
	Success  bool          `json:"success"`            // 成功フラグ
	Projects []ProjectInfo `json:"projects,omitempty"` // プロジェクト一覧
	Error    string        `json:"error,omitempty"`    // エラーメッセージ
}

// DestroyRequest はプロジェクト削除リクエストの構造体です
type DestroyRequest struct {
	Path string `json:"path"`
}

// ProjectsHandler はプロジェクト一覧関連のHTTPハンドラを提供します
//
// BaseDir はスキャン対象のベースディレクトリ。
// ゼロ値の場合はデフォルトパス /Users/user/ を使用する。
// HomeDir はプロジェクト削除時のパス制限に使用するホームディレクトリ。
// テスト時にディレクトリを差し替え可能にするために公開フィールドとしている。
type ProjectsHandler struct {
	BaseDir          string
	HomeDir          string
	PatrolConfigPath string
}

// NewProjectsHandler は新しいProjectsHandlerを生成します。
// patrolConfigPath が指定された場合、Handle は patrol_projects.json に
// 登録されたプロジェクトのみを返す。空文字の場合は従来どおり全ディレクトリをスキャンする。
func NewProjectsHandler(patrolConfigPath string) *ProjectsHandler {
	return &ProjectsHandler{
		PatrolConfigPath: patrolConfigPath,
	}
}

// baseDir はスキャン対象のベースディレクトリを返します
func (h *ProjectsHandler) baseDir() string {
	if h.BaseDir != "" {
		return h.BaseDir
	}
	return "/Users/user/"
}

// homeDir はプロジェクト削除時のパス制限に使用するホームディレクトリを返します
func (h *ProjectsHandler) homeDir() string {
	if h.HomeDir != "" {
		return h.HomeDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/Users/user"
	}
	return home
}

// HandleDestroy はプロジェクトディレクトリを削除します。
//
// ホームディレクトリ直下のディレクトリのみ削除を許可する。
// docker-compose.yml が存在する場合は docker compose down -v を実行してから削除する。
// docker compose の実行に失敗してもディレクトリ削除は続行する。
//
// POST /api/projects/destroy
//
// レスポンス:
//   - 200: 削除成功
//   - 400: リクエスト不正（パス未指定、パストラバーサル検出）
//   - 404: 対象ディレクトリが存在しない
//   - 500: 削除失敗
func (h *ProjectsHandler) HandleDestroy(c *gin.Context) {
	var req DestroyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストが不正です",
		})
		return
	}

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "パスが指定されていません",
		})
		return
	}

	cleanPath := filepath.Clean(req.Path)
	home := h.homeDir()

	log.Printf("[ProjectsHandler] HandleDestroy started: path=%s, cleanPath=%s, homeDir=%s", req.Path, cleanPath, home)

	// パストラバーサル防止: ホームディレクトリ直下のみ許可
	if filepath.Dir(cleanPath) != home {
		log.Printf("[ProjectsHandler] HandleDestroy rejected: path is not directly under home directory, cleanPath=%s, homeDir=%s", cleanPath, home)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "ホームディレクトリ直下のプロジェクトのみ削除できます",
		})
		return
	}

	// 対象ディレクトリの存在チェック
	info, err := os.Stat(cleanPath)
	if os.IsNotExist(err) {
		log.Printf("[ProjectsHandler] HandleDestroy failed: directory not found, path=%s", cleanPath)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "指定されたディレクトリが見つかりません",
		})
		return
	}
	if err != nil {
		log.Printf("[ProjectsHandler] HandleDestroy failed: failed to stat directory, path=%s, error=%v", cleanPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "ディレクトリの確認に失敗しました",
		})
		return
	}
	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "指定されたパスはディレクトリではありません",
		})
		return
	}

	// docker-compose.yml が存在する場合は docker compose down -v を実行
	composePath := filepath.Join(cleanPath, "docker-compose.yml")
	if _, err := os.Stat(composePath); err == nil {
		log.Printf("[ProjectsHandler] HandleDestroy: docker-compose.yml found, running docker compose down -v, path=%s", cleanPath)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "docker", "compose", "down", "-v")
		cmd.Dir = cleanPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("[ProjectsHandler] HandleDestroy: docker compose down -v failed (continuing with deletion), path=%s, error=%v, output=%s", cleanPath, err, string(output))
		} else {
			log.Printf("[ProjectsHandler] HandleDestroy: docker compose down -v completed, path=%s", cleanPath)
		}
	}

	// ディレクトリ削除
	if err := os.RemoveAll(cleanPath); err != nil {
		log.Printf("[ProjectsHandler] HandleDestroy failed: failed to remove directory, path=%s, error=%v", cleanPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "ディレクトリの削除に失敗しました",
		})
		return
	}

	log.Printf("[ProjectsHandler] HandleDestroy completed: path=%s", cleanPath)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// Handle はプロジェクト一覧を取得する。
//
// PatrolConfigPath が設定されている場合は patrol_projects.json に登録された
// プロジェクトのみを返す。未設定の場合はベースディレクトリ直下の全ディレクトリを返す（後方互換）。
//
// レスポンス:
//   - 200: 成功（ProjectsResponse.Projects にプロジェクト一覧）
//   - 500: 読み取りエラー
func (h *ProjectsHandler) Handle(c *gin.Context) {
	// patrol_projects.json が指定されていればそこから読む
	if h.PatrolConfigPath != "" {
		h.handleFromPatrolConfig(c)
		return
	}

	// フォールバック: ベースディレクトリ全スキャン（後方互換）
	h.handleFromDirectory(c)
}

// handleFromPatrolConfig は patrol_projects.json に登録されたプロジェクトを返す
func (h *ProjectsHandler) handleFromPatrolConfig(c *gin.Context) {
	log.Printf("[ProjectsHandler] Handle started: patrolConfig=%s", h.PatrolConfigPath)

	data, err := os.ReadFile(h.PatrolConfigPath)
	if err != nil {
		log.Printf("[ProjectsHandler] Handle failed: failed to read patrol config, path=%s, error=%v", h.PatrolConfigPath, err)
		// フォールバック: ディレクトリスキャン
		h.handleFromDirectory(c)
		return
	}

	var config patrolConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("[ProjectsHandler] Handle failed: failed to parse patrol config, error=%v", err)
		h.handleFromDirectory(c)
		return
	}

	var projects []ProjectInfo
	for _, p := range config.Projects {
		projects = append(projects, ProjectInfo{
			Name: p.Name,
			Path: p.Path,
		})
	}

	log.Printf("[ProjectsHandler] Handle completed: patrolConfig=%s, projects=%d", h.PatrolConfigPath, len(projects))

	c.JSON(http.StatusOK, ProjectsResponse{
		Success:  true,
		Projects: projects,
	})
}

// handleFromDirectory はベースディレクトリ直下のディレクトリ一覧を返す（後方互換）
func (h *ProjectsHandler) handleFromDirectory(c *gin.Context) {
	dir := h.baseDir()

	log.Printf("[ProjectsHandler] Handle started: baseDir=%s", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[ProjectsHandler] Handle failed: failed to read directory, baseDir=%s, error=%v", dir, err)
		c.JSON(http.StatusInternalServerError, ProjectsResponse{
			Success: false,
			Error:   "ディレクトリ一覧の取得に失敗しました",
		})
		return
	}

	var projects []ProjectInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		if strings.HasPrefix(name, ".") {
			continue
		}

		projects = append(projects, ProjectInfo{
			Name: name,
			Path: filepath.Join(dir, name),
		})
	}

	log.Printf("[ProjectsHandler] Handle completed: baseDir=%s, projects=%d", dir, len(projects))

	c.JSON(http.StatusOK, ProjectsResponse{
		Success:  true,
		Projects: projects,
	})
}
