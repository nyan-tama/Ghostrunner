import type { FilesResponse, ProjectsResponse, CommandRequest, ContinueRequest } from "@/types";
import type { GeminiTokenResponse } from "@/types/gemini";
import type { OpenAITokenResponse } from "@/types/openai";

// ローカル開発時はバックエンド直接、本番はプロキシ経由
const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";

export async function fetchProjects(): Promise<ProjectsResponse> {
  try {
    const response = await fetch(`${API_BASE}/api/projects`);
    if (!response.ok) {
      return { success: false, projects: [] };
    }
    return response.json();
  } catch {
    return { success: false, projects: [] };
  }
}

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

/**
 * Gemini Live API 用のエフェメラルトークンを取得
 * @param expireSeconds トークン有効期限（秒）、デフォルト3600
 * @returns エフェメラルトークン文字列
 * @throws トークン取得に失敗した場合
 */
export async function fetchGeminiToken(expireSeconds?: number): Promise<string> {
  const response = await fetch(`${API_BASE}/api/gemini/token`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(expireSeconds ? { expireSeconds } : {}),
  });

  const data: GeminiTokenResponse = await response.json();

  if (!response.ok || !data.success) {
    throw new Error(data.error || "Failed to get ephemeral token");
  }

  if (!data.token) {
    throw new Error("Token not found in response");
  }

  return data.token;
}

/**
 * OpenAI Realtime API 用のエフェメラルトークンを取得
 * @param model 使用するモデル（オプション）
 * @param voice 音声タイプ（オプション）
 * @returns エフェメラルトークン文字列
 * @throws トークン取得に失敗した場合
 */
export async function fetchOpenAIRealtimeToken(model?: string, voice?: string): Promise<string> {
  const response = await fetch(`${API_BASE}/api/openai/realtime/session`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ model, voice }),
  });

  const data: OpenAITokenResponse = await response.json();

  if (!response.ok || !data.success) {
    throw new Error(data.error || "Failed to get ephemeral token");
  }

  if (!data.token) {
    throw new Error("Token not found in response");
  }

  return data.token;
}
