"use client";

import { useEffect } from "react";
import type { FileInfo } from "@/types";
import { COMMANDS } from "@/lib/constants";

interface CommandFormProps {
  projectPath: string;
  onProjectChange: (path: string) => void;
  command: string;
  onCommandChange: (command: string) => void;
  selectedFile: string;
  onFileChange: (file: string) => void;
  args: string;
  onArgsChange: (args: string) => void;
  groupedFiles: { folder: string; files: FileInfo[] }[];
  onLoadFiles: (project: string) => void;
  onSubmit: () => void;
  isSubmitting: boolean;
}

export default function CommandForm({
  projectPath,
  onProjectChange,
  command,
  onCommandChange,
  selectedFile,
  onFileChange,
  args,
  onArgsChange,
  groupedFiles,
  onLoadFiles,
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

  return (
    <form onSubmit={handleSubmit}>
      <div className="mb-4">
        <label className="block mb-2 font-semibold text-gray-800">
          Project Path
        </label>
        <input
          type="text"
          value={projectPath}
          onChange={(e) => onProjectChange(e.target.value)}
          placeholder="/Users/user/myproject"
          required
          className="w-full p-3 border border-gray-200 rounded-lg text-base bg-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
        />
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
          value={selectedFile}
          onChange={(e) => onFileChange(e.target.value)}
          className="w-full p-3 border border-gray-200 rounded-lg text-base bg-white focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
        >
          <option value="">-- Select a file or type below --</option>
          {groupedFiles.map((group) => (
            <optgroup key={group.folder} label={group.folder}>
              {group.files.map((file) => (
                <option key={file.path} value={file.path}>
                  {file.name}
                </option>
              ))}
            </optgroup>
          ))}
        </select>
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
