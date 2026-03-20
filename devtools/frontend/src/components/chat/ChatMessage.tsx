"use client";

import type { ChatItem } from "@/hooks/useProjectChat";

interface ChatMessageProps {
  item: ChatItem;
}

// **bold** と `code` を簡易レンダリング
function renderText(text: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  const regex = /(\*\*(.+?)\*\*|`([^`]+)`)/g;
  let lastIndex = 0;
  let match;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index));
    }
    if (match[2]) {
      parts.push(<strong key={match.index}>{match[2]}</strong>);
    } else if (match[3]) {
      parts.push(
        <code key={match.index} className="px-1 py-0.5 bg-black/10 rounded text-xs font-mono">
          {match[3]}
        </code>
      );
    }
    lastIndex = regex.lastIndex;
  }

  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }

  return parts;
}

export default function ChatMessage({ item }: ChatMessageProps) {
  switch (item.type) {
    case "ai":
      return (
        <div className="flex justify-start">
          <div className="max-w-[80%] px-4 py-3 rounded-2xl rounded-tl-sm bg-gray-200 text-gray-800 text-sm whitespace-pre-wrap">
            {renderText(item.content)}
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
