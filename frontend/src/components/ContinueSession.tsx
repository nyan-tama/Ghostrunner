"use client";

import { useState } from "react";

interface ContinueSessionProps {
  visible: boolean;
  isLoading: boolean;
  onContinue: (message: string) => void;
}

export default function ContinueSession({
  visible,
  isLoading,
  onContinue,
}: ContinueSessionProps) {
  const [message, setMessage] = useState("");

  if (!visible) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (message.trim() && !isLoading) {
      onContinue(message.trim());
      setMessage("");
    }
  };

  return (
    <div className="px-5 py-4 border-t border-gray-200 bg-blue-50">
      <form onSubmit={handleSubmit} className="flex gap-2">
        <input
          type="text"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder="回答を入力してセッションを続ける..."
          disabled={isLoading}
          className="flex-1 p-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100 disabled:bg-gray-100"
        />
        <button
          type="submit"
          disabled={isLoading || !message.trim()}
          className="px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium hover:bg-blue-600 disabled:bg-gray-400 disabled:cursor-not-allowed"
        >
          {isLoading ? "送信中..." : "送信"}
        </button>
      </form>
    </div>
  );
}
