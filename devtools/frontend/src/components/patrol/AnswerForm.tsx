"use client";

import { useState } from "react";
import type { Question } from "@/types";

interface AnswerFormProps {
  projectPath: string;
  question: Question;
  isSubmitting: boolean;
  onSubmit: (projectPath: string, answer: string) => void;
}

export default function AnswerForm({ projectPath, question, isSubmitting, onSubmit }: AnswerFormProps) {
  const [customAnswer, setCustomAnswer] = useState("");
  const [selectedOptions, setSelectedOptions] = useState<string[]>([]);

  const handleOptionClick = (label: string) => {
    if (question.multiSelect) {
      setSelectedOptions((prev) => {
        const isSelected = prev.includes(label);
        return isSelected ? prev.filter((l) => l !== label) : [...prev, label];
      });
    } else {
      onSubmit(projectPath, label);
    }
  };

  const handleSubmit = () => {
    const trimmed = customAnswer.trim();
    if (trimmed) {
      onSubmit(projectPath, trimmed);
      setCustomAnswer("");
      setSelectedOptions([]);
      return;
    }
    if (question.multiSelect && selectedOptions.length > 0) {
      onSubmit(projectPath, selectedOptions.join(", "));
      setCustomAnswer("");
      setSelectedOptions([]);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div className="mt-3 p-3 bg-yellow-50 border border-yellow-300 rounded-lg">
      <div className="text-xs font-semibold text-yellow-700 mb-1">
        {question.header || "Question"}
      </div>
      <div className="text-sm text-gray-800 mb-2">{question.question}</div>

      {question.options.length > 0 && (
        <div className="flex flex-col gap-1.5 mb-2">
          {question.options.map((opt) => {
            const isSelected = selectedOptions.includes(opt.label);
            return (
              <button
                key={opt.label}
                type="button"
                disabled={isSubmitting}
                onClick={() => handleOptionClick(opt.label)}
                className={`w-full py-2 px-3 border-2 rounded-lg text-left text-sm cursor-pointer transition-all disabled:opacity-50 disabled:cursor-not-allowed ${
                  isSelected
                    ? "border-blue-500 bg-blue-50"
                    : "border-gray-200 bg-white hover:border-blue-500 hover:bg-gray-50"
                }`}
              >
                <span className="font-medium text-gray-800 block">{opt.label}</span>
                {opt.description && (
                  <span className="text-xs text-gray-500 mt-0.5 block">{opt.description}</span>
                )}
              </button>
            );
          })}
        </div>
      )}

      <input
        type="text"
        value={customAnswer}
        onChange={(e) => setCustomAnswer(e.target.value)}
        onKeyDown={handleKeyDown}
        disabled={isSubmitting}
        placeholder="自由テキストで回答..."
        className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100 disabled:opacity-50 disabled:cursor-not-allowed"
      />

      <button
        type="button"
        onClick={handleSubmit}
        disabled={isSubmitting}
        className="w-full mt-2 py-2 px-4 bg-green-500 text-white rounded-lg font-medium text-sm cursor-pointer transition-colors hover:bg-green-600 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isSubmitting ? "送信中..." : question.multiSelect ? "選択を送信" : "送信"}
      </button>
    </div>
  );
}
