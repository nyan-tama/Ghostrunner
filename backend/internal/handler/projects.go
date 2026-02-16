// Package handler はHTTPハンドラーを提供します
package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// ProjectsHandler はプロジェクト一覧関連のHTTPハンドラを提供します
//
// BaseDir はスキャン対象のベースディレクトリ。
// ゼロ値の場合はデフォルトパス /Users/user/ を使用する。
// テスト時にベースディレクトリを差し替え可能にするために公開フィールドとしている。
type ProjectsHandler struct {
	BaseDir string
}

// NewProjectsHandler は新しいProjectsHandlerを生成します
func NewProjectsHandler() *ProjectsHandler {
	return &ProjectsHandler{}
}

// baseDir はスキャン対象のベースディレクトリを返します
func (h *ProjectsHandler) baseDir() string {
	if h.BaseDir != "" {
		return h.BaseDir
	}
	return "/Users/user/"
}

// Handle はベースディレクトリ直下のディレクトリ一覧を取得する。
//
// /Users/user/ 直下のディレクトリをスキャンし、プロジェクト候補として返却する。
// 隠しディレクトリ（.で始まるもの）とファイル、シンボリックリンクはスキップする。
// os.ReadDir はエントリをファイル名のアルファベット順で返すため、追加ソートは不要。
//
// レスポンス:
//   - 200: 成功（ProjectsResponse.Projects にディレクトリ一覧）
//   - 500: ディレクトリ読み取りエラー
func (h *ProjectsHandler) Handle(c *gin.Context) {
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
		// ディレクトリ以外はスキップ（ファイル、シンボリックリンクを除外）
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// 隠しディレクトリはスキップ
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
