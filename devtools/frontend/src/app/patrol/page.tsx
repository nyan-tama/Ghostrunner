"use client";

import { useState } from "react";
import Link from "next/link";
import { usePatrol } from "@/hooks/usePatrol";
import PatrolHeader from "@/components/patrol/PatrolHeader";
import ProjectRegister from "@/components/patrol/ProjectRegister";
import ProjectCard from "@/components/patrol/ProjectCard";

export default function PatrolPage() {
  const {
    projects,
    projectStates,
    isRunning,
    isPolling,
    error,
    connectionStatus,
    registerProject,
    removeProject,
    handleStartPatrol,
    handleStopPatrol,
    handleSendAnswer,
    handleTogglePolling,
  } = usePatrol();

  const [isAnswerSubmitting, setIsAnswerSubmitting] = useState(false);
  const [isPatrolLoading, setIsPatrolLoading] = useState(false);

  const handleAnswer = async (projectPath: string, answer: string) => {
    setIsAnswerSubmitting(true);
    try {
      await handleSendAnswer(projectPath, answer);
    } finally {
      setIsAnswerSubmitting(false);
    }
  };

  const handleStart = async () => {
    setIsPatrolLoading(true);
    try {
      await handleStartPatrol();
    } finally {
      setIsPatrolLoading(false);
    }
  };

  const handleStop = async () => {
    setIsPatrolLoading(true);
    try {
      await handleStopPatrol();
    } finally {
      setIsPatrolLoading(false);
    }
  };

  return (
    <div className="max-w-[900px] mx-auto px-5 py-5 bg-gray-100 min-h-screen">
      {/* ヘッダー */}
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-gray-800 text-2xl font-bold">巡回ダッシュボード</h1>
        <Link
          href="/"
          className="px-3 py-1 text-xs bg-gray-200 text-gray-600 rounded hover:bg-gray-300 transition-colors"
        >
          Back
        </Link>
      </div>

      {/* エラー表示 */}
      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {error}
        </div>
      )}

      {/* 巡回制御 */}
      <div className="mb-4">
        <PatrolHeader
          isRunning={isRunning}
          isPolling={isPolling}
          isLoading={isPatrolLoading}
          connectionStatus={connectionStatus}
          onStart={handleStart}
          onStop={handleStop}
          onTogglePolling={handleTogglePolling}
        />
      </div>

      {/* プロジェクト登録 */}
      <div className="mb-4">
        <ProjectRegister
          registeredPaths={projects.map((p) => p.path)}
          onRegister={registerProject}
        />
      </div>

      {/* プロジェクト一覧 */}
      {projects.length === 0 ? (
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-8 text-center">
          <p className="text-sm text-gray-400">
            巡回対象のプロジェクトがありません。上の「追加」ボタンからプロジェクトを登録してください。
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {projects.map((project) => (
            <ProjectCard
              key={project.path}
              project={project}
              state={projectStates[project.path]}
              onRemove={removeProject}
              onAnswer={handleAnswer}
              isAnswerSubmitting={isAnswerSubmitting}
            />
          ))}
        </div>
      )}
    </div>
  );
}
