"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

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
        // Tailwind の typography プラグイン未導入のため、子要素セレクタで最低限の体裁を整える。
        // GFM (テーブル・打消し線・タスクリスト) は remark-gfm で対応。
        <div
          className="text-sm text-gray-800 break-words
            [&_h1]:text-base [&_h1]:font-bold [&_h1]:mt-3 [&_h1]:mb-2
            [&_h2]:text-base [&_h2]:font-semibold [&_h2]:mt-3 [&_h2]:mb-1
            [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:mt-2 [&_h3]:mb-1
            [&_p]:my-2 [&_p]:leading-relaxed
            [&_ul]:list-disc [&_ul]:pl-5 [&_ul]:my-2
            [&_ol]:list-decimal [&_ol]:pl-5 [&_ol]:my-2
            [&_li]:my-0.5
            [&_strong]:font-semibold
            [&_em]:italic
            [&_a]:text-blue-600 [&_a]:underline
            [&_code]:bg-gray-100 [&_code]:text-pink-700 [&_code]:px-1 [&_code]:py-0.5 [&_code]:rounded [&_code]:text-xs
            [&_pre]:bg-gray-50 [&_pre]:border [&_pre]:border-gray-200 [&_pre]:rounded [&_pre]:p-2 [&_pre]:my-2 [&_pre]:overflow-x-auto
            [&_pre_code]:bg-transparent [&_pre_code]:text-gray-800 [&_pre_code]:p-0
            [&_blockquote]:border-l-4 [&_blockquote]:border-gray-300 [&_blockquote]:pl-3 [&_blockquote]:text-gray-600 [&_blockquote]:my-2
            [&_table]:border-collapse [&_table]:my-2 [&_table]:text-xs
            [&_th]:border [&_th]:border-gray-300 [&_th]:bg-gray-50 [&_th]:px-2 [&_th]:py-1 [&_th]:font-semibold
            [&_td]:border [&_td]:border-gray-300 [&_td]:px-2 [&_td]:py-1
            [&_hr]:my-3 [&_hr]:border-gray-200"
        >
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {responseText}
          </ReactMarkdown>
        </div>
      )}
    </div>
  );
}
