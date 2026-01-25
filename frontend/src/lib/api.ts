import type { FilesResponse, CommandRequest, ContinueRequest } from "@/types";

// ローカル開発時はバックエンド直接、本番はプロキシ経由
const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";

export async function fetchFiles(project: string): Promise<FilesResponse> {
  const response = await fetch(
    `${API_BASE}/api/files?project=${encodeURIComponent(project)}`
  );
  if (!response.ok) {
    return { success: false, error: "Request failed" };
  }
  return response.json();
}

export async function executeCommandStream(
  request: CommandRequest,
  signal?: AbortSignal
): Promise<Response> {
  const response = await fetch(`${API_BASE}/api/command/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    signal,
  });
  return response;
}

export async function continueSessionStream(
  request: ContinueRequest,
  signal?: AbortSignal
): Promise<Response> {
  const response = await fetch(`${API_BASE}/api/command/continue/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    signal,
  });
  return response;
}
