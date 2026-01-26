"use client";

import type { DisplayEvent, Question } from "@/types";
import LoadingIndicator from "./LoadingIndicator";
import EventList from "./EventList";
import QuestionSection from "./QuestionSection";
import PlanApproval from "./PlanApproval";
import ContinueSession from "./ContinueSession";

interface ProgressContainerProps {
  visible: boolean;
  prompt: string;
  events: DisplayEvent[];
  loadingText: string;
  isLoading: boolean;
  questions: Question[];
  showQuestions: boolean;
  currentQuestionIndex: number;
  showPlanApproval: boolean;
  resultOutput: string;
  resultType: "success" | "error" | null;
  sessionId: string | null;
  totalCost: number;
  onAnswer: (answer: string) => void;
  onApprove: () => void;
  onReject: () => void;
  onAbort: () => void;
  canAbort: boolean;
}

export default function ProgressContainer({
  visible,
  prompt,
  events,
  loadingText,
  isLoading,
  questions,
  showQuestions,
  currentQuestionIndex,
  showPlanApproval,
  resultOutput,
  resultType,
  sessionId,
  totalCost,
  onAnswer,
  onApprove,
  onReject,
  onAbort,
  canAbort,
}: ProgressContainerProps) {
  if (!visible) return null;

  // 完了後でセッションがあり、質問UIやプラン承認UIが出ていない場合に継続UIを表示
  const showContinue =
    resultType === "success" &&
    !!sessionId &&
    !showQuestions &&
    !showPlanApproval &&
    !isLoading;

  return (
    <div className="mt-6 bg-white rounded-xl shadow-md overflow-hidden">
      <div className="px-5 py-4 bg-gray-50 border-b border-gray-200">
        <div className="font-mono text-sm text-gray-800 bg-white p-3 rounded-md border border-gray-200">
          {prompt}
        </div>
      </div>

      <LoadingIndicator
        text={loadingText}
        visible={isLoading}
        showAbort={canAbort}
        onAbort={onAbort}
      />

      <EventList events={events} />

      <QuestionSection
        key={currentQuestionIndex}
        questions={questions}
        visible={showQuestions}
        currentQuestionIndex={currentQuestionIndex}
        onAnswer={onAnswer}
      />

      <PlanApproval
        visible={showPlanApproval}
        onApprove={onApprove}
        onReject={onReject}
        isLoading={isLoading}
      />

      {resultOutput && resultType && (
        <div
          className={`px-5 py-4 border-t border-gray-200 ${
            resultType === "success" ? "bg-green-100" : "bg-red-100"
          }`}
        >
          <div className="bg-white p-3 rounded-md font-mono text-sm whitespace-pre-wrap break-words max-h-96 overflow-y-auto">
            {resultOutput}
          </div>
        </div>
      )}

      <ContinueSession
        visible={showContinue}
        isLoading={isLoading}
        onContinue={onAnswer}
      />

      <div className="px-5 py-3 bg-gray-50 border-t border-gray-200 flex justify-between text-xs text-gray-500">
        <span>
          {sessionId
            ? `Session: ${sessionId.substring(0, 8)}...`
            : "-"}
        </span>
        <span>{totalCost > 0 ? `Cost: $${totalCost.toFixed(4)}` : "-"}</span>
      </div>
    </div>
  );
}
