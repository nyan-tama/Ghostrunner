"use client";

import { useEffect, useRef, useState } from "react";
import type { PatrolSSEEvent } from "@/types/patrol";

export type SSEConnectionStatus = "connected" | "connecting" | "disconnected";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";
const INITIAL_RETRY_DELAY = 1000;
const MAX_RETRY_DELAY = 30000;
const MAX_RETRIES = 10;

interface UsePatrolSSEProps {
  onEvent: (event: PatrolSSEEvent) => void;
}

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

  useEffect(() => {
    function cleanup() {
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
        retryTimeoutRef.current = null;
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    }

    function connect() {
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
    }

    connect();
    return cleanup;
  }, []);

  return { connectionStatus };
}
