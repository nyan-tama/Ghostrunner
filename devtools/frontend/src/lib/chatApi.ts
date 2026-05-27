import type { ChatSession, ChatHistoryItem } from "@/types/chat";
import { GHOSTRUNNER_CWD } from "@/lib/constants";

export async function listSessions(opts?: {
  cwd?: string;
  provider?: string;
  limit?: number;
}): Promise<ChatSession[]> {
  const params = new URLSearchParams();
  if (opts?.cwd) params.set("cwd", opts.cwd);
  if (opts?.provider) params.set("provider", opts.provider);
  if (opts?.limit) params.set("limit", String(opts.limit));
  const res = await fetch(`/api/sessions?${params}`);
  if (!res.ok) return [];
  return res.json();
}

export async function sendPrompt(req: {
  sessionId: string | null;
  text: string;
  cwd?: string;
}): Promise<Response> {
  const body: Record<string, unknown> = {
    text: req.text,
    provider: "claude",
    cwd: req.cwd ?? GHOSTRUNNER_CWD,
  };
  if (req.sessionId) {
    body.sessionId = req.sessionId;
  }
  return fetch("/api/prompt", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

// 履歴取得（背景復帰時の整合性確保用）。失敗時は空配列を返す
export async function getHistory(
  sessionId: string,
  limit: number
): Promise<ChatHistoryItem[]> {
  const params = new URLSearchParams();
  params.set("limit", String(limit));
  const res = await fetch(
    `/api/sessions/${encodeURIComponent(sessionId)}/history?${params}`
  );
  if (!res.ok) return [];
  try {
    const data = await res.json();
    if (Array.isArray(data)) return data as ChatHistoryItem[];
    if (Array.isArray((data as { items?: unknown }).items)) {
      return (data as { items: ChatHistoryItem[] }).items;
    }
    return [];
  } catch {
    return [];
  }
}

export function openEventStream(sessionId: string): EventSource {
  return new EventSource(
    `/api/events?sessionId=${encodeURIComponent(sessionId)}`
  );
}
