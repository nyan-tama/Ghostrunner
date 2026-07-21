package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// j は map をコンパクト JSON 文字列にします（テスト用ビルダー）。map の Marshal は失敗しないため error は無視します。
func j(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func asstText(ts, cwd, text string) string {
	return j(map[string]any{
		"type": "assistant", "cwd": cwd, "timestamp": ts,
		"message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "text", "text": text},
		}},
	})
}

func asstAsk(ts, cwd string, questions ...string) string {
	qs := make([]any, 0, len(questions))
	for _, q := range questions {
		qs = append(qs, map[string]any{"question": q})
	}
	return j(map[string]any{
		"type": "assistant", "cwd": cwd, "timestamp": ts,
		"message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "tool_use", "name": "AskUserQuestion", "input": map[string]any{"questions": qs}},
		}},
	})
}

func asstBash(ts, cwd string) string {
	return j(map[string]any{
		"type": "assistant", "cwd": cwd, "timestamp": ts,
		"message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "tool_use", "name": "Bash", "input": map[string]any{"command": "ls"}},
		}},
	})
}

func asstThinking(ts, cwd string) string {
	return j(map[string]any{
		"type": "assistant", "cwd": cwd, "timestamp": ts,
		"message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "thinking", "thinking": "hmm"},
		}},
	})
}

func userEntry(cwd string) string {
	return j(map[string]any{
		"type": "user", "cwd": cwd,
		"message": map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "tool_result", "content": "done"},
		}},
	})
}

func lastPromptEntry(cwd, prompt string) string {
	return j(map[string]any{"type": "last-prompt", "cwd": cwd, "lastPrompt": prompt})
}

func noiseEntry(typ, cwd string) string {
	return j(map[string]any{"type": typ, "cwd": cwd})
}

// writeLines は lines を1行1JSONの JSONL として一時ファイルに書き、パスを返します。
func writeLines(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write session: %v", err)
	}
	return path
}

func epoch(t *testing.T, ts string) int64 {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Fatalf("parse ts %q: %v", ts, err)
	}
	return parsed.Unix()
}

const cwd = "/Users/x/app"

// TestParseTail_Classification は最終実質エントリ種別ごとの待機判定を検証します（section1）。
func TestParseTail_Classification(t *testing.T) {
	tests := []struct {
		name              string
		lines             []string
		wantParseOK       bool
		wantWaiting       bool
		wantLastAssistant string
		wantLastPrompt    string
	}{
		{
			name:              "末尾assistant textで待機",
			lines:             []string{lastPromptEntry(cwd, "start"), asstText("2026-07-20T10:00:00Z", cwd, "hello")},
			wantParseOK:       true,
			wantWaiting:       true,
			wantLastAssistant: "hello",
			wantLastPrompt:    "start",
		},
		{
			name:              "assistant textの改行保持",
			lines:             []string{asstText("2026-07-20T10:00:00Z", cwd, "line1\nline2")},
			wantParseOK:       true,
			wantWaiting:       true,
			wantLastAssistant: "line1\nline2",
		},
		{
			name:              "末尾AskUserQuestion未応答で待機・質問文preview",
			lines:             []string{asstAsk("2026-07-20T10:00:00Z", cwd, "案Aと案Bどちら?", "追加で確認したい点は?")},
			wantParseOK:       true,
			wantWaiting:       true,
			wantLastAssistant: "案Aと案Bどちら?\n追加で確認したい点は?",
		},
		{
			name:        "通常tool_use(Bash)結果未着=busy=非待機",
			lines:       []string{asstBash("2026-07-20T10:00:00Z", cwd)},
			wantParseOK: true,
			wantWaiting: false,
		},
		{
			name:        "AskUserQuestionにuser(tool_result)が続く=応答済=非待機",
			lines:       []string{asstAsk("2026-07-20T10:00:00Z", cwd, "案Aと案Bどちら?"), userEntry(cwd)},
			wantParseOK: true,
			wantWaiting: false,
		},
		{
			name:        "末尾userは非待機",
			lines:       []string{asstText("2026-07-20T10:00:00Z", cwd, "hello"), userEntry(cwd)},
			wantParseOK: true,
			wantWaiting: false,
		},
		{
			name:              "末尾last-prompt帳簿は無視し直前assistant textで待機・LastPrompt抽出(allowlist)",
			lines:             []string{asstText("2026-07-20T10:00:00Z", cwd, "hello"), lastPromptEntry(cwd, "私の質問")},
			wantParseOK:       true,
			wantWaiting:       true,
			wantLastAssistant: "hello",
			wantLastPrompt:    "私の質問",
		},
		{
			name: "末尾ai-title/last-prompt帳簿を跨いで直前AskUserQuestionで待機(false-negative修正)",
			lines: []string{
				asstAsk("2026-07-20T10:00:00Z", cwd, "案Aと案Bどちら?"),
				noiseEntry("ai-title", cwd),
				lastPromptEntry(cwd, "私の質問"),
			},
			wantParseOK:       true,
			wantWaiting:       true,
			wantLastAssistant: "案Aと案Bどちら?",
			wantLastPrompt:    "私の質問",
		},
		{
			name: "ノイズ行を除外して直前assistantで待機判定(W1・permission-mode/worktree-state)",
			lines: []string{
				asstText("2026-07-20T10:00:00Z", cwd, "hello"),
				noiseEntry("file-history-snapshot", cwd),
				noiseEntry("permission-mode", cwd),
				noiseEntry("worktree-state", cwd),
			},
			wantParseOK:       true,
			wantWaiting:       true,
			wantLastAssistant: "hello",
		},
		{
			name:        "assistant thinking途中は非待機",
			lines:       []string{asstThinking("2026-07-20T10:00:00Z", cwd)},
			wantParseOK: true,
			wantWaiting: false,
		},
		{
			name: "壊れ行skip・直前完全エントリで判定",
			lines: []string{
				asstText("2026-07-20T09:00:00Z", cwd, "keep\nme"),
				`{"type":"assistant","message":{"role":"assist`, // partial JSON(生成途中の最終行想定)
			},
			wantParseOK:       true,
			wantWaiting:       true,
			wantLastAssistant: "keep\nme",
		},
		{
			name: "content非配列(公式非サポート形式)は保守的に非待機",
			lines: []string{
				`{"type":"assistant","timestamp":"2026-07-20T10:00:00Z","message":{"role":"assistant","content":"just a string"}}`,
			},
			wantParseOK: false,
			wantWaiting: false,
		},
		{
			name: "message欠落assistantは保守的に非待機",
			lines: []string{
				`{"type":"assistant","timestamp":"2026-07-20T10:00:00Z"}`,
			},
			wantParseOK: false,
			wantWaiting: false,
		},
		{
			name: "未知の帳簿type末尾は無視し直前assistantで待機(allowlist)",
			lines: []string{
				asstText("2026-07-20T10:00:00Z", cwd, "hello"),
				j(map[string]any{"type": "brand-new-unknown-type", "cwd": cwd}),
			},
			wantParseOK:       true,
			wantWaiting:       true,
			wantLastAssistant: "hello",
		},
		{
			name: "全行ノイズは実質エントリ0でParseOK=false",
			lines: []string{
				noiseEntry("file-history-snapshot", cwd),
				noiseEntry("system", cwd),
			},
			wantParseOK: false,
			wantWaiting: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeLines(t, tt.lines...)
			tail, err := parseTail(path)
			if err != nil {
				t.Fatalf("parseTail error: %v", err)
			}
			if tail.ParseOK != tt.wantParseOK {
				t.Errorf("ParseOK = %v, want %v", tail.ParseOK, tt.wantParseOK)
			}
			if tail.IsWaiting != tt.wantWaiting {
				t.Errorf("IsWaiting = %v, want %v", tail.IsWaiting, tt.wantWaiting)
			}
			if tail.LastAssistant != tt.wantLastAssistant {
				t.Errorf("LastAssistant = %q, want %q", tail.LastAssistant, tt.wantLastAssistant)
			}
			if tail.LastPrompt != tt.wantLastPrompt {
				t.Errorf("LastPrompt = %q, want %q", tail.LastPrompt, tt.wantLastPrompt)
			}
		})
	}
}

// TestParseTail_EmptyFile は空ファイルが ParseOK=false になることを検証します（section1 case10）。
func TestParseTail_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	tail, err := parseTail(path)
	if err != nil {
		t.Fatalf("parseTail error: %v", err)
	}
	if tail.ParseOK || tail.IsWaiting {
		t.Errorf("empty file: ParseOK=%v IsWaiting=%v, want both false", tail.ParseOK, tail.IsWaiting)
	}
}

// TestParseTail_CwdExtraction は cwd が最後の非空フィールドから取得されることを検証します。
func TestParseTail_CwdExtraction(t *testing.T) {
	lines := []string{
		asstText("2026-07-20T09:00:00Z", "/Users/x/old", "prev"),
		noiseEntry("file-history-snapshot", ""),
		asstText("2026-07-20T10:00:00Z", "/Users/x/app", "hello"),
	}
	path := writeLines(t, lines...)
	tail, err := parseTail(path)
	if err != nil {
		t.Fatalf("parseTail error: %v", err)
	}
	if tail.Cwd != "/Users/x/app" {
		t.Errorf("Cwd = %q, want /Users/x/app", tail.Cwd)
	}
}

// TestParseTail_FullReadFallback は tail 窓に実質エントリが無い（巨大 AskUserQuestion）場合に
// full-read で拾えることを検証します（W2・section1 case8）。
func TestParseTail_FullReadFallback(t *testing.T) {
	// 1行が tailReadBytes(128KB) を超える巨大 AskUserQuestion。末尾窓読みでは行の途中で
	// partial JSON になり実質エントリ0 → full-read fallback が発火する。
	needle := "NEEDLE_START"
	giant := needle + strings.Repeat("x", 200*1024)
	line := asstAsk("2026-07-20T10:00:00Z", cwd, giant)
	if len(line) <= tailReadBytes {
		t.Fatalf("giant line too small (%d bytes), test precondition broken", len(line))
	}
	path := writeLines(t, line)

	tail, err := parseTail(path)
	if err != nil {
		t.Fatalf("parseTail error: %v", err)
	}
	if !tail.ParseOK || !tail.IsWaiting {
		t.Fatalf("full-read fallback failed: ParseOK=%v IsWaiting=%v", tail.ParseOK, tail.IsWaiting)
	}
	if !strings.HasPrefix(tail.LastAssistant, needle) {
		t.Errorf("LastAssistant did not start with needle; got prefix %q", tail.LastAssistant[:min(len(tail.LastAssistant), 20)])
	}
}

// TestParseTail_EntryTimeStability は Marker.Timestamp 用 entry-time の安定性を検証します（C1・section2）。
func TestParseTail_EntryTimeStability(t *testing.T) {
	t.Run("entry-time=最終assistantのtimestamp", func(t *testing.T) {
		ts := "2026-07-20T10:00:00Z"
		path := writeLines(t, asstText(ts, cwd, "hello"))
		tail, err := parseTail(path)
		if err != nil {
			t.Fatalf("parseTail: %v", err)
		}
		if tail.LastAssistantAt != epoch(t, ts) {
			t.Errorf("LastAssistantAt = %d, want %d", tail.LastAssistantAt, epoch(t, ts))
		}
	})

	t.Run("ノイズ追記でmtime変化してもentry-time不変", func(t *testing.T) {
		ts := "2026-07-20T10:00:00Z"
		dir := t.TempDir()
		path := filepath.Join(dir, "s.jsonl")
		if err := os.WriteFile(path, []byte(asstText(ts, cwd, "hello")+"\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		first, err := parseTail(path)
		if err != nil {
			t.Fatalf("parseTail 1: %v", err)
		}
		e0 := first.LastAssistantAt

		// ノイズ行を追記してファイル mtime を進める
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatalf("open append: %v", err)
		}
		if _, err := f.WriteString(noiseEntry("file-history-snapshot", cwd) + "\n" + noiseEntry("worktree-state", cwd) + "\n"); err != nil {
			t.Fatalf("append: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}

		second, err := parseTail(path)
		if err != nil {
			t.Fatalf("parseTail 2: %v", err)
		}
		if !second.IsWaiting {
			t.Errorf("still waiting expected after noise append")
		}
		if second.LastAssistantAt != e0 {
			t.Errorf("entry-time changed after noise append: %d -> %d (must be mtime-independent)", e0, second.LastAssistantAt)
		}
	})

	t.Run("新assistantエントリでentry-time前進", func(t *testing.T) {
		ts0 := "2026-07-20T10:00:00Z"
		ts1 := "2026-07-20T10:05:00Z"
		path := writeLines(t, asstText(ts0, cwd, "old"), asstText(ts1, cwd, "new"))
		tail, err := parseTail(path)
		if err != nil {
			t.Fatalf("parseTail: %v", err)
		}
		if tail.LastAssistantAt != epoch(t, ts1) {
			t.Errorf("LastAssistantAt = %d, want %d (advanced to new entry)", tail.LastAssistantAt, epoch(t, ts1))
		}
		if tail.LastAssistant != "new" {
			t.Errorf("LastAssistant = %q, want new", tail.LastAssistant)
		}
	})

	t.Run("timestamp欠落は本文署名で安定(raw mtime非依存)", func(t *testing.T) {
		// timestamp フィールドの無い assistant text
		line := `{"type":"assistant","cwd":"/Users/x/app","message":{"role":"assistant","content":[{"type":"text","text":"same body"}]}}`
		path := writeLines(t, line)
		a, err := parseTail(path)
		if err != nil {
			t.Fatalf("parseTail a: %v", err)
		}
		if a.LastAssistantAt != 0 {
			t.Errorf("LastAssistantAt = %d, want 0 (timestamp欠落)", a.LastAssistantAt)
		}
		if a.ContentHash == "" {
			t.Fatalf("ContentHash empty; expected fnv signature for stable key")
		}
		// 同一本文なら別ファイルでも同一署名（安定キー）
		path2 := writeLines(t, line)
		b, err := parseTail(path2)
		if err != nil {
			t.Fatalf("parseTail b: %v", err)
		}
		if a.ContentHash != b.ContentHash {
			t.Errorf("ContentHash unstable for same body: %q != %q", a.ContentHash, b.ContentHash)
		}
	})
}
