package transcript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// tailReadBytes は会話ログ JSONL の末尾から読み取るバイト数です。
// 待機判定は最終実質エントリのみで足りるため全読みは避け、末尾のみを読みます。
// AskUserQuestion の長い input.questions を1エントリで含められるよう十分大きくします（W2）。
const tailReadBytes = 128 * 1024

// noiseEntryTypes は待機判定で無視する非実質エントリ（bookkeeping）の type 集合です。
// これらは待機中でも追記されうる（file-history-snapshot・ユーザーの shift+tab による
// permission-mode 切替等）ため、「最終実質エントリ」の判定から除外します。
// noise 集合に無い type は「未知＝実質エントリ」扱いになり、text 待機中に追記されると
// 非待機と誤判定して質問待ちカードが消える（false-negative）ため、実在する bookkeeping 型は
// 漏れなく列挙します。
//
// 運用注記: 会話ログは公式非サポート形式（バージョンで変わる）。新しい bookkeeping type
// （*-mode / *-state 等）が現れたら、本集合への追加が必要になる（未追加は待機取りこぼしに直結）。
var noiseEntryTypes = map[string]struct{}{
	"attachment":            {},
	"queue-operation":       {},
	"file-history-snapshot": {},
	"file-history-delta":    {},
	"system":                {},
	"ai-title":              {},
	"mode":                  {},
	"permission-mode":       {},
	"worktree-state":        {},
}

// transcriptTail は会話ログ末尾の待機判定結果です。
// 会話ログは公式非サポート形式のため best-effort で解釈し、
// 解釈不能時は ParseOK=false として呼び出し側が保守的（非待機）に倒せるようにします。
type transcriptTail struct {
	// LastAssistant は待機中の場合の末尾 assistant テキスト、または AskUserQuestion の質問文です。
	LastAssistant string
	// LastPrompt は最後の last-prompt エントリのユーザー発言です（要約の材料）。
	LastPrompt string
	// LastAssistantAt は最後の assistant エントリ自身の timestamp（epoch秒・C1）です。
	// mtime ではなく待機 episode の安定同一性を担保します。取得できなければ 0 です。
	LastAssistantAt int64
	// ContentHash は LastAssistantAt が取得できない版での安定キー用の本文署名です。
	// 呼び出し側が「同一署名なら初回検出時刻を保持」してキーを安定化します（raw mtime fallback 禁止）。
	ContentHash string
	// Cwd はセッションの実 cwd です（帰属判定に MatchProject で使用・C2）。
	Cwd string
	// IsWaiting は最終実質エントリが未応答の assistant（質問待ち）かを表します。
	IsWaiting bool
	// ParseOK は待機状態を判定できたかを表します。false のとき呼び出し側は保守的に非待機扱いにします。
	ParseOK bool
}

// logEntry は JSONL 1行の共通フィールドを表します（必要な範囲のみ）。
type logEntry struct {
	Type       string      `json:"type"`
	Cwd        string      `json:"cwd"`
	Timestamp  string      `json:"timestamp"`
	LastPrompt string      `json:"lastPrompt"`
	Message    *logMessage `json:"message"`
}

// logMessage は assistant/user エントリの message 部を表します。
type logMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// contentItem は message.content[] の1要素を表します。
type contentItem struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// parseTail は会話ログ末尾を読み、最終実質エントリから待機状態を判定します。
// まず末尾 tailReadBytes だけを読み、窓内に実質エントリが1つも無ければ
// full-read で再走査します（W2）。読み取り自体に失敗した場合のみ error を返します。
func parseTail(path string) (transcriptTail, error) {
	data, err := readTailBytes(path, tailReadBytes)
	if err != nil {
		return transcriptTail{}, fmt.Errorf("failed to read transcript tail %s: %w", path, err)
	}

	tail, found := analyzeEntries(data)
	if found {
		return tail, nil
	}

	// W2: tail 窓に実質エントリが無い（巨大な input.questions やノイズで埋まった等）→ 全読み再走査。
	full, err := os.ReadFile(path)
	if err != nil {
		return transcriptTail{}, fmt.Errorf("failed to read transcript %s: %w", path, err)
	}
	tail, _ = analyzeEntries(full)
	return tail, nil
}

// readTailBytes はファイル末尾から最大 n バイトを読み取ります。
// n 以上のファイルは末尾 n バイトのみを ReadAt で読みます（先頭の途中行は呼び出し側で skip）。
func readTailBytes(path string, n int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("[transcript] failed to close transcript file: path=%s, error=%v", path, cerr)
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size <= n {
		return io.ReadAll(f)
	}

	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, size-n); err != nil && err != io.EOF {
		return nil, err
	}
	return buf, nil
}

// analyzeEntries は JSONL バイト列を行単位で解釈し、最終実質エントリから待機状態を判定します。
// 壊れ行は skip します。found は実質エントリを1つ以上見つけたかを表し、
// false のとき呼び出し側は full-read fallback を行います。
func analyzeEntries(data []byte) (tail transcriptTail, found bool) {
	lines := bytes.Split(data, []byte("\n"))

	var (
		lastSub    *logEntry // 最終実質エントリ（ノイズ除外後）
		lastPrompt string
		cwd        string
	)

	for _, raw := range lines {
		line := bytes.TrimSpace(raw)
		if len(line) == 0 {
			continue
		}

		var e logEntry
		if err := json.Unmarshal(line, &e); err != nil {
			// 壊れ行（先頭の途中行・書き込み途中の最終行等）は skip
			continue
		}

		if e.Cwd != "" {
			cwd = e.Cwd
		}

		if e.Type == "last-prompt" {
			lastPrompt = e.LastPrompt
			entry := e
			lastSub = &entry
			continue
		}

		if _, noise := noiseEntryTypes[e.Type]; noise {
			continue
		}

		// assistant / user / 未知の非ノイズ type はすべて実質エントリ扱い
		entry := e
		lastSub = &entry
	}

	tail.LastPrompt = lastPrompt
	tail.Cwd = cwd

	if lastSub == nil {
		// 実質エントリ皆無 → full-read fallback を促す
		tail.ParseOK = false
		return tail, false
	}

	if lastSub.Type != "assistant" {
		// user / last-prompt / 未知 type が最終 → 非待機（確定）
		tail.ParseOK = true
		tail.IsWaiting = false
		return tail, true
	}

	// 最終が assistant: content の末尾要素で待機/busy を判定
	if lastSub.Message == nil {
		// message 欠落で解釈不能 → 保守的に非待機
		tail.ParseOK = false
		return tail, true
	}

	isWaiting, text, parsed := classifyAssistant(lastSub.Message.Content)
	if !parsed {
		// content 解釈不能 → 保守的に非待機
		tail.ParseOK = false
		return tail, true
	}

	tail.ParseOK = true
	tail.IsWaiting = isWaiting
	if isWaiting {
		tail.LastAssistant = text
		tail.LastAssistantAt, tail.ContentHash = entryTimeOrHash(lastSub.Timestamp, text)
	}
	return tail, true
}

// classifyAssistant は assistant の message.content から待機/busy を判定します（W1）。
//   - 末尾が text → 待機（回答提示後のユーザー返信待ち）
//   - 末尾が AskUserQuestion tool_use → 待機（質問文を preview に）
//   - 末尾が通常 tool_use（Bash/Edit 等）で結果未着 → busy（非待機）
//   - 末尾が thinking 等 → 生成途中とみなし非待機（保守的）
//
// parsed=false は content が解釈不能（配列でない・空）で ParseOK=false に倒すべき場合です。
func classifyAssistant(content json.RawMessage) (isWaiting bool, text string, parsed bool) {
	var items []contentItem
	if err := json.Unmarshal(content, &items); err != nil {
		return false, "", false
	}
	if len(items) == 0 {
		return false, "", false
	}

	last := items[len(items)-1]
	switch last.Type {
	case "text":
		return true, last.Text, true
	case "tool_use":
		if last.Name == "AskUserQuestion" {
			q := extractQuestions(last.Input)
			if q == "" {
				// 質問文が取れなければ直前の text を preview に流用
				q = lastTextBefore(items)
			}
			return true, q, true
		}
		// 通常 tool_use は結果未着 = busy（非待機）
		return false, "", true
	default:
		// thinking 等の生成途中 → 非待機
		return false, "", true
	}
}

// extractQuestions は AskUserQuestion の input.questions[].question を改行連結します。
func extractQuestions(input json.RawMessage) string {
	var in struct {
		Questions []struct {
			Question string `json:"question"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return ""
	}
	questions := make([]string, 0, len(in.Questions))
	for _, q := range in.Questions {
		if q.Question != "" {
			questions = append(questions, q.Question)
		}
	}
	return strings.Join(questions, "\n")
}

// lastTextBefore は content 内の最後の text 要素を返します（AskUserQuestion の preview fallback）。
func lastTextBefore(items []contentItem) string {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Type == "text" && items[i].Text != "" {
			return items[i].Text
		}
	}
	return ""
}

// entryTimeOrHash は assistant エントリの timestamp を epoch秒に変換します（C1）。
// timestamp が無い/解釈不能な版では、raw mtime に頼らず本文の fnv 署名を返し、
// 呼び出し側が「同一署名なら初回検出時刻を保持」してキーを安定化します。
func entryTimeOrHash(timestamp, text string) (epoch int64, contentHash string) {
	if timestamp != "" {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			return t.Unix(), ""
		}
	}
	h := fnv.New64a()
	// Write は error を返さない実装のため戻り値は無視して問題ない
	_, _ = h.Write([]byte(text))
	return 0, fmt.Sprintf("%x", h.Sum64())
}
