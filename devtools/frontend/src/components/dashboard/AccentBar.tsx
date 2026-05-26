"use client";

import type { Attention } from "@/types/dashboard";

interface AccentBarProps {
  attention: Attention;
  hasUnanswered: boolean;
}

function getBarColor(attention: Attention, hasUnanswered: boolean): string {
  switch (attention) {
    case "required":
      return "bg-red-500";
    case "progress":
      return hasUnanswered ? "bg-yellow-400" : "bg-blue-500";
    case "watching":
      return "bg-gray-300";
  }
}

export default function AccentBar({ attention, hasUnanswered }: AccentBarProps) {
  return (
    <div
      className={`absolute left-0 top-0 bottom-0 w-1 rounded-l-lg ${getBarColor(attention, hasUnanswered)}`}
    />
  );
}
