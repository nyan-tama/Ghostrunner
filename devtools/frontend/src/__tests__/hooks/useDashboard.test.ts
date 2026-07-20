import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import type { DashboardState } from "@/types/dashboard";
import type { DashboardConnectionState } from "@/hooks/useDashboardSSE";
import { LOCAL_STORAGE_POLLING_ENABLED_KEY } from "@/lib/constants";

// dashboardApi モック
const mockFetchDashboardState = vi.fn();
vi.mock("@/lib/dashboardApi", () => ({
  fetchDashboardState: (...args: unknown[]) => mockFetchDashboardState(...args),
  submitAnswer: vi.fn(),
}));

// useDashboardSSE モック（connectionState を制御し onSnapshot を捕捉）
// 名前は "mock" 接頭辞で vi.mock ホイスティングの参照制約を満たす
const mockSSE: {
  connectionState: DashboardConnectionState;
  onSnapshot: ((s: DashboardState) => void) | null;
} = { connectionState: "live", onSnapshot: null };

vi.mock("@/hooks/useDashboardSSE", () => ({
  useDashboardSSE: ({
    onSnapshot,
  }: {
    onSnapshot: (s: DashboardState) => void;
  }) => {
    mockSSE.onSnapshot = onSnapshot;
    return { connectionState: mockSSE.connectionState, reconnect: vi.fn() };
  },
}));

const MOCK_STATE: DashboardState = {
  projects: [
    {
      name: "TestProject",
      path: "/test/path",
      isSelf: false,
      attention: "watching",
      kanban: { reviewing: 0, waiting: 1, running: 0, done: 5 },
      unanswered: [],
      ops: [],
      opsOptedIn: false,
      warnings: [],
    },
  ],
  generatedAt: "2026-07-20T00:00:00Z",
};

const SNAPSHOT_STATE: DashboardState = {
  projects: [],
  generatedAt: "2026-07-20T01:00:00Z",
};

// jsdom がこの環境で localStorage を提供しないため、最小スタブを注入
function createLocalStorageStub() {
  let store: Record<string, string> = {};
  return {
    getItem: (k: string) => (k in store ? store[k] : null),
    setItem: (k: string, v: string) => {
      store[k] = String(v);
    },
    removeItem: (k: string) => {
      delete store[k];
    },
    clear: () => {
      store = {};
    },
    key: () => null,
    length: 0,
  };
}

function setVisibility(value: "visible" | "hidden") {
  Object.defineProperty(document, "visibilityState", {
    value,
    configurable: true,
  });
  document.dispatchEvent(new Event("visibilitychange"));
}

function flushPromises() {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, 0);
    vi.advanceTimersByTime(0);
  });
}

async function mountDashboard() {
  const { useDashboard } = await import("@/hooks/useDashboard");
  const rendered = renderHook(() => useDashboard());
  await act(async () => {
    await flushPromises();
  });
  return rendered;
}

describe("useDashboard", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.stubGlobal("localStorage", createLocalStorageStub());
    mockSSE.connectionState = "live";
    mockSSE.onSnapshot = null;
    mockFetchDashboardState.mockReset();
    mockFetchDashboardState.mockResolvedValue(MOCK_STATE);
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("fetches once on mount (SSE 初回 push 前の白画面防止)", async () => {
    // live のときはポーリングしないので mount fetch のみ
    await mountDashboard();
    expect(mockFetchDashboardState).toHaveBeenCalledTimes(1);
  });

  describe("shouldPoll 状態機械 (FC2)", () => {
    it("SSE 接続 (live) の間はポーリングしない", async () => {
      mockSSE.connectionState = "live";
      await mountDashboard();

      const afterMount = mockFetchDashboardState.mock.calls.length;
      await act(async () => {
        vi.advanceTimersByTime(60000);
        await flushPromises();
      });

      // 周期 fetch が発生しない
      expect(mockFetchDashboardState.mock.calls.length).toBe(afterMount);
    });

    it("切断 (reconnecting) かつ polling ON かつ visible でフォールバック起動", async () => {
      mockSSE.connectionState = "reconnecting";
      await mountDashboard();

      const afterMount = mockFetchDashboardState.mock.calls.length;
      await act(async () => {
        vi.advanceTimersByTime(15000);
        await flushPromises();
      });

      expect(mockFetchDashboardState.mock.calls.length).toBeGreaterThan(
        afterMount
      );
    });

    it("offline でもフォールバック起動する", async () => {
      mockSSE.connectionState = "offline";
      await mountDashboard();

      const afterMount = mockFetchDashboardState.mock.calls.length;
      await act(async () => {
        vi.advanceTimersByTime(15000);
        await flushPromises();
      });

      expect(mockFetchDashboardState.mock.calls.length).toBeGreaterThan(
        afterMount
      );
    });

    it("polling OFF なら切断中でもフォールバックしない", async () => {
      mockSSE.connectionState = "reconnecting";
      const { result } = await mountDashboard();

      await act(async () => {
        result.current.setPolling(false);
        await flushPromises();
      });

      const base = mockFetchDashboardState.mock.calls.length;
      await act(async () => {
        vi.advanceTimersByTime(45000);
        await flushPromises();
      });

      expect(mockFetchDashboardState.mock.calls.length).toBe(base);
    });

    it("visibility hidden で停止し visible 復帰で再開する", async () => {
      mockSSE.connectionState = "reconnecting";
      await mountDashboard();

      // hidden にする → shouldPoll false で停止
      await act(async () => {
        setVisibility("hidden");
        await flushPromises();
      });
      const whileHidden = mockFetchDashboardState.mock.calls.length;
      await act(async () => {
        vi.advanceTimersByTime(45000);
        await flushPromises();
      });
      expect(mockFetchDashboardState.mock.calls.length).toBe(whileHidden);

      // visible 復帰 → 即時 fetch で再開
      await act(async () => {
        setVisibility("visible");
        await flushPromises();
      });
      expect(mockFetchDashboardState.mock.calls.length).toBeGreaterThan(
        whileHidden
      );
    });

    it("interval は高々1本（1 tick で fetch は +1 のみ・二重起動しない）", async () => {
      mockSSE.connectionState = "reconnecting";
      await mountDashboard();

      const base = mockFetchDashboardState.mock.calls.length;
      await act(async () => {
        vi.advanceTimersByTime(15000);
        await flushPromises();
      });

      // interval が二重なら +2 になる
      expect(mockFetchDashboardState.mock.calls.length - base).toBe(1);
    });
  });

  describe("SSE 受信", () => {
    it("snapshot 受信で state を更新し connectionState を公開する", async () => {
      mockSSE.connectionState = "live";
      const { result } = await mountDashboard();

      expect(result.current.connectionState).toBe("live");

      await act(async () => {
        mockSSE.onSnapshot?.(SNAPSHOT_STATE);
        await flushPromises();
      });

      expect(result.current.state).toEqual(SNAPSHOT_STATE);
    });
  });

  describe("既存の基本挙動", () => {
    it("refresh() で即時 fetch する", async () => {
      const { result } = await mountDashboard();

      const base = mockFetchDashboardState.mock.calls.length;
      await act(async () => {
        result.current.refresh();
        await flushPromises();
      });

      expect(mockFetchDashboardState.mock.calls.length).toBeGreaterThan(base);
    });

    it("fetch 失敗時は error をセットしつつ直前の state を保持する", async () => {
      const { result } = await mountDashboard();

      // mount fetch 成功で state が入っている
      expect(result.current.state).toEqual(MOCK_STATE);

      mockFetchDashboardState.mockRejectedValueOnce(new Error("Network error"));
      await act(async () => {
        result.current.refresh();
        await flushPromises();
      });

      expect(result.current.error).toBe("Network error");
      expect(result.current.state).toEqual(MOCK_STATE);
    });

    it("setPolling(false) は localStorage に永続化する", async () => {
      const { result } = await mountDashboard();

      await act(async () => {
        result.current.setPolling(false);
        await flushPromises();
      });

      expect(
        localStorage.getItem(LOCAL_STORAGE_POLLING_ENABLED_KEY)
      ).toBe("false");
      expect(result.current.polling).toBe(false);
    });
  });
});
