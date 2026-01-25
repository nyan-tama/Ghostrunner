import type { FilesResponse, CommandRequest, ContinueRequest } from "@/types";

export async function fetchFiles(project: string): Promise<FilesResponse> {
  const response = await fetch(`/api/files?project=${encodeURIComponent(project)}`);
  if (!response.ok) {
    return { success: false, error: "Request failed" };
  }
  return response.json();
}

export async function executeCommandStream(
  request: CommandRequest
): Promise<Response> {
  const response = await fetch("/api/command/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  return response;
}

export async function continueSessionStream(
  request: ContinueRequest
): Promise<Response> {
  const response = await fetch("/api/command/continue/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  return response;
}
