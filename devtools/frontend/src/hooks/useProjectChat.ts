"use client";

import { useState, useCallback, useRef } from "react";
import type { StreamEvent, Question } from "@/types";
import { executeCommandStream, continueSessionStream } from "@/lib/api";
import { useSSEStream } from "@/hooks/useSSEStream";

export interface ChatItem {
  id: string;
  type: "ai" | "user" | "tool" | "error";
  content: string;
  toolName?: string;
}

export type ChatPhase = "idle" | "chatting" | "complete" | "error";

interface UseProjectChatReturn {
  messages: ChatItem[];
  isStreaming: boolean;
  sessionId: string | null;
  phase: ChatPhase;
  currentQuestion: Question | null;
  createdPath: string | null;
  startChat: () => void;
  sendAnswer: (answer: string) => void;
  reset: () => void;
}

function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}

export function useProjectChat(): UseProjectChatReturn {
  const [messages, setMessages] = useState<ChatItem[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [phase, setPhase] = useState<ChatPhase>("idle");
  const [currentQuestion, setCurrentQuestion] = useState<Question | null>(null);
  const [createdPath, setCreatedPath] = useState<string | null>(null);

  const abortControllerRef = useRef<AbortController | null>(null);
  // 連続する text イベントをマージするための参照
  const lastTextIdRef = useRef<string | null>(null);

  const appendMessage = useCallback((item: ChatItem) => {
    setMessages((prev) => [...prev, item]);
  }, []);

  const handleStreamEvent = useCallback(
    (event: StreamEvent) => {
      if (event.session_id) {
        setSessionId(event.session_id);
      }

      switch (event.type) {
        case "text": {
          if (!event.message) break;
          // 連続する text イベントは1つのメッセージにマージ
          if (lastTextIdRef.current) {
            const lastId = lastTextIdRef.current;
            setMessages((prev) =>
              prev.map((m) =>
                m.id === lastId
                  ? { ...m, content: m.content + event.message }
                  : m
              )
            );
          } else {
            const id = generateId();
            lastTextIdRef.current = id;
            appendMessage({ id, type: "ai", content: event.message });
          }
          break;
        }

        case "question": {
          // text マージをリセット
          lastTextIdRef.current = null;
          setIsStreaming(false);
          if (event.result?.questions && event.result.questions.length > 0) {
            const q = event.result.questions[0];
            // 質問テキストをAIメッセージとして追加
            const questionText = q.header
              ? `${q.header}\n\n${q.question}`
              : q.question;
            appendMessage({
              id: generateId(),
              type: "ai",
              content: questionText,
            });
            setCurrentQuestion(q);
          }
          break;
        }

        case "tool_use": {
          // text マージをリセット
          lastTextIdRef.current = null;
          if (event.tool_name) {
            const detail = event.message || event.tool_name;
            appendMessage({
              id: generateId(),
              type: "tool",
              content: detail,
              toolName: event.tool_name,
            });
          }
          break;
        }

        case "complete": {
          // text マージをリセット
          lastTextIdRef.current = null;
          setIsStreaming(false);

          if (event.result) {
            if (event.result.questions && event.result.questions.length > 0) {
              // complete に質問が含まれる場合
              const q = event.result.questions[0];
              const questionText = q.header
                ? `${q.header}\n\n${q.question}`
                : q.question;
              appendMessage({
                id: generateId(),
                type: "ai",
                content: questionText,
              });
              setCurrentQuestion(q);
            } else {
              // 完了
              const output = event.result.output || "";
              if (output) {
                appendMessage({
                  id: generateId(),
                  type: "ai",
                  content: output,
                });
              }
              // output からプロジェクトパスを抽出
              const pathMatch = output.match(/生成先:\s*(\/Users\/\S+)/);
              if (pathMatch) {
                setCreatedPath(pathMatch[1].replace(/\/$/, ""));
              } else {
                // output にない場合、全メッセージから抽出を試みる
                setMessages((prev) => {
                  const allText = prev.filter((m) => m.type === "ai").map((m) => m.content).join("\n");
                  const msgMatch = allText.match(/生成先[:：]\s*(\/Users\/\S+)/);
                  if (msgMatch) {
                    setCreatedPath(msgMatch[1].replace(/\/$/, ""));
                  } else {
                    // /Users/user/xxx パターンで最後のマッチを使う
                    const fallback = allText.match(/\/Users\/user\/[a-z0-9][a-z0-9-]*/g);
                    if (fallback && fallback.length > 0) {
                      setCreatedPath(fallback[fallback.length - 1]);
                    }
                  }
                  return prev;
                });
              }
              setPhase("complete");
            }
          } else {
            setPhase("complete");
          }
          break;
        }

        case "error": {
          // text マージをリセット
          lastTextIdRef.current = null;
          setIsStreaming(false);
          appendMessage({
            id: generateId(),
            type: "error",
            content: event.message || "エラーが発生しました",
          });
          setPhase("error");
          break;
        }

        // init, thinking はストリーム中表示で十分なので、メッセージには追加しない
        default:
          break;
      }
    },
    [appendMessage]
  );

  const handleError = useCallback(
    (error: string) => {
      lastTextIdRef.current = null;
      setIsStreaming(false);
      appendMessage({
        id: generateId(),
        type: "error",
        content: error,
      });
      setPhase("error");
    },
    [appendMessage]
  );

  const handleComplete = useCallback(() => {
    setIsStreaming(false);
  }, []);

  const { processStream } = useSSEStream({
    onEvent: handleStreamEvent,
    onError: handleError,
    onComplete: handleComplete,
  });

  const startChat = useCallback(() => {
    // 既存の接続を中断
    abortControllerRef.current?.abort();
    const controller = new AbortController();
    abortControllerRef.current = controller;

    // 状態をリセット
    setMessages([]);
    setSessionId(null);
    setCurrentQuestion(null);
    lastTextIdRef.current = null;
    setPhase("chatting");
    setIsStreaming(true);

    executeCommandStream(
      { project: "", command: "init", args: "" },
      controller.signal
    )
      .then((response) => processStream(response))
      .catch((error) => {
        if (error instanceof Error && error.name === "AbortError") {
          return;
        }
        handleError(
          "接続に失敗しました: " +
            (error instanceof Error ? error.message : "Unknown error")
        );
      });
  }, [processStream, handleError]);

  const sendAnswer = useCallback(
    (answer: string) => {
      if (!sessionId) {
        handleError("セッションが見つかりません");
        return;
      }

      // ユーザーメッセージを追加
      appendMessage({
        id: generateId(),
        type: "user",
        content: answer,
      });

      // 質問をクリア
      setCurrentQuestion(null);
      lastTextIdRef.current = null;
      setIsStreaming(true);

      // 既存の接続を中断
      abortControllerRef.current?.abort();
      const controller = new AbortController();
      abortControllerRef.current = controller;

      continueSessionStream(
        { project: "", session_id: sessionId, answer },
        controller.signal
      )
        .then((response) => processStream(response))
        .catch((error) => {
          if (error instanceof Error && error.name === "AbortError") {
            return;
          }
          handleError(
            "接続に失敗しました: " +
              (error instanceof Error ? error.message : "Unknown error")
          );
        });
    },
    [sessionId, processStream, handleError, appendMessage]
  );

  const reset = useCallback(() => {
    abortControllerRef.current?.abort();
    abortControllerRef.current = null;
    setMessages([]);
    setIsStreaming(false);
    setSessionId(null);
    setPhase("idle");
    setCurrentQuestion(null);
    setCreatedPath(null);
    lastTextIdRef.current = null;
  }, []);

  return {
    messages,
    isStreaming,
    sessionId,
    phase,
    currentQuestion,
    createdPath,
    startChat,
    sendAnswer,
    reset,
  };
}
