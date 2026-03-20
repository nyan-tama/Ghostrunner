import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { usePatrol } from "@/hooks/usePatrol";
import type { PatrolSSEEvent } from "@/types/patrol";

// usePatrolSSE モック
let capturedOnEvent: ((event: PatrolSSEEvent) => void) | null = null;

vi.mock("@/hooks/usePatrolSSE", () => ({
  usePatrolSSE: ({ onEvent }: { onEvent: (event: PatrolSSEEvent) => void }) => {
    capturedOnEvent = onEvent;
    return { connectionStatus: "connected" as const };
  },
}));

// patrolApi モック
const mockFetchPatrolProjects = vi.fn();
const mockFetchPatrolStates = vi.fn();
const mockRegisterPatrolProject = vi.fn();
const mockRemovePatrolProject = vi.fn();
const mockStartPatrol = vi.fn();
const mockStopPatrol = vi.fn();
const mockSendPatrolAnswer = vi.fn();
const mockStartPolling = vi.fn();
const mockStopPolling = vi.fn();

vi.mock("@/lib/patrolApi", () => ({
  fetchPatrolProjects: (...args: unknown[]) => mockFetchPatrolProjects(...args),
  fetchPatrolStates: (...args: unknown[]) => mockFetchPatrolStates(...args),
  registerPatrolProject: (...args: unknown[]) => mockRegisterPatrolProject(...args),
  removePatrolProject: (...args: unknown[]) => mockRemovePatrolProject(...args),
  startPatrol: (...args: unknown[]) => mockStartPatrol(...args),
  stopPatrol: (...args: unknown[]) => mockStopPatrol(...args),
  sendPatrolAnswer: (...args: unknown[]) => mockSendPatrolAnswer(...args),
  startPolling: (...args: unknown[]) => mockStartPolling(...args),
  stopPolling: (...args: unknown[]) => mockStopPolling(...args),
}));

describe("usePatrol", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedOnEvent = null;
    mockFetchPatrolProjects.mockResolvedValue({
      success: true,
      projects: [
        { path: "/proj/a", name: "Project A" },
        { path: "/proj/b", name: "Project B" },
      ],
    });
    mockFetchPatrolStates.mockResolvedValue({
      success: true,
      states: {
        "/proj/a": {
          project_path: "/proj/a",
          status: "idle",
          recent_commits: [],
          pending_tasks: 0,
        },
      },
    });
  });

  it("loads initial projects and states on mount", async () => {
    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    expect(result.current.projects[0].name).toBe("Project A");
    expect(result.current.projectStates["/proj/a"].status).toBe("idle");
    expect(result.current.error).toBeNull();
  });

  it("sets error when initial fetch fails", async () => {
    mockFetchPatrolProjects.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.error).toBe("Network error");
    });
  });

  it("updates projectStates when SSE event with state is received", async () => {
    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    act(() => {
      capturedOnEvent?.({
        type: "project_started",
        project_path: "/proj/a",
        state: {
          project_path: "/proj/a",
          status: "running",
          recent_commits: ["fix: something"],
          pending_tasks: 3,
        },
      });
    });

    expect(result.current.projectStates["/proj/a"].status).toBe("running");
    expect(result.current.isRunning).toBe(true);
  });

  it("updates state on scan_completed event", async () => {
    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    act(() => {
      capturedOnEvent?.({
        type: "scan_completed",
        project_path: "/proj/b",
        state: {
          project_path: "/proj/b",
          status: "completed",
          recent_commits: [],
          pending_tasks: 0,
        },
      });
    });

    expect(result.current.projectStates["/proj/b"].status).toBe("completed");
  });

  it("handleStartPatrol calls API and sets isRunning", async () => {
    mockStartPatrol.mockResolvedValue({ success: true });
    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    await act(async () => {
      await result.current.handleStartPatrol();
    });

    expect(mockStartPatrol).toHaveBeenCalled();
    expect(result.current.isRunning).toBe(true);
  });

  it("handleStopPatrol calls API and clears isRunning", async () => {
    mockStopPatrol.mockResolvedValue({ success: true });
    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    // Start first
    mockStartPatrol.mockResolvedValue({ success: true });
    await act(async () => {
      await result.current.handleStartPatrol();
    });
    expect(result.current.isRunning).toBe(true);

    // Stop
    await act(async () => {
      await result.current.handleStopPatrol();
    });
    expect(result.current.isRunning).toBe(false);
  });

  it("handleSendAnswer updates project state to running", async () => {
    mockSendPatrolAnswer.mockResolvedValue({ success: true });
    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    // Set waiting_approval state via SSE
    act(() => {
      capturedOnEvent?.({
        type: "project_question",
        project_path: "/proj/a",
        state: {
          project_path: "/proj/a",
          status: "waiting_approval",
          recent_commits: [],
          pending_tasks: 0,
          question: { question: "Proceed?", header: "Q", options: [], multiSelect: false },
        },
      });
    });

    expect(result.current.projectStates["/proj/a"].status).toBe("waiting_approval");

    await act(async () => {
      await result.current.handleSendAnswer("/proj/a", "yes");
    });

    expect(mockSendPatrolAnswer).toHaveBeenCalledWith("/proj/a", "yes");
    expect(result.current.projectStates["/proj/a"].status).toBe("running");
    expect(result.current.projectStates["/proj/a"].question).toBeUndefined();
  });

  it("handleTogglePolling toggles polling state", async () => {
    mockStartPolling.mockResolvedValue({ success: true });
    mockStopPolling.mockResolvedValue({ success: true });
    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    expect(result.current.isPolling).toBe(false);

    // Start polling
    await act(async () => {
      await result.current.handleTogglePolling();
    });
    expect(mockStartPolling).toHaveBeenCalled();
    expect(result.current.isPolling).toBe(true);

    // Stop polling
    await act(async () => {
      await result.current.handleTogglePolling();
    });
    expect(mockStopPolling).toHaveBeenCalled();
    expect(result.current.isPolling).toBe(false);
  });

  it("registerProject calls API and refreshes projects list", async () => {
    mockRegisterPatrolProject.mockResolvedValue({ success: true });
    mockFetchPatrolProjects.mockResolvedValueOnce({
      success: true,
      projects: [
        { path: "/proj/a", name: "Project A" },
        { path: "/proj/b", name: "Project B" },
      ],
    }).mockResolvedValueOnce({
      success: true,
      projects: [
        { path: "/proj/a", name: "Project A" },
        { path: "/proj/b", name: "Project B" },
        { path: "/proj/c", name: "Project C" },
      ],
    });

    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    await act(async () => {
      await result.current.registerProject("/proj/c");
    });

    expect(mockRegisterPatrolProject).toHaveBeenCalledWith("/proj/c");
    expect(result.current.projects).toHaveLength(3);
  });

  it("removeProject removes from local state", async () => {
    mockRemovePatrolProject.mockResolvedValue({ success: true });
    const { result } = renderHook(() => usePatrol());

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    await act(async () => {
      await result.current.removeProject("/proj/a");
    });

    expect(mockRemovePatrolProject).toHaveBeenCalledWith("/proj/a");
    expect(result.current.projects).toHaveLength(1);
    expect(result.current.projects[0].path).toBe("/proj/b");
    expect(result.current.projectStates["/proj/a"]).toBeUndefined();
  });
});
