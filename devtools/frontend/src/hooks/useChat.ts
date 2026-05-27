"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  listSessions,
  sendPrompt,
  openEventStream,
  getHistory,
} from "@/lib/chatApi";
import { LOCAL_STORAGE_ACTIVE_SESSION_ID_KEY, GHOSTRUNNER_CWD } from "@/lib/constants";
import type { ChatSession, ChatStreamEvent, ChatHistoryItem } from "@/types/chat";
const MAX_RETRIES = 10;
const SILENCE_TIMEOUT_MS = 3000;
const HISTORY_REPLAY_LIMIT = 5;

type ChatStatus = "idle" | "busy" | "error";
type ConnectionState = "live" | "reconnecting" | "offline";

interface UseChatProps {
  onComplete?: (fullText: string) => void;
  // session 切替時に親側で TTS をキャンセルするためのコールバック
  onSessionSwitch?: () => void;
}

interface UseChatReturn {
  send: (text: string) => Promise<void>;
  responseText: string;
  isStreaming: boolean;
  status: ChatStatus;
  error: string | null;
  sessionId: string | null;
  sessions: ChatSession[];
  connectionState: ConnectionState;
  refresh: () => Promise<void>;
  switchSession: (sid: string) => void;
  startNewSession: () => void;
  fetchSessions: () => Promise<void>;
}

// 履歴アイテムから「最後のアシスタント発話のみ」を抽出する
// 配列を逆順スキャンし、user が現れた時点で停止することで複数 turn を取り込まない
function extractAssistantText(items: ChatHistoryItem[]): string {
  const chunks: string[] = [];
  for (let i = items.length - 1; i >= 0; i--) {
    const it = items[i];
    const role = typeof it.role === "string" ? it.role : "";
    const type = typeof it.type === "string" ? it.type : "";
    if (role === "user") {
      break;
    }
    const isAssistant =
      role === "assistant" || type === "assistant" || type === "text_delta";
    if (isAssistant && typeof it.text === "string") {
      chunks.unshift(it.text);
    }
  }
  return chunks.join("");
}

export function useChat(props?: UseChatProps): UseChatReturn {
  const [responseText, setResponseText] = useState("");
  const [isStreaming, setIsStreaming] = useState(false);
  const [status, setStatus] = useState<ChatStatus>("idle");
  const [error, setError] = useState<string | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [connectionState, setConnectionState] = useState<ConnectionState>("offline");

  const eventSourceRef = useRef<EventSource | null>(null);
  const retryCountRef = useRef(0);
  const retryTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const silenceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const accumulatedTextRef = useRef("");
  const isStreamingRef = useRef(false);
  const onCompleteRef = useRef(props?.onComplete);
  const onSessionSwitchRef = useRef(props?.onSessionSwitch);
  const sessionIdRef = useRef<string | null>(null);
  const receivedAnyEventRef = useRef(false);
  const isNewSessionRef = useRef(false);
  // visible 復帰時の getHistory 多重発火を防ぐ in-flight ガード
  const isHistoryReplayInFlightRef = useRef(false);

  useEffect(() => {
    onCompleteRef.current = props?.onComplete;
  }, [props?.onComplete]);

  useEffect(() => {
    onSessionSwitchRef.current = props?.onSessionSwitch;
  }, [props?.onSessionSwitch]);

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

      setConnectionState("reconnecting");

      const es = openEventStream(sid);
      eventSourceRef.current = es;

      es.onopen = () => {
        retryCountRef.current = 0;
        setConnectionState("live");
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
          setConnectionState("reconnecting");
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
          setConnectionState("offline");
        }
      };
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [handleCompletion]
  );

  // セッション一覧を取得して最新を選択
  const fetchAndSelectSession = useCallback(async (): Promise<string | null> => {
    // cwd 指定で取得
    let list = await listSessions({
      cwd: GHOSTRUNNER_CWD,
      provider: "claude",
    });

    // 空なら cwd 未指定で再試行
    if (list.length === 0) {
      list = await listSessions({ provider: "claude" });
    }

    setSessions(list);

    if (list.length === 0) {
      return null;
    }

    const latest = list[0];
    setSessionId(latest.id);
    sessionIdRef.current = latest.id;
    localStorage.setItem(LOCAL_STORAGE_ACTIVE_SESSION_ID_KEY, latest.id);
    return latest.id;
  }, []);

  // セッション一覧のみ取得（state 更新）
  const fetchSessions = useCallback(async (): Promise<void> => {
    try {
      let list = await listSessions({
        cwd: GHOSTRUNNER_CWD,
        provider: "claude",
      });
      if (list.length === 0) {
        list = await listSessions({ provider: "claude" });
      }
      setSessions(list);
    } catch {
      // 一覧取得失敗は黙ってスキップ（既存 session には影響しない）
    }
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
        // sessions state も初期化（picker 表示用）
        void fetchSessions();
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
        setConnectionState("offline");
      } else {
        // visible: 再接続 + 履歴 replay
        if (sessionIdRef.current && !eventSourceRef.current) {
          retryCountRef.current = 0;
          connectSSE(sessionIdRef.current);

          // 背景中に流れた text_delta の取りこぼし対策。失敗は黙ってスキップ
          // TTS は autoplay 制約のため呼ばない（静かに反映するのみ）
          // iOS Safari の visibilitychange 多重発火対策として in-flight ガードを掛ける
          const sidForReplay = sessionIdRef.current;
          if (!isHistoryReplayInFlightRef.current) {
            isHistoryReplayInFlightRef.current = true;
            (async () => {
              try {
                const items = await getHistory(sidForReplay, HISTORY_REPLAY_LIMIT);
                const text = extractAssistantText(items);
                // SSE の text_delta が先に到着しているケースを想定し、
                // 履歴の方が「進んでいる」（長い）場合のみ反映する
                if (text && text.length > accumulatedTextRef.current.length) {
                  accumulatedTextRef.current = text;
                  setResponseText(text);
                }
              } catch {
                // 履歴取得失敗は黙ってスキップ
              } finally {
                isHistoryReplayInFlightRef.current = false;
              }
            })();
          }
        }
      }
    }

    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [connectSSE]);

  // 内部 state リセット（switch/new で共有）
  const resetChatState = useCallback(() => {
    setResponseText("");
    setError(null);
    setStatus("idle");
    setIsStreaming(false);
    accumulatedTextRef.current = "";
    receivedAnyEventRef.current = false;
    clearSilenceTimer();
  }, []);

  // 既存セッションへの切替
  const switchSession = useCallback(
    (sid: string) => {
      if (sid === sessionIdRef.current) {
        return;
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
        // close と次の connectSSE("reconnecting") の中間状態を明示する
        // startNewSession との一貫性のため
        setConnectionState("offline");
      }
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
        retryTimeoutRef.current = null;
      }
      resetChatState();

      sessionIdRef.current = sid;
      setSessionId(sid);
      localStorage.setItem(LOCAL_STORAGE_ACTIVE_SESSION_ID_KEY, sid);
      isNewSessionRef.current = false;
      retryCountRef.current = 0;

      connectSSE(sid);

      try {
        onSessionSwitchRef.current?.();
      } catch {
        // コールバック例外は黙って無視（TTS cancel 等が失敗しても致命ではない）
      }
    },
    [connectSSE, resetChatState]
  );

  // 新規セッション開始
  const startNewSession = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (retryTimeoutRef.current) {
      clearTimeout(retryTimeoutRef.current);
      retryTimeoutRef.current = null;
    }
    resetChatState();

    sessionIdRef.current = null;
    setSessionId(null);
    localStorage.removeItem(LOCAL_STORAGE_ACTIVE_SESSION_ID_KEY);
    isNewSessionRef.current = true;
    retryCountRef.current = 0;
    setConnectionState("offline");

    try {
      onSessionSwitchRef.current?.();
    } catch {
      // コールバック例外は黙って無視
    }
  }, [resetChatState]);

  // POST /api/prompt のレスポンス body から新 SID を取得
  async function readNewSidFromResponse(res: Response): Promise<string | null> {
    try {
      const json = (await res.clone().json()) as { sessionId?: string };
      if (json && typeof json.sessionId === "string" && json.sessionId.length > 0) {
        return json.sessionId;
      }
      return null;
    } catch {
      return null;
    }
  }

  // 送信
  const send = useCallback(
    async (text: string) => {
      setError(null);
      accumulatedTextRef.current = "";
      receivedAnyEventRef.current = false;
      setResponseText("");
      setStatus("busy");
      setIsStreaming(false);

      // 新規 session 開始モード: sessionId 省略で POST
      if (isNewSessionRef.current || sessionIdRef.current === null) {
        try {
          const res = await sendPrompt({
            sessionId: null,
            text,
            cwd: GHOSTRUNNER_CWD,
          });

          if (!res.ok) {
            setError(`送信に失敗しました: ${res.status}`);
            setStatus("error");
            return;
          }

          const newSid = await readNewSidFromResponse(res);
          if (!newSid) {
            setError("新規セッションの発行に失敗しました");
            setStatus("error");
            return;
          }

          sessionIdRef.current = newSid;
          setSessionId(newSid);
          localStorage.setItem(LOCAL_STORAGE_ACTIVE_SESSION_ID_KEY, newSid);
          isNewSessionRef.current = false;
          retryCountRef.current = 0;
          connectSSE(newSid);
          return;
        } catch {
          setError("送信に失敗しました");
          setStatus("error");
          return;
        }
      }

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
    sessions,
    connectionState,
    refresh,
    switchSession,
    startNewSession,
    fetchSessions,
  };
}
