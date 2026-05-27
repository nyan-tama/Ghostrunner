"use client";

import { useEffect, useRef, useState } from "react";

interface SidCopyButtonProps {
  sessionId: string | null;
}

// secure context でない場合のフォールバックコピー
function fallbackCopy(text: string): boolean {
  if (typeof document === "undefined") return false;
  try {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.setAttribute("readonly", "");
    ta.style.position = "fixed";
    ta.style.top = "0";
    ta.style.left = "0";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand("copy");
    document.body.removeChild(ta);
    return ok;
  } catch {
    return false;
  }
}

export default function SidCopyButton({ sessionId }: SidCopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const [showManual, setShowManual] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const handleClick = async () => {
    if (!sessionId) return;

    let ok = false;

    if (
      typeof navigator !== "undefined" &&
      navigator.clipboard &&
      typeof navigator.clipboard.writeText === "function"
    ) {
      try {
        await navigator.clipboard.writeText(sessionId);
        ok = true;
      } catch {
        ok = false;
      }
    }

    if (!ok) {
      ok = fallbackCopy(sessionId);
    }

    if (ok) {
      setCopied(true);
      setShowManual(false);
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => {
        setCopied(false);
      }, 2000);
    } else {
      // 両方失敗: 手動コピー用の input を露出
      setShowManual(true);
    }
  };

  const disabled = sessionId === null;
  const label = copied ? "Copied" : "SID";

  return (
    <div className="flex items-center gap-2">
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled}
        title={sessionId ?? "セッション未確立"}
        className="min-h-[40px] px-3 py-1.5 text-sm rounded-lg border border-gray-300 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed text-gray-700"
      >
        {label}
      </button>
      {showManual && sessionId && (
        <input
          type="text"
          readOnly
          value={sessionId}
          onFocus={(e) => e.target.select()}
          className="text-xs px-2 py-1 border border-gray-300 rounded w-40"
          aria-label="セッションID（手動コピー）"
        />
      )}
    </div>
  );
}
