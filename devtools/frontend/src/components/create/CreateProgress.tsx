"use client";

import type { CreateStep } from "@/types";

interface CreateProgressProps {
  steps: CreateStep[];
  progress: number;
}

function StepIcon({ status }: { status: CreateStep["status"] }) {
  switch (status) {
    case "done":
      return (
        <svg className="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
    case "active":
      return (
        <span className="w-5 h-5 border-2 border-blue-400 border-t-blue-600 rounded-full animate-spin inline-block" />
      );
    case "error":
      return (
        <svg className="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
    case "pending":
    default:
      return (
        <span className="w-5 h-5 rounded-full border-2 border-gray-300 inline-block" />
      );
  }
}

function stepTextColor(status: CreateStep["status"]): string {
  switch (status) {
    case "done":
      return "text-green-700";
    case "active":
      return "text-blue-700 font-medium";
    case "error":
      return "text-red-700";
    default:
      return "text-gray-400";
  }
}

export default function CreateProgress({ steps, progress }: CreateProgressProps) {
  return (
    <div className="space-y-4">
      {/* プログレスバー */}
      <div>
        <div className="flex justify-between text-xs text-gray-500 mb-1">
          <span>Progress</span>
          <span>{progress}%</span>
        </div>
        <div className="w-full h-2 bg-gray-200 rounded-full overflow-hidden">
          <div
            className="h-full bg-blue-500 rounded-full transition-all duration-300"
            style={{ width: `${progress}%` }}
          />
        </div>
      </div>

      {/* ステップリスト */}
      <ul className="space-y-2">
        {steps.map((step) => (
          <li key={step.id} className="flex items-center gap-3">
            <StepIcon status={step.status} />
            <span className={`text-sm ${stepTextColor(step.status)}`}>
              {step.label}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
