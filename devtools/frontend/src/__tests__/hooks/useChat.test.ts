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
const mockGetHistory = vi.fn();

vi.mock("@/lib/chatApi", () => ({
  listSessions: (...args: unknown[]) => mockListSessions(...args),
  sendPrompt: (...args: unknown[]) => mockSendPrompt(...args),
  getHistory: (...args: unknown[]) => mockGetHistory(...args),
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
    mockGetHistory.mockResolvedValue([]);
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

  // ===== FE-17: 背景復帰時の整合性対策 =====

  it("FE-17: visible 復帰時に getHistory を 1 回呼び transcript に反映する（TTS は呼ばない）", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    mockGetHistory.mockResolvedValueOnce([
      { role: "assistant", text: "restored " },
      { role: "assistant", text: "text" },
    ]);

    const onComplete = vi.fn();
    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat({ onComplete }));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const initialHistoryCalls = mockGetHistory.mock.calls.length;

    // hidden -> visible
    await act(async () => {
      Object.defineProperty(document, "visibilityState", {
        value: "hidden",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    await act(async () => {
      Object.defineProperty(document, "visibilityState", {
        value: "visible",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
      await vi.runAllTimersAsync();
    });

    expect(mockGetHistory.mock.calls.length).toBe(initialHistoryCalls + 1);
    expect(mockGetHistory).toHaveBeenLastCalledWith("session-1", 5);
    expect(result.current.responseText).toBe("restored text");
    // TTS（onComplete）は visible 復帰では呼ばれない
    expect(onComplete).not.toHaveBeenCalled();

    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });
  });

  it("FE-17: connectionState が live/reconnecting/offline に遷移する", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // SSE open で live
    const es = MockEventSource.latest();
    act(() => {
      es.onopen?.(new Event("open"));
    });
    expect(result.current.connectionState).toBe("live");

    // onerror で reconnecting（バックオフ待ち）
    act(() => {
      es.onerror?.(new Event("error"));
    });
    expect(result.current.connectionState).toBe("reconnecting");

    // 上限超えで offline に至る
    // 既に 1 回失敗済み。再接続→onerror を MAX_RETRIES 上限まで繰り返す
    for (let i = 0; i < 12; i++) {
      act(() => {
        vi.advanceTimersByTime(10000);
      });
      const latest = MockEventSource.latest();
      act(() => {
        latest.onerror?.(new Event("error"));
      });
    }

    expect(result.current.connectionState).toBe("offline");
  });

  it("FE-17: reconnecting から live に遷移する（再接続成功パス）", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // 初期の SSE を live にしておく
    const es = MockEventSource.latest();
    act(() => {
      es.onopen?.(new Event("open"));
    });
    expect(result.current.connectionState).toBe("live");

    // onerror で reconnecting
    act(() => {
      es.onerror?.(new Event("error"));
    });
    expect(result.current.connectionState).toBe("reconnecting");

    // 1s 経過で新規 EventSource が生成される
    const countBefore = MockEventSource.instances.length;
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(MockEventSource.instances.length).toBe(countBefore + 1);

    // 新 EventSource の onopen で live に戻る
    const newEs = MockEventSource.latest();
    act(() => {
      newEs.onopen?.(new Event("open"));
    });
    expect(result.current.connectionState).toBe("live");
  });

  it("FE-17: visibilitychange 多重発火でも getHistory は 1 回しか走らない", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    // getHistory は解決を遅らせて in-flight 中に多重 dispatch する状況を作る
    let resolveHistory: ((value: unknown) => void) | null = null;
    mockGetHistory.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveHistory = resolve;
        })
    );

    const { useChat } = await import("@/hooks/useChat");
    renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const initialHistoryCalls = mockGetHistory.mock.calls.length;

    // hidden -> visible を 2 回連続で dispatch
    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "hidden",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "visible",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    // 2 回目の visible（in-flight 中）
    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "hidden",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    act(() => {
      Object.defineProperty(document, "visibilityState", {
        value: "visible",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    // in-flight ガードにより getHistory は +1 のみ
    expect(mockGetHistory.mock.calls.length).toBe(initialHistoryCalls + 1);

    // 解決させて in-flight フラグを解放
    await act(async () => {
      resolveHistory?.([]);
      await vi.runAllTimersAsync();
    });

    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });
  });

  it("FE-17: getHistory 失敗時はエラーをセットしない", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    mockGetHistory.mockRejectedValueOnce(new Error("history fetch failed"));

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // hidden -> visible
    await act(async () => {
      Object.defineProperty(document, "visibilityState", {
        value: "hidden",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    await act(async () => {
      Object.defineProperty(document, "visibilityState", {
        value: "visible",
        configurable: true,
      });
      document.dispatchEvent(new Event("visibilitychange"));
      await vi.runAllTimersAsync();
    });

    // history 失敗は error にしない
    expect(result.current.error).toBeNull();

    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });
  });

  // ===== B: switchSession / startNewSession / fetchSessions =====

  it("switchSession: SSE を切り替え localStorage を更新する", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const oldEs = MockEventSource.latest();
    const countBefore = MockEventSource.instances.length;

    act(() => {
      result.current.switchSession("session-2");
    });

    // 旧 EventSource が close され、新規が作られる
    expect(oldEs.close).toHaveBeenCalled();
    expect(MockEventSource.instances.length).toBe(countBefore + 1);

    expect(result.current.sessionId).toBe("session-2");
    expect(localStorage.getItem("ghostrunner_active_session_id")).toBe("session-2");
  });

  it("switchSession: onSessionSwitch コールバックを呼び出す", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const onSessionSwitch = vi.fn();
    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat({ onSessionSwitch }));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    act(() => {
      result.current.switchSession("session-2");
    });

    expect(onSessionSwitch).toHaveBeenCalledTimes(1);
  });

  it("startNewSession: sessionId を null にし localStorage から削除する", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    act(() => {
      result.current.startNewSession();
    });

    expect(result.current.sessionId).toBeNull();
    expect(localStorage.getItem("ghostrunner_active_session_id")).toBeNull();
  });

  it("startNewSession 後の send は sessionId なしで POST し、レスポンスから新 SID を取得する", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // startNewSession
    act(() => {
      result.current.startNewSession();
    });

    // sendPrompt のモックを「sessionId 入れずに OK 応答 + 新 SID body」に上書き
    mockSendPrompt.mockReset();
    mockSendPrompt.mockResolvedValueOnce({
      ok: true,
      status: 200,
      clone() {
        return {
          json: async () => ({ sessionId: "new-sid-xyz", provider: "claude" }),
        };
      },
    });

    await act(async () => {
      await result.current.send("hello");
      await vi.runAllTimersAsync();
    });

    // sendPrompt は sessionId: null で呼ばれている
    const lastCall = mockSendPrompt.mock.calls[mockSendPrompt.mock.calls.length - 1];
    expect(lastCall[0].sessionId).toBeNull();

    // 新 SID が確定し localStorage に保存されている
    expect(result.current.sessionId).toBe("new-sid-xyz");
    expect(localStorage.getItem("ghostrunner_active_session_id")).toBe("new-sid-xyz");
  });

  it("fetchSessions: listSessions を呼び sessions state を更新する", async () => {
    localStorage.setItem("ghostrunner_active_session_id", "session-1");

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // fetchSessions を呼ぶ
    mockListSessions.mockReset();
    mockListSessions.mockResolvedValue([
      { id: "a", title: "a" },
      { id: "b", title: "b" },
      { id: "c", title: "c" },
    ]);

    await act(async () => {
      await result.current.fetchSessions();
    });

    expect(result.current.sessions.length).toBe(3);
    expect(result.current.sessions[0].id).toBe("a");
  });

  it("初期化時に sessions state が listSessions の結果で設定される", async () => {
    // localStorage に session 無し（新規取得経路）
    mockListSessions.mockReset();
    mockListSessions.mockResolvedValue([
      { id: "x", title: "X" },
      { id: "y", title: "Y" },
    ]);

    const { useChat } = await import("@/hooks/useChat");
    const { result } = renderHook(() => useChat());

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(result.current.sessions.length).toBe(2);
  });
});
