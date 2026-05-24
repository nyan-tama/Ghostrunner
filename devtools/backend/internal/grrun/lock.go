package grrun

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// lockKey はプロジェクトパスからロックファイル名を生成します。
// フォーマット: <basename>-<sha256先頭12文字>.lock
func lockKey(projectPath string) string {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		// Abs が失敗するのは極めて稀（getcwd 失敗時のみ）
		absPath = projectPath
	}
	hash := sha256.Sum256([]byte(absPath))
	hex := fmt.Sprintf("%x", hash[:])
	return filepath.Base(absPath) + "-" + hex[:12] + ".lock"
}

// AcquireLock はプロジェクト単位の排他ロックを取得します。
// 戻り値:
//   - *os.File: ロックファイルのハンドル（呼び出し元がGC防止のため保持すること）
//   - bool: ロック取得成功なら true、他プロセスが保持中なら false
//   - error: ロック取得以外のエラー（ディレクトリ作成失敗等）
func AcquireLock(locksDir, projectPath string) (*os.File, bool, error) {
	if err := os.MkdirAll(locksDir, 0755); err != nil {
		return nil, false, fmt.Errorf("failed to create locks directory %s: %w", locksDir, err)
	}

	lockPath := filepath.Join(locksDir, lockKey(projectPath))
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, false, fmt.Errorf("failed to open lock file %s: %w", lockPath, err)
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		// EWOULDBLOCK と EAGAIN は同じ値（macOS/Linux共通で35/11）
		if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to acquire flock on %s: %w", lockPath, err)
	}

	return f, true, nil
}
