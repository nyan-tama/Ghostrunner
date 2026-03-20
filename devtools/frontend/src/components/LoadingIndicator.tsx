"use client";

interface LoadingIndicatorProps {
  text: string;
  visible: boolean;
  showAbort?: boolean;
  onAbort?: () => void;
}

export default function LoadingIndicator({
  text,
  visible,
  showAbort = false,
  onAbort,
}: LoadingIndicatorProps) {
  if (!visible) return null;

  return (
    <div className="flex items-center justify-between px-5 py-3 bg-blue-50">
      <div className="flex items-center">
        <span className="w-4 h-4 border-2 border-blue-200 border-t-blue-500 rounded-full animate-spin mr-3" />
        <span className="text-sm text-blue-800 font-medium">{text}</span>
      </div>
      {showAbort && onAbort && (
        <button
          onClick={onAbort}
          className="py-1.5 px-3 bg-red-500 text-white rounded-md font-medium text-sm cursor-pointer transition-colors hover:bg-red-600 border-none"
        >
          Abort
        </button>
      )}
    </div>
  );
}
