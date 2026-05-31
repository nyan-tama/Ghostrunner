"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import type { PatrolSSEEvent } from "@/types/patrol";

export type SSEConnectionStatus = "connected" | "connecting" | "disconnected";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";
const INITIAL_RETRY_DELAY = 1000;
const MAX_RETRY_DELAY = 30000;
const MAX_RETRIES = 10;

interface UsePatrolSSEProps {
  onEvent: (event: PatrolSSEEvent) => void;
}

/**
 * Patrol SSE 接続フック。
 *
 * - exponential backoff で最大 MAX_RETRIES 回再接続を試みる
 * - visibilitychange でタブ復帰時にリトライカウントをリセットし即再接続
 * - reconnect() を外部に公開し、手動再接続も可能
 */
export function usePatrolSSE({ onEvent }: UsePatrolSSEProps) {
  const [connectionStatus, setConnectionStatus] = useState<SSEConnectionStatus>("disconnected");
  const eventSourceRef = useRef<EventSource | null>(null);
  const retryCountRef = useRef(0);
  const retryTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onEventRef = useRef(onEvent);

  // onEvent の最新値を常に参照できるようにする
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

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
    setConnectionStatus("connecting");

    const es = new EventSource(`${API_BASE}/api/patrol/stream`);
    eventSourceRef.current = es;

    es.onopen = () => {
      setConnectionStatus("connected");
      retryCountRef.current = 0;
    };

    es.onmessage = (event) => {
      try {
        const data: PatrolSSEEvent = JSON.parse(event.data);
        onEventRef.current(data);
      } catch {
        // パースエラーは無視
      }
    };

    es.onerror = () => {
      es.close();
      eventSourceRef.current = null;
      setConnectionStatus("disconnected");

      if (retryCountRef.current < MAX_RETRIES) {
        const delay = Math.min(
          INITIAL_RETRY_DELAY * Math.pow(2, retryCountRef.current),
          MAX_RETRY_DELAY
        );
        retryCountRef.current += 1;
        retryTimeoutRef.current = setTimeout(connect, delay);
      }
    };
  }, [cleanup]);

  // 手動再接続（visibilitychange 復帰時やUIボタンから呼ぶ）
  const reconnect = useCallback(() => {
    retryCountRef.current = 0;
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
        // 既に接続中なら何もしない
        if (
          eventSourceRef.current &&
          eventSourceRef.current.readyState === EventSource.OPEN
        ) {
          return;
        }
        // リトライ上限に達していても復帰時はリセットして再接続
        reconnect();
      }
    }

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [reconnect]);

  return { connectionStatus, reconnect };
}
