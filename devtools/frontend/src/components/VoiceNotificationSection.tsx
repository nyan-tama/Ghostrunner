"use client";

import type { OpenAIConnectionStatus } from "@/types/openai";

interface VoiceNotificationSectionProps {
  enabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  connectionStatus: OpenAIConnectionStatus;
  isRecording: boolean;
  error: string | null;
  onStartRecording: () => void;
  onStopRecording: () => void;
}

/**
 * 接続状態に応じたドットの色を返す
 */
function getStatusColor(status: OpenAIConnectionStatus): string {
  switch (status) {
    case "connected":
      return "bg-green-500";
    case "connecting":
      return "bg-yellow-500";
    case "error":
      return "bg-red-500";
    case "disconnected":
    default:
      return "bg-gray-400";
  }
}

/**
 * 接続状態に応じたツールチップテキストを返す
 */
function getStatusTooltip(status: OpenAIConnectionStatus, error: string | null): string {
  switch (status) {
    case "connected":
      return "Connected";
    case "connecting":
      return "Connecting...";
    case "error":
      return error || "Connection error";
    case "disconnected":
    default:
      return "Disconnected";
  }
}

/**
 * 音声通知セクション
 * - トグルスイッチ
 * - 接続状態インジケーター
 * - マイクボタン
 */
export default function VoiceNotificationSection({
  enabled,
  onEnabledChange,
  connectionStatus,
  isRecording,
  error,
  onStartRecording,
  onStopRecording,
}: VoiceNotificationSectionProps) {
  const statusColor = getStatusColor(connectionStatus);
  const statusTooltip = getStatusTooltip(connectionStatus, error);

  const handleMicClick = () => {
    if (isRecording) {
      onStopRecording();
    } else {
      onStartRecording();
    }
  };

  return (
    <div className="flex items-center gap-3">
      {/* トグルスイッチ */}
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={enabled}
          onChange={(e) => onEnabledChange(e.target.checked)}
          className="sr-only peer"
        />
        <div className="w-9 h-5 bg-gray-300 rounded-full peer peer-checked:bg-blue-500 peer-focus:ring-2 peer-focus:ring-blue-100 after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full peer-checked:after:border-white" />
      </label>

      {/* ラベルと接続状態ドット */}
      <div className="flex items-center gap-2">
        <span className="text-sm text-gray-700">Voice notification</span>
        <div
          className={`w-2 h-2 rounded-full ${statusColor}`}
          title={statusTooltip}
        />
      </div>

      {/* マイクボタン（有効時のみ表示） */}
      {enabled && connectionStatus === "connected" && (
        <button
          type="button"
          onClick={handleMicClick}
          className={`p-1.5 rounded-full transition-colors ${
            isRecording
              ? "bg-red-500 text-white hover:bg-red-600"
              : "bg-gray-200 text-gray-600 hover:bg-gray-300"
          }`}
          title={isRecording ? "Stop recording" : "Start recording"}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="currentColor"
            className="w-4 h-4"
          >
            <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z" />
            <path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z" />
          </svg>
        </button>
      )}

      {/* エラー表示 */}
      {error && connectionStatus === "error" && (
        <span className="text-xs text-red-600" title={error}>
          Error
        </span>
      )}
    </div>
  );
}
