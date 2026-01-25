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

// DevFolders はスキャン対象のフォルダ一覧です
var DevFolders = []string{
	"実装/実装待ち",
	"実装/完了",
	"検討中",
	"資料",
}

// FileInfo はファイル情報を表します
type FileInfo struct {
	Name string `json:"name"` // ファイル名
	Path string `json:"path"` // 相対パス（開発/から）
}

// FilesResponse は/api/filesレスポンスの構造体です
type FilesResponse struct {
	Success bool                  `json:"success"`         // 成功フラグ
	Files   map[string][]FileInfo `json:"files,omitempty"` // フォルダ別ファイル一覧
	Error   string                `json:"error,omitempty"` // エラーメッセージ
}

// FilesHandler はFiles関連のHTTPハンドラを提供します
type FilesHandler struct{}

// NewFilesHandler は新しいFilesHandlerを生成します
func NewFilesHandler() *FilesHandler {
	return &FilesHandler{}
}

// Handle は開発フォルダ内のmdファイル一覧を取得する。
//
// プロジェクトの「開発」ディレクトリ配下にある指定フォルダをスキャンし、
// 各フォルダ内の .md ファイル一覧を返却する。
// ディレクトリや隠しファイル（.で始まるファイル）はスキップする。
//
// クエリパラメータ:
//   - project: プロジェクトの絶対パス（必須）
//
// レスポンス:
//   - 200: 成功（FilesResponse.Files にフォルダ別ファイル一覧）
//   - 400: projectパラメータ未指定、無効なパス
//   - 404: 開発ディレクトリが存在しない
//   - 500: フォルダ読み取りエラー
func (h *FilesHandler) Handle(c *gin.Context) {
	project := c.Query("project")

	log.Printf("[FilesHandler] Handle started: project=%s", project)

	// プロジェクトパスのバリデーション
	if err := validateProjectPath(project); err != nil {
		log.Printf("[FilesHandler] Handle failed: invalid project path, project=%s, error=%v", project, err)
		c.JSON(http.StatusBadRequest, FilesResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 開発ディレクトリのパス
	devDir := filepath.Join(project, "開発")

	// 開発ディレクトリの存在確認
	info, err := os.Stat(devDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[FilesHandler] Handle failed: dev directory not found, project=%s, error=%v", project, err)
			c.JSON(http.StatusNotFound, FilesResponse{
				Success: false,
				Error:   "開発ディレクトリが存在しません",
			})
			return
		}
		log.Printf("[FilesHandler] Handle failed: failed to check dev directory, project=%s, error=%v", project, err)
		c.JSON(http.StatusInternalServerError, FilesResponse{
			Success: false,
			Error:   "開発ディレクトリの確認に失敗しました",
		})
		return
	}

	if !info.IsDir() {
		log.Printf("[FilesHandler] Handle failed: dev path is not a directory, project=%s", project)
		c.JSON(http.StatusBadRequest, FilesResponse{
			Success: false,
			Error:   "開発パスがディレクトリではありません",
		})
		return
	}

	// ファイル収集
	files := make(map[string][]FileInfo)
	totalFiles := 0

	for _, folder := range DevFolders {
		folderPath := filepath.Join(devDir, folder)

		// フォルダが存在するか確認
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			// フォルダが存在しない場合は空の配列を設定
			files[folder] = []FileInfo{}
			continue
		}

		// フォルダ内のファイルを取得
		entries, err := os.ReadDir(folderPath)
		if err != nil {
			log.Printf("[FilesHandler] Handle failed: failed to read folder, folder=%s, error=%v", folder, err)
			c.JSON(http.StatusInternalServerError, FilesResponse{
				Success: false,
				Error:   "フォルダの読み取りに失敗しました: " + folder,
			})
			return
		}

		fileList := []FileInfo{}
		for _, entry := range entries {
			// ディレクトリはスキップ
			if entry.IsDir() {
				continue
			}

			name := entry.Name()

			// .mdファイルのみ対象
			if !strings.HasSuffix(name, ".md") {
				continue
			}

			// .gitkeepなどの隠しファイルはスキップ
			if strings.HasPrefix(name, ".") {
				continue
			}

			fileList = append(fileList, FileInfo{
				Name: name,
				Path: filepath.Join("開発", folder, name),
			})
		}

		files[folder] = fileList
		totalFiles += len(fileList)
	}

	log.Printf("[FilesHandler] Handle completed: project=%s, folders=%d, files=%d", project, len(DevFolders), totalFiles)

	c.JSON(http.StatusOK, FilesResponse{
		Success: true,
		Files:   files,
	})
}
