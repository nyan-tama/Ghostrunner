"use client";

import type { KanbanCounts } from "@/types/dashboard";

interface DevSummaryProps {
  kanban: KanbanCounts;
}

export default function DevSummary({ kanban }: DevSummaryProps) {
  const items = [
    { label: "レビュー", count: kanban.reviewing, color: "text-purple-600" },
    { label: "待ち", count: kanban.waiting, color: "text-orange-600" },
    { label: "実行中", count: kanban.running, color: "text-blue-600" },
    { label: "完了", count: kanban.done, color: "text-green-600" },
  ];

  return (
    <div className="flex gap-3 text-sm">
      {items.map((item) => (
        <span key={item.label} className="flex items-center gap-1">
          <span className="text-gray-500">{item.label}</span>
          <span className={`font-semibold ${item.color}`}>{item.count}</span>
        </span>
      ))}
    </div>
  );
}
