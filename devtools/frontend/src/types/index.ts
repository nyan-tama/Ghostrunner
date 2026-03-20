export interface Option {
  label: string;
  description: string;
}

export interface Question {
  question: string;
  header: string;
  options: Option[];
  multiSelect: boolean;
}

export interface CommandResult {
  session_id: string;
  output: string;
  questions?: Question[];
  completed: boolean;
  cost_usd?: number;
}

export interface StreamEvent {
  type: EventType;
  session_id?: string;
  message?: string;
  tool_name?: string;
  tool_input?: ToolInput;
  result?: CommandResult;
}

export type EventType =
  | "init"
  | "thinking"
  | "tool_use"
  | "tool_result"
  | "text"
  | "question"
  | "complete"
  | "error";

export interface ToolInput {
  file_path?: string;
  offset?: number;
  limit?: number;
  content?: string;
  old_string?: string;
  new_string?: string;
  pattern?: string;
  path?: string;
  glob?: string;
  command?: string;
  description?: string;
  prompt?: string;
  subagent_type?: string;
  todos?: unknown[];
  url?: string;
  query?: string;
}

export interface FileInfo {
  name: string;
  path: string;
}

export interface FilesResponse {
  success: boolean;
  files?: Record<string, FileInfo[]>;
  error?: string;
}

export interface ProjectInfo {
  name: string;
  path: string;
}

export interface ProjectsResponse {
  success: boolean;
  projects?: ProjectInfo[];
  error?: string;
}

export interface ImageData {
  name: string;
  data: string;
  mimeType: string;
}

export interface CommandRequest {
  project: string;
  command: string;
  args: string;
  images?: ImageData[];
}

export interface ContinueRequest {
  project: string;
  session_id: string;
  answer: string;
}

export type EventDotType = "tool" | "task" | "text" | "info" | "error" | "question";

export interface DisplayEvent {
  id: string;
  type: EventDotType;
  title: string;
  detail?: string;
  fullText?: string;
}

export type RestartStatus = "idle" | "restarting" | "success" | "error" | "timeout";
