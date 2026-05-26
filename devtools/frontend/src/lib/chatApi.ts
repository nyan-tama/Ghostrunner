import type { ChatSession } from "@/types/chat";
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
  sessionId: string;
  text: string;
  cwd?: string;
}): Promise<Response> {
  return fetch("/api/prompt", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      sessionId: req.sessionId,
      text: req.text,
      provider: "claude",
      cwd: req.cwd ?? GHOSTRUNNER_CWD,
    }),
  });
}

export function openEventStream(sessionId: string): EventSource {
  return new EventSource(
    `/api/events?sessionId=${encodeURIComponent(sessionId)}`
  );
}
