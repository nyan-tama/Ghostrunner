"use client";

import { useEffect, useState } from "react";
import type { IdleState } from "@/types/dashboard";

interface WaitingBadgeProps {
  idle: IdleState;
  // テスト用に現在時刻（ms）を注入可能。省略時は内部クロック（毎分自走更新）
  now?: number;
}

// idle.timestamp（RFC3339）から現在までの待機分を算出する
function waitingMinutes(timestamp: string, nowMs: number): number {
  const started = new Date(timestamp).getTime();
  if (Number.isNaN(started)) {
    return 0;
  }
  const diffMs = nowMs - started;
  if (diffMs < 0) {
    return 0;
  }
  return Math.floor(diffMs / 60000);
}

// 質問待ちバッジ。`[質問待ち N分]` + 「何を待っているか」の1行を表示する（描画のみ）。
// summary があれば summary、無ければ preview を暫定表示。summary 未生成なら「(要約中…)」、
// preview も summary も無ければ「(プレビューなし)」。
export default function WaitingBadge({ idle, now }: WaitingBadgeProps) {
  // 現在時刻は render 外（内部クロック）で保持し、prop 未指定時のみ毎分更新する。
  // now prop が渡された場合（テスト）は interval を張らずその値を使う。
  const [tick, setTick] = useState<number>(() => Date.now());

  useEffect(() => {
    if (now !== undefined) {
      return;
    }
    const timer = setInterval(() => setTick(Date.now()), 60000);
    return () => clearInterval(timer);
  }, [now]);

  const nowMs = now ?? tick;
  const minutes = waitingMinutes(idle.timestamp, nowMs);

  const summaryText = idle.summary.trim();
  const previewText = idle.preview.trim();
  const summarizing = summaryText === "";
  const detail = summarizing ? previewText || "(プレビューなし)" : summaryText;

  return (
    <div className="mt-2 flex flex-col gap-0.5">
      <span className="inline-flex w-fit items-center rounded bg-red-100 px-1.5 py-0.5 text-xs font-semibold text-red-700">
        [質問待ち {minutes}分]
      </span>
      <div className="text-xs text-gray-700">
        {summarizing && <span className="mr-1 text-gray-400">(要約中…)</span>}
        <span>{detail}</span>
      </div>
    </div>
  );
}
