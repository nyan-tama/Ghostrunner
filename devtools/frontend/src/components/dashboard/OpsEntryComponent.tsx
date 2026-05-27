"use client";

import type { OpsEntry } from "@/types/dashboard";
import { isProgressShape, isTodayShape, isStatsShape } from "@/types/dashboard";

interface OpsEntryComponentProps {
  entry: OpsEntry;
}

export default function OpsEntryComponent({ entry }: OpsEntryComponentProps) {
  return (
    <div className="text-sm border border-gray-200 rounded-lg p-2 bg-gray-50">
      <div className="flex items-center justify-between">
        <span className="font-medium text-gray-800">
          {entry.kind} / {entry.account}
        </span>
        <span
          className={`text-xs px-1.5 py-0.5 rounded ${
            entry.status === "running"
              ? "bg-blue-100 text-blue-700"
              : "bg-gray-200 text-gray-600"
          }`}
        >
          {entry.status}
        </span>
      </div>

      <div className="mt-1 flex flex-wrap gap-2 text-xs text-gray-600">
        {isProgressShape(entry.progress) && (
          <span>
            進捗: {entry.progress.index}/{entry.progress.total}
          </span>
        )}
        {isTodayShape(entry.today) && (
          <span>
            本日: {entry.today.count}/{entry.today.target}
          </span>
        )}
        {isStatsShape(entry.stats) && (
          <span>
            実行:{entry.stats.followed} 既存:{entry.stats.already} skip:
            {entry.stats.skipped} err:{entry.stats.error}
          </span>
        )}
        {!isProgressShape(entry.progress) &&
          !isTodayShape(entry.today) &&
          !isStatsShape(entry.stats) &&
          entry.rawExtra && (
            <span className="text-gray-400 break-all">
              {JSON.stringify(entry.rawExtra)}
            </span>
          )}
      </div>

      {entry.stale && (
        <div className="mt-1 text-xs text-red-600 font-medium">
          {entry.staleHours}時間無更新（実行停止疑い）
        </div>
      )}

      {entry.consecutiveErrors >= 3 && (
        <div className="mt-1 text-xs text-red-600 font-medium">
          連続エラー: {entry.consecutiveErrors}回
        </div>
      )}
    </div>
  );
}
