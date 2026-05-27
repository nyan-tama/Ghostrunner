package projects

import (
	"encoding/json"
	"fmt"
	"os"
)

// Project は巡回対象プロジェクトを表します
type Project struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// Config はpatrol_projects.jsonの構造を表します
type Config struct {
	Projects []Project `json:"projects"`
}

// LoadProjects はpatrol_projects.jsonからプロジェクト一覧を読み込みます。
// ファイルが存在しない場合は(nil, nil)を返します。
// JSONが不正な場合のみエラーを返します。
func LoadProjects(configPath string) ([]Project, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config %s: %w", configPath, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", configPath, err)
	}

	return config.Projects, nil
}
