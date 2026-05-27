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
}

export interface DashboardState {
  projects: ProjectCardData[];
  generatedAt: string;
}

// 型ガード関数
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
