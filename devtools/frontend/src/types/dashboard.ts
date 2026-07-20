export type Attention = "required" | "progress" | "watching";

export interface KanbanCounts {
  reviewing: number;
  waiting: number;
  running: number;
  done: number;
}

export interface UnansweredItem {
  planPath: string;
  lineStart: number;
  lineEnd: number;
  questionText: string;
  heading: string;
}

export interface OpsProgress {
  index: number;
  total: number;
}

export interface OpsToday {
  count: number;
  target: number;
}

export interface OpsStats {
  followed: number;
  already: number;
  skipped: number;
  error: number;
}

// 質問待ち状態（バックエンド `idle` オブジェクトと同一フィールド）。
// キーの存在自体が「質問待ち」を意味する（undefined/null = 質問待ちでない）。
export interface IdleState {
  timestamp: string; // RFC3339。バッジの「N分」はフロントが now - timestamp で算出
  preview: string; // rawTail.lastAssistant 先頭80字 / summary 未生成時の暫定
  sessionCount: number; // 同プロジェクトの質問待ちセッション数（代表1件＝最長待機）
  summary: string; // 「何を待っているか」の日本語1行要約（生成前は ""）
  summarizedAt: string; // 要約生成時刻（RFC3339・未生成は ""）
}

export interface OpsEntry {
  account: string;
  kind: string;
  status: string;
  progress?: OpsProgress;
  today?: OpsToday;
  stats?: OpsStats;
  consecutiveErrors: number;
  updatedAt: string;
  stale: boolean;
  staleHours: number;
  sourceFile: string;
  rawExtra?: Record<string, unknown>;
}

export interface ProjectCardData {
  name: string;
  path: string;
  isSelf: boolean;
  attention: Attention;
  kanban: KanbanCounts;
  unanswered: UnansweredItem[];
  ops: OpsEntry[];
  opsOptedIn: boolean;
  warnings: string[];
  idle?: IdleState | null; // キー欠落 or null = 質問待ちでない（FC3）
}

export interface DashboardState {
  projects: ProjectCardData[];
  generatedAt: string;
}

// 型ガード関数

// SSE 生 payload の DashboardState を軽く検証する（fe-W5）
export function isDashboardStateShape(v: unknown): v is DashboardState {
  return (
    typeof v === "object" &&
    v !== null &&
    "projects" in v &&
    Array.isArray((v as DashboardState).projects) &&
    "generatedAt" in v &&
    typeof (v as DashboardState).generatedAt === "string"
  );
}

// SSE 生 payload の idle を軽く検証する（fe-W9・検証目的のみ）
export function isIdleShape(v: unknown): v is IdleState {
  return (
    typeof v === "object" &&
    v !== null &&
    "timestamp" in v &&
    "preview" in v &&
    "sessionCount" in v &&
    "summary" in v &&
    "summarizedAt" in v &&
    typeof (v as IdleState).timestamp === "string" &&
    typeof (v as IdleState).preview === "string" &&
    typeof (v as IdleState).sessionCount === "number" &&
    typeof (v as IdleState).summary === "string" &&
    typeof (v as IdleState).summarizedAt === "string"
  );
}

export function isProgressShape(v: unknown): v is OpsProgress {
  return (
    typeof v === "object" &&
    v !== null &&
    "index" in v &&
    "total" in v &&
    typeof (v as OpsProgress).index === "number" &&
    typeof (v as OpsProgress).total === "number"
  );
}

export function isTodayShape(v: unknown): v is OpsToday {
  return (
    typeof v === "object" &&
    v !== null &&
    "count" in v &&
    "target" in v &&
    typeof (v as OpsToday).count === "number" &&
    typeof (v as OpsToday).target === "number"
  );
}

export function isStatsShape(v: unknown): v is OpsStats {
  return (
    typeof v === "object" &&
    v !== null &&
    "followed" in v &&
    "already" in v &&
    "skipped" in v &&
    "error" in v &&
    typeof (v as OpsStats).followed === "number" &&
    typeof (v as OpsStats).already === "number" &&
    typeof (v as OpsStats).skipped === "number" &&
    typeof (v as OpsStats).error === "number"
  );
}
