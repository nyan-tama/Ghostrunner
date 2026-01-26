"use client";

import { useEffect } from "react";
import type { FileInfo, ImageData } from "@/types";
import { COMMANDS } from "@/lib/constants";
import ImageUploader from "@/components/ImageUploader";

interface CommandFormProps {
  projectPath: string;
  onProjectChange: (path: string) => void;
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
          Project Path
        </label>
        <div className="flex gap-2">
          <input
            type="text"
            value={projectPath}
            onChange={(e) => onProjectChange(e.target.value)}
            placeholder="/Users/user/myproject"
            required
            className="flex-1 p-3 border border-gray-200 rounded-lg text-base bg-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
          />
          {projectHistory.length > 0 && (
            <select
              value=""
              onChange={handleHistorySelect}
              className="p-3 border border-gray-200 rounded-lg text-base bg-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
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
        <label className="block mb-2 font-semibold text-gray-800">Command</label>
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
          File (optional)
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
          <option value="">-- Select files to add --</option>
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
                  title="Remove file"
                >
                  x
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="mb-4">
        <label className="block mb-2 font-semibold text-gray-800">Arguments</label>
        <textarea
          value={args}
          onChange={(e) => onArgsChange(e.target.value)}
          placeholder="Describe what you want to implement..."
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
        Execute Command
      </button>
    </form>
  );
}
