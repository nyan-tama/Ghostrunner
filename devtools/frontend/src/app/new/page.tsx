"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import type { DataService } from "@/types";
import { useProjectCreate } from "@/hooks/useProjectCreate";
import ProjectForm from "@/components/create/ProjectForm";
import CreateProgress from "@/components/create/CreateProgress";
import CreateComplete from "@/components/create/CreateComplete";

export default function NewProjectPage() {
  const {
    phase,
    steps,
    progress,
    errorMessage,
    createdProject,
    startCreate,
    resetToForm,
  } = useProjectCreate();

  // エラー時にフォームに戻す際、入力値を保持するための状態
  const [lastInput, setLastInput] = useState<{
    name: string;
    description: string;
    services: DataService[];
  }>({ name: "", description: "", services: [] });

  const handleFormSubmit = useCallback(
    (name: string, description: string, services: DataService[]) => {
      setLastInput({ name, description, services });
      startCreate(name, description, services);
    },
    [startCreate]
  );

  const handleCreateAnother = useCallback(() => {
    setLastInput({ name: "", description: "", services: [] });
    resetToForm();
  }, [resetToForm]);

  return (
    <div className="max-w-[600px] mx-auto px-5 py-5 bg-gray-100 min-h-screen">
      {/* ヘッダー */}
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-gray-800 text-2xl font-bold">New Project</h1>
        <Link
          href="/"
          className="px-3 py-1 text-xs bg-gray-200 text-gray-600 rounded hover:bg-gray-300 transition-colors"
        >
          Back
        </Link>
      </div>

      {/* メインコンテンツ */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
        {/* Form フェーズ */}
        {(phase === "form" || phase === "error") && (
          <div className="space-y-4">
            {phase === "error" && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-700">
                {errorMessage}
              </div>
            )}
            <ProjectForm
              onSubmit={handleFormSubmit}
              initialName={lastInput.name}
              initialDescription={lastInput.description}
              initialServices={lastInput.services}
            />
          </div>
        )}

        {/* Creating フェーズ */}
        {phase === "creating" && (
          <div className="space-y-4">
            <h2 className="text-sm font-medium text-gray-700">
              Creating {lastInput.name}...
            </h2>
            <CreateProgress steps={steps} progress={progress} />
          </div>
        )}

        {/* Complete フェーズ */}
        {phase === "complete" && createdProject && (
          <CreateComplete
            project={createdProject}
            onCreateAnother={handleCreateAnother}
          />
        )}
      </div>
    </div>
  );
}
