package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceTemplateDir(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		want        string
	}{
		{
			name:        "database は with-db を返す",
			serviceName: "database",
			want:        "with-db",
		},
		{
			name:        "storage は with-storage を返す",
			serviceName: "storage",
			want:        "with-storage",
		},
		{
			name:        "cache は with-redis を返す",
			serviceName: "cache",
			want:        "with-redis",
		},
		{
			name:        "未知のサービス名は空文字を返す",
			serviceName: "unknown",
			want:        "",
		},
		{
			name:        "空文字は空文字を返す",
			serviceName: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceTemplateDir(tt.serviceName)
			if got != tt.want {
				t.Errorf("serviceTemplateDir(%q): got %q, want %q", tt.serviceName, got, tt.want)
			}
		})
	}
}

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		// バイナリファイル
		{name: "PNG", path: "image.png", want: true},
		{name: "JPG", path: "photo.jpg", want: true},
		{name: "JPEG", path: "photo.jpeg", want: true},
		{name: "GIF", path: "anim.gif", want: true},
		{name: "ICO", path: "favicon.ico", want: true},
		{name: "WOFF", path: "font.woff", want: true},
		{name: "WOFF2", path: "font.woff2", want: true},
		{name: "TTF", path: "font.ttf", want: true},
		{name: "EOT", path: "font.eot", want: true},
		{name: "ZIP", path: "archive.zip", want: true},
		{name: "TAR", path: "archive.tar", want: true},
		{name: "GZ", path: "archive.gz", want: true},
		{name: "BIN", path: "data.bin", want: true},
		{name: "EXE", path: "app.exe", want: true},
		{name: "DLL", path: "lib.dll", want: true},
		{name: "SO", path: "lib.so", want: true},
		{name: "DYLIB", path: "lib.dylib", want: true},
		{name: "PDF", path: "doc.pdf", want: true},
		{name: "WEBP", path: "image.webp", want: true},
		// 大文字の拡張子もバイナリ判定される（ToLower処理あり）
		{name: "PNG大文字", path: "image.PNG", want: true},
		// テキストファイル
		{name: "Go", path: "main.go", want: false},
		{name: "YAML", path: "config.yml", want: false},
		{name: "JSON", path: "data.json", want: false},
		{name: "Markdown", path: "README.md", want: false},
		{name: "TypeScript", path: "app.tsx", want: false},
		{name: "拡張子なし", path: "Makefile", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinaryFile(tt.path)
			if got != tt.want {
				t.Errorf("isBinaryFile(%q): got %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestMergeYAMLMaps(t *testing.T) {
	tests := []struct {
		name     string
		dest     map[string]interface{}
		src      map[string]interface{}
		checkFn  func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "servicesがマージされる",
			dest: map[string]interface{}{
				"services": map[string]interface{}{
					"app": map[string]interface{}{"image": "app:latest"},
				},
			},
			src: map[string]interface{}{
				"services": map[string]interface{}{
					"db": map[string]interface{}{"image": "postgres:16"},
				},
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				services := result["services"].(map[string]interface{})
				if _, ok := services["app"]; !ok {
					t.Error("services should contain 'app'")
				}
				if _, ok := services["db"]; !ok {
					t.Error("services should contain 'db'")
				}
			},
		},
		{
			name: "volumesがマージされる",
			dest: map[string]interface{}{
				"volumes": map[string]interface{}{
					"data1": nil,
				},
			},
			src: map[string]interface{}{
				"volumes": map[string]interface{}{
					"data2": nil,
				},
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				volumes := result["volumes"].(map[string]interface{})
				if _, ok := volumes["data1"]; !ok {
					t.Error("volumes should contain 'data1'")
				}
				if _, ok := volumes["data2"]; !ok {
					t.Error("volumes should contain 'data2'")
				}
			},
		},
		{
			name: "destにservicesがない場合に新規作成される",
			dest: map[string]interface{}{},
			src: map[string]interface{}{
				"services": map[string]interface{}{
					"redis": map[string]interface{}{"image": "redis:7"},
				},
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				services := result["services"].(map[string]interface{})
				if _, ok := services["redis"]; !ok {
					t.Error("services should contain 'redis'")
				}
			},
		},
		{
			name: "destにvolumesがない場合に新規作成される",
			dest: map[string]interface{}{},
			src: map[string]interface{}{
				"volumes": map[string]interface{}{
					"redis-data": nil,
				},
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				volumes := result["volumes"].(map[string]interface{})
				if _, ok := volumes["redis-data"]; !ok {
					t.Error("volumes should contain 'redis-data'")
				}
			},
		},
		{
			name: "servicesとvolumes以外のキーはマージされない",
			dest: map[string]interface{}{
				"version": "3.8",
			},
			src: map[string]interface{}{
				"version": "3.9",
			},
			checkFn: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				if result["version"] != "3.8" {
					t.Errorf("version should remain '3.8', got %v", result["version"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeYAMLMaps(tt.dest, tt.src)
			tt.checkFn(t, tt.dest)
		})
	}
}

func TestTemplateService_ReplacePlaceholders(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string // ファイル名 -> 内容
		projectName string
		wantFiles   map[string]string // ファイル名 -> 期待される内容
	}{
		{
			name: "プレースホルダーが置換される",
			files: map[string]string{
				"go.mod":     "module {{PROJECT_NAME}}/backend\n",
				"config.yml": "name: {{PROJECT_NAME}}\nport: 8080\n",
			},
			projectName: "my-app",
			wantFiles: map[string]string{
				"go.mod":     "module my-app/backend\n",
				"config.yml": "name: my-app\nport: 8080\n",
			},
		},
		{
			name: "プレースホルダーがないファイルは変更されない",
			files: map[string]string{
				"README.md": "Hello World\n",
			},
			projectName: "my-app",
			wantFiles: map[string]string{
				"README.md": "Hello World\n",
			},
		},
		{
			name: "複数のプレースホルダーが同一ファイル内で置換される",
			files: map[string]string{
				"main.go": "package main\n// {{PROJECT_NAME}} server\nconst name = \"{{PROJECT_NAME}}\"\n",
			},
			projectName: "test-project",
			wantFiles: map[string]string{
				"main.go": "package main\n// test-project server\nconst name = \"test-project\"\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destDir := t.TempDir()

			// テストファイルを作成
			for name, content := range tt.files {
				filePath := filepath.Join(destDir, name)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to create test file %s: %v", name, err)
				}
			}

			svc := NewTemplateService("")
			if err := svc.ReplacePlaceholders(destDir, tt.projectName); err != nil {
				t.Fatalf("ReplacePlaceholders() error: %v", err)
			}

			// 結果を検証
			for name, wantContent := range tt.wantFiles {
				filePath := filepath.Join(destDir, name)
				gotBytes, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("failed to read result file %s: %v", name, err)
				}
				got := string(gotBytes)
				if got != wantContent {
					t.Errorf("file %s: got %q, want %q", name, got, wantContent)
				}
			}
		})
	}
}

func TestBuildClaudeMD(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		description string
		services    []string
		wantContain []string // 生成結果に含まれるべき文字列
	}{
		{
			name:        "基本構成_サービスなし",
			projectName: "my-app",
			description: "テスト用のアプリケーション",
			services:    nil,
			wantContain: []string{
				"# my-app",
				"テスト用のアプリケーション",
				"## 技術スタック",
				"Go (Gin)",
				"Next.js 15",
				"Tailwind CSS",
				"## 開発サーバー",
				"make dev",
				"## ビルド",
				"go build",
				"go test",
				"npm run build",
				"## プロジェクト構造",
				"my-app/",
				"backend/",
				"frontend/",
			},
		},
		{
			name:        "databaseサービス選択時にPostgreSQLとDB構造が含まれる",
			projectName: "db-app",
			description: "",
			services:    []string{"database"},
			wantContain: []string{
				"# db-app",
				"PostgreSQL 16 + GORM",
				"db/",
			},
		},
		{
			name:        "storageサービス選択時にMinIOが含まれる",
			projectName: "storage-app",
			description: "",
			services:    []string{"storage"},
			wantContain: []string{
				"MinIO (S3互換)",
			},
		},
		{
			name:        "cacheサービス選択時にRedisが含まれる",
			projectName: "cache-app",
			description: "",
			services:    []string{"cache"},
			wantContain: []string{
				"Redis 7",
			},
		},
		{
			name:        "descriptionが空の場合はプロジェクト名の後に概要が出力されない",
			projectName: "no-desc",
			description: "",
			services:    nil,
			wantContain: []string{
				"# no-desc",
				"## 技術スタック",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildClaudeMD(tt.projectName, tt.description, tt.services)

			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("buildClaudeMD() output does not contain %q\ngot:\n%s", want, got)
				}
			}
		})
	}
}
