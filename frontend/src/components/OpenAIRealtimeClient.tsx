"use client";

import { useCallback } from "react";
import { useOpenAIRealtime } from "@/hooks/useOpenAIRealtime";
import type { OpenAIConnectionStatus } from "@/types/openai";

// 接続状態に応じた色を返す
function getStatusColor(status: OpenAIConnectionStatus): string {
  switch (status) {
    case "connected":
      return "bg-green-500";
    case "connecting":
      return "bg-yellow-500";
    case "error":
      return "bg-red-500";
    default:
      return "bg-gray-400";
  }
}

// 接続状態に応じたテキストを返す
function getStatusText(status: OpenAIConnectionStatus): string {
  switch (status) {
    case "connected":
      return "Connected";
    case "connecting":
      return "Connecting...";
    case "error":
      return "Error";
    default:
      return "Disconnected";
  }
}

export default function OpenAIRealtimeClient() {
  const {
    connectionStatus,
    isRecording,
    error,
    connect,
    disconnect,
    startRecording,
    stopRecording,
  } = useOpenAIRealtime({
    instructions: "You are a helpful voice assistant. Respond in a conversational and friendly manner.",
  });

  const isConnected = connectionStatus === "connected";
  const isConnecting = connectionStatus === "connecting";

  const handleConnectClick = useCallback(() => {
    if (isConnected || isConnecting) {
      disconnect();
    } else {
      connect();
    }
  }, [isConnected, isConnecting, connect, disconnect]);

  const handleRecordClick = useCallback(() => {
    if (isRecording) {
      stopRecording();
    } else {
      startRecording();
    }
  }, [isRecording, startRecording, stopRecording]);

  return (
    <div className="max-w-[600px] mx-auto px-5 py-10 bg-gray-100 min-h-screen">
      <h1 className="text-gray-800 text-2xl font-bold mb-8 text-center">
        OpenAI Realtime
      </h1>

      {/* 接続状態インジケーター */}
      <div className="flex items-center justify-center gap-3 mb-8">
        <div
          className={`w-3 h-3 rounded-full ${getStatusColor(connectionStatus)}`}
        />
        <span className="text-gray-600">{getStatusText(connectionStatus)}</span>
      </div>

      {/* エラー表示 */}
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-6">
          {error}
        </div>
      )}

      {/* ボタンエリア */}
      <div className="flex flex-col items-center gap-4">
        {/* 接続ボタン */}
        <button
          onClick={handleConnectClick}
          disabled={isConnecting}
          className={`
            w-48 px-6 py-3 rounded-lg font-medium transition-colors
            ${
              isConnected || isConnecting
                ? "bg-red-600 text-white hover:bg-red-700"
                : "bg-blue-600 text-white hover:bg-blue-700"
            }
            disabled:opacity-50 disabled:cursor-not-allowed
          `}
        >
          {isConnecting
            ? "Connecting..."
            : isConnected
              ? "Disconnect"
              : "Connect"}
        </button>

        {/* マイクボタン */}
        <button
          onClick={handleRecordClick}
          disabled={!isConnected}
          className={`
            w-48 px-6 py-3 rounded-lg font-medium transition-colors
            ${
              isRecording
                ? "bg-orange-600 text-white hover:bg-orange-700"
                : "bg-green-600 text-white hover:bg-green-700"
            }
            disabled:opacity-50 disabled:cursor-not-allowed
          `}
        >
          {isRecording ? "Stop Recording" : "Start Recording"}
        </button>
      </div>

      {/* 使い方説明 */}
      <div className="mt-12 text-gray-500 text-sm text-center">
        <p className="mb-2">
          1. Click &quot;Connect&quot; to establish connection with OpenAI Realtime API
        </p>
        <p className="mb-2">
          2. Click &quot;Start Recording&quot; to begin voice input
        </p>
        <p>
          3. Speak into your microphone and listen to the AI response
        </p>
      </div>

      {/* デバッグ情報（開発時のみ） */}
      {process.env.NODE_ENV === "development" && (
        <div className="mt-8 p-4 bg-gray-200 rounded text-xs font-mono">
          <p>Connection Status: {connectionStatus}</p>
          <p>Is Recording: {isRecording ? "Yes" : "No"}</p>
          <p>Error: {error || "None"}</p>
        </div>
      )}
    </div>
  );
}
