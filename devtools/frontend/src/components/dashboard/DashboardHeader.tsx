"use client";

import PollingToggle from "./PollingToggle";
import TTSToggle from "./TTSToggle";
import ProgressGraspButton from "./ProgressGraspButton";

interface DashboardHeaderProps {
  polling: boolean;
  onPollingToggle: (v: boolean) => void;
  ttsEnabled: boolean;
  onTTSToggle: (v: boolean) => void;
  onGrasp: () => void;
  graspDisabled: boolean;
}

export default function DashboardHeader({
  polling,
  onPollingToggle,
  ttsEnabled,
  onTTSToggle,
  onGrasp,
  graspDisabled,
}: DashboardHeaderProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
      <h1 className="text-lg font-bold text-gray-900">統括ダッシュボード</h1>
      <div className="flex items-center gap-2">
        <PollingToggle polling={polling} onToggle={onPollingToggle} />
        <TTSToggle enabled={ttsEnabled} onToggle={onTTSToggle} />
        <ProgressGraspButton onGrasp={onGrasp} disabled={graspDisabled} />
      </div>
    </div>
  );
}
