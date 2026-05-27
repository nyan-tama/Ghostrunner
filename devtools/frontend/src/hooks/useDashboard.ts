"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { fetchDashboardState } from "@/lib/dashboardApi";
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
}

export function useDashboard(): UseDashboardReturn {
  const [state, setState] = useState<DashboardState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [polling, setPollingState] = useState(true);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollingRef = useRef(true);

  // localStorage から復元（SSR セーフ）
  useEffect(() => {
    const stored = localStorage.getItem(LOCAL_STORAGE_POLLING_ENABLED_KEY);
    if (stored === "false") {
      setPollingState(false);
      pollingRef.current = false;
    }
  }, []);

  // pollingRef を同期
  useEffect(() => {
    pollingRef.current = polling;
  }, [polling]);

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

  // インターバルの開始・停止
  const startInterval = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }
    intervalRef.current = setInterval(doFetch, DASHBOARD_POLL_INTERVAL_MS);
  }, [doFetch]);

  const stopInterval = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  // マウント時: 必ず1回 fetch（白画面防止）
  useEffect(() => {
    doFetch();
  }, [doFetch]);

  // polling 状態に応じたインターバル管理
  useEffect(() => {
    if (polling) {
      startInterval();
    } else {
      stopInterval();
    }
    return stopInterval;
  }, [polling, startInterval, stopInterval]);

  // visibilitychange 連動（polling ON の時のみ）
  useEffect(() => {
    function handleVisibility() {
      if (!pollingRef.current) return;

      if (document.visibilityState === "hidden") {
        stopInterval();
      } else {
        doFetch();
        startInterval();
      }
    }

    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [doFetch, startInterval, stopInterval]);

  const setPolling = useCallback(
    (v: boolean) => {
      setPollingState(v);
      pollingRef.current = v;
      localStorage.setItem(LOCAL_STORAGE_POLLING_ENABLED_KEY, String(v));
      if (!v) {
        stopInterval();
      }
    },
    [stopInterval]
  );

  const refresh = useCallback(() => {
    doFetch();
    // polling 中ならインターバルをリセット
    if (pollingRef.current) {
      startInterval();
    }
  }, [doFetch, startInterval]);

  return { state, error, loading, polling, setPolling, refresh };
}
