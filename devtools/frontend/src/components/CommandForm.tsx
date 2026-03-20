"use client";

import { useEffect } from "react";
import type { FileInfo, ImageData, ProjectInfo } from "@/types";
import { COMMANDS } from "@/lib/constants";
import ImageUploader from "@/components/ImageUploader";

interface CommandFormProps {
  projectPath: string;
  onProjectChange: (path: string) => void;
  projects: ProjectInfo[];
  projectHistory: string[];
  command: string;
  onCommandChange: (command: string) => void;
  selectedFiles: string[];
  onAddFile: (file: string) => void;
  onRemoveFile: (file: string) => void;
  args: string;
  onArgsChange: (args: string) => void;
  images: ImageData[];
  onImagesChange: (images: ImageData[]) => void;
  groupedFiles: { folder: string; files: FileInfo[] }[];
  onLoadFiles: (project: string) => void;
  onRefreshFiles: (project: string) => void;
  onSubmit: () => void;
  isSubmitting: boolean;
}

export default function CommandForm({
  projectPath,
  onProjectChange,
  projects,
  projectHistory,
  command,
  onCommandChange,
  selectedFiles,
  onAddFile,
  onRemoveFile,
  args,
  onArgsChange,
  images,
  onImagesChange,
  groupedFiles,
  onLoadFiles,
  onRefreshFiles,
  onSubmit,
  isSubmitting,
}: CommandFormProps) {
  useEffect(() => {
    if (projectPath) {
      onLoadFiles(projectPath);
    }
  }, [projectPath, onLoadFiles]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit();
  };

  const handleHistorySelect = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    if (value) {
      onProjectChange(value);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="mb-4">
        <label className="block mb-2 font-semibold text-gray-800">
          プロジェクト
        </label>
        <div className="flex gap-2">
          <select
            value={projectPath}
            onChange={(e) => onProjectChange(e.target.value)}
            required
            className="flex-1 min-w-0 p-3 border border-gray-200 rounded-lg text-base bg-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
          >
            <option value="" disabled>-- プロジェクトを選択 --</option>
            {projectPath && !projects.some((p) => p.path === projectPath) && (
              <option value={projectPath}>
                {projectPath.split("/").pop()} (custom)
              </option>
            )}
            {projects.map((project) => (
              <option key={project.path} value={project.path}>
                {project.name}
              </option>
            ))}
          </select>
          {projectHistory.length > 0 && (
            <select
              value=""
              onChange={handleHistorySelect}
              className="w-20 shrink-0 p-3 border border-gray-200 rounded-lg text-base bg-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
              title="履歴から選択"
            >
              <option value="">履歴</option>
              {projectHistory.map((path) => (
                <option key={path} value={path}>
                  {path.split("/").slice(-2).join("/")}
                </option>
              ))}
            </select>
          )}
        </div>
      </div>

      <div className="mb-4">
        <label className="block mb-2 font-semibold text-gray-800">コマンド</label>
        <select
          value={command}
          onChange={(e) => onCommandChange(e.target.value)}
          className="w-full p-3 border border-gray-200 rounded-lg text-base bg-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
        >
          {COMMANDS.map((cmd) => (
            <option key={cmd.value} value={cmd.value}>
              {cmd.label}
            </option>
          ))}
        </select>
      </div>

      <div className="mb-4">
        <label className="block mb-2 font-semibold text-gray-800">
          ファイル（任意）
        </label>
        <select
          value=""
          onChange={(e) => {
            if (e.target.value) {
              onAddFile(e.target.value);
            }
          }}
          onFocus={() => onRefreshFiles(projectPath)}
          className="w-full p-3 border border-gray-200 rounded-lg text-base bg-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
        >
          <option value="">-- ファイルを追加 --</option>
          {groupedFiles.map((group) => (
            <optgroup key={group.folder} label={group.folder}>
              {group.files.map((file) => {
                const isSelected = selectedFiles.includes(file.path);
                return (
                  <option
                    key={file.path}
                    value={file.path}
                    disabled={isSelected}
                  >
                    {isSelected ? `\u2713 ${file.name}` : file.name}
                  </option>
                );
              })}
            </optgroup>
          ))}
        </select>
        {selectedFiles.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-2">
            {selectedFiles.map((file) => (
              <div
                key={file}
                className="inline-flex items-center gap-1 px-2 py-1 bg-blue-50 border border-blue-200 rounded text-sm text-gray-700"
              >
                <span>{file.split("/").pop()}</span>
                <button
                  type="button"
                  onClick={() => onRemoveFile(file)}
                  className="text-gray-500 hover:text-red-600 focus:outline-none"
                  title="削除"
                >
                  x
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="mb-4">
        <label className="block mb-2 font-semibold text-gray-800">内容</label>
        <textarea
          value={args}
          onChange={(e) => onArgsChange(e.target.value)}
          placeholder="実装したい内容を入力..."
          required
          className="w-full p-3 border border-gray-200 rounded-lg text-base bg-white min-h-20 resize-y focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
        />
      </div>

      <ImageUploader images={images} onImagesChange={onImagesChange} />

      <button
        type="submit"
        disabled={isSubmitting}
        className="w-full py-3.5 px-6 bg-blue-500 text-white rounded-lg font-semibold text-base cursor-pointer transition-colors hover:bg-blue-600 disabled:bg-gray-400 disabled:cursor-not-allowed border-none"
      >
        実行
      </button>
    </form>
  );
}
