export interface ChatSession {
  id: string;
  cwd?: string;
  provider?: string;
  createdAt?: string;
  title?: string;
  timestamp?: string;
  status?: string | null;
}

// 履歴 1 行（even-terminal の getHistory レスポンス想定）
export interface ChatHistoryItem {
  role?: string;
  text?: string;
  type?: string;
  [key: string]: unknown;
}

export interface PromptRequest {
  sessionId: string;
  text: string;
  provider?: string;
  cwd?: string;
}

// even-terminal Claude provider が emit するイベント型
export type ChatStreamEvent =
  | { type: "text_delta"; text: string; sessionId?: string }
  | { type: "result"; sessionId?: string; [key: string]: unknown }
  | { type: "status"; state: "busy" | "idle"; sessionId?: string }
  | { type: "error"; message: string; sessionId?: string }
  | { type: "user_prompt"; text?: string; sessionId?: string }
  | { type: "running_stats"; sessionId?: string; [key: string]: unknown }
  | { type: "tool_start"; tool?: string; sessionId?: string; [key: string]: unknown }
  | { type: "tool_end"; tool?: string; sessionId?: string; [key: string]: unknown }
  | { type: "task_progress"; sessionId?: string; [key: string]: unknown }
  | { type: "notification"; message?: string; sessionId?: string; [key: string]: unknown }
  | { type: "user_question"; sessionId?: string; [key: string]: unknown }
  | { type: "permission_request"; sessionId?: string; [key: string]: unknown };
