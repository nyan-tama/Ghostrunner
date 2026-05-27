package dashboard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ghostrunner/backend/internal/projects"
)

// ErrAlreadyAnswered は既に回答済みか行がずれている場合のエラーです
var ErrAlreadyAnswered = errors.New("already answered or line shifted")

// ErrValidation はバリデーションエラーです
var ErrValidation = errors.New("validation error")

// AnswerRequest は確認事項への回答リクエストを表します
type AnswerRequest struct {
	ProjectPath string `json:"projectPath"`
	PlanPath    string `json:"planPath"`
	LineStart   int    `json:"lineStart"`
	Answer      string `json:"answer"`
}

// AnswerQuestion は確認事項に回答を書き戻します
func AnswerQuestion(req AnswerRequest, allowedProjects []projects.Project) error {
	// バリデーション
	if err := validateAnswerRequest(req, allowedProjects); err != nil {
		return err
	}

	absPath := filepath.Clean(filepath.Join(req.ProjectPath, req.PlanPath))

	// パストラバーサル防止
	cleanProject := filepath.Clean(req.ProjectPath)
	if !strings.HasPrefix(absPath, cleanProject+string(os.PathSeparator)) {
		return fmt.Errorf("%w: path traversal detected", ErrValidation)
	}

	// ファイル読み込み
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read plan file: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	// 検索ウィンドウ内で未回答行を探す(LineStart-2 ~ LineStart+2、0-indexed)
	targetIdx := req.LineStart - 1 // 0-based
	windowStart := targetIdx - 2
	if windowStart < 0 {
		windowStart = 0
	}
	windowEnd := targetIdx + 2
	if windowEnd >= len(lines) {
		windowEnd = len(lines) - 1
	}

	// ウィンドウ内の未回答行を検索
	matchIdx := -1
	bestDist := len(lines)
	for i := windowStart; i <= windowEnd; i++ {
		if unansweredRe.MatchString(lines[i]) {
			dist := i - targetIdx
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist {
				bestDist = dist
				matchIdx = i
			}
		}
	}

	if matchIdx < 0 {
		return ErrAlreadyAnswered
	}

	// ステータスを回答済みに変更
	lines[matchIdx] = strings.Replace(lines[matchIdx], "未回答", "回答済", 1)

	// 回答行の挿入・置換
	answerLine := fmt.Sprintf("**回答**: %s", strings.TrimSpace(req.Answer))
	nextIdx := matchIdx + 1

	if nextIdx >= len(lines) {
		// ファイル末尾の場合は追加
		lines = append(lines, answerLine)
	} else if strings.TrimSpace(lines[nextIdx]) == "" {
		// 空行の場合は回答行を挿入
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:nextIdx]...)
		newLines = append(newLines, answerLine)
		newLines = append(newLines, lines[nextIdx:]...)
		lines = newLines
	} else if strings.HasPrefix(strings.TrimSpace(lines[nextIdx]), "**回答**:") {
		// 既存の回答行を置換
		lines[nextIdx] = answerLine
	} else {
		// その他（見出し等）の場合は手前に挿入
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:nextIdx]...)
		newLines = append(newLines, answerLine)
		newLines = append(newLines, lines[nextIdx:]...)
		lines = newLines
	}

	// アトミック書き込み（同一ディレクトリにtmpファイルを作成してrename）
	content := strings.Join(lines, "\n")
	dir := filepath.Dir(absPath)
	tmpFile, err := os.CreateTemp(dir, ".plan.tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, absPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// validateAnswerRequest はリクエストのバリデーションを行います
func validateAnswerRequest(req AnswerRequest, allowedProjects []projects.Project) error {
	// projectPathが許可リストに含まれるか
	allowed := false
	cleanReqPath := filepath.Clean(req.ProjectPath)
	for _, p := range allowedProjects {
		if filepath.Clean(p.Path) == cleanReqPath {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("%w: project not in allowed list: %s", ErrValidation, req.ProjectPath)
	}

	// planPathが許可ディレクトリ配下か
	cleanPlan := filepath.Clean(req.PlanPath)
	validPrefixes := []string{
		filepath.Join("開発", "実装", "実装待ち") + string(os.PathSeparator),
		filepath.Join("開発", "実装", "実行中") + string(os.PathSeparator),
	}
	validPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(cleanPlan, prefix) {
			validPrefix = true
			break
		}
	}
	if !validPrefix {
		return fmt.Errorf("%w: plan path not in allowed directory: %s", ErrValidation, req.PlanPath)
	}

	// 拡張子チェック
	if filepath.Ext(cleanPlan) != ".md" {
		return fmt.Errorf("%w: plan path must be .md file: %s", ErrValidation, req.PlanPath)
	}

	// LineStartチェック
	if req.LineStart < 1 {
		return fmt.Errorf("%w: lineStart must be >= 1", ErrValidation)
	}

	// Answerチェック
	if strings.TrimSpace(req.Answer) == "" {
		return fmt.Errorf("%w: answer must not be empty", ErrValidation)
	}

	return nil
}
