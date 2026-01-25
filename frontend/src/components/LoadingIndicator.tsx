"use client";

interface LoadingIndicatorProps {
  text: string;
  visible: boolean;
}

export default function LoadingIndicator({ text, visible }: LoadingIndicatorProps) {
  if (!visible) return null;

  return (
    <div className="flex items-center px-5 py-3 bg-blue-50">
      <span className="w-4 h-4 border-2 border-blue-200 border-t-blue-500 rounded-full animate-spin mr-3" />
      <span className="text-sm text-blue-800 font-medium">{text}</span>
    </div>
  );
}
