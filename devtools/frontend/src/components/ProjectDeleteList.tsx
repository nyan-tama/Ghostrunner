"use client";

import type { ProjectInfo } from "@/types";

interface ProjectDeleteListProps {
  projects: ProjectInfo[];
  onDelete: (path: string) => void;
  deletingPath: string | null;
}

export default function ProjectDeleteList({
  projects,
  onDelete,
  deletingPath,
}: ProjectDeleteListProps) {
  if (projects.length === 0) {
    return (
      <div className="bg-white rounded-xl border border-red-200 p-6">
        <p className="text-gray-500 text-sm text-center">
          登録されているプロジェクトがありません
        </p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-xl border border-red-200 p-4">
      <h2 className="text-sm font-bold text-red-700 mb-3">
        プロジェクト削除
      </h2>
      <ul className="space-y-2">
        {projects.map((project) => {
          const isDeleting = deletingPath === project.path;
          return (
            <li
              key={project.path}
              className="flex items-center justify-between gap-3 px-3 py-2 rounded-lg bg-gray-50 border border-gray-200"
            >
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-gray-800 truncate">
                  {project.name}
                </p>
                <p className="text-xs text-gray-500 truncate">
                  {project.path}
                </p>
              </div>
              <button
                onClick={() => onDelete(project.path)}
                disabled={isDeleting}
                className="shrink-0 px-3 py-1 text-xs font-medium text-white bg-red-600 rounded hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {isDeleting ? "削除中..." : "削除"}
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
