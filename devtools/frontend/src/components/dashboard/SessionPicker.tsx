"use client";

import { useEffect, useRef, useState } from "react";
import type { ChatSession } from "@/types/chat";

interface SessionPickerProps {
  sessions: ChatSession[];
  currentSessionId: string | null;
  onSwitch: (sid: string) => void;
  onNewSession: () => void;
  onOpen?: () => void;
  disabled?: boolean;
}

// 相対時刻フォーマット（簡易版）
function formatRelative(iso?: string): string {
  if (!iso) return "";
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "";
  const diffMs = Date.now() - t;
  if (diffMs < 0) return "";
  const sec = Math.floor(diffMs / 1000);
  if (sec < 60) return `${sec}秒前`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}分前`;
  const hour = Math.floor(min / 60);
  if (hour < 24) return `${hour}時間前`;
  const day = Math.floor(hour / 24);
  return `${day}日前`;
}

function getDisplayLabel(session: ChatSession): string {
  if (session.title && session.title.length > 0) return session.title;
  return session.id.slice(0, 8);
}

export default function SessionPicker({
  sessions,
  currentSessionId,
  onSwitch,
  onNewSession,
  onOpen,
  disabled = false,
}: SessionPickerProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  // 外部クリックで閉じる
  useEffect(() => {
    if (!open) return;
    function handleClickOutside(e: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [open]);

  const handleToggle = () => {
    if (disabled) return;
    const next = !open;
    setOpen(next);
    if (next) {
      try {
        onOpen?.();
      } catch {
        // onOpen 例外は黙って無視（fetchSessions 失敗等）
      }
    }
  };

  const handleSelect = (sid: string) => {
    setOpen(false);
    if (sid !== currentSessionId) {
      onSwitch(sid);
    }
  };

  const handleNew = () => {
    setOpen(false);
    onNewSession();
  };

  const currentLabel = (() => {
    if (!currentSessionId) return "新規セッション";
    const found = sessions.find((s) => s.id === currentSessionId);
    if (found) return getDisplayLabel(found);
    return currentSessionId.slice(0, 8);
  })();

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={handleToggle}
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="min-h-[40px] px-3 py-1.5 text-sm rounded-lg border border-gray-300 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
      >
        <span className="font-medium text-gray-700 truncate max-w-[140px]">
          {currentLabel}
        </span>
        <span className="text-gray-400 text-xs">▼</span>
      </button>

      {open && (
        <div
          role="listbox"
          className="absolute left-0 top-full mt-1 z-20 w-72 max-h-80 overflow-y-auto bg-white border border-gray-200 rounded-lg shadow-lg"
        >
          <button
            type="button"
            onClick={handleNew}
            className="w-full text-left px-3 py-2 min-h-[44px] border-b border-gray-100 hover:bg-blue-50 text-sm text-blue-700 font-medium"
          >
            + 新規 session
          </button>

          {sessions.length === 0 ? (
            <div className="px-3 py-3 text-sm text-gray-400">
              セッションなし
            </div>
          ) : (
            <ul>
              {sessions.map((s) => {
                const isCurrent = s.id === currentSessionId;
                return (
                  <li key={s.id}>
                    <button
                      type="button"
                      role="option"
                      aria-selected={isCurrent}
                      onClick={() => handleSelect(s.id)}
                      className={`w-full text-left px-3 py-2 min-h-[44px] hover:bg-gray-50 text-sm border-b border-gray-50 last:border-b-0 ${
                        isCurrent ? "bg-blue-50 text-blue-700" : "text-gray-700"
                      }`}
                    >
                      <div className="font-medium truncate">
                        {getDisplayLabel(s)}
                      </div>
                      <div className="text-xs text-gray-400 flex gap-2">
                        <span>{formatRelative(s.timestamp)}</span>
                        {s.status && <span>· {s.status}</span>}
                      </div>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
