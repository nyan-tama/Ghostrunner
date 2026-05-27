import type { DashboardState } from "@/types/dashboard";

export async function fetchDashboardState(): Promise<DashboardState> {
  const res = await fetch("/api/dashboard/state");
  if (!res.ok) throw new Error(`Dashboard fetch failed: ${res.status}`);
  return res.json();
}

export async function submitAnswer(req: {
  projectPath: string;
  planPath: string;
  lineStart: number;
  answer: string;
}): Promise<{ success: boolean; error?: string }> {
  const res = await fetch("/api/dashboard/answer", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  return res.json();
}
