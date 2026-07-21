package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghostrunner/backend/internal/projects"
)

// sessionFile は走査対象の会話ログファイル1件を表します。
type sessionFile struct {
	path      string
	modTime   time.Time
	sessionID string
}

// deriveProjectID は cwd 絶対パスの非英数字を "-" に置換した project-id を返します。
// 例: /Users/user/j-board -> -Users-user-j-board。
// これは lossy 変換で帰属決定には使えず、走査ディレクトリの絞り込み（性能）専用です（C2/W6）。
func deriveProjectID(cwd string) string {
	var b strings.Builder
	b.Grow(len(cwd))
	for _, r := range cwd {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// discoverSessions は登録プロジェクトごとに project-id 前方一致 glob で候補セッションを列挙します。
// glob は lossy な project-id に基づくため取りこぼしうる。1件も拾えない場合は
// ~/.claude/projects/ 全ディレクトリ走査に fallback します（W6）。
// 帰属の最終判定は呼び出し側が実 cwd + MatchProject で行うため、
// ここでの過剰マッチ（兄弟ディレクトリ等）は許容されます。
func discoverSessions(homeDir string, projs []projects.Project) ([]sessionFile, error) {
	projectsDir := filepath.Join(homeDir, ".claude", "projects")

	seen := make(map[string]struct{})
	out := make([]sessionFile, 0)
	for _, p := range projs {
		projID := deriveProjectID(p.Path)
		pattern := filepath.Join(projectsDir, projID+"*", "*.jsonl")
		paths, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob sessions %s: %w", pattern, err)
		}
		for _, path := range paths {
			appendSession(&out, seen, path)
		}
	}

	if len(out) > 0 {
		return out, nil
	}

	// W6: 絞り込み glob が空 → 全走査 fallback（帰属は実 cwd + MatchProject が担保）
	return scanAllSessions(projectsDir)
}

// scanAllSessions は ~/.claude/projects/ 配下の全ディレクトリの *.jsonl を列挙します。
// projects ディレクトリ自体が存在しない場合は空スライスを返します（error にしない）。
func scanAllSessions(projectsDir string) ([]sessionFile, error) {
	dirs, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []sessionFile{}, nil
		}
		return nil, fmt.Errorf("failed to read projects dir %s: %w", projectsDir, err)
	}

	seen := make(map[string]struct{})
	out := make([]sessionFile, 0)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		paths, err := filepath.Glob(filepath.Join(projectsDir, d.Name(), "*.jsonl"))
		if err != nil {
			return nil, fmt.Errorf("failed to glob sessions in %s: %w", d.Name(), err)
		}
		for _, path := range paths {
			appendSession(&out, seen, path)
		}
	}
	return out, nil
}

// appendSession は path の mtime を取得し重複排除して out に追加します。
// stat 失敗（削除競合等）のファイルは黙って skip します。
func appendSession(out *[]sessionFile, seen map[string]struct{}, path string) {
	if _, ok := seen[path]; ok {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	seen[path] = struct{}{}
	*out = append(*out, sessionFile{
		path:      path,
		modTime:   info.ModTime(),
		sessionID: sessionIDFromPath(path),
	})
}

// sessionIDFromPath はファイルパスから拡張子を除いた session-id を返します。
func sessionIDFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
