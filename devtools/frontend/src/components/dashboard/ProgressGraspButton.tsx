"use client";

interface ProgressGraspButtonProps {
  onGrasp: () => void;
  disabled: boolean;
}

export default function ProgressGraspButton({
  onGrasp,
  disabled,
}: ProgressGraspButtonProps) {
  return (
    <button
      type="button"
      onClick={onGrasp}
      disabled={disabled}
      className="px-4 py-1.5 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
    >
      {disabled ? "確認中..." : "状況は？"}
    </button>
  );
}
