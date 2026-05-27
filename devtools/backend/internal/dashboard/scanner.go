package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ghostrunner/backend/internal/grrun"
)

var unansweredPattern = grrun.UnansweredPattern

var unansweredRe = regexp.MustCompile(unansweredPattern)

// headingRe は確認事項の見出し行を検出する正規表現です
var headingRe = regexp.MustCompile(`^### Q\d+:`)

// GetPatternForTest はSSOT検証用にパターン文字列を返します
func GetPatternForTest() string {
	return unansweredPattern
}

// ScanProject は1つのプロジェクトの状態を集約します
func ScanProject(projectPath, ghostrunnerRoot string, now time.Time) (ProjectState, error) {
	name := filepath.Base(projectPath)
	state := ProjectState{
		Name:       name,
		Path:       projectPath,
		IsSelf:     filepath.Clean(projectPath) == filepath.Clean(ghostrunnerRoot),
		Unanswered: []UnansweredQuestion{},
		Ops:        []OpsEntry{},
		Warnings:   []string{},
	}

	// カンバンカウント
	state.Kanban = countKanban(projectPath, &state.Warnings)

	// 未回答確認事項の検出（実装待ちと実行中）
	scanDirs := []string{
		filepath.Join(projectPath, "開発", "実装", "実装待ち"),
		filepath.Join(projectPath, "開発", "実装", "実行中"),
	}
	for _, dir := range scanDirs {
		questions := scanUnanswered(projectPath, dir, &state.Warnings)
		state.Unanswered = append(state.Unanswered, questions...)
	}

	// 運用状態
	opsDir := filepath.Join(projectPath, "運用")
	if info, err := os.Stat(opsDir); err == nil && info.IsDir() {
		state.OpsOptedIn = true
		state.Ops = scanOps(projectPath, now, &state.Warnings)
	}

	// Attention判定
	state.Attention = determineAttention(state)

	return state, nil
}

// countKanban はカンバンディレクトリの.mdファイル数をカウントします
func countKanban(projectPath string, warnings *[]string) KanbanCounts {
	counts := KanbanCounts{}
	dirs := map[string]*int{
		filepath.Join(projectPath, "開発", "実装", "レビュー"): &counts.Reviewing,
		filepath.Join(projectPath, "開発", "実装", "実装待ち"): &counts.Waiting,
		filepath.Join(projectPath, "開発", "実装", "実行中"):  &counts.Running,
		filepath.Join(projectPath, "開発", "実装", "完了"):   &counts.Done,
	}

	for dir, counter := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("failed to glob %s: %v", dir, err))
			continue
		}
		*counter = len(matches)
	}

	return counts
}

// scanUnanswered はディレクトリ内の.mdファイルから未回答確認事項を検出します
func scanUnanswered(projectPath, dir string, warnings *[]string) []UnansweredQuestion {
	var questions []UnansweredQuestion

	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("failed to glob %s: %v", dir, err))
		return questions
	}

	for _, filePath := range matches {
		data, err := os.ReadFile(filePath)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("failed to read %s: %v", filePath, err))
			continue
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if !unansweredRe.MatchString(line) {
				continue
			}

			lineNum := i + 1 // 1-based

			// 見出しを後方検索
			heading := ""
			questionText := ""
			for j := i - 1; j >= 0; j-- {
				if headingRe.MatchString(lines[j]) {
					heading = strings.TrimSpace(lines[j])
					// 見出しとステータス行の間のテキストを質問文とする
					var textParts []string
					for k := j + 1; k < i; k++ {
						trimmed := strings.TrimSpace(lines[k])
						if trimmed != "" && !strings.HasPrefix(trimmed, "**") {
							textParts = append(textParts, trimmed)
						}
					}
					questionText = strings.Join(textParts, " ")
					break
				}
			}

			// プロジェクトパスからの相対パスを計算
			relPath, err := filepath.Rel(projectPath, filePath)
			if err != nil {
				relPath = filePath
			}

			questions = append(questions, UnansweredQuestion{
				PlanPath:     relPath,
				LineStart:    lineNum,
				LineEnd:      lineNum,
				QuestionText: questionText,
				Heading:      heading,
			})
		}
	}

	return questions
}

// opsJSON は運用状態JSONファイルの構造を表します
type opsJSON struct {
	Account           string       `json:"account"`
	Kind              string       `json:"kind"`
	Status            string       `json:"status"`
	Progress          *OpsProgress `json:"progress,omitempty"`
	Today             *OpsToday    `json:"today,omitempty"`
	Stats             *OpsStats    `json:"stats,omitempty"`
	ConsecutiveErrors int          `json:"consecutiveErrors"`
	UpdatedAt         string       `json:"updatedAt"`
}

// scanOps は運用/状態ディレクトリからOpsEntryを収集します
func scanOps(projectPath string, now time.Time, warnings *[]string) []OpsEntry {
	var entries []OpsEntry
	stateDir := filepath.Join(projectPath, "運用", "状態")

	matches, err := filepath.Glob(filepath.Join(stateDir, "*.json"))
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("failed to glob ops state: %v", err))
		return entries
	}

	for _, filePath := range matches {
		info, err := os.Stat(filePath)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("failed to stat %s: %v", filePath, err))
			continue
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("failed to read %s: %v", filePath, err))
			continue
		}

		var raw opsJSON
		if err := json.Unmarshal(data, &raw); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("failed to parse %s: %v", filePath, err))
			continue
		}

		// staleness判定: ファイル更新から3時間以上かつstatus=running
		staleHours := int(now.Sub(info.ModTime()).Hours())
		stale := now.Sub(info.ModTime()).Hours() >= 3.0 && raw.Status == "running"

		// rawExtraとしてJSON全体を保持
		var rawExtra map[string]any
		if err := json.Unmarshal(data, &rawExtra); err == nil {
			// 既知フィールドを除外
			for _, key := range []string{"account", "kind", "status", "progress", "today", "stats", "consecutiveErrors", "updatedAt"} {
				delete(rawExtra, key)
			}
			if len(rawExtra) == 0 {
				rawExtra = nil
			}
		}

		relPath, err := filepath.Rel(projectPath, filePath)
		if err != nil {
			relPath = filePath
		}

		entries = append(entries, OpsEntry{
			Account:           raw.Account,
			Kind:              raw.Kind,
			Status:            raw.Status,
			Progress:          raw.Progress,
			Today:             raw.Today,
			Stats:             raw.Stats,
			ConsecutiveErrors: raw.ConsecutiveErrors,
			UpdatedAt:         raw.UpdatedAt,
			Stale:             stale,
			StaleHours:        staleHours,
			SourceFile:        relPath,
			RawExtra:          rawExtra,
		})
	}

	return entries
}

// determineAttention はプロジェクトの注目度を判定します
func determineAttention(state ProjectState) Attention {
	// required: 未回答あり、またはops異常
	if len(state.Unanswered) > 0 {
		return AttentionRequired
	}
	for _, op := range state.Ops {
		if op.Status == "blocked" || op.Stale || op.ConsecutiveErrors >= 3 {
			return AttentionRequired
		}
	}

	// progress: カンバンにrunning/waitingあり、またはops正常稼働中
	if state.Kanban.Running > 0 || state.Kanban.Waiting > 0 {
		return AttentionProgress
	}
	for _, op := range state.Ops {
		if op.Status == "running" && !op.Stale {
			return AttentionProgress
		}
	}

	return AttentionWatching
}
