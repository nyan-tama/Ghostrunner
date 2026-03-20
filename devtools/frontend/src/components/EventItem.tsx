"use client";

import type { DisplayEvent } from "@/types";
import OutputText from "./OutputText";

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
  const dotColor = DOT_COLORS[event.type] || "bg-gray-500";

  return (
    <div className="flex items-start px-5 py-3 border-b border-gray-100 last:border-b-0 animate-slideIn">
      <div className={`w-2 h-2 rounded-full ${dotColor} mr-3 mt-1.5 flex-shrink-0`} />
      <div className="flex-1 min-w-0">
        <div className="font-semibold text-gray-800 text-sm">
          {event.title}
        </div>
        {event.fullText ? (
          <div className="mt-1">
            <OutputText text={event.fullText} />
          </div>
        ) : (
          event.detail && (
            <div className="font-mono text-sm text-gray-500 mt-1 break-all">
              {event.detail}
            </div>
          )
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
