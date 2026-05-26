"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  listSessions,
  sendPrompt,
  openEventStream,
} from "@/lib/chatApi";
import { LOCAL_STORAGE_ACTIVE_SESSION_ID_KEY, GHOSTRUNNER_CWD } from "@/lib/constants";
import type { ChatStreamEvent } from "@/types/chat";
const MAX_RETRIES = 10;
const SILENCE_TIMEOUT_MS = 3000;

type ChatStatus = "idle" | "busy" | "error";

interface UseChatProps {
  onComplete?: (fullText: string) => void;
}

interface UseChatReturn {
  send: (text: string) => Promise<void>;
  responseText: string;
  isStreaming: boolean;
  status: ChatStatus;
  error: string | null;
  sessionId: string | null;
  refresh: () => Promise<void>;
}

export function useChat(props?: UseChatProps): UseChatReturn {
  const [responseText, setResponseText] = useState("");
  const [isStreaming, setIsStreaming] = useState(false);
  const [status, setStatus] = useState<ChatStatus>("idle");
  const [error, setError] = useState<string | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);

  const eventSourceRef = useRef<EventSource | null>(null);
  const retryCountRef = useRef(0);
  const retryTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const silenceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const accumulatedTextRef = useRef("");
  const isStreamingRef = useRef(false);
  const onCompleteRef = useRef(props?.onComplete);
  const sessionIdRef = useRef<string | null>(null);
  const receivedAnyEventRef = useRef(false);

  useEffect(() => {
    onCompleteRef.current = props?.onComplete;
  }, [props?.onComplete]);

  useEffect(() => {
    isStreamingRef.current = isStreaming;
  }, [isStreaming]);

  // セッション完了処理
  const handleCompletion = useCallback(() => {
    setIsStreaming(false);
    setStatus("idle");
    clearSilenceTimer();
    if (accumulatedTextRef.current && onCompleteRef.current) {
      onCompleteRef.current(accumulatedTextRef.current);
    }
  }, []);

  function clearSilenceTimer() {
    if (silenceTimerRef.current) {
      clearTimeout(silenceTimerRef.current);
      silenceTimerRef.current = null;
    }
  }

  function resetSilenceTimer(onTimeout: () => void) {
    clearSilenceTimer();
    silenceTimerRef.current = setTimeout(onTimeout, SILENCE_TIMEOUT_MS);
  }

  // SSE 接続
  const connectSSE = useCallback(
    (sid: string) => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }

      const es = openEventStream(sid);
      eventSourceRef.current = es;

      es.onopen = () => {
        retryCountRef.current = 0;
      };

      es.onmessage = (event) => {
        try {
          const data: ChatStreamEvent = JSON.parse(event.data);
          receivedAnyEventRef.current = true;

          switch (data.type) {
            case "text_delta": {
              accumulatedTextRef.current += data.text;
              setResponseText(accumulatedTextRef.current);
              setIsStreaming(true);
              setStatus("busy");
              resetSilenceTimer(handleCompletion);
              break;
            }
            case "result": {
              handleCompletion();
              break;
            }
            case "status": {
              if (data.state === "busy") {
                setStatus("busy");
              } else if (data.state === "idle") {
                // idle で streaming 中なら完了扱い
                if (accumulatedTextRef.current) {
                  handleCompletion();
                } else {
                  setStatus("idle");
                }
              }
              break;
            }
            case "error": {
              setError(data.message);
              setStatus("error");
              setIsStreaming(false);
              clearSilenceTimer();
              break;
            }
            default:
              // running_stats, tool_start, tool_end 等は MVP では無視
              // 無音タイマーはリセット（イベントが来ている証拠）
              if (isStreamingRef.current) {
                resetSilenceTimer(handleCompletion);
              }
              break;
          }
        } catch {
          // JSON パースエラーは無視
        }
      };

      es.onerror = () => {
        es.close();
        eventSourceRef.current = null;

        if (retryCountRef.current < MAX_RETRIES) {
          const delay = Math.min(
            1000 * Math.pow(2, retryCountRef.current),
            8000
          );
          retryCountRef.current += 1;
          retryTimeoutRef.current = setTimeout(() => {
            retryTimeoutRef.current = null;
            if (sessionIdRef.current) {
              connectSSE(sessionIdRef.current);
            }
          }, delay);
        } else {
          setError("SSE 接続に失敗しました（再接続上限）");
          setStatus("error");
          setIsStreaming(false);
        }
      };
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [handleCompletion]
  );

  // セッション一覧を取得して最新を選択
  const fetchAndSelectSession = useCallback(async (): Promise<string | null> => {
    // cwd 指定で取得
    let sessions = await listSessions({
      cwd: GHOSTRUNNER_CWD,
      provider: "claude",
    });

    // 空なら cwd 未指定で再試行
    if (sessions.length === 0) {
      sessions = await listSessions({ provider: "claude" });
    }

    if (sessions.length === 0) {
      return null;
    }

    const latest = sessions[0];
    setSessionId(latest.id);
    sessionIdRef.current = latest.id;
    localStorage.setItem(LOCAL_STORAGE_ACTIVE_SESSION_ID_KEY, latest.id);
    return latest.id;
  }, []);

  // マウント時: セッション復元・取得 + SSE 接続
  useEffect(() => {
    async function init() {
      // localStorage から復元
      const stored = localStorage.getItem(LOCAL_STORAGE_ACTIVE_SESSION_ID_KEY);
      if (stored) {
        setSessionId(stored);
        sessionIdRef.current = stored;
        connectSSE(stored);
        return;
      }

      const sid = await fetchAndSelectSession();
      if (sid) {
        connectSSE(sid);
      }
    }
    init();

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
      }
      clearSilenceTimer();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // visibilitychange 連動
  useEffect(() => {
    function handleVisibility() {
      if (document.visibilityState === "hidden") {
        if (eventSourceRef.current) {
          eventSourceRef.current.close();
          eventSourceRef.current = null;
        }
        if (retryTimeoutRef.current) {
          clearTimeout(retryTimeoutRef.current);
          retryTimeoutRef.current = null;
        }
      } else {
        // visible: 再接続
        if (sessionIdRef.current && !eventSourceRef.current) {
          retryCountRef.current = 0;
          connectSSE(sessionIdRef.current);
        }
      }
    }

    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [connectSSE]);

  // 送信
  const send = useCallback(
    async (text: string) => {
      if (!sessionIdRef.current) {
        setError("セッションが選択されていません");
        return;
      }

      setError(null);
      accumulatedTextRef.current = "";
      receivedAnyEventRef.current = false;
      setResponseText("");
      setStatus("busy");
      setIsStreaming(false);

      try {
        const res = await sendPrompt({
          sessionId: sessionIdRef.current,
          text,
          cwd: GHOSTRUNNER_CWD,
        });

        // 4xx: セッション無効 -> 再取得してリトライ1回
        if (res.status >= 400 && res.status < 500) {
          const newSid = await fetchAndSelectSession();
          if (!newSid) {
            setError("有効なセッションが見つかりません");
            setStatus("error");
            return;
          }
          connectSSE(newSid);
          const retryRes = await sendPrompt({
            sessionId: newSid,
            text,
            cwd: GHOSTRUNNER_CWD,
          });
          if (!retryRes.ok) {
            setError(`送信に失敗しました: ${retryRes.status}`);
            setStatus("error");
            return;
          }
        } else if (!res.ok) {
          setError(`送信に失敗しました: ${res.status}`);
          setStatus("error");
        }
      } catch {
        setError("送信に失敗しました");
        setStatus("error");
      }
    },
    [connectSSE, fetchAndSelectSession]
  );

  // セッション再取得
  const refresh = useCallback(async () => {
    const sid = await fetchAndSelectSession();
    if (sid) {
      retryCountRef.current = 0;
      connectSSE(sid);
    }
  }, [fetchAndSelectSession, connectSSE]);

  return {
    send,
    responseText,
    isStreaming,
    status,
    error,
    sessionId,
    refresh,
  };
}
