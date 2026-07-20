import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useDashboardSSE } from "@/hooks/useDashboardSSE";
import type { DashboardState } from "@/types/dashboard";

// EventSource モック（usePatrolSSE.test.ts と同じ様式）
class MockEventSource {
  static instances: MockEventSource[] = [];
  static OPEN = 1;

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

const VALID_SNAPSHOT: DashboardState = {
  projects: [],
  generatedAt: "2026-07-20T00:00:00Z",
};

describe("useDashboardSSE", () => {
  beforeEach(() => {
    MockEventSource.reset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("connects to /api/dashboard/stream on mount with initial 'reconnecting'", () => {
    const onSnapshot = vi.fn();
    const { result } = renderHook(() => useDashboardSSE({ onSnapshot }));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.latest().url).toBe("/api/dashboard/stream");
    // 初回接続を即座に張るため初期状態は reconnecting（マウント時の setState 回避）
    expect(result.current.connectionState).toBe("reconnecting");
  });

  it("sets connectionState to 'live' on open", () => {
    const onSnapshot = vi.fn();
    const { result } = renderHook(() => useDashboardSSE({ onSnapshot }));

    act(() => {
      MockEventSource.latest().onopen?.(new Event("open"));
    });

    expect(result.current.connectionState).toBe("live");
  });

  it("parses a raw DashboardState (no {type,data} envelope) and calls onSnapshot", () => {
    const onSnapshot = vi.fn();
    renderHook(() => useDashboardSSE({ onSnapshot }));

    const es = MockEventSource.latest();
    act(() => {
      es.onopen?.(new Event("open"));
      es.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(VALID_SNAPSHOT) })
      );
    });

    expect(onSnapshot).toHaveBeenCalledTimes(1);
    expect(onSnapshot).toHaveBeenCalledWith(VALID_SNAPSHOT);
  });

  it("ignores messages with invalid JSON (onSnapshot not called)", () => {
    const onSnapshot = vi.fn();
    renderHook(() => useDashboardSSE({ onSnapshot }));

    const es = MockEventSource.latest();
    act(() => {
      es.onmessage?.(new MessageEvent("message", { data: "not-json{" }));
    });

    expect(onSnapshot).not.toHaveBeenCalled();
  });

  it("ignores payloads that are not DashboardState-shaped", () => {
    const onSnapshot = vi.fn();
    renderHook(() => useDashboardSSE({ onSnapshot }));

    const es = MockEventSource.latest();
    act(() => {
      es.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify({ foo: 1 }) })
      );
    });

    expect(onSnapshot).not.toHaveBeenCalled();
  });

  it("sets 'reconnecting' on error below the retry limit and reconnects with backoff", () => {
    const onSnapshot = vi.fn();
    const { result } = renderHook(() => useDashboardSSE({ onSnapshot }));

    const es = MockEventSource.latest();
    act(() => {
      es.onopen?.(new Event("open"));
    });
    expect(result.current.connectionState).toBe("live");

    act(() => {
      es.onerror?.(new Event("error"));
    });

    // バックオフ中を落とさず reconnecting を維持（fe-W4）
    expect(result.current.connectionState).toBe("reconnecting");
    expect(es.close).toHaveBeenCalled();

    // 初回リトライ遅延（1000ms）後に再接続
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(MockEventSource.instances).toHaveLength(2);
  });

  it("sets 'offline' after exceeding MAX_RETRIES (10)", () => {
    const onSnapshot = vi.fn();
    const { result } = renderHook(() => useDashboardSSE({ onSnapshot }));

    // 10 回のエラー＋再接続で retryCount を上限まで進める
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

    // 上限到達後の次のエラーで offline
    act(() => {
      MockEventSource.latest().onerror?.(new Event("error"));
    });

    expect(result.current.connectionState).toBe("offline");
  });

  it("does not recreate EventSource when onSnapshot identity changes (onSnapshotRef)", () => {
    const first = vi.fn();
    const second = vi.fn();
    const { rerender } = renderHook(
      ({ cb }) => useDashboardSSE({ onSnapshot: cb }),
      { initialProps: { cb: first } }
    );

    expect(MockEventSource.instances).toHaveLength(1);

    // onSnapshot を別関数に差し替えても再接続ループは起きない
    rerender({ cb: second });
    expect(MockEventSource.instances).toHaveLength(1);

    // 最新の onSnapshot（second）が呼ばれる
    const es = MockEventSource.latest();
    act(() => {
      es.onmessage?.(
        new MessageEvent("message", { data: JSON.stringify(VALID_SNAPSHOT) })
      );
    });

    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
  });

  it("closes EventSource on unmount", () => {
    const onSnapshot = vi.fn();
    const { unmount } = renderHook(() => useDashboardSSE({ onSnapshot }));

    const es = MockEventSource.latest();
    unmount();

    expect(es.close).toHaveBeenCalled();
  });
});
