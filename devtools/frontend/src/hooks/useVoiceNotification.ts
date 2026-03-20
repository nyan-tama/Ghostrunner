import { useState, useCallback, useRef, useEffect } from "react";
import { useOpenAIRealtime } from "@/hooks/useOpenAIRealtime";
import { LOCAL_STORAGE_VOICE_NOTIFICATION_KEY } from "@/lib/constants";
import type { OpenAIConnectionStatus } from "@/types/openai";

// 通知メッセージの最大長
const MAX_MESSAGE_LENGTH = 200;

interface PendingNotification {
  type: "completion" | "error";
  message: string;
}

interface UseVoiceNotificationReturn {
  enabled: boolean;
  setEnabled: (enabled: boolean) => void;
  connectionStatus: OpenAIConnectionStatus;
  isRecording: boolean;
  error: string | null;
  notifyCompletion: (summary: string) => void;
  notifyError: (error: string) => void;
  startRecording: () => Promise<void>;
  stopRecording: () => void;
}

/**
 * 音声通知機能をカプセル化するフック
 * - トグルONで OpenAI Realtime API に自動接続
 * - 処理完了/エラー時に音声で通知
 * - マイクボタンで対話モードに移行可能
 */
export function useVoiceNotification(): UseVoiceNotificationReturn {
  // localStorage からトグル状態を復元
  const [enabled, setEnabledState] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem(LOCAL_STORAGE_VOICE_NOTIFICATION_KEY) === "true";
  });

  // 通知キュー（再接続中に通知が発生した場合に使用）
  const pendingNotificationsRef = useRef<PendingNotification[]>([]);

  const {
    connectionStatus,
    isRecording,
    error,
    connect,
    disconnect,
    startRecording: openaiStartRecording,
    stopRecording: openaiStopRecording,
    sendText,
  } = useOpenAIRealtime({
    instructions: "あなたはGhostrunnerの音声アシスタントです。処理結果を簡潔に日本語で伝えてください。",
  });

  /**
   * 通知キューを処理
   */
  const processNotificationQueue = useCallback(() => {
    if (connectionStatus !== "connected") return;
    if (pendingNotificationsRef.current.length === 0) return;

    // キューから1つずつ取り出して送信
    const notification = pendingNotificationsRef.current.shift();
    if (notification) {
      const prefix = notification.type === "completion" ? "処理が完了しました。" : "エラーが発生しました。";
      sendText(`${prefix}${notification.message}`);
    }
  }, [connectionStatus, sendText]);

  // 接続状態が connected になったらキューを処理
  useEffect(() => {
    if (connectionStatus === "connected") {
      processNotificationQueue();
    }
  }, [connectionStatus, processNotificationQueue]);

  /**
   * トグル状態を変更（true で接続、false で切断）
   */
  const setEnabled = useCallback((newEnabled: boolean) => {
    setEnabledState(newEnabled);
    localStorage.setItem(LOCAL_STORAGE_VOICE_NOTIFICATION_KEY, String(newEnabled));

    if (newEnabled) {
      connect();
    } else {
      disconnect();
      pendingNotificationsRef.current = [];
    }
  }, [connect, disconnect]);

  /**
   * 初回マウント時: enabled が true なら接続
   */
  useEffect(() => {
    if (enabled) {
      connect();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /**
   * メッセージを切り詰める
   */
  const truncateMessage = (message: string): string => {
    if (message.length <= MAX_MESSAGE_LENGTH) {
      return message;
    }
    return message.substring(0, MAX_MESSAGE_LENGTH) + "...";
  };

  /**
   * 完了通知を送信
   */
  const notifyCompletion = useCallback((summary: string) => {
    if (!enabled) return;
    if (isRecording) return; // 録音中は自動通知を抑制

    const truncatedSummary = truncateMessage(summary);

    if (connectionStatus === "connected") {
      sendText(`処理が完了しました。${truncatedSummary}`);
    } else {
      // 再接続中の場合はキューに追加
      pendingNotificationsRef.current.push({
        type: "completion",
        message: truncatedSummary,
      });
    }
  }, [enabled, isRecording, connectionStatus, sendText]);

  /**
   * エラー通知を送信
   */
  const notifyError = useCallback((errorMessage: string) => {
    if (!enabled) return;
    if (isRecording) return; // 録音中は自動通知を抑制

    const truncatedError = truncateMessage(errorMessage);

    if (connectionStatus === "connected") {
      sendText(`エラーが発生しました。${truncatedError}`);
    } else {
      // 再接続中の場合はキューに追加
      pendingNotificationsRef.current.push({
        type: "error",
        message: truncatedError,
      });
    }
  }, [enabled, isRecording, connectionStatus, sendText]);

  /**
   * マイク入力を開始
   */
  const startRecording = useCallback(async () => {
    await openaiStartRecording();
  }, [openaiStartRecording]);

  /**
   * マイク入力を停止
   */
  const stopRecording = useCallback(() => {
    openaiStopRecording();
  }, [openaiStopRecording]);

  return {
    enabled,
    setEnabled,
    connectionStatus,
    isRecording,
    error,
    notifyCompletion,
    notifyError,
    startRecording,
    stopRecording,
  };
}
