// Package service はビジネスロジックを提供します
package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

// CreateEvent はプロジェクト生成の進捗イベントを表します
type CreateEvent struct {
	Type     string `json:"type"`            // "progress" | "complete" | "error"
	Step     string `json:"step"`            // ステップID
	Message  string `json:"message"`         // 表示メッセージ
	Progress int    `json:"progress"`        // 進捗率 (0-100)
	Path     string `json:"path,omitempty"`  // 生成されたプロジェクトパス（completeのみ）
	Error    string `json:"error,omitempty"` // エラーメッセージ（errorのみ）
}

// CreateRequest はプロジェクト生成リクエストを表します
type CreateRequest struct {
	Name        string   `json:"name"`        // プロジェクト名
	Description string   `json:"description"` // プロジェクト概要
	Services    []string `json:"services"`    // 選択されたサービス ("database", "storage", "cache")
}

// ValidateResult はバリデーション結果を表します
type ValidateResult struct {
	Valid bool   `json:"valid"`           // バリデーション成功かどうか
	Path  string `json:"path,omitempty"`  // 生成先パス
	Error string `json:"error,omitempty"` // エラーメッセージ
}

// projectNameRegexp はプロジェクト名のバリデーション用正規表現
var projectNameRegexp = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// AllowedServices はプロジェクト生成で許可されるサービス名のリストです
var AllowedServices = map[string]bool{
	"database": true,
	"storage":  true,
	"cache":    true,
}

// CreateProjectService はプロジェクト生成のインターフェースです
type CreateProjectService interface {
	ValidateProjectName(name string) *ValidateResult
	CreateProject(ctx context.Context, req *CreateRequest, eventCh chan<- CreateEvent)
	OpenInVSCode(path string) error
	ProjectBaseDir() string
}

// CreateService はプロジェクト生成のオーケストレーションを担当します
type CreateService struct {
	templateService *TemplateService
	projectBaseDir  string
}

// NewCreateService は新しいCreateServiceを生成します
func NewCreateService(templateService *TemplateService, projectBaseDir string) *CreateService {
	return &CreateService{
		templateService: templateService,
		projectBaseDir:  projectBaseDir,
	}
}

// ProjectBaseDir はプロジェクト生成先のベースディレクトリを返します
func (s *CreateService) ProjectBaseDir() string {
	return s.projectBaseDir
}

// ValidateProjectName はプロジェクト名をバリデーションします
func (s *CreateService) ValidateProjectName(name string) *ValidateResult {
	log.Printf("[CreateService] ValidateProjectName started: name=%s", name)

	if name == "" {
		return &ValidateResult{Valid: false, Error: "プロジェクト名を入力してください"}
	}

	if !projectNameRegexp.MatchString(name) {
		return &ValidateResult{Valid: false, Error: "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）"}
	}

	projectPath := filepath.Join(s.projectBaseDir, name)

	if _, err := os.Stat(projectPath); err == nil {
		return &ValidateResult{Valid: false, Path: projectPath, Error: "同名のディレクトリが既に存在します"}
	}

	log.Printf("[CreateService] ValidateProjectName completed: name=%s, path=%s", name, projectPath)
	return &ValidateResult{Valid: true, Path: projectPath}
}

// CreateProject はプロジェクトを生成します（各ステップの進捗をeventChに送信）
func (s *CreateService) CreateProject(ctx context.Context, req *CreateRequest, eventCh chan<- CreateEvent) {
	defer close(eventCh)

	projectPath := filepath.Join(s.projectBaseDir, req.Name)

	log.Printf("[CreateService] CreateProject started: name=%s, path=%s, services=%v", req.Name, projectPath, req.Services)

	// ステップ定義
	type step struct {
		id      string
		message string
		pct     int
		run     func() error
	}

	steps := []step{
		{"template_copy", "テンプレートをコピー中...", 10, func() error { return s.stepTemplateCopy(projectPath, req.Services) }},
		{"placeholder_replace", "プロジェクト名を設定中...", 20, func() error { return s.templateService.ReplacePlaceholders(projectPath, req.Name) }},
		{"env_create", "環境設定ファイルを作成中...", 30, func() error { return s.templateService.CreateEnvFile(projectPath, req.Name, req.Services) }},
		{"dependency_install", "依存パッケージをインストール中...", 40, func() error { return s.stepDependencyInstall(ctx, projectPath) }},
		{"claude_assets", "開発支援ツールを設定中...", 50, func() error { return s.stepClaudeAssets(projectPath, req.Services) }},
		{"claude_md", "プロジェクト設定を生成中...", 60, func() error { return s.templateService.GenerateClaudeMD(projectPath, req.Name, req.Description, req.Services) }},
		{"devtools_link", "devtools を接続中...", 70, func() error { return s.templateService.CreateDevtoolsLink(projectPath) }},
		{"git_init", "バージョン管理を初期化中...", 80, func() error { return s.stepGitInit(ctx, projectPath) }},
		{"server_start", "サーバーを起動中...", 90, func() error { return s.stepServerStart(ctx, projectPath) }},
		{"health_check", "動作確認中...", 95, func() error { return s.stepHealthCheck(ctx) }},
	}

	for _, st := range steps {
		// クライアント切断チェック
		if ctx.Err() != nil {
			s.sendError(eventCh, st.id, fmt.Errorf("canceled: %w", ctx.Err()))
			return
		}
		s.sendProgress(eventCh, st.id, st.message, st.pct)
		if err := st.run(); err != nil {
			s.sendError(eventCh, st.id, err)
			return
		}
	}

	// 完了
	log.Printf("[CreateService] CreateProject completed: name=%s, path=%s", req.Name, projectPath)
	eventCh <- CreateEvent{
		Type:     "complete",
		Step:     "done",
		Message:  "プロジェクトの作成が完了しました",
		Progress: 100,
		Path:     projectPath,
	}
}

// stepTemplateCopy はbaseテンプレートとオプションサービスのテンプレートをコピーします
func (s *CreateService) stepTemplateCopy(projectPath string, services []string) error {
	// baseテンプレートをコピー
	if err := s.templateService.CopyBase(projectPath); err != nil {
		return fmt.Errorf("failed to copy base template: %w", err)
	}

	// オプションサービスのテンプレートをコピー
	for _, svc := range services {
		if err := s.templateService.CopyServiceTemplate(projectPath, svc); err != nil {
			return fmt.Errorf("failed to copy service template %s: %w", svc, err)
		}
	}

	// docker-compose.yml のマージ（サービスが選択されている場合）
	if len(services) > 0 {
		if err := s.templateService.MergeDockerCompose(projectPath, services); err != nil {
			return fmt.Errorf("failed to merge docker-compose: %w", err)
		}
	}

	return nil
}

// stepDependencyInstall は依存パッケージをインストールします
func (s *CreateService) stepDependencyInstall(ctx context.Context, projectPath string) error {
	// go mod tidy
	goCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	goCmd.Dir = filepath.Join(projectPath, "backend")
	if output, err := goCmd.CombinedOutput(); err != nil {
		log.Printf("[CreateService] go mod tidy failed: output=%s, error=%v", string(output), err)
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	// npm install
	npmCmd := exec.CommandContext(ctx, "npm", "install")
	npmCmd.Dir = filepath.Join(projectPath, "frontend")
	if output, err := npmCmd.CombinedOutput(); err != nil {
		log.Printf("[CreateService] npm install failed: output=%s, error=%v", string(output), err)
		return fmt.Errorf("npm install failed: %w", err)
	}

	return nil
}

// stepClaudeAssets は.claude/資産をコピーし、不要なエージェントを削除します
func (s *CreateService) stepClaudeAssets(projectPath string, services []string) error {
	if err := s.templateService.CopyClaudeAssets(projectPath); err != nil {
		return fmt.Errorf("failed to copy claude assets: %w", err)
	}

	if err := s.templateService.RemoveUnusedAgents(projectPath, services); err != nil {
		return fmt.Errorf("failed to remove unused agents: %w", err)
	}

	return nil
}

// stepGitInit はgitリポジトリを初期化してコミットします
func (s *CreateService) stepGitInit(ctx context.Context, projectPath string) error {
	commands := []struct {
		name string
		args []string
	}{
		{"git init", []string{"init"}},
		{"git add", []string{"add", "-A"}},
		{"git commit", []string{"commit", "-m", "feat: プロジェクト初期構築（Ghostrunnerで生成）"}},
	}

	for _, cmd := range commands {
		gitCmd := exec.CommandContext(ctx, "git", cmd.args...)
		gitCmd.Dir = projectPath
		if output, err := gitCmd.CombinedOutput(); err != nil {
			log.Printf("[CreateService] %s failed: output=%s, error=%v", cmd.name, string(output), err)
			return fmt.Errorf("%s failed: %w", cmd.name, err)
		}
	}

	return nil
}

// stepServerStart はMakefileでバックエンドとフロントエンドを起動します
func (s *CreateService) stepServerStart(ctx context.Context, projectPath string) error {
	// バックエンドをバックグラウンドで起動
	backendCmd := exec.CommandContext(ctx, "make", "start-backend")
	backendCmd.Dir = projectPath
	// start-backend はバックグラウンドで起動するため、出力は /tmp/backend.log に書かれる
	// tail -f が残るが、ここではそれを待たずに完了扱いにする
	if err := backendCmd.Start(); err != nil {
		log.Printf("[CreateService] make start-backend failed: error=%v", err)
		return fmt.Errorf("make start-backend failed: %w", err)
	}
	go backendCmd.Wait()

	// start-backend の完了を少し待つ（バックグラウンドプロセス起動まで）
	select {
	case <-ctx.Done():
		return fmt.Errorf("server start canceled: %w", ctx.Err())
	case <-time.After(3 * time.Second):
		// 起動完了見込み
	}

	return nil
}

// stepHealthCheck はバックエンドのヘルスチェックをポーリングします
func (s *CreateService) stepHealthCheck(ctx context.Context) error {
	client := &http.Client{Timeout: 2 * time.Second}
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get("http://localhost:8080/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Printf("[CreateService] Health check passed: attempt=%d", i+1)
				return nil
			}
		}

		log.Printf("[CreateService] Health check attempt %d/%d failed: error=%v", i+1, maxRetries, err)

		select {
		case <-ctx.Done():
			return fmt.Errorf("health check canceled: %w", ctx.Err())
		case <-time.After(2 * time.Second):
			// 次のリトライへ
		}
	}

	return fmt.Errorf("health check failed after %d attempts", maxRetries)
}

// sendProgress は進捗イベントを送信します
func (s *CreateService) sendProgress(eventCh chan<- CreateEvent, step, message string, progress int) {
	log.Printf("[CreateService] Step: %s (%d%%)", step, progress)
	eventCh <- CreateEvent{
		Type:     "progress",
		Step:     step,
		Message:  message,
		Progress: progress,
	}
}

// sendError はエラーイベントを送信します
func (s *CreateService) sendError(eventCh chan<- CreateEvent, step string, err error) {
	log.Printf("[CreateService] Step failed: step=%s, error=%v", step, err)
	eventCh <- CreateEvent{
		Type:    "error",
		Step:    step,
		Message: "エラーが発生しました",
		Error:   err.Error(),
	}
}

// OpenInVSCode はVS Codeでプロジェクトを開きます
func (s *CreateService) OpenInVSCode(path string) error {
	log.Printf("[CreateService] OpenInVSCode started: path=%s", path)

	// macOS: open コマンドで VS Code を起動（PATH に code が無くても動作する）
	cmd := exec.Command("open", "-a", "Visual Studio Code", path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open VS Code: %w", err)
	}

	// VS Code は独立プロセスとして動作するため、待機せずリリース
	go cmd.Wait()

	log.Printf("[CreateService] OpenInVSCode completed: path=%s", path)
	return nil
}
