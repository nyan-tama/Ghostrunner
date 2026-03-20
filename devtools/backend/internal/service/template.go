// Package service はビジネスロジックを提供します
package service

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TemplateService はテンプレートのコピーと加工を担当します
type TemplateService struct {
	// ghostrunnerRoot はGhostrunnerリポジトリのルートディレクトリ
	ghostrunnerRoot string
}

// NewTemplateService は新しいTemplateServiceを生成します
func NewTemplateService(ghostrunnerRoot string) *TemplateService {
	return &TemplateService{
		ghostrunnerRoot: ghostrunnerRoot,
	}
}

// CopyBase はbaseテンプレートを指定ディレクトリにコピーします
func (s *TemplateService) CopyBase(destDir string) error {
	srcDir := filepath.Join(s.ghostrunnerRoot, "templates", "base")

	log.Printf("[TemplateService] CopyBase started: src=%s, dest=%s", srcDir, destDir)

	if err := copyDir(srcDir, destDir); err != nil {
		return fmt.Errorf("failed to copy base template: %w", err)
	}

	log.Printf("[TemplateService] CopyBase completed: dest=%s", destDir)
	return nil
}

// CopyServiceTemplate はオプションのサービステンプレートをコピーします
// サービス名: "database", "storage", "cache"
func (s *TemplateService) CopyServiceTemplate(destDir string, serviceName string) error {
	templateDir := serviceTemplateDir(serviceName)
	if templateDir == "" {
		return fmt.Errorf("unknown service: %s", serviceName)
	}

	srcDir := filepath.Join(s.ghostrunnerRoot, "templates", templateDir)

	log.Printf("[TemplateService] CopyServiceTemplate started: service=%s, src=%s, dest=%s", serviceName, srcDir, destDir)

	// docker-compose.yml はマージで処理するためスキップ
	if err := copyDirSkip(srcDir, destDir, "docker-compose.yml"); err != nil {
		return fmt.Errorf("failed to copy service template %s: %w", serviceName, err)
	}

	log.Printf("[TemplateService] CopyServiceTemplate completed: service=%s, dest=%s", serviceName, destDir)
	return nil
}

// ReplacePlaceholders はプロジェクト内の全ファイルで {{PROJECT_NAME}} を置換します
func (s *TemplateService) ReplacePlaceholders(destDir string, projectName string) error {
	log.Printf("[TemplateService] ReplacePlaceholders started: dest=%s, projectName=%s", destDir, projectName)

	count := 0
	err := filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// バイナリファイルはスキップ
		if isBinaryFile(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		original := string(content)
		replaced := strings.ReplaceAll(original, "{{PROJECT_NAME}}", projectName)

		if original != replaced {
			if err := os.WriteFile(path, []byte(replaced), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", path, err)
			}
			count++
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to replace placeholders: %w", err)
	}

	log.Printf("[TemplateService] ReplacePlaceholders completed: dest=%s, filesModified=%d", destDir, count)
	return nil
}

// MergeDockerCompose はbase と選択サービスの docker-compose.yml をマージします
func (s *TemplateService) MergeDockerCompose(destDir string, services []string) error {
	log.Printf("[TemplateService] MergeDockerCompose started: dest=%s, services=%v", destDir, services)

	// database が含まれる場合は with-db のdocker-compose.ymlをベースにする
	var basePath string
	hasDatabaseService := false
	for _, svc := range services {
		if svc == "database" {
			hasDatabaseService = true
			break
		}
	}

	if hasDatabaseService {
		basePath = filepath.Join(s.ghostrunnerRoot, "templates", "with-db", "docker-compose.yml")
	} else {
		basePath = filepath.Join(destDir, "docker-compose.yml")
	}

	baseContent, err := os.ReadFile(basePath)
	if err != nil {
		return fmt.Errorf("failed to read base docker-compose.yml: %w", err)
	}

	var baseMap map[string]interface{}
	if err := yaml.Unmarshal(baseContent, &baseMap); err != nil {
		return fmt.Errorf("failed to parse base docker-compose.yml: %w", err)
	}

	// 各サービスのdocker-compose.ymlをマージ
	for _, svc := range services {
		// database は既にベースに含まれている
		if svc == "database" {
			continue
		}

		templateDir := serviceTemplateDir(svc)
		if templateDir == "" {
			continue
		}

		svcPath := filepath.Join(s.ghostrunnerRoot, "templates", templateDir, "docker-compose.yml")
		svcContent, err := os.ReadFile(svcPath)
		if err != nil {
			return fmt.Errorf("failed to read docker-compose.yml for %s: %w", svc, err)
		}

		var svcMap map[string]interface{}
		if err := yaml.Unmarshal(svcContent, &svcMap); err != nil {
			return fmt.Errorf("failed to parse docker-compose.yml for %s: %w", svc, err)
		}

		mergeYAMLMaps(baseMap, svcMap)
	}

	// マージ結果を書き出し
	output, err := yaml.Marshal(baseMap)
	if err != nil {
		return fmt.Errorf("failed to marshal merged docker-compose.yml: %w", err)
	}

	destPath := filepath.Join(destDir, "docker-compose.yml")
	if err := os.WriteFile(destPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write merged docker-compose.yml: %w", err)
	}

	log.Printf("[TemplateService] MergeDockerCompose completed: dest=%s", destDir)
	return nil
}

// CopyClaudeAssets は .claude/ ディレクトリをプロジェクトにコピーします
func (s *TemplateService) CopyClaudeAssets(destDir string) error {
	srcDir := filepath.Join(s.ghostrunnerRoot, ".claude")
	destClaudeDir := filepath.Join(destDir, ".claude")

	log.Printf("[TemplateService] CopyClaudeAssets started: src=%s, dest=%s", srcDir, destClaudeDir)

	// agents と skills をコピー（存在するもののみ）
	for _, subDir := range []string{"agents", "skills"} {
		src := filepath.Join(srcDir, subDir)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		dest := filepath.Join(destClaudeDir, subDir)
		if err := copyDir(src, dest); err != nil {
			return fmt.Errorf("failed to copy .claude/%s: %w", subDir, err)
		}
	}

	// settings.json をコピー
	settingsSrc := filepath.Join(srcDir, "settings.json")
	settingsDest := filepath.Join(destClaudeDir, "settings.json")
	if err := copyFile(settingsSrc, settingsDest); err != nil {
		return fmt.Errorf("failed to copy .claude/settings.json: %w", err)
	}

	log.Printf("[TemplateService] CopyClaudeAssets completed: dest=%s", destClaudeDir)
	return nil
}

// RemoveUnusedAgents はサービスに応じて不要なエージェントを削除します
func (s *TemplateService) RemoveUnusedAgents(destDir string, services []string) error {
	log.Printf("[TemplateService] RemoveUnusedAgents started: dest=%s, services=%v", destDir, services)

	hasDB := false
	for _, svc := range services {
		if svc == "database" {
			hasDB = true
			break
		}
	}

	agentsDir := filepath.Join(destDir, ".claude", "agents")

	// DB関連エージェントはdatabaseサービスが選択されていない場合に削除
	if !hasDB {
		dbAgents := []string{"pg-impl.md", "pg-planner.md", "pg-reviewer.md", "pg-tester.md"}
		for _, agent := range dbAgents {
			agentPath := filepath.Join(agentsDir, agent)
			if err := os.Remove(agentPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove agent %s: %w", agent, err)
			}
		}
	}

	log.Printf("[TemplateService] RemoveUnusedAgents completed: dest=%s", destDir)
	return nil
}

// GenerateClaudeMD はプロジェクト用の CLAUDE.md を生成します
func (s *TemplateService) GenerateClaudeMD(destDir string, projectName string, description string, services []string) error {
	log.Printf("[TemplateService] GenerateClaudeMD started: dest=%s, projectName=%s", destDir, projectName)

	content := buildClaudeMD(projectName, description, services)

	destPath := filepath.Join(destDir, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	log.Printf("[TemplateService] GenerateClaudeMD completed: dest=%s", destDir)
	return nil
}

// CreateEnvFile は .env ファイルを生成します
func (s *TemplateService) CreateEnvFile(destDir string, projectName string, services []string) error {
	log.Printf("[TemplateService] CreateEnvFile started: dest=%s, projectName=%s, services=%v", destDir, projectName, services)

	// .env.example をベースに .env を生成
	envExamplePath := filepath.Join(destDir, "backend", ".env.example")
	content, err := os.ReadFile(envExamplePath)
	if err != nil {
		return fmt.Errorf("failed to read .env.example: %w", err)
	}

	envContent := string(content)

	// サービスごとに環境変数を追加
	for _, svc := range services {
		switch svc {
		case "database":
			envContent += fmt.Sprintf("\nDATABASE_URL=postgres://postgres:postgres@localhost:5432/%s?sslmode=disable\n", projectName)
		case "storage":
			envContent += "\nMINIO_ENDPOINT=localhost:9000\nMINIO_ACCESS_KEY=minioadmin\nMINIO_SECRET_KEY=minioadmin\nMINIO_BUCKET=uploads\nMINIO_USE_SSL=false\n"
		case "cache":
			envContent += "\nREDIS_URL=localhost:6379\n"
		}
	}

	envPath := filepath.Join(destDir, "backend", ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env: %w", err)
	}

	log.Printf("[TemplateService] CreateEnvFile completed: dest=%s", destDir)
	return nil
}

// CreateDevtoolsLink はdevtoolsフロントエンドへのシンボリックリンクを作成します
func (s *TemplateService) CreateDevtoolsLink(destDir string) error {
	log.Printf("[TemplateService] CreateDevtoolsLink started: dest=%s", destDir)

	linkPath := filepath.Join(destDir, ".devtools")
	target := filepath.Join(s.ghostrunnerRoot, "devtools", "frontend")

	if err := os.Symlink(target, linkPath); err != nil {
		return fmt.Errorf("failed to create devtools symlink: %w", err)
	}

	log.Printf("[TemplateService] CreateDevtoolsLink completed: dest=%s, target=%s", destDir, target)
	return nil
}

// serviceTemplateDir はサービス名からテンプレートディレクトリ名を返します
func serviceTemplateDir(serviceName string) string {
	switch serviceName {
	case "database":
		return "with-db"
	case "storage":
		return "with-storage"
	case "cache":
		return "with-redis"
	default:
		return ""
	}
}

// copyDir はディレクトリを再帰的にコピーします
func copyDir(src, dest string) error {
	return copyDirSkip(src, dest, "")
}

// copyDirSkip はディレクトリを再帰的にコピーします（skipFileNameに一致するファイルはスキップ）
func copyDirSkip(src, dest, skipFileName string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// スキップ対象のファイル
		if skipFileName != "" && d.Name() == skipFileName {
			return nil
		}

		return copyFile(path, destPath)
	})
}

// copyFile はファイルをコピーします
func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// 元ファイルのパーミッションを保持
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if err := os.WriteFile(dest, content, info.Mode()); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// isBinaryFile はファイルがバイナリかどうかを判定します（拡張子ベース）
func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".ico": true, ".woff": true, ".woff2": true, ".ttf": true,
		".eot": true, ".zip": true, ".tar": true, ".gz": true,
		".bin": true, ".exe": true, ".dll": true, ".so": true,
		".dylib": true, ".pdf": true, ".webp": true,
	}
	return binaryExts[ext]
}

// mergeYAMLMaps はsrcMapの内容をdestMapにマージします
// services と volumes のキーをマージ対象とします
func mergeYAMLMaps(destMap, srcMap map[string]interface{}) {
	// services のマージ
	if srcServices, ok := srcMap["services"].(map[string]interface{}); ok {
		destServices, ok := destMap["services"].(map[string]interface{})
		if !ok {
			destServices = make(map[string]interface{})
			destMap["services"] = destServices
		}
		for k, v := range srcServices {
			destServices[k] = v
		}
	}

	// volumes のマージ
	if srcVolumes, ok := srcMap["volumes"].(map[string]interface{}); ok {
		destVolumes, ok := destMap["volumes"].(map[string]interface{})
		if !ok {
			destVolumes = make(map[string]interface{})
			destMap["volumes"] = destVolumes
		}
		for k, v := range srcVolumes {
			destVolumes[k] = v
		}
	}
}

// buildClaudeMD はプロジェクト用のCLAUDE.mdを生成します
func buildClaudeMD(projectName, description string, services []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", projectName))

	if description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", description))
	}

	sb.WriteString("## 技術スタック\n\n")
	sb.WriteString("- Go (Gin) - バックエンド\n")
	sb.WriteString("- Next.js 15 (App Router) + React 19 + TypeScript - フロントエンド\n")
	sb.WriteString("- Tailwind CSS - スタイリング\n")

	for _, svc := range services {
		switch svc {
		case "database":
			sb.WriteString("- PostgreSQL 16 + GORM - データベース\n")
		case "storage":
			sb.WriteString("- MinIO (S3互換) - オブジェクトストレージ\n")
		case "cache":
			sb.WriteString("- Redis 7 - キャッシュ\n")
		}
	}

	sb.WriteString("\n## 開発サーバー\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("make dev        # バックエンド + フロントエンド + devtools を並列起動\n")
	sb.WriteString("make backend    # バックエンドのみ起動\n")
	sb.WriteString("make frontend   # フロントエンドのみ起動\n")
	sb.WriteString("make stop       # 全サーバーを停止\n")
	sb.WriteString("make health     # ヘルスチェック\n")
	sb.WriteString("```\n")

	sb.WriteString("\n## ビルド\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("cd backend && go build ./...     # バックエンドビルド確認\n")
	sb.WriteString("cd backend && go test ./...      # バックエンドテスト\n")
	sb.WriteString("cd frontend && npm run build     # フロントエンドビルド\n")
	sb.WriteString("cd frontend && npm test          # フロントエンドテスト\n")
	sb.WriteString("```\n")

	sb.WriteString("\n## プロジェクト構造\n\n")
	sb.WriteString("```\n")
	sb.WriteString(fmt.Sprintf("%s/\n", projectName))
	sb.WriteString("├── backend/          # Go (Gin) API サーバー\n")
	sb.WriteString("│   ├── cmd/server/   # エントリーポイント\n")
	sb.WriteString("│   └── internal/     # 内部パッケージ\n")
	sb.WriteString("├── frontend/         # Next.js フロントエンド\n")
	sb.WriteString("│   └── src/app/      # App Router ページ\n")

	for _, svc := range services {
		if svc == "database" {
			sb.WriteString("├── db/               # DB初期化SQL\n")
		}
	}

	sb.WriteString("└── docker-compose.yml\n")
	sb.WriteString("```\n")

	return sb.String()
}
