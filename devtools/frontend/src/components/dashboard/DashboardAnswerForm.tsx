"use client";

import { useState } from "react";
import AnswerForm from "@/components/patrol/AnswerForm";
import { submitAnswer } from "@/lib/dashboardApi";
import type { UnansweredItem } from "@/types/dashboard";
import type { Question } from "@/types";

interface DashboardAnswerFormProps {
  projectPath: string;
  item: UnansweredItem;
  onAnswered: () => void;
}

// UnansweredItem から AnswerForm 用の Question を組み立てる
function buildQuestion(item: UnansweredItem): Question {
  return {
    question: item.questionText,
    header: item.heading ?? "",
    options: [],
    multiSelect: false,
  };
}

export default function DashboardAnswerForm({
  projectPath,
  item,
  onAnswered,
}: DashboardAnswerFormProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (_projectPath: string, answer: string) => {
    setIsSubmitting(true);
    try {
      const result = await submitAnswer({
        projectPath,
        planPath: item.planPath,
        lineStart: item.lineStart,
        answer,
      });
      if (result.success) {
        onAnswered();
      } else {
        alert(result.error || "回答の送信に失敗しました");
      }
    } catch {
      alert("回答の送信に失敗しました");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <AnswerForm
      projectPath={projectPath}
      question={buildQuestion(item)}
      isSubmitting={isSubmitting}
      onSubmit={handleSubmit}
    />
  );
}
