import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

// Mock EventSource
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

// Mock chatApi
const mockListSessions = vi.fn();
const mockSendPrompt = vi.fn();

vi.mock("@/lib/chatApi", () => ({
  listSessions: (...args: unknown[]) => mockListSessions(...args),
  sendPrompt: (...args: unknown[]) => mockSendPrompt(...args),
  openEventStream: (sessionId: string) =>
    new MockEventSource(`/api/events?sessionId=${sessionId}`),
}));

function sendSSEMessage(es: MockEventSource, data: Record<string, unknown>) {
  es.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
}

describe("useChat", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
    MockEventSource.reset();
    mockListSessions.mockResolvedValue([{ id: "session-1", cwd: "/test" }]);
    mockSendPrompt.mockResolvedValue({ ok: true, status: 200 });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("accumulates text_delta events", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.latest();

    act(() => {
      sendSSEMessage(es, { type: "text_delta", text: "Hello " });
    });
    act(() => {
      sendSSEMessage(es, { type: "text_delta", text: "World" });
    });

    expect(result.current.responseText).toBe("Hello World");
  });

  it("type: result triggers onComplete with accumulated text", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const onComplete = vi.fn();
    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat({ onComplete }));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.latest();

    act(() => {
      sendSSEMessage(es, { type: "text_delta", text: "response text" });
    });
    act(() => {
      sendSSEMessage(es, { type: "result" });
    });

    expect(onComplete).toHaveBeenCalledWith("response text");
    expect(result.current.isStreaming).toBe(false);
  });

  it("3s silence fallback triggers completion", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const onComplete = vi.fn();
    const { useChat } = await import("@/hooks/useChat");
    renderHook(() => useChat({ onComplete }));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.latest();

    act(() => {
      sendSSEMessage(es, { type: "text_delta", text: "partial" });
    });

    // Advance 3 seconds for silence timeout
    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(onComplete).toHaveBeenCalledWith("partial");
  });

  it("onerror triggers exponential backoff starting at 1s", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const initialCount = MockEventSource.instances.length;
    const es = MockEventSource.latest();

    // Trigger error
    act(() => {
      es.onerror?.(new Event("error"));
    });

    expect(es.close).toHaveBeenCalled();

    // Before 1s: no new EventSource
    act(() => {
      vi.advanceTimersByTime(999);
    });
    expect(MockEventSource.instances.length).toBe(initialCount);

    // At 1s: new EventSource created
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(MockEventSource.instances.length).toBe(initialCount + 1);
  });

  it("onerror exponential backoff doubles: 1s, 2s, 4s, 8s", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const delays = [1000, 2000, 4000, 8000];

    for (const delay of delays) {
      const countBefore = MockEventSource.instances.length;
      const es = MockEventSource.latest();

      act(() => {
        es.onerror?.(new Event("error"));
      });

      // Just before delay: no new instance
      act(() => {
        vi.advanceTimersByTime(delay - 1);
      });
      expect(MockEventSource.instances.length).toBe(countBefore);

      // At delay: new instance
      act(() => {
        vi.advanceTimersByTime(1);
      });
      expect(MockEventSource.instances.length).toBe(countBefore + 1);
    }
  });

  it("stops retrying after 10 failures and sets error", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // Fail 10 times
    for (let i = 0; i < 10; i++) {
      const es = MockEventSource.latest();
      act(() => {
        es.onerror?.(new Event("error"));
      });
      // Advance enough time for reconnect (max 8s)
      act(() => {
        vi.advanceTimersByTime(10000);
      });
    }

    // 11th error - should not reconnect
    const countBefore = MockEventSource.instances.length;
    const es = MockEventSource.latest();
    act(() => {
      es.onerror?.(new Event("error"));
    });

    // Advance plenty of time
    act(() => {
      vi.advanceTimersByTime(20000);
    });

    // No new EventSource
    expect(MockEventSource.instances.length).toBe(countBefore);
    expect(result.current.error).toContain("再接続上限");
    expect(result.current.status).toBe("error");
  });

  it("session invalid (4xx) retries with fresh session", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // sendPrompt returns 4xx first time, then ok
    mockSendPrompt
      .mockResolvedValueOnce({ ok: false, status: 401 })
      .mockResolvedValueOnce({ ok: true, status: 200 });
    mockListSessions.mockResolvedValueOnce([{ id: "session-2", cwd: "/test" }]);

    await act(async () => {
      await result.current.send("hello");
      await vi.runAllTimersAsync();
    });

    // Should have called listSessions to get fresh session
    expect(mockListSessions).toHaveBeenCalled();
    // Should have retried with new session
    expect(mockSendPrompt).toHaveBeenCalledTimes(2);
    expect(result.current.sessionId).toBe("session-2");
  });

  it("visibilitychange hidden closes EventSource", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.latest();

    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "hidden",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    expect(es.close).toHaveBeenCalled();

    // Restore
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });
  });

  it("visibilitychange visible reconnects EventSource", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const countAfterInit = MockEventSource.instances.length;

    // Go hidden
    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "hidden",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    // Go visible
    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "visible",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    // New EventSource should be created
    expect(MockEventSource.instances.length).toBeGreaterThan(countAfterInit);

    // Restore
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });
  });

  it("persists session ID to localStorage", async () => {
    // No stored session
    mockListSessions.mockResolvedValueOnce([{ id: "new-session", cwd: "/test" }]);

    const { useChat } = await import("@/hooks/useChat");
    renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(localStorage.getItem("ghostrunner_active_session_id")).toBe(
      "new-session"
    );
  });

  it("retries listSessions without cwd when first call returns empty", async () => {
    // No stored session - clear any default mock
    mockListSessions.mockReset();
    mockListSessions
      .mockResolvedValueOnce([]) // first call with cwd returns empty
      .mockResolvedValueOnce([{ id: "fallback-session" }]) // second without cwd
      .mockResolvedValue([]); // any subsequent calls

    const { useChat } = await import("@/hooks/useChat");
    renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // Should have been called at least twice (with cwd, then without cwd)
    expect(mockListSessions.mock.calls.length).toBeGreaterThanOrEqual(2);

    // Find the pair: first call with cwd, immediately followed by call without cwd
    const calls = mockListSessions.mock.calls;
    let foundPair = false;
    for (let i = 0; i < calls.length - 1; i++) {
      const call = calls[i][0] as Record<string, unknown>;
      const nextCall = calls[i + 1][0] as Record<string, unknown>;
      if (call.cwd && !("cwd" in nextCall)) {
        foundPair = true;
        expect(call.provider).toBe("claude");
        expect(nextCall.provider).toBe("claude");
        break;
      }
    }
    expect(foundPair).toBe(true);
  });
});
