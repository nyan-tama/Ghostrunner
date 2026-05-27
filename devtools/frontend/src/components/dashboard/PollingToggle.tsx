"use client";

interface PollingToggleProps {
  polling: boolean;
  onToggle: (v: boolean) => void;
}

export default function PollingToggle({ polling, onToggle }: PollingToggleProps) {
  return (
    <button
      type="button"
      onClick={() => onToggle(!polling)}
      className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${
        polling
          ? "bg-green-600 text-white border-green-600"
          : "bg-white text-gray-600 border-gray-300 hover:border-gray-400"
      }`}
    >
      {polling ? "自動更新" : "手動更新（聞いたら返す）"}
    </button>
  );
}
