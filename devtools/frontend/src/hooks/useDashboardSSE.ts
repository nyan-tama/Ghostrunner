"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { DASHBOARD_SSE_PATH } from "@/lib/constants";
import { isDashboardStateShape, isIdleShape } from "@/types/dashboard";
import type { DashboardState } from "@/types/dashboard";

// ConnectionIndicator が期待する 3 値（fe-W4）
export type DashboardConnectionState = "live" | "reconnecting" | "offline";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";
const INITIAL_RETRY_DELAY = 1000;
const MAX_RETRY_DELAY = 30000;
const MAX_RETRIES = 10;

interface UseDashboardSSEProps {
  // 生 DashboardState スナップショットを受け取る（fe-W5・エンベロープ無し）
  onSnapshot: (state: DashboardState) => void;
}

// SSE 生 payload の idle を軽く検証し、壊れていれば null に落とす（fe-W9・イミュータブル）
function normalizeSnapshot(state: DashboardState): DashboardState {
  return {
    ...state,
    projects: state.projects.map((p) =>
      p.idle && !isIdleShape(p.idle) ? { ...p, idle: null } : p
    ),
  };
}

/**
 * dashboard SSE 接続フック（/api/dashboard/stream）。
 *
 * - onmessage は patrol の `{type,data}` エンベロープではなく **生 DashboardState** を parse（fe-W5）
 * - onerror でバックオフ再接続中は `reconnecting`、リトライ上限で `offline`、接続で `live`（fe-W4）
 * - onEventRef パターン（onSnapshotRef）で onSnapshot 同一性による再接続ループを防ぐ
 * - visibilitychange でタブ復帰時にリトライカウントをリセットし即再接続
 */
export function useDashboardSSE({ onSnapshot }: UseDashboardSSEProps): {
  connectionState: DashboardConnectionState;
  reconnect: () => void;
} {
  // 初回接続を即座に張るため、初期状態は reconnecting（マウント時の setState を回避）
  const [connectionState, setConnectionState] =
    useState<DashboardConnectionState>("reconnecting");
  const eventSourceRef = useRef<EventSource | null>(null);
  const retryCountRef = useRef(0);
  const retryTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onSnapshotRef = useRef(onSnapshot);
  // connect の自己参照を避けるための間接参照（バックオフ再接続の setTimeout 用）
  const connectRef = useRef<() => void>(() => {});

  // onSnapshot の最新値を常に参照できるようにする（再接続ループ防止）
  useEffect(() => {
    onSnapshotRef.current = onSnapshot;
  }, [onSnapshot]);

  const cleanup = useCallback(() => {
    if (retryTimeoutRef.current) {
      clearTimeout(retryTimeoutRef.current);
      retryTimeoutRef.current = null;
    }
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
  }, []);

  const connect = useCallback(() => {
    cleanup();
    // reconnecting の設定は呼び出し側（reconnect / onerror）で行い、
    // effect 内での同期 setState を避ける

    const es = new EventSource(`${API_BASE}${DASHBOARD_SSE_PATH}`);
    eventSourceRef.current = es;

    es.onopen = () => {
      setConnectionState("live");
      retryCountRef.current = 0;
    };

    es.onmessage = (event) => {
      try {
        const parsed: unknown = JSON.parse(event.data);
        if (isDashboardStateShape(parsed)) {
          onSnapshotRef.current(normalizeSnapshot(parsed));
        }
      } catch {
        // パースエラーは無視（次の push を待つ）
      }
    };

    es.onerror = () => {
      es.close();
      eventSourceRef.current = null;

      if (retryCountRef.current < MAX_RETRIES) {
        const delay = Math.min(
          INITIAL_RETRY_DELAY * Math.pow(2, retryCountRef.current),
          MAX_RETRY_DELAY
        );
        retryCountRef.current += 1;
        // バックオフ中を落とさず reconnecting を維持（fe-W4）
        setConnectionState("reconnecting");
        retryTimeoutRef.current = setTimeout(() => connectRef.current(), delay);
      } else {
        setConnectionState("offline");
      }
    };
  }, [cleanup]);

  // connect の最新値を ref に同期（setTimeout からの間接呼び出し用・依存に connect を増やさない）
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  // 手動再接続（visibilitychange 復帰時に呼ぶ）。ハンドラ内なので setState は安全
  const reconnect = useCallback(() => {
    retryCountRef.current = 0;
    setConnectionState("reconnecting");
    connect();
  }, [connect]);

  // 初回接続
  useEffect(() => {
    connect();
    return cleanup;
  }, [connect, cleanup]);

  // visibilitychange: タブ復帰時にリトライカウントをリセットして即再接続
  useEffect(() => {
    function handleVisibilityChange() {
      if (document.visibilityState === "visible") {
        if (
          eventSourceRef.current &&
          eventSourceRef.current.readyState === EventSource.OPEN
        ) {
          return;
        }
        reconnect();
      }
    }
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [reconnect]);

  return { connectionState, reconnect };
}
