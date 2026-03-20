import type { CreateProjectRequest } from "@/types";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";

interface ValidateResponse {
  valid: boolean;
  path: string;
  error?: string;
}

export async function validateProjectName(
  name: string,
  signal?: AbortSignal
): Promise<ValidateResponse> {
  const response = await fetch(
    `${API_BASE}/api/projects/validate?name=${encodeURIComponent(name)}`,
    { signal }
  );

  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Validation request failed");
  }

  return response.json();
}

export async function createProjectStream(
  request: CreateProjectRequest,
  signal?: AbortSignal
): Promise<Response> {
  const response = await fetch(`${API_BASE}/api/projects/create/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    signal,
  });

  return response;
}

interface OpenResponse {
  success: boolean;
  message: string;
}

export async function openInVSCode(path: string): Promise<OpenResponse> {
  const response = await fetch(`${API_BASE}/api/projects/open`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });

  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to open project");
  }

  return response.json();
}
