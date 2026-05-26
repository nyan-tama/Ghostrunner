"use client";

interface ChatTranscriptProps {
  responseText: string;
  status: "idle" | "busy" | "error";
  error: string | null;
}

export default function ChatTranscript({
  responseText,
  status,
  error,
}: ChatTranscriptProps) {
  if (!responseText && status === "idle" && !error) {
    return null;
  }

  return (
    <div className="border border-gray-200 rounded-lg p-3 bg-white">
      <div className="flex items-center gap-2 mb-2">
        <span className="text-xs font-medium text-gray-500">応答</span>
        {status === "busy" && (
          <span className="text-xs text-blue-600">応答中...</span>
        )}
        {status === "error" && (
          <span className="text-xs text-red-600">エラー</span>
        )}
      </div>

      {error && (
        <div className="text-sm text-red-600 mb-2">{error}</div>
      )}

      {responseText && (
        <div className="text-sm text-gray-800 whitespace-pre-wrap break-words">
          {responseText}
        </div>
      )}
    </div>
  );
}
