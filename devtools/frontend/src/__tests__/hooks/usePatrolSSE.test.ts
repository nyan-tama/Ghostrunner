import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { usePatrolSSE } from "@/hooks/usePatrolSSE";

// EventSource モック
class MockEventSource {
  static instances: MockEventSource[] = [];

  url: string;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  readyState = 0;
  close = vi.fn();

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  static reset() {
    MockEventSource.instances = [];
  }

  static latest() {
    return MockEventSource.instances[MockEventSource.instances.length - 1];
  }
}

vi.stubGlobal("EventSource", MockEventSource);

describe("usePatrolSSE", () => {
  beforeEach(() => {
    MockEventSource.reset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("connects to /api/patrol/stream on mount", () => {
    const onEvent = vi.fn();
    renderHook(() => usePatrolSSE({ onEvent }));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.latest().url).toBe("/api/patrol/stream");
  });

  it("sets connectionStatus to 'connecting' on mount, then 'connected' on open", () => {
    const onEvent = vi.fn();
    const { result } = renderHook(() => usePatrolSSE({ onEvent }));

    // After mount, status should be 'connecting'
    expect(result.current.connectionStatus).toBe("connecting");

    // Simulate open
    act(() => {
      MockEventSource.latest().onopen?.(new Event("open"));
    });

    expect(result.current.connectionStatus).toBe("connected");
  });

  it("parses and dispatches SSE messages via onEvent callback", () => {
    const onEvent = vi.fn();
    renderHook(() => usePatrolSSE({ onEvent }));

    const es = MockEventSource.latest();
    act(() => {
      es.onopen?.(new Event("open"));
    });

    const eventData = {
      type: "project_started",
      project_path: "/path/to/project",
      state: { project_path: "/path/to/project", status: "running", recent_commits: [], pending_tasks: 0 },
    };

    act(() => {
      es.onmessage?.(new MessageEvent("message", { data: JSON.stringify(eventData) }));
    });

    expect(onEvent).toHaveBeenCalledTimes(1);
    expect(onEvent).toHaveBeenCalledWith(eventData);
  });

  it("ignores messages with invalid JSON", () => {
    const onEvent = vi.fn();
    renderHook(() => usePatrolSSE({ onEvent }));

    const es = MockEventSource.latest();
    act(() => {
      es.onopen?.(new Event("open"));
    });

    act(() => {
      es.onmessage?.(new MessageEvent("message", { data: "not-json" }));
    });

    expect(onEvent).not.toHaveBeenCalled();
  });

  it("sets connectionStatus to 'disconnected' on error and retries with backoff", () => {
    const onEvent = vi.fn();
    const { result } = renderHook(() => usePatrolSSE({ onEvent }));

    const es = MockEventSource.latest();
    act(() => {
      es.onopen?.(new Event("open"));
    });
    expect(result.current.connectionStatus).toBe("connected");

    // Trigger error
    act(() => {
      es.onerror?.(new Event("error"));
    });
    expect(result.current.connectionStatus).toBe("disconnected");
    expect(es.close).toHaveBeenCalled();

    // After 1000ms (initial retry delay), should reconnect
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    // A new EventSource should have been created
    expect(MockEventSource.instances).toHaveLength(2);
  });

  it("closes EventSource on unmount", () => {
    const onEvent = vi.fn();
    const { unmount } = renderHook(() => usePatrolSSE({ onEvent }));

    const es = MockEventSource.latest();

    unmount();

    expect(es.close).toHaveBeenCalled();
  });

  it("stops retrying after MAX_RETRIES (10)", () => {
    const onEvent = vi.fn();
    renderHook(() => usePatrolSSE({ onEvent }));

    // Simulate 10 errors with advancing timers between each
    for (let i = 0; i < 10; i++) {
      const es = MockEventSource.latest();
      act(() => {
        es.onerror?.(new Event("error"));
      });
      const delay = Math.min(1000 * Math.pow(2, i), 30000);
      act(() => {
        vi.advanceTimersByTime(delay);
      });
    }

    // After 10 retries, the 11th error should not create a new instance
    const countBefore = MockEventSource.instances.length;
    const es = MockEventSource.latest();
    act(() => {
      es.onerror?.(new Event("error"));
    });
    act(() => {
      vi.advanceTimersByTime(60000);
    });

    expect(MockEventSource.instances).toHaveLength(countBefore);
  });
});
