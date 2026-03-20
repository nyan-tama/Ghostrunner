import type { Question } from "@/types";

export type PatrolProjectStatus =
  | "idle"
  | "running"
  | "waiting_approval"
  | "queued"
  | "completed"
  | "error";

export interface PatrolProject {
  path: string;
  name: string;
}

export interface PatrolProjectState {
  project_path: string;
  status: PatrolProjectStatus;
  recent_commits: string[];
  pending_tasks: number;
  question?: Question;
  error?: string;
}

export type PatrolSSEEventType =
  | "project_started"
  | "project_question"
  | "project_completed"
  | "project_error"
  | "scan_completed";

export interface PatrolSSEEvent {
  type: PatrolSSEEventType;
  project_path: string;
  state?: PatrolProjectState;
  message?: string;
}

export interface PatrolProjectsResponse {
  success: boolean;
  projects?: PatrolProject[];
  error?: string;
}

export interface PatrolStatesResponse {
  success: boolean;
  states?: Record<string, PatrolProjectState>;
  error?: string;
}

export interface PatrolActionResponse {
  success: boolean;
  message?: string;
  error?: string;
}

export interface PatrolScanResponse {
  success: boolean;
  states?: Record<string, PatrolProjectState>;
  error?: string;
}
