import type {
  PatrolProjectsResponse,
  PatrolStatesResponse,
  PatrolActionResponse,
  PatrolScanResponse,
} from "@/types/patrol";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";

export async function fetchPatrolProjects(): Promise<PatrolProjectsResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/projects`);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to fetch patrol projects");
  }
  return response.json();
}

export async function registerPatrolProject(path: string): Promise<PatrolActionResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/projects`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to register patrol project");
  }
  return response.json();
}

export async function removePatrolProject(path: string): Promise<PatrolActionResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/projects/remove`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to remove patrol project");
  }
  return response.json();
}

export async function startPatrol(): Promise<PatrolActionResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/start`, {
    method: "POST",
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to start patrol");
  }
  return response.json();
}

export async function stopPatrol(): Promise<PatrolActionResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/stop`, {
    method: "POST",
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to stop patrol");
  }
  return response.json();
}

export async function sendPatrolAnswer(
  projectPath: string,
  answer: string
): Promise<PatrolActionResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/resume`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ projectPath, answer }),
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to send answer");
  }
  return response.json();
}

export async function fetchPatrolStates(): Promise<PatrolStatesResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/states`);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to fetch patrol states");
  }
  return response.json();
}

export async function fetchPatrolScan(): Promise<PatrolScanResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/scan`, {
    method: "POST",
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to scan patrol projects");
  }
  return response.json();
}

export async function startPolling(): Promise<PatrolActionResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/polling/start`, {
    method: "POST",
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to start polling");
  }
  return response.json();
}

export async function stopPolling(): Promise<PatrolActionResponse> {
  const response = await fetch(`${API_BASE}/api/patrol/polling/stop`, {
    method: "POST",
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "Failed to stop polling");
  }
  return response.json();
}
