package grrun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// mockNotifier records Notify and NotifyError calls
type mockNotifier struct {
	mu          sync.Mutex
	notifyCalls []notifyCall
	errorCalls  []notifyCall
}

type notifyCall struct {
	title   string
	message string
}

func (m *mockNotifier) Notify(title, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifyCalls = append(m.notifyCalls, notifyCall{title, message})
}

func (m *mockNotifier) NotifyError(title, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalls = append(m.errorCalls, notifyCall{title, message})
}

func (m *mockNotifier) notifyCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.notifyCalls)
}

func (m *mockNotifier) errorCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.errorCalls)
}

// setupProject creates a temp project with kanban structure and a task file in waiting
func setupProject(t *testing.T, taskFile string) string {
	t.Helper()
	projDir := t.TempDir()
	waitDir := filepath.Join(projDir, RelWaiting)
	if err := os.MkdirAll(waitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(waitDir, taskFile), []byte("task content"), 0644); err != nil {
		t.Fatal(err)
	}
	return projDir
}

// makeExecutor creates a CommandExecutor stub that returns the given exitCode/err
// and runs an optional side-effect function to simulate Claude behavior (file moves etc.)
func makeExecutor(exitCode int, err error, sideEffect func(projectPath, taskFile string)) CommandExecutor {
	return func(ctx context.Context, projectPath, taskFile string) (int, error) {
		if sideEffect != nil {
			sideEffect(projectPath, taskFile)
		}
		return exitCode, err
	}
}

func TestRunner_Run(t *testing.T) {
	taskFile := "T.md"

	t.Run("completed: executor returns 0 and file moved to done", func(t *testing.T) {
		projDir := setupProject(t, taskFile)
		locksDir := t.TempDir()
		notif := &mockNotifier{}

		executor := makeExecutor(0, nil, func(projPath, tf string) {
			// Simulate Claude moving file from running to done
			doneDir := filepath.Join(projPath, RelDone)
			os.MkdirAll(doneDir, 0755)
			os.Rename(
				filepath.Join(projPath, RelRunning, tf),
				filepath.Join(doneDir, tf),
			)
		})

		runner := NewRunner(Config{
			ProjectPath: projDir,
			TaskFile:    taskFile,
			LocksDir:    locksDir,
		}, notif, executor)

		result := runner.Run(context.Background())

		if result.Outcome != OutcomeCompleted {
			t.Errorf("outcome = %q, want %q", result.Outcome, OutcomeCompleted)
		}
		if notif.notifyCount() != 1 {
			t.Errorf("Notify called %d times, want 1", notif.notifyCount())
		}
		if notif.errorCount() != 0 {
			t.Errorf("NotifyError called %d times, want 0", notif.errorCount())
		}
	})

	t.Run("waiting_answer: executor returns 0, file in running with unanswered", func(t *testing.T) {
		projDir := setupProject(t, taskFile)
		locksDir := t.TempDir()
		notif := &mockNotifier{}

		executor := makeExecutor(0, nil, func(projPath, tf string) {
			// Write unanswered marker to the file in running dir
			runPath := filepath.Join(projPath, RelRunning, tf)
			os.WriteFile(runPath, []byte("**ステータス**: 未回答"), 0644)
		})

		runner := NewRunner(Config{
			ProjectPath: projDir,
			TaskFile:    taskFile,
			LocksDir:    locksDir,
		}, notif, executor)

		result := runner.Run(context.Background())

		if result.Outcome != OutcomeWaitingAnswer {
			t.Errorf("outcome = %q, want %q", result.Outcome, OutcomeWaitingAnswer)
		}
		if notif.notifyCount() != 1 {
			t.Errorf("Notify called %d times, want 1", notif.notifyCount())
		}
	})

	t.Run("abnormal: executor returns 1, file in running no marker", func(t *testing.T) {
		projDir := setupProject(t, taskFile)
		locksDir := t.TempDir()
		notif := &mockNotifier{}

		executor := makeExecutor(1, nil, nil) // file stays in running with original content

		runner := NewRunner(Config{
			ProjectPath: projDir,
			TaskFile:    taskFile,
			LocksDir:    locksDir,
		}, notif, executor)

		result := runner.Run(context.Background())

		if result.Outcome != OutcomeAbnormal {
			t.Errorf("outcome = %q, want %q", result.Outcome, OutcomeAbnormal)
		}
		if notif.errorCount() != 1 {
			t.Errorf("NotifyError called %d times, want 1", notif.errorCount())
		}
	})

	t.Run("needs_check: executor returns 0, file in running no marker", func(t *testing.T) {
		projDir := setupProject(t, taskFile)
		locksDir := t.TempDir()
		notif := &mockNotifier{}

		executor := makeExecutor(0, nil, nil) // file stays in running with original content

		runner := NewRunner(Config{
			ProjectPath: projDir,
			TaskFile:    taskFile,
			LocksDir:    locksDir,
		}, notif, executor)

		result := runner.Run(context.Background())

		if result.Outcome != OutcomeNeedsCheck {
			t.Errorf("outcome = %q, want %q", result.Outcome, OutcomeNeedsCheck)
		}
		if notif.notifyCount() != 1 {
			t.Errorf("Notify called %d times, want 1", notif.notifyCount())
		}
	})

	t.Run("start failure: executor returns -1 with error", func(t *testing.T) {
		projDir := setupProject(t, taskFile)
		locksDir := t.TempDir()
		notif := &mockNotifier{}

		executor := makeExecutor(-1, fmt.Errorf("command not found"), nil)

		runner := NewRunner(Config{
			ProjectPath: projDir,
			TaskFile:    taskFile,
			LocksDir:    locksDir,
		}, notif, executor)

		result := runner.Run(context.Background())

		if result.Outcome != OutcomeAbnormal {
			t.Errorf("outcome = %q, want %q", result.Outcome, OutcomeAbnormal)
		}
		if notif.errorCount() != 1 {
			t.Errorf("NotifyError called %d times, want 1", notif.errorCount())
		}
	})

	t.Run("lock busy: pre-acquired lock prevents run", func(t *testing.T) {
		projDir := setupProject(t, taskFile)
		locksDir := t.TempDir()
		notif := &mockNotifier{}

		// Pre-acquire the lock
		lockFile, ok, err := AcquireLock(locksDir, projDir)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("pre-acquire should succeed")
		}
		defer lockFile.Close()

		executorCalled := false
		executor := func(ctx context.Context, projectPath, tf string) (int, error) {
			executorCalled = true
			return 0, nil
		}

		runner := NewRunner(Config{
			ProjectPath: projDir,
			TaskFile:    taskFile,
			LocksDir:    locksDir,
		}, notif, executor)

		result := runner.Run(context.Background())

		if result.Outcome != OutcomeLockBusy {
			t.Errorf("outcome = %q, want %q", result.Outcome, OutcomeLockBusy)
		}
		if executorCalled {
			t.Error("executor should not have been called when lock is busy")
		}
		if notif.notifyCount() != 0 {
			t.Errorf("Notify called %d times, want 0", notif.notifyCount())
		}
		if notif.errorCount() != 0 {
			t.Errorf("NotifyError called %d times, want 0", notif.errorCount())
		}
	})

	t.Run("notifier=nil: no panic", func(t *testing.T) {
		projDir := setupProject(t, taskFile)
		locksDir := t.TempDir()

		executor := makeExecutor(0, nil, func(projPath, tf string) {
			doneDir := filepath.Join(projPath, RelDone)
			os.MkdirAll(doneDir, 0755)
			os.Rename(
				filepath.Join(projPath, RelRunning, tf),
				filepath.Join(doneDir, tf),
			)
		})

		runner := NewRunner(Config{
			ProjectPath: projDir,
			TaskFile:    taskFile,
			LocksDir:    locksDir,
		}, nil, executor) // nil notifier

		// Should not panic
		result := runner.Run(context.Background())

		if result.Outcome != OutcomeCompleted {
			t.Errorf("outcome = %q, want %q", result.Outcome, OutcomeCompleted)
		}
	})
}
