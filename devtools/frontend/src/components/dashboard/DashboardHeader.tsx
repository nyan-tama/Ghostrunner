"use client";

import PollingToggle from "./PollingToggle";
import TTSToggle from "./TTSToggle";
import ProgressGraspButton from "./ProgressGraspButton";
import SessionPicker from "./SessionPicker";
import SidCopyButton from "./SidCopyButton";
import ConnectionIndicator from "./ConnectionIndicator";
import type { ChatSession } from "@/types/chat";

interface DashboardHeaderProps {
  polling: boolean;
  onPollingToggle: (v: boolean) => void;
  ttsEnabled: boolean;
  onTTSToggle: (v: boolean) => void;
  onGrasp: () => void;
  graspDisabled: boolean;
  sessions: ChatSession[];
  currentSessionId: string | null;
  onSessionSwitch: (sid: string) => void;
  onNewSession: () => void;
  onSessionPickerOpen?: () => void;
  sessionSwitchDisabled?: boolean;
  connectionState: "live" | "reconnecting" | "offline";
}

export default function DashboardHeader({
  polling,
  onPollingToggle,
  ttsEnabled,
  onTTSToggle,
  onGrasp,
  graspDisabled,
  sessions,
  currentSessionId,
  onSessionSwitch,
  onNewSession,
  onSessionPickerOpen,
  sessionSwitchDisabled,
  connectionState,
}: DashboardHeaderProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
      <div className="flex items-center gap-2 flex-wrap">
        <h1 className="text-lg font-bold text-gray-900">統括</h1>
        <SessionPicker
          sessions={sessions}
          currentSessionId={currentSessionId}
          onSwitch={onSessionSwitch}
          onNewSession={onNewSession}
          onOpen={onSessionPickerOpen}
          disabled={sessionSwitchDisabled}
        />
        <SidCopyButton sessionId={currentSessionId} />
      </div>
      <div className="flex items-center gap-2 flex-wrap">
        <PollingToggle polling={polling} onToggle={onPollingToggle} />
        <TTSToggle enabled={ttsEnabled} onToggle={onTTSToggle} />
        <ProgressGraspButton onGrasp={onGrasp} disabled={graspDisabled} />
        <ConnectionIndicator state={connectionState} />
      </div>
    </div>
  );
}
