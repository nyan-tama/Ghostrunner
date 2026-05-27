"use client";

import { useCallback } from "react";
import { useDashboard } from "@/hooks/useDashboard";
import { useChat } from "@/hooks/useChat";
import { useTTS } from "@/hooks/useTTS";
import DashboardHeader from "@/components/dashboard/DashboardHeader";
import DashboardCard from "@/components/dashboard/DashboardCard";
import ChatTranscript from "@/components/dashboard/ChatTranscript";
import ChatInput from "@/components/dashboard/ChatInput";

export default function DashboardPage() {
  const tts = useTTS();

  const handleChatComplete = useCallback(
    (fullText: string) => {
      tts.speak(fullText);
    },
    // tts.speak は useCallback で安定しているので依存に含める
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [tts.speak]
  );

  const dashboard = useDashboard();
  const chat = useChat({
    onComplete: handleChatComplete,
    onSessionSwitch: tts.cancel,
  });

  // エラー集約: 最初に見つかったエラーを表示
  const topError = chat.error ?? dashboard.error ?? tts.error;

  const handleGrasp = useCallback(() => {
    chat.send("状況は？");
    dashboard.refresh();
  }, [chat, dashboard]);

  const handleChatSend = useCallback(
    (text: string) => {
      chat.send(text);
    },
    [chat]
  );

  return (
    <div className="max-w-[900px] mx-auto px-4 py-4">
      {topError && (
        <div className="mb-4 px-3 py-2 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {topError}
        </div>
      )}

      <DashboardHeader
        polling={dashboard.polling}
        onPollingToggle={dashboard.setPolling}
        ttsEnabled={tts.enabled}
        onTTSToggle={tts.setEnabled}
        onGrasp={handleGrasp}
        graspDisabled={chat.status === "busy"}
        sessions={chat.sessions}
        currentSessionId={chat.sessionId}
        onSessionSwitch={chat.switchSession}
        onNewSession={chat.startNewSession}
        onSessionPickerOpen={chat.fetchSessions}
        sessionSwitchDisabled={chat.status === "busy"}
        connectionState={chat.connectionState}
      />

      {dashboard.loading && !dashboard.state && (
        <div className="text-center text-gray-500 py-8">読み込み中...</div>
      )}

      {dashboard.state && (
        <div className="space-y-3 mb-4">
          {dashboard.state.projects.map((project) => (
            <DashboardCard
              key={project.path}
              project={project}
              onAnswered={dashboard.refresh}
            />
          ))}
          {dashboard.state.projects.length === 0 && (
            <div className="text-center text-gray-400 py-8">
              プロジェクトが登録されていません
            </div>
          )}
        </div>
      )}

      <div className="space-y-3">
        <ChatTranscript
          responseText={chat.responseText}
          status={chat.status}
          error={chat.error}
        />

        <ChatInput
          onSend={handleChatSend}
          disabled={chat.status === "busy"}
        />
      </div>

      {dashboard.state && (
        <div className="mt-4 text-xs text-gray-400 text-center">
          最終更新: {dashboard.state.generatedAt}
        </div>
      )}
    </div>
  );
}
