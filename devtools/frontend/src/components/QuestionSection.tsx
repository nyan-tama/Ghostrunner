"use client";

import { useState } from "react";
import type { Question } from "@/types";

interface QuestionSectionProps {
  questions: Question[];
  visible: boolean;
  currentQuestionIndex: number;
  onAnswer: (answer: string) => void;
}

export default function QuestionSection({
  questions,
  visible,
  currentQuestionIndex,
  onAnswer,
}: QuestionSectionProps) {
  // key propによってcurrentQuestionIndexが変わるとコンポーネントがリマウントされ、状態がリセットされる
  const [customAnswer, setCustomAnswer] = useState("");
  const [selectedOptions, setSelectedOptions] = useState<string[]>([]);

  // 境界チェック: 表示しない条件
  if (!visible || questions.length === 0 || currentQuestionIndex >= questions.length) {
    return null;
  }

  const currentQuestion = questions[currentQuestionIndex];
  const totalQuestions = questions.length;

  const handleOptionClick = (label: string, isMultiSelect: boolean) => {
    if (isMultiSelect) {
      setSelectedOptions((prev) => {
        const isSelected = prev.includes(label);
        return isSelected
          ? prev.filter((l) => l !== label)
          : [...prev, label];
      });
    } else {
      onAnswer(label);
      setCustomAnswer("");
      setSelectedOptions([]);
    }
  };

  const handleSubmit = (isMultiSelect: boolean) => {
    const trimmedCustomAnswer = customAnswer.trim();
    if (trimmedCustomAnswer) {
      onAnswer(trimmedCustomAnswer);
      setCustomAnswer("");
      setSelectedOptions([]);
      return;
    }

    if (isMultiSelect && selectedOptions.length > 0) {
      onAnswer(selectedOptions.join(", "));
      setCustomAnswer("");
      setSelectedOptions([]);
    }
  };

  return (
    <div className="px-5 py-4 bg-yellow-50 border-t border-yellow-400">
      {/* 進捗表示 */}
      <div className="text-sm text-gray-500 mb-2">
        質問 {currentQuestionIndex + 1}/{totalQuestions}
      </div>

      <div className="mb-4">
        <div className="font-semibold text-yellow-700 mb-3">
          {currentQuestion.header || "Question"}
        </div>
        <div className="text-base text-gray-800 mb-3">{currentQuestion.question}</div>

        <div className="flex flex-col gap-2">
          {currentQuestion.options.map((opt) => {
            const isSelected = selectedOptions.includes(opt.label);
            return (
              <button
                key={opt.label}
                type="button"
                onClick={() => handleOptionClick(opt.label, currentQuestion.multiSelect)}
                className={`w-full py-3 px-4 bg-white border-2 rounded-lg text-left cursor-pointer transition-all ${
                  isSelected
                    ? "border-blue-500 bg-blue-50"
                    : "border-gray-200 hover:border-blue-500 hover:bg-gray-50"
                }`}
              >
                <span className="font-semibold text-gray-800 block">
                  {opt.label}
                </span>
                {opt.description && (
                  <span className="text-sm text-gray-500 mt-1 block">
                    {opt.description}
                  </span>
                )}
              </button>
            );
          })}
        </div>

        <div className="mt-3">
          <input
            type="text"
            value={customAnswer}
            onChange={(e) => setCustomAnswer(e.target.value)}
            placeholder="Or type custom answer..."
            className="w-full py-3 px-4 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
          />
        </div>

        <button
          type="button"
          onClick={() => handleSubmit(currentQuestion.multiSelect)}
          className="w-full mt-3 py-3.5 px-6 bg-green-500 text-white rounded-lg font-semibold text-base cursor-pointer transition-colors hover:bg-green-600 border-none"
        >
          {currentQuestion.multiSelect ? "Submit Selected" : "Submit"}
        </button>
      </div>
    </div>
  );
}
