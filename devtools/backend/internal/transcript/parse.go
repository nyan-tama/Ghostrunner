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

// substantiveEntryTypes は会話ターンを構成する「実質エントリ」の type 集合（allowlist）です。
// 会話ターンは user と assistant のみが実質で、待機判定はこの2種の「最終実質エントリ」だけで行います。
//
// これ以外の type（attachment / queue-operation / file-history-snapshot / file-history-delta /
// system / permission-mode / worktree-state / ai-title / last-prompt / その他未知の帳簿型）は
// 全て bookkeeping として無視します。会話ログ末尾には実会話ターンの後に ai-title（自動タイトル）や
// last-prompt（最終プロンプト記録）等の帳簿エントリが追記されうるため、denylist で個別に除外すると
// 新しい帳簿型が増えるたびに列挙漏れ→「未知＝実質」誤判定で待機を取りこぼす（false-negative）。
// allowlist なら新しい帳簿型が現れても自動で無視され、この種の取りこぼしが構造的に根絶されます。
var substantiveEntryTypes = map[string]struct{}{
	"user":      {},
	"assistant": {},
}

// tailKind は最終実質エントリの種別です。mtime 鮮度の合成前の「内容だけ」の分類で、
// reader が classifyRepresentative で mtime を合わせて最終 status（waiting/running/none）を確定します。
type tailKind int

const (
	// kindNone は応答済/解釈不能/実質エントリ皆無（ParseOK=false 含む）を表します。
	kindNone tailKind = iota
	// kindWaiting は末尾が未応答 assistant（text / AskUserQuestion）で質問待ちを表します。
	kindWaiting
	// kindMidTurn は生成途中（未応答通常tool_use / thinking / user末尾で assistant 未応答）を表します。
	kindMidTurn
)

// transcriptTail は会話ログ末尾の種別判定結果です。
// 会話ログは公式非サポート形式のため best-effort で解釈し、
// 解釈不能時は ParseOK=false / Kind=kindNone として呼び出し側が保守的に倒せるようにします。
type transcriptTail struct {
	// LastAssistant は kindWaiting なら末尾 assistant テキスト/質問文、kindMidTurn なら
	// running preview 材料（直前の text 要素、無ければ空・W-5）です。
	LastAssistant string
	// LastPrompt は最後の last-prompt エントリのユーザー発言です（要約の材料）。
	LastPrompt string
	// LastAssistantAt は最後の assistant エントリ自身の timestamp（epoch秒・C1）です。
	// mtime ではなく待機 episode の安定同一性を担保します。取得できなければ 0 です。
	// kindWaiting でのみ設定します（running の Marker.Timestamp は reader が mtime を使う）。
	LastAssistantAt int64
	// ContentHash は LastAssistantAt が取得できない版での安定キー用の本文署名です。
	// 呼び出し側が「同一署名なら初回検出時刻を保持」してキーを安定化します（raw mtime fallback 禁止）。
	ContentHash string
	// Cwd はセッションの実 cwd です（帰属判定に MatchProject で使用・C2）。
	Cwd string
	// Kind は最終実質エントリの種別（waiting/midTurn/none）です。
	Kind tailKind
	// ParseOK は種別を判定できたかを表します。false のとき Kind は kindNone です。
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
		lastSub      *logEntry // 最終実質エントリ（allowlist=user/assistant のみ）
		lastPrompt   string
		cwd          string
		lastAsstText string // tail 内で最後に見た assistant テキスト（running preview 材料・W-5改）
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

		// LastPrompt は last-prompt 帳簿エントリから抽出します（要約の材料）。
		// これは「最終実質エントリ判定」とは独立で、last-prompt 自体は実質エントリではありません。
		if e.Type == "last-prompt" {
			lastPrompt = e.LastPrompt
			continue
		}

		// allowlist: 会話ターンは user / assistant のみが実質エントリ。
		// それ以外（ai-title / last-prompt / *-mode / *-state / 未知の帳簿型）は全て無視します。
		if _, ok := substantiveEntryTypes[e.Type]; !ok {
			continue
		}

		entry := e
		lastSub = &entry

		// running(midTurn) preview 用に、tail 全体で最後に見た assistant テキストを保持する。
		// 最終実質が tool_use や user(tool_result) でも、直近の「Claude が最後に言ったこと」を
		// preview に出せるようにする（waiting と同じ最後の assistant テキストブロックを流用）。
		if e.Type == "assistant" && e.Message != nil {
			if txt := lastAssistantText(e.Message.Content); txt != "" {
				lastAsstText = txt
			}
		}
	}

	tail.LastPrompt = lastPrompt
	tail.Cwd = cwd

	if lastSub == nil {
		// 実質エントリ皆無 → full-read fallback を促す
		tail.ParseOK = false
		tail.Kind = kindNone
		return tail, false
	}

	if lastSub.Type != "assistant" {
		// 最終実質が user（実ユーザー返信 または tool_result）で assistant 未応答 → midTurn。
		// Claude がこれから応答生成する途中とみなす。放置/クラッシュは reader の mtime 鮮度上限で none へ落ちる。
		// preview は直近の assistant テキスト（何をやっているか）を出す。無ければ空。
		tail.ParseOK = true
		tail.Kind = kindMidTurn
		tail.LastAssistant = lastAsstText
		return tail, true
	}

	// 最終が assistant: content の末尾要素で種別（waiting/midTurn）を判定
	if lastSub.Message == nil {
		// message 欠落で解釈不能 → 保守的に none
		tail.ParseOK = false
		tail.Kind = kindNone
		return tail, true
	}

	kind, text, parsed := classifyAssistant(lastSub.Message.Content)
	if !parsed {
		// content 解釈不能 → 保守的に none
		tail.ParseOK = false
		tail.Kind = kindNone
		return tail, true
	}

	tail.ParseOK = true
	tail.Kind = kind
	if kind == kindWaiting {
		// waiting は分類済みテキスト（末尾 text または質問文）をそのまま preview に使う。
		tail.LastAssistant = text
		// entry-time は待機 episode の安定同一性（要約 key）に使う（C1）。running は reader が mtime を使う。
		tail.LastAssistantAt, tail.ContentHash = entryTimeOrHash(lastSub.Timestamp, text)
	} else {
		// midTurn（末尾 tool_use / thinking）: running preview は tail 全体で最後に見た
		// assistant テキストを使う（同一 assistant 内に text が無くても直近の発言を出せる）。
		tail.LastAssistant = lastAsstText
	}
	return tail, true
}

// classifyAssistant は assistant の message.content から種別を判定します。
//   - 末尾が text → kindWaiting（回答提示後のユーザー返信待ち。text を preview に）
//   - 末尾が AskUserQuestion tool_use → kindWaiting（質問文を preview に）
//   - 末尾が通常 tool_use（Bash/Edit 等）で結果未着 → kindMidTurn（生成途中・running 候補）
//   - 末尾が thinking 等 → kindMidTurn（生成途中・running 候補）
//
// text は kindWaiting でのみ意味を持ちます（末尾 text または質問文）。midTurn の running preview は
// 呼び出し側が tail 全体で最後に見た assistant テキストを使うため、ここでは "" を返します（W-5改）。
// parsed=false は content が解釈不能（配列でない・空）で ParseOK=false / kindNone に倒すべき場合です。
func classifyAssistant(content json.RawMessage) (kind tailKind, text string, parsed bool) {
	var items []contentItem
	if err := json.Unmarshal(content, &items); err != nil {
		return kindNone, "", false
	}
	if len(items) == 0 {
		return kindNone, "", false
	}

	last := items[len(items)-1]
	switch last.Type {
	case "text":
		return kindWaiting, last.Text, true
	case "tool_use":
		if last.Name == "AskUserQuestion" {
			q := extractQuestions(last.Input)
			if q == "" {
				// 質問文が取れなければ直前の text を preview に流用
				q = lastTextBefore(items)
			}
			return kindWaiting, q, true
		}
		// 通常 tool_use は結果未着 = 生成途中 → midTurn（preview は呼び出し側が lastAsstText を使う）
		return kindMidTurn, "", true
	default:
		// thinking 等の生成途中 → midTurn（preview は呼び出し側が lastAsstText を使う）
		return kindMidTurn, "", true
	}
}

// lastAssistantText は assistant の message.content から最後の text ブロックを返します。
// running preview 材料として、tool_use / thinking で終わる assistant メッセージでも
// その中に含まれる直近の発言テキストを取り出します。text が無ければ空を返します。
func lastAssistantText(content json.RawMessage) string {
	var items []contentItem
	if err := json.Unmarshal(content, &items); err != nil {
		return ""
	}
	return lastTextBefore(items)
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
