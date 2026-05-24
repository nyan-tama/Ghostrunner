package grrun

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// --- ClaimTask tests ---

func TestClaimTask(t *testing.T) {
	t.Run("normal claim moves file to running", func(t *testing.T) {
		projDir := t.TempDir()
		waitDir := filepath.Join(projDir, RelWaiting)
		runDir := filepath.Join(projDir, RelRunning)

		if err := os.MkdirAll(waitDir, 0755); err != nil {
			t.Fatal(err)
		}
		taskFile := "T.md"
		if err := os.WriteFile(filepath.Join(waitDir, taskFile), []byte("task content"), 0644); err != nil {
			t.Fatal(err)
		}

		err := ClaimTask(projDir, taskFile)
		if err != nil {
			t.Fatalf("ClaimTask error: %v", err)
		}

		// File should be in running, not in waiting
		if _, err := os.Stat(filepath.Join(runDir, taskFile)); err != nil {
			t.Errorf("file not found in running dir: %v", err)
		}
		if _, err := os.Stat(filepath.Join(waitDir, taskFile)); err == nil {
			t.Error("file still exists in waiting dir")
		}
	})

	t.Run("running dir already exists", func(t *testing.T) {
		projDir := t.TempDir()
		waitDir := filepath.Join(projDir, RelWaiting)
		runDir := filepath.Join(projDir, RelRunning)

		if err := os.MkdirAll(waitDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(runDir, 0755); err != nil {
			t.Fatal(err)
		}
		taskFile := "T.md"
		if err := os.WriteFile(filepath.Join(waitDir, taskFile), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		err := ClaimTask(projDir, taskFile)
		if err != nil {
			t.Fatalf("ClaimTask error when running dir exists: %v", err)
		}

		if _, err := os.Stat(filepath.Join(runDir, taskFile)); err != nil {
			t.Errorf("file not found in running dir: %v", err)
		}
	})

	t.Run("source missing returns error", func(t *testing.T) {
		projDir := t.TempDir()

		err := ClaimTask(projDir, "nonexistent.md")
		if err == nil {
			t.Fatal("expected error for missing source, got nil")
		}
	})
}

// --- ClassifyResult tests ---

func TestClassifyResult(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		// setupFn places the task file and writes content
		setupFn  func(projDir, taskFile string)
		expected Outcome
	}{
		{
			name:     "exit=0, file in done -> completed",
			exitCode: 0,
			setupFn: func(projDir, taskFile string) {
				doneDir := filepath.Join(projDir, RelDone)
				os.MkdirAll(doneDir, 0755)
				os.WriteFile(filepath.Join(doneDir, taskFile), []byte("done"), 0644)
			},
			expected: OutcomeCompleted,
		},
		{
			name:     "exit=0, file in running with unanswered -> waiting_answer",
			exitCode: 0,
			setupFn: func(projDir, taskFile string) {
				runDir := filepath.Join(projDir, RelRunning)
				os.MkdirAll(runDir, 0755)
				os.WriteFile(filepath.Join(runDir, taskFile), []byte("**ステータス**: 未回答"), 0644)
			},
			expected: OutcomeWaitingAnswer,
		},
		{
			name:     "exit=1, file in running with unanswered -> waiting_answer (confirmation priority)",
			exitCode: 1,
			setupFn: func(projDir, taskFile string) {
				runDir := filepath.Join(projDir, RelRunning)
				os.MkdirAll(runDir, 0755)
				os.WriteFile(filepath.Join(runDir, taskFile), []byte("**ステータス**: 未回答"), 0644)
			},
			expected: OutcomeWaitingAnswer,
		},
		{
			name:     "exit=1, file in running no marker -> abnormal",
			exitCode: 1,
			setupFn: func(projDir, taskFile string) {
				runDir := filepath.Join(projDir, RelRunning)
				os.MkdirAll(runDir, 0755)
				os.WriteFile(filepath.Join(runDir, taskFile), []byte("no markers here"), 0644)
			},
			expected: OutcomeAbnormal,
		},
		{
			name:     "exit=0, file in running no marker -> needs_check",
			exitCode: 0,
			setupFn: func(projDir, taskFile string) {
				runDir := filepath.Join(projDir, RelRunning)
				os.MkdirAll(runDir, 0755)
				os.WriteFile(filepath.Join(runDir, taskFile), []byte("normal content"), 0644)
			},
			expected: OutcomeNeedsCheck,
		},
		{
			name:     "exit=137 (SIGKILL), file in running no marker -> abnormal",
			exitCode: 137,
			setupFn: func(projDir, taskFile string) {
				runDir := filepath.Join(projDir, RelRunning)
				os.MkdirAll(runDir, 0755)
				os.WriteFile(filepath.Join(runDir, taskFile), []byte("killed process"), 0644)
			},
			expected: OutcomeAbnormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projDir := t.TempDir()
			taskFile := "T.md"
			tt.setupFn(projDir, taskFile)

			result := ClassifyResult(projDir, taskFile, tt.exitCode)
			if result != tt.expected {
				t.Errorf("ClassifyResult() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// --- hasUnansweredQuestion tests ---

func TestHasUnansweredQuestion(t *testing.T) {
	tests := []struct {
		name    string
		content string // empty string means file should not be created
		noFile  bool
		want    bool
	}{
		{
			name:    "normal form detected",
			content: "**ステータス**: 未回答",
			want:    true,
		},
		{
			name:    "answered not detected",
			content: "**ステータス**: 回答済",
			want:    false,
		},
		{
			name:    "mixed: 1 answered + 1 unanswered -> true",
			content: "**ステータス**: 回答済\n**ステータス**: 未回答",
			want:    true,
		},
		{
			name:    "no bold not detected",
			content: "ステータス: 未回答",
			want:    false,
		},
		{
			name:    "value variation not detected (space in value)",
			content: "**ステータス**: 未 回答",
			want:    false,
		},
		{
			name:    "space tolerance: 0 spaces after colon",
			content: "**ステータス**:未回答",
			want:    true,
		},
		{
			name:    "space tolerance: 2 spaces after colon",
			content: "**ステータス**:  未回答",
			want:    true,
		},
		{
			name:   "file not found returns false",
			noFile: true,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.noFile {
				path = filepath.Join(t.TempDir(), "nonexistent.md")
			} else {
				dir := t.TempDir()
				path = filepath.Join(dir, "plan.md")
				if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := hasUnansweredQuestion(path)
			if got != tt.want {
				t.Errorf("hasUnansweredQuestion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasUnansweredQuestion_SSOTMatch(t *testing.T) {
	// Read the chief-director.md to extract the SSOT pattern
	chiefDirectorPath := "/Users/user/Ghostrunner/.claude/agents/chief-director.md"
	data, err := os.ReadFile(chiefDirectorPath)
	if err != nil {
		t.Skipf("chief-director.md not found, skipping SSOT check: %v", err)
	}

	content := string(data)

	// Extract the pattern from the file: \*\*ステータス\*\*:\s*未回答
	// The pattern appears in the file as documented
	extractRe := regexp.MustCompile(`\\` + `\*\\` + `\*ステータス\\` + `\*\\` + `\*:\\s\*未回答`)
	if !extractRe.MatchString(content) {
		// Try a more flexible extraction: look for the literal pattern string
		if !strings.Contains(content, `\*\*ステータス\*\*:\s*未回答`) {
			t.Fatalf("could not find SSOT pattern in chief-director.md")
		}
	}

	// The UnansweredPattern constant must match the pattern documented in chief-director.md
	expectedPattern := `\*\*ステータス\*\*:\s*未回答`
	if UnansweredPattern != expectedPattern {
		t.Errorf("UnansweredPattern = %q, want %q (SSOT mismatch with chief-director.md)", UnansweredPattern, expectedPattern)
	}
}
