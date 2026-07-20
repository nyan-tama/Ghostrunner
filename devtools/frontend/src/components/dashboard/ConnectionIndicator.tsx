"use client";

interface ConnectionIndicatorProps {
  state: "live" | "reconnecting" | "offline";
  // 複数系統（chat / 盤）を並べる時の識別用ラベル
  caption?: string;
}

// SSE 接続状態のドット表示。live=緑実線 / reconnecting=黄点滅 / offline=灰
export default function ConnectionIndicator({
  state,
  caption,
}: ConnectionIndicatorProps) {
  let dotClass = "bg-gray-300";
  let label = "切断";
  let title = "SSE 未接続";

  if (state === "live") {
    dotClass = "bg-green-500";
    label = "接続";
    title = "SSE 接続中";
  } else if (state === "reconnecting") {
    dotClass = "bg-yellow-400 animate-pulse";
    label = "再接続";
    title = "SSE 再接続中（バックオフ待ち）";
  }

  return (
    <span
      className="inline-flex items-center gap-1 text-xs text-gray-600"
      title={caption ? `${caption}: ${title}` : title}
      aria-label={`${caption ? `${caption} ` : ""}接続状態: ${label}`}
    >
      {caption && <span className="text-gray-400">{caption}</span>}
      <span className={`inline-block w-2 h-2 rounded-full ${dotClass}`} />
      <span>{label}</span>
    </span>
  );
}
