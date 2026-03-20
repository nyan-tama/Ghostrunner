"use client";

interface PlanApprovalProps {
  visible: boolean;
  onApprove: () => void;
  onReject: () => void;
  isLoading?: boolean;
}

export default function PlanApproval({ visible, onApprove, onReject, isLoading }: PlanApprovalProps) {
  if (!visible) return null;

  return (
    <div className="flex gap-3 px-5 pb-4">
      <button
        onClick={onApprove}
        disabled={isLoading}
        className="flex-1 py-3.5 px-6 bg-green-500 text-white rounded-lg font-semibold text-base cursor-pointer transition-colors hover:bg-green-600 border-none disabled:bg-gray-400 disabled:cursor-not-allowed"
      >
        Approve Plan
      </button>
      <button
        onClick={onReject}
        disabled={isLoading}
        className="flex-1 py-3.5 px-6 bg-red-500 text-white rounded-lg font-semibold text-base cursor-pointer transition-colors hover:bg-red-600 border-none disabled:bg-gray-400 disabled:cursor-not-allowed"
      >
        Reject
      </button>
    </div>
  );
}
