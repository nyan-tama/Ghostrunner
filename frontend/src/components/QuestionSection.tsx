"use client";

import { useState } from "react";
import type { Question } from "@/types";

interface QuestionSectionProps {
  questions: Question[];
  visible: boolean;
  onAnswer: (answer: string) => void;
}

export default function QuestionSection({
  questions,
  visible,
  onAnswer,
}: QuestionSectionProps) {
  const [customAnswers, setCustomAnswers] = useState<Record<number, string>>({});
  const [selectedOptions, setSelectedOptions] = useState<Record<number, string[]>>({});

  if (!visible || questions.length === 0) return null;

  const handleOptionClick = (questionIdx: number, label: string, isMultiSelect: boolean) => {
    if (isMultiSelect) {
      setSelectedOptions((prev) => {
        const current = prev[questionIdx] || [];
        const isSelected = current.includes(label);
        return {
          ...prev,
          [questionIdx]: isSelected
            ? current.filter((l) => l !== label)
            : [...current, label],
        };
      });
    } else {
      onAnswer(label);
      setCustomAnswers({});
      setSelectedOptions({});
    }
  };

  const handleSubmit = (questionIdx: number, isMultiSelect: boolean) => {
    const customValue = customAnswers[questionIdx]?.trim();
    if (customValue) {
      onAnswer(customValue);
      setCustomAnswers({});
      setSelectedOptions({});
      return;
    }

    if (isMultiSelect) {
      const selected = selectedOptions[questionIdx] || [];
      if (selected.length > 0) {
        onAnswer(selected.join(", "));
        setCustomAnswers({});
        setSelectedOptions({});
      }
    }
  };

  return (
    <div className="px-5 py-4 bg-yellow-50 border-t border-yellow-400">
      {questions.map((q, idx) => (
        <div key={idx} className="mb-4 last:mb-0">
          <div className="font-semibold text-yellow-700 mb-3">
            {q.header || "Question"}
          </div>
          <div className="text-base text-gray-800 mb-3">{q.question}</div>

          <div className="flex flex-col gap-2">
            {q.options.map((opt) => {
              const isSelected = (selectedOptions[idx] || []).includes(opt.label);
              return (
                <button
                  key={opt.label}
                  type="button"
                  onClick={() => handleOptionClick(idx, opt.label, q.multiSelect)}
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
              value={customAnswers[idx] || ""}
              onChange={(e) =>
                setCustomAnswers((prev) => ({ ...prev, [idx]: e.target.value }))
              }
              placeholder="Or type custom answer..."
              className="w-full py-3 px-4 border border-gray-200 rounded-lg text-sm focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            />
          </div>

          <button
            type="button"
            onClick={() => handleSubmit(idx, q.multiSelect)}
            className="w-full mt-3 py-3.5 px-6 bg-green-500 text-white rounded-lg font-semibold text-base cursor-pointer transition-colors hover:bg-green-600 border-none"
          >
            {q.multiSelect ? "Submit Selected" : "Submit"}
          </button>
        </div>
      ))}
    </div>
  );
}
