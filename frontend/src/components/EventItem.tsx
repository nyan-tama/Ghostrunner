"use client";

import { useState } from "react";
import type { DisplayEvent } from "@/types";

interface EventItemProps {
  event: DisplayEvent;
}

const DOT_COLORS: Record<string, string> = {
  tool: "bg-blue-500",
  task: "bg-purple-500",
  text: "bg-green-500",
  info: "bg-cyan-500",
  error: "bg-red-500",
  question: "bg-yellow-500",
};

export default function EventItem({ event }: EventItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const dotColor = DOT_COLORS[event.type] || "bg-gray-500";
  const hasFullText = event.fullText && event.fullText.length > 200;

  return (
    <div className="flex items-start px-5 py-3 border-b border-gray-100 last:border-b-0 animate-slideIn">
      <div className={`w-2 h-2 rounded-full ${dotColor} mr-3 mt-1.5 flex-shrink-0`} />
      <div className="flex-1 min-w-0">
        <div className="font-semibold text-gray-800 text-sm">
          {event.title}
          {hasFullText && (
            <span className="font-normal text-gray-400 text-xs ml-2">
              ({event.fullText!.length} chars)
            </span>
          )}
        </div>
        {event.detail && !hasFullText && (
          <div className="font-mono text-sm text-gray-500 mt-1 break-all">
            {event.detail}
          </div>
        )}
        {hasFullText && (
          <>
            {!isExpanded && (
              <div className="font-mono text-sm text-gray-500 mt-1 break-all">
                {event.fullText!.substring(0, 150)}...
              </div>
            )}
            {isExpanded && (
              <div className="font-mono text-sm text-gray-500 mt-1 whitespace-pre-wrap max-h-72 overflow-y-auto bg-gray-50 p-2 rounded">
                {event.fullText}
              </div>
            )}
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="text-xs text-blue-500 hover:underline mt-1 bg-transparent border-none cursor-pointer p-0"
            >
              {isExpanded ? "[Show less]" : "[Show more]"}
            </button>
          </>
        )}
        {event.type === "task" && event.detail && (
          <div className="bg-purple-50 border border-purple-200 rounded-lg p-3 mt-2 ml-5">
            <div className="text-sm text-gray-600 bg-white p-2.5 rounded whitespace-pre-wrap">
              {event.detail}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
