"use client";

import { useEffect, useState } from "react";

interface TTSToggleProps {
  enabled: boolean;
  onToggle: (v: boolean) => void;
}

export default function TTSToggle({ enabled, onToggle }: TTSToggleProps) {
  const [isSupported, setIsSupported] = useState(false);

  useEffect(() => {
    setIsSupported("speechSynthesis" in window);
  }, []);

  return (
    <button
      type="button"
      onClick={() => onToggle(!enabled)}
      disabled={!isSupported}
      className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${
        enabled
          ? "bg-blue-600 text-white border-blue-600"
          : "bg-white text-gray-600 border-gray-300 hover:border-gray-400"
      } disabled:opacity-40 disabled:cursor-not-allowed`}
      title={isSupported ? "音声読み上げ" : "このブラウザは音声合成に対応していません"}
    >
      TTS {enabled ? "ON" : "OFF"}
    </button>
  );
}
