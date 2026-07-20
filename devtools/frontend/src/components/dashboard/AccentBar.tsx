"use client";

import type { Attention } from "@/types/dashboard";

interface AccentBarProps {
  attention: Attention;
  hasUnanswered: boolean;
  // 質問待ちを独立軸として最優先色にする（fe-W6・attention に畳まない）
  hasWaiting?: boolean;
}

function getBarColor(
  attention: Attention,
  hasUnanswered: boolean,
  hasWaiting: boolean
): string {
  // 質問待ちは「未回答由来 required」と色を分離し、最優先で強調する
  if (hasWaiting) {
    return "bg-red-600 animate-pulse";
  }
  switch (attention) {
    case "required":
      return "bg-red-500";
    case "progress":
      return hasUnanswered ? "bg-yellow-400" : "bg-blue-500";
    case "watching":
      return "bg-gray-300";
  }
}

export default function AccentBar({
  attention,
  hasUnanswered,
  hasWaiting = false,
}: AccentBarProps) {
  return (
    <div
      className={`absolute left-0 top-0 bottom-0 w-1 rounded-l-lg ${getBarColor(attention, hasUnanswered, hasWaiting)}`}
    />
  );
}
