import type { ChatSession, ChatHistoryItem } from "@/types/chat";
import { GHOSTRUNNER_CWD } from "@/lib/constants";

// even-terminal の認証トークン。.env.local の NEXT_PUBLIC_EVEN_TERMINAL_TOKEN から読む。
// 未設定の場合は空文字（401 になる）。EventSource はカスタムヘッダ不可なので、
// 全エンドポイント統一で ?token=<X> のクエリ付与方式を採用。
const TOKEN = process.env.NEXT_PUBLIC_EVEN_TERMINAL_TOKEN ?? "";

function withToken(url: string): string {
  if (!TOKEN) return url;
  const sep = url.includes("?") ? "&" : "?";
  return `${url}${sep}token=${encodeURIComponent(TOKEN)}`;
}

export async function listSessions(opts?: {
  cwd?: string;
  provider?: string;
  limit?: number;
}): Promise<ChatSession[]> {
  const params = new URLSearchParams();
  if (opts?.cwd) params.set("cwd", opts.cwd);
  if (opts?.provider) params.set("provider", opts.provider);
  if (opts?.limit) params.set("limit", String(opts.limit));
  const res = await fetch(withToken(`/api/sessions?${params}`));
  if (!res.ok) return [];
  try {
    // even-terminal は { sessions: [...] } の形で返すのでアンラップ。
    // 一応 raw array で返ってくる別 provider 互換性も維持する。
    const data = await res.json();
    if (Array.isArray(data)) return data as ChatSession[];
    if (Array.isArray((data as { sessions?: unknown }).sessions)) {
      return (data as { sessions: ChatSession[] }).sessions;
    }
    return [];
  } catch {
    return [];
  }
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
  return fetch(withToken("/api/prompt"), {
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
    withToken(
      `/api/sessions/${encodeURIComponent(sessionId)}/history?${params}`
    )
  );
  if (!res.ok) return [];
  try {
    const data = await res.json();
    if (Array.isArray(data)) return data as ChatHistoryItem[];
    // even-terminal は { history: [...] } で返す。他provider用に items も保持
    if (Array.isArray((data as { history?: unknown }).history)) {
      return (data as { history: ChatHistoryItem[] }).history;
    }
    if (Array.isArray((data as { items?: unknown }).items)) {
      return (data as { items: ChatHistoryItem[] }).items;
    }
    return [];
  } catch {
    return [];
  }
}

// セッションの状態を取得（even-terminal の権威ある状態判定）
export async function getSessionStatus(
  sessionId: string
): Promise<"idle" | "busy" | "awaiting" | null> {
  try {
    const res = await fetch(
      withToken(
        `/api/status?sessionId=${encodeURIComponent(sessionId)}&provider=claude`
      )
    );
    if (!res.ok) return null;
    const data = (await res.json()) as { state?: string };
    const s = data.state;
    if (s === "idle" || s === "busy" || s === "awaiting") return s;
    return null;
  } catch {
    return null;
  }
}

export function openEventStream(sessionId: string): EventSource {
  return new EventSource(
    withToken(`/api/events?sessionId=${encodeURIComponent(sessionId)}`)
  );
}
