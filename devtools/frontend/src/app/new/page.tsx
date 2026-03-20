"use client";

import { useRef, useEffect, useState } from "react";
import Link from "next/link";
import { useProjectChat } from "@/hooks/useProjectChat";
import { openInVSCode } from "@/lib/createApi";
import ChatMessage from "@/components/chat/ChatMessage";
import ChatInput from "@/components/chat/ChatInput";

function CompleteView({ createdPath, onReset }: { createdPath: string | null; onReset: () => void }) {
  const [isOpening, setIsOpening] = useState(false);
  const [openError, setOpenError] = useState("");

  const handleOpenVSCode = async () => {
    if (!createdPath) return;
    setIsOpening(true);
    setOpenError("");
    try {
      await openInVSCode(createdPath);
    } catch (err) {
      setOpenError(err instanceof Error ? err.message : "VS Codeの起動に失敗しました");
    } finally {
      setIsOpening(false);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-8">
      <div className="text-center mb-6">
        <h2 className="text-lg font-semibold text-green-700 mb-2">
          プロジェクトの作成が完了しました
        </h2>
      </div>

      {createdPath && (
        <div className="mb-6 p-4 bg-gray-50 rounded-lg">
          <div className="text-xs text-gray-500 mb-1">生成先</div>
          <div className="text-sm font-mono text-gray-800">{createdPath}</div>
        </div>
      )}

      <div className="flex gap-3 justify-center">
        <button
          type="button"
          onClick={handleOpenVSCode}
          disabled={!createdPath || isOpening}
          className="px-5 py-2.5 bg-blue-600 text-white rounded-xl text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {isOpening ? "起動中..." : "VS Codeで開く"}
        </button>
        <button
          type="button"
          onClick={onReset}
          className="px-5 py-2.5 bg-gray-200 text-gray-700 rounded-xl text-sm font-medium hover:bg-gray-300 transition-colors"
        >
          もう1つ作る
        </button>
      </div>

      {openError && (
        <p className="mt-3 text-center text-xs text-red-600">{openError}</p>
      )}
    </div>
  );
}

export default function NewProjectPage() {
  const {
    messages,
    isStreaming,
    phase,
    currentQuestion,
    startChat,
    sendAnswer,
    reset,
    createdPath,
  } = useProjectChat();

  const scrollRef = useRef<HTMLDivElement>(null);
  const [selectedOptions, setSelectedOptions] = useState<string[]>([]);
  const prevQuestionRef = useRef(currentQuestion);

  // 質問が変わったら選択状態をリセット（useEffect を使わずレンダー中に処理）
  if (prevQuestionRef.current !== currentQuestion) {
    prevQuestionRef.current = currentQuestion;
    setSelectedOptions([]);
  }

  // メッセージ追加時に自動スクロール
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isStreaming]);

  const handleOptionClick = (label: string) => {
    if (!currentQuestion) return;

    if (currentQuestion.multiSelect) {
      setSelectedOptions((prev) => {
        const isSelected = prev.includes(label);
        return isSelected
          ? prev.filter((l) => l !== label)
          : [...prev, label];
      });
    } else {
      sendAnswer(label);
    }
  };

  const handleMultiSelectSubmit = () => {
    if (selectedOptions.length > 0) {
      sendAnswer(selectedOptions.join(", "));
      setSelectedOptions([]);
    }
  };

  return (
    <div className="max-w-[700px] mx-auto px-5 py-5 bg-gray-100 min-h-screen flex flex-col">
      {/* ヘッダー */}
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-gray-800 text-2xl font-bold">New Project</h1>
        <Link
          href="/"
          className="px-3 py-1 text-xs bg-gray-200 text-gray-600 rounded hover:bg-gray-300 transition-colors"
        >
          Back
        </Link>
      </div>

      {/* idle: 説明 + 開始ボタン */}
      {phase === "idle" && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-8 text-center">
          <h2 className="text-lg font-semibold text-gray-800 mb-3">
            対話形式でプロジェクトを作成
          </h2>
          <p className="text-sm text-gray-500 mb-6 leading-relaxed">
            AIと対話しながら、プロジェクト名や構成を決めていきます。
            <br />
            質問に答えるだけで、新しいプロジェクトが自動生成されます。
          </p>
          <button
            type="button"
            onClick={startChat}
            className="px-6 py-3 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700 transition-colors"
          >
            対話を始める
          </button>
        </div>
      )}

      {/* chatting: チャットエリア + 入力 */}
      {phase === "chatting" && (
        <div className="flex-1 flex flex-col bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden min-h-0">
          {/* チャットメッセージ領域 */}
          <div
            ref={scrollRef}
            className="flex-1 overflow-y-auto p-4 space-y-3"
            style={{ maxHeight: "calc(100vh - 280px)" }}
          >
            {messages.map((msg) => (
              <ChatMessage key={msg.id} item={msg} />
            ))}
            {isStreaming && (
              <div className="flex justify-start">
                <div className="text-xs text-gray-400 py-1 px-2 animate-pulse">
                  処理中...
                </div>
              </div>
            )}
          </div>

          {/* 選択肢ボタン */}
          {currentQuestion &&
            currentQuestion.options.length > 0 &&
            !isStreaming && (
              <div className="px-4 py-3 border-t border-gray-100 space-y-2">
                <div className="flex flex-col gap-2">
                  {currentQuestion.options.map((opt) => {
                    const isSelected = selectedOptions.includes(opt.label);
                    return (
                      <button
                        key={opt.label}
                        type="button"
                        onClick={() => handleOptionClick(opt.label)}
                        className={`w-full py-2.5 px-4 border-2 rounded-lg text-left text-sm cursor-pointer transition-all ${
                          isSelected
                            ? "border-blue-500 bg-blue-50"
                            : "border-gray-200 bg-white hover:border-blue-500 hover:bg-gray-50"
                        }`}
                      >
                        <span className="font-medium text-gray-800 block">
                          {opt.label}
                        </span>
                        {opt.description && (
                          <span className="text-xs text-gray-500 mt-0.5 block">
                            {opt.description}
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
                {currentQuestion.multiSelect && selectedOptions.length > 0 && (
                  <button
                    type="button"
                    onClick={handleMultiSelectSubmit}
                    className="w-full py-2.5 px-4 bg-green-500 text-white rounded-lg font-medium text-sm hover:bg-green-600 transition-colors"
                  >
                    選択を送信
                  </button>
                )}
              </div>
            )}

          {/* 入力エリア */}
          <div className="p-4 border-t border-gray-200">
            <ChatInput onSend={sendAnswer} disabled={isStreaming} />
          </div>
        </div>
      )}

      {/* complete: 完了画面 */}
      {phase === "complete" && (
        <CompleteView createdPath={createdPath} onReset={reset} />
      )}

      {/* error: エラー画面 */}
      {phase === "error" && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-8">
          <div className="text-center mb-6">
            <h2 className="text-lg font-semibold text-red-700 mb-2">
              エラーが発生しました
            </h2>
          </div>

          {/* エラーメッセージを表示 */}
          {messages.length > 0 && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700 whitespace-pre-wrap">
              {messages
                .filter((m) => m.type === "error")
                .slice(-1)
                .map((m) => m.content)
                .join("")}
            </div>
          )}

          <div className="flex gap-3 justify-center">
            <button
              type="button"
              onClick={reset}
              className="px-5 py-2.5 bg-blue-600 text-white rounded-xl text-sm font-medium hover:bg-blue-700 transition-colors"
            >
              やり直す
            </button>
            <Link
              href="/"
              className="px-5 py-2.5 bg-gray-200 text-gray-700 rounded-xl text-sm font-medium hover:bg-gray-300 transition-colors"
            >
              ダッシュボードに戻る
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
