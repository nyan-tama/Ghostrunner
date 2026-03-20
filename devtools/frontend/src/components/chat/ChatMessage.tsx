"use client";

import type { ChatItem } from "@/hooks/useProjectChat";

interface ChatMessageProps {
  item: ChatItem;
}

export default function ChatMessage({ item }: ChatMessageProps) {
  switch (item.type) {
    case "ai":
      return (
        <div className="flex justify-start">
          <div className="max-w-[80%] px-4 py-3 rounded-2xl rounded-tl-sm bg-gray-200 text-gray-800 text-sm whitespace-pre-wrap">
            {item.content}
          </div>
        </div>
      );

    case "user":
      return (
        <div className="flex justify-end">
          <div className="max-w-[80%] px-4 py-3 rounded-2xl rounded-tr-sm bg-blue-600 text-white text-sm whitespace-pre-wrap">
            {item.content}
          </div>
        </div>
      );

    case "tool":
      return (
        <div className="flex justify-start">
          <div className="text-xs text-gray-400 py-1 px-2">
            {item.toolName ? `[${item.toolName}] ` : ""}
            {item.content}
          </div>
        </div>
      );

    case "error":
      return (
        <div className="flex justify-start">
          <div className="max-w-[80%] px-4 py-3 rounded-2xl bg-red-100 text-red-700 text-sm whitespace-pre-wrap border border-red-200">
            {item.content}
          </div>
        </div>
      );
  }
}
