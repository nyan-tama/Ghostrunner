import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

// Mock dashboardApi
const mockFetchDashboardState = vi.fn();
vi.mock("@/lib/dashboardApi", () => ({
  fetchDashboardState: (...args: unknown[]) => mockFetchDashboardState(...args),
  submitAnswer: vi.fn(),
}));

import type { DashboardState } from "@/types/dashboard";

const MOCK_STATE: DashboardState = {
  projects: [
    {
      name: "TestProject",
      path: "/test/path",
      isSelf: false,
      attention: "watching" as const,
      kanban: { reviewing: 0, waiting: 1, running: 0, done: 5 },
      unanswered: [],
      ops: [],
      opsOptedIn: false,
      warnings: [],
    },
  ],
  generatedAt: "2026-05-26T00:00:00Z",
};

// Helper to flush pending promises (microtask queue)
function flushPromises() {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, 0);
    vi.advanceTimersByTime(0);
  });
}

describe("useDashboard", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
    mockFetchDashboardState.mockResolvedValue(MOCK_STATE);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("fetches once on mount", async () => {
    const { useDashboard } = await import("@/hooks/useDashboard");
    const { result } = renderHook(() => useDashboard());

    // Flush mount effect + fetch promise
    await act(async () => {
      await flushPromises();
    });

    expect(mockFetchDashboardState).toHaveBeenCalledTimes(1);
  });

  it("polls every 15 seconds", async () => {
    const { useDashboard } = await import("@/hooks/useDashboard");
    renderHook(() => useDashboard());

    // Flush mount
    await act(async () => {
      await flushPromises();
    });

    const initialCalls = mockFetchDashboardState.mock.calls.length;

    // Advance 15 seconds
    await act(async () => {
      vi.advanceTimersByTime(15000);
      await flushPromises();
    });

    expect(mockFetchDashboardState.mock.calls.length).toBeGreaterThan(initialCalls);
  });

  it("stops polling when visibilityState becomes hidden", async () => {
    const { useDashboard } = await import("@/hooks/useDashboard");
    renderHook(() => useDashboard());

    await act(async () => {
      await flushPromises();
    });

    // Go hidden
    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "hidden",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    const callsBeforeAdvance = mockFetchDashboardState.mock.calls.length;

    // Advance 15s - should NOT trigger additional fetches
    await act(async () => {
      vi.advanceTimersByTime(15000);
      await flushPromises();
    });

    expect(mockFetchDashboardState.mock.calls.length).toBe(callsBeforeAdvance);

    // Restore visibility
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });
  });

  it("resumes with immediate fetch when visibilityState becomes visible", async () => {
    const { useDashboard } = await import("@/hooks/useDashboard");
    renderHook(() => useDashboard());

    await act(async () => {
      await flushPromises();
    });

    // Go hidden
    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "hidden",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    const callsBefore = mockFetchDashboardState.mock.calls.length;

    // Go visible
    await act(async () => {
      Object.defineProperty(document, "visibilityState", {
        value: "visible",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
      await flushPromises();
    });

    // Should have fetched immediately on visible
    expect(mockFetchDashboardState.mock.calls.length).toBeGreaterThan(callsBefore);
  });

  it("refresh() triggers an immediate fetch", async () => {
    const { useDashboard } = await import("@/hooks/useDashboard");
    const { result } = renderHook(() => useDashboard());

    await act(async () => {
      await flushPromises();
    });

    const callsBefore = mockFetchDashboardState.mock.calls.length;

    await act(async () => {
      result.current.refresh();
      await flushPromises();
    });

    expect(mockFetchDashboardState.mock.calls.length).toBeGreaterThan(callsBefore);
  });

  it("sets error string on fetch failure while preserving previous state", async () => {
    const { useDashboard } = await import("@/hooks/useDashboard");
    const { result } = renderHook(() => useDashboard());

    // First successful fetch
    await act(async () => {
      await flushPromises();
    });

    expect(result.current.state).toEqual(MOCK_STATE);

    // Make next fetch fail
    mockFetchDashboardState.mockRejectedValueOnce(new Error("Network error"));

    await act(async () => {
      result.current.refresh();
      await flushPromises();
    });

    expect(result.current.error).toBe("Network error");
    // Previous state is preserved
    expect(result.current.state).toEqual(MOCK_STATE);
  });
});
