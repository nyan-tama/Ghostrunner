"use client";

import { useState, useEffect } from "react";
import type { ProjectInfo } from "@/types";
import { fetchProjects } from "@/lib/api";

interface ProjectRegisterProps {
  registeredPaths: string[];
  onRegister: (path: string) => void;
}

export default function ProjectRegister({ registeredPaths, onRegister }: ProjectRegisterProps) {
  const [allProjects, setAllProjects] = useState<ProjectInfo[]>([]);
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    fetchProjects().then((res) => {
      if (res.success && res.projects) {
        setAllProjects(res.projects);
      }
    }).catch((err) => {
      console.error("Failed to fetch projects:", err);
    });
  }, [isOpen]);

  const availableProjects = allProjects.filter(
    (p) => !registeredPaths.includes(p.path)
  );

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-700">巡回対象プロジェクト</h2>
        <button
          type="button"
          onClick={() => setIsOpen((prev) => !prev)}
          className="px-3 py-1 text-xs bg-blue-100 text-blue-700 rounded hover:bg-blue-200 transition-colors"
        >
          {isOpen ? "閉じる" : "追加"}
        </button>
      </div>

      {isOpen && (
        <div className="mt-3 space-y-1.5">
          {availableProjects.length === 0 ? (
            <p className="text-xs text-gray-400 py-2">
              追加可能なプロジェクトがありません
            </p>
          ) : (
            availableProjects.map((proj) => (
              <button
                key={proj.path}
                type="button"
                onClick={() => {
                  onRegister(proj.path);
                  setIsOpen(false);
                }}
                className="w-full py-2 px-3 border border-gray-200 rounded-lg text-left hover:border-blue-500 hover:bg-blue-50 transition-all"
              >
                <span className="text-sm font-medium text-gray-800 block">{proj.name}</span>
                <span className="text-xs text-gray-400 block truncate">{proj.path}</span>
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
