// Package service はビジネスロジックを提供します
package service

// ClaudeResponse はClaude CLIのJSON出力を表します
type ClaudeResponse struct {
	SessionID          string              `json:"session_id"`
	Result             string              `json:"result,omitempty"`
	PermissionDenials  []PermissionDenial  `json:"permission_denials,omitempty"`
	CostUSD            float64             `json:"cost_usd,omitempty"`
	TotalCostUSD       float64             `json:"total_cost_usd,omitempty"`
	IsError            bool                `json:"is_error,omitempty"`
	DurationMS         int                 `json:"duration_ms,omitempty"`
	DurationAPIMS      int                 `json:"duration_api_ms,omitempty"`
	NumTurns           int                 `json:"num_turns,omitempty"`
}

// PermissionDenial は拒否されたツール呼び出しを表します
type PermissionDenial struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// Question はAskUserQuestionの質問を表します
type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header"`
	Options     []Option `json:"options"`
	MultiSelect bool     `json:"multiSelect"`
}

// Option は質問の選択肢を表します
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// PlanResult はExecutePlanの結果を表します
type PlanResult struct {
	SessionID  string     `json:"session_id"`            // セッションID（継続用）
	Output     string     `json:"output"`                // 出力テキスト
	Questions  []Question `json:"questions,omitempty"`   // 質問がある場合
	Completed  bool       `json:"completed"`             // 完了したかどうか
	CostUSD    float64    `json:"cost_usd,omitempty"`    // コスト
}

// StreamEvent はストリーミングイベントを表します
type StreamEvent struct {
	Type      string      `json:"type"`                 // イベントタイプ
	SessionID string      `json:"session_id,omitempty"` // セッションID
	Message   string      `json:"message,omitempty"`    // メッセージ
	ToolName  string      `json:"tool_name,omitempty"`  // ツール名
	ToolInput interface{} `json:"tool_input,omitempty"` // ツール入力
	Result    *PlanResult `json:"result,omitempty"`     // 最終結果
}

// StreamEventType はストリーミングイベントのタイプを定義します
const (
	EventTypeInit       = "init"        // セッション開始
	EventTypeThinking   = "thinking"    // 思考中
	EventTypeToolUse    = "tool_use"    // ツール使用開始
	EventTypeToolResult = "tool_result" // ツール使用完了
	EventTypeText       = "text"        // テキスト出力
	EventTypeQuestion   = "question"    // 質問
	EventTypeComplete   = "complete"    // 完了
	EventTypeError      = "error"       // エラー
)

// ClaudeStreamMessage はClaude CLIのstream-json出力の1行を表します
type ClaudeStreamMessage struct {
	Type       string                 `json:"type"`
	SessionID  string                 `json:"session_id,omitempty"`
	Message    map[string]interface{} `json:"message,omitempty"`
	ToolName   string                 `json:"tool_name,omitempty"`
	ToolInput  map[string]interface{} `json:"tool_input,omitempty"`
	ToolResult string                 `json:"tool_result,omitempty"`
	Content    string                 `json:"content,omitempty"`
	Result     *ClaudeResponse        `json:"result,omitempty"`
}
