package grrun

import (
	"os"
	"regexp"
	"syscall"
	"testing"
)

func TestLockKey_Stability(t *testing.T) {
	path := "/tmp/some/project"
	key1 := lockKey(path)
	key2 := lockKey(path)
	if key1 != key2 {
		t.Errorf("lockKey is not stable: got %q and %q for same path", key1, key2)
	}
}

func TestLockKey_CollisionAvoidance(t *testing.T) {
	// Same basename, different parent directories
	keyA := lockKey("/home/user/projectA/myapp")
	keyB := lockKey("/home/user/projectB/myapp")
	if keyA == keyB {
		t.Errorf("lockKey collision: both paths produced %q", keyA)
	}
}

func TestLockKey_Format(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"absolute path", "/home/user/project"},
		{"nested path", "/a/b/c/d/e"},
		{"simple name", "/myproject"},
	}

	formatRe := regexp.MustCompile(`^.+-[0-9a-f]{12}\.lock$`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := lockKey(tt.path)
			if !formatRe.MatchString(key) {
				t.Errorf("lockKey(%q) = %q, does not match <basename>-<12hex>.lock", tt.path, key)
			}
		})
	}
}

func TestAcquireLock(t *testing.T) {
	t.Run("acquire succeeds on empty locksDir", func(t *testing.T) {
		locksDir := t.TempDir()
		f, ok, err := AcquireLock(locksDir, "/tmp/projectA")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("expected ok=true, got false")
		}
		if f == nil {
			t.Fatal("expected non-nil file")
		}
		f.Close()
	})

	t.Run("auto-create locksDir", func(t *testing.T) {
		base := t.TempDir()
		locksDir := base + "/sub/locks"

		f, ok, err := AcquireLock(locksDir, "/tmp/projectA")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("expected ok=true")
		}
		f.Close()

		// Verify directory was created
		info, err := os.Stat(locksDir)
		if err != nil {
			t.Fatalf("locksDir not created: %v", err)
		}
		if !info.IsDir() {
			t.Fatal("locksDir is not a directory")
		}
	})

	t.Run("double acquire same path returns busy", func(t *testing.T) {
		locksDir := t.TempDir()
		projectPath := "/tmp/projectA"

		f1, ok1, err := AcquireLock(locksDir, projectPath)
		if err != nil {
			t.Fatalf("first acquire error: %v", err)
		}
		if !ok1 {
			t.Fatal("first acquire should succeed")
		}
		defer f1.Close()

		// Open a SECOND file descriptor to the same lock file and try flock
		lockPath := locksDir + "/" + lockKey(projectPath)
		f2, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			t.Fatalf("failed to open second fd: %v", err)
		}
		defer f2.Close()

		err = syscall.Flock(int(f2.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			t.Fatal("expected flock to fail on second fd, but it succeeded")
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			t.Fatalf("expected EWOULDBLOCK/EAGAIN, got: %v", err)
		}

		// Also test via AcquireLock API
		f3, ok3, err := AcquireLock(locksDir, projectPath)
		if err != nil {
			t.Fatalf("second AcquireLock error: %v", err)
		}
		if ok3 {
			t.Fatal("second AcquireLock should return ok=false (busy)")
		}
		if f3 != nil {
			f3.Close()
			t.Fatal("second AcquireLock should return nil file when busy")
		}
	})

	t.Run("separate paths are independent", func(t *testing.T) {
		locksDir := t.TempDir()

		fA, okA, err := AcquireLock(locksDir, "/tmp/projectA")
		if err != nil {
			t.Fatalf("acquire A error: %v", err)
		}
		if !okA {
			t.Fatal("acquire A should succeed")
		}
		defer fA.Close()

		fB, okB, err := AcquireLock(locksDir, "/tmp/projectB")
		if err != nil {
			t.Fatalf("acquire B error: %v", err)
		}
		if !okB {
			t.Fatal("acquire B should succeed while A is held")
		}
		fB.Close()
	})

	t.Run("re-acquire after release succeeds", func(t *testing.T) {
		locksDir := t.TempDir()
		projectPath := "/tmp/projectA"

		f1, ok1, err := AcquireLock(locksDir, projectPath)
		if err != nil {
			t.Fatalf("first acquire error: %v", err)
		}
		if !ok1 {
			t.Fatal("first acquire should succeed")
		}
		f1.Close() // release

		f2, ok2, err := AcquireLock(locksDir, projectPath)
		if err != nil {
			t.Fatalf("re-acquire error: %v", err)
		}
		if !ok2 {
			t.Fatal("re-acquire should succeed after release")
		}
		f2.Close()
	})
}
