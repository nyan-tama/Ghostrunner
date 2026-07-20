"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { fetchDashboardState } from "@/lib/dashboardApi";
import {
  useDashboardSSE,
  type DashboardConnectionState,
} from "@/hooks/useDashboardSSE";
import {
  DASHBOARD_POLL_INTERVAL_MS,
  LOCAL_STORAGE_POLLING_ENABLED_KEY,
} from "@/lib/constants";
import type { DashboardState } from "@/types/dashboard";

interface UseDashboardReturn {
  state: DashboardState | null;
  error: string | null;
  loading: boolean;
  polling: boolean;
  setPolling: (v: boolean) => void;
  refresh: () => void;
  connectionState: DashboardConnectionState;
}

export function useDashboard(): UseDashboardReturn {
  const [state, setState] = useState<DashboardState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  // polling は「フォールバック自動更新の可否」に意味変更（SSE 接続中は休眠・fe-W7）
  const [polling, setPollingState] = useState(true);
  // visibility を state 化して shouldPoll の再評価トリガにする（stale closure 回避）
  const [visible, setVisible] = useState(true);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // SSE スナップショット受信で state を更新
  const handleSnapshot = useCallback((snapshot: DashboardState) => {
    setState(snapshot);
    setError(null);
    setLoading(false);
  }, []);

  const { connectionState } = useDashboardSSE({ onSnapshot: handleSnapshot });

  // localStorage から復元（SSR セーフ）
  useEffect(() => {
    const stored = localStorage.getItem(LOCAL_STORAGE_POLLING_ENABLED_KEY);
    if (stored === "false") {
      setPollingState(false);
    }
  }, []);

  const doFetch = useCallback(async () => {
    try {
      const data = await fetchDashboardState();
      setState(data);
      setError(null);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "ダッシュボードの取得に失敗しました";
      setError(message);
    } finally {
      setLoading(false);
    }
  }, []);

  // マウント時: 必ず1回 fetch（SSE 初回 push 前の白画面防止）
  useEffect(() => {
    doFetch();
  }, [doFetch]);

  // visibility を state に反映（shouldPoll の再評価トリガ）
  useEffect(() => {
    function handleVisibility() {
      setVisible(document.visibilityState === "visible");
    }
    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, []);

  // 単一の派生値（FC2）: フォールバック有効 かつ SSE 未接続 かつ 可視 のときだけポーリング
  const shouldPoll = polling && connectionState !== "live" && visible;

  // interval を start/stop する useEffect を1本に集約（FC2・常に高々1本）
  useEffect(() => {
    if (!shouldPoll) {
      return;
    }
    // フォールバック開始時に即時取得してから周期実行（SSE 切断直後の 15 秒待ちを避ける）
    doFetch();
    intervalRef.current = setInterval(doFetch, DASHBOARD_POLL_INTERVAL_MS);
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [shouldPoll, doFetch]);

  const setPolling = useCallback((v: boolean) => {
    setPollingState(v);
    localStorage.setItem(LOCAL_STORAGE_POLLING_ENABLED_KEY, String(v));
  }, []);

  const refresh = useCallback(() => {
    doFetch();
  }, [doFetch]);

  return {
    state,
    error,
    loading,
    polling,
    setPolling,
    refresh,
    connectionState,
  };
}
