"use client";

import type { SSEConnectionStatus } from "@/hooks/usePatrolSSE";

interface PatrolHeaderProps {
  isRunning: boolean;
  isPolling: boolean;
  isLoading: boolean;
  connectionStatus: SSEConnectionStatus;
  onStart: () => void;
  onStop: () => void;
  onTogglePolling: () => void;
}

function ConnectionIndicator({ status }: { status: SSEConnectionStatus }) {
  const colorClass =
    status === "connected"
      ? "bg-green-500"
      : status === "connecting"
        ? "bg-yellow-500 animate-pulse"
        : "bg-red-500";

  const label =
    status === "connected"
      ? "接続済み"
      : status === "connecting"
        ? "接続中..."
        : "切断";

  return (
    <div className="flex items-center gap-1.5 text-xs text-gray-500">
      <span className={`w-2 h-2 rounded-full ${colorClass}`} />
      <span>{label}</span>
    </div>
  );
}

export default function PatrolHeader({
  isRunning,
  isPolling,
  isLoading,
  connectionStatus,
  onStart,
  onStop,
  onTogglePolling,
}: PatrolHeaderProps) {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4 flex items-center justify-between">
      <div className="flex items-center gap-4">
        {isRunning ? (
          <button
            type="button"
            onClick={onStop}
            disabled={isLoading}
            className="px-4 py-2 bg-red-600 text-white text-sm font-medium rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? "処理中..." : "巡回停止"}
          </button>
        ) : (
          <button
            type="button"
            onClick={onStart}
            disabled={isLoading}
            className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? "処理中..." : "巡回開始"}
          </button>
        )}

        <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={isPolling}
            onChange={onTogglePolling}
            className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          ポーリング
        </label>
      </div>

      <ConnectionIndicator status={connectionStatus} />
    </div>
  );
}
