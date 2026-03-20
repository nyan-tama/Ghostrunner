"use client";

import { useState, useCallback } from "react";
import type { CreatedProject } from "@/types";
import { openInVSCode } from "@/lib/createApi";

interface CreateCompleteProps {
  project: CreatedProject;
  onCreateAnother: () => void;
}

export default function CreateComplete({ project, onCreateAnother }: CreateCompleteProps) {
  const [isOpening, setIsOpening] = useState(false);
  const [openError, setOpenError] = useState("");

  const handleOpenVSCode = useCallback(async () => {
    setIsOpening(true);
    setOpenError("");

    try {
      const result = await openInVSCode(project.path);
      if (!result.success) {
        setOpenError(result.message || "Failed to open VS Code");
      }
    } catch (err) {
      setOpenError(err instanceof Error ? err.message : "Failed to open VS Code");
    } finally {
      setIsOpening(false);
    }
  }, [project.path]);

  return (
    <div className="space-y-6">
      {/* 成功メッセージ */}
      <div className="text-center">
        <svg className="w-12 h-12 text-green-500 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <h2 className="text-lg font-bold text-gray-800">Project Created</h2>
        <p className="text-sm text-gray-500 mt-1">
          {project.name} is ready for development
        </p>
      </div>

      {/* プロジェクトパス */}
      <div className="bg-gray-50 rounded-lg p-4">
        <label className="block text-xs text-gray-500 mb-1">Project Path</label>
        <code className="text-sm text-gray-800 font-mono break-all">{project.path}</code>
      </div>

      {/* アクションボタン */}
      <div className="flex gap-3">
        <button
          onClick={handleOpenVSCode}
          disabled={isOpening}
          className="flex-1 py-2.5 bg-blue-600 text-white rounded-lg font-medium text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {isOpening ? "Opening..." : "Open in VS Code"}
        </button>
        <button
          onClick={onCreateAnother}
          className="flex-1 py-2.5 bg-gray-200 text-gray-700 rounded-lg font-medium text-sm hover:bg-gray-300 transition-colors"
        >
          Create Another
        </button>
      </div>

      {openError && (
        <p className="text-sm text-red-600 text-center">{openError}</p>
      )}
    </div>
  );
}
