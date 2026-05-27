"use client";

import { useState } from "react";
import DashboardAnswerForm from "./DashboardAnswerForm";
import type { UnansweredItem } from "@/types/dashboard";

interface UnansweredListProps {
  projectPath: string;
  items: UnansweredItem[];
  onAnswered: () => void;
}

export default function UnansweredList({
  projectPath,
  items,
  onAnswered,
}: UnansweredListProps) {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);

  if (items.length === 0) return null;

  return (
    <div className="mt-2 space-y-1">
      <div className="text-xs font-medium text-yellow-700">
        未回答: {items.length}件
      </div>
      {items.map((item, idx) => (
        <div key={`${item.planPath}-${item.lineStart}`} className="text-sm">
          <button
            type="button"
            onClick={() =>
              setExpandedIndex(expandedIndex === idx ? null : idx)
            }
            className="w-full text-left px-2 py-1.5 rounded hover:bg-yellow-50 transition-colors"
          >
            <span className="text-gray-700">{item.heading || item.questionText}</span>
            <span className="ml-1 text-xs text-gray-400">
              ({item.planPath.split("/").pop()}:{item.lineStart})
            </span>
          </button>

          {expandedIndex === idx && (
            <div className="ml-2">
              <DashboardAnswerForm
                projectPath={projectPath}
                item={item}
                onAnswered={onAnswered}
              />
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
