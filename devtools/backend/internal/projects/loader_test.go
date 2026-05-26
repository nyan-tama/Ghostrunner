package projects

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjects(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		fileExists bool
		wantCount  int
		wantNil    bool
		wantErr    bool
	}{
		{
			name:       "正常系: プロジェクト2件",
			content:    `{"projects":[{"path":"/tmp/a","name":"a"},{"path":"/tmp/b","name":"b"}]}`,
			fileExists: true,
			wantCount:  2,
		},
		{
			name:       "正常系: 空のプロジェクト配列",
			content:    `{"projects":[]}`,
			fileExists: true,
			wantCount:  0,
		},
		{
			name:       "正常系: ファイル不在はnilを返す",
			fileExists: false,
			wantNil:    true,
		},
		{
			name:       "異常系: 不正なJSON",
			content:    `{invalid}`,
			fileExists: true,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "patrol_projects.json")

			if tt.fileExists {
				if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			}

			projects, err := LoadProjects(configPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if projects != nil {
					t.Errorf("expected nil, got %v", projects)
				}
				return
			}

			if len(projects) != tt.wantCount {
				t.Errorf("expected %d projects, got %d", tt.wantCount, len(projects))
			}
		})
	}
}
