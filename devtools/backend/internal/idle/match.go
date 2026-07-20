package idle

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghostrunner/backend/internal/projects"
)

// MatchProject は cwd がどのプロジェクトに属するかを判定します。
// filepath.Clean を cwd と各 project.Path の両辺に適用し、
// 完全一致またはセグメント境界を担保した前方一致で判定します。
// 複数一致する場合は最長一致（最も深いパス）を優先します。
// これにより /a/b が /a/bc に誤マッチしません（W7）。
func MatchProject(cwd string, projs []projects.Project) (matched string, ok bool) {
	cleanCwd := filepath.Clean(cwd)

	bestLen := -1
	for _, p := range projs {
		projPath := filepath.Clean(p.Path)
		if cleanCwd == projPath || strings.HasPrefix(cleanCwd, projPath+string(os.PathSeparator)) {
			if len(projPath) > bestLen {
				bestLen = len(projPath)
				matched = projPath
				ok = true
			}
		}
	}

	return matched, ok
}

// IsExpired はマーカーが TTL を超過して失効しているかを判定します。
// now - timestamp(epoch秒) が ttl を超えた場合に true を返します。
func IsExpired(m Marker, now time.Time, ttl time.Duration) bool {
	elapsed := now.Sub(time.Unix(m.Timestamp, 0))
	return elapsed > ttl
}
