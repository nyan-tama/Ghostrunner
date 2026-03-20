"use client";

import { useState, useCallback, useEffect } from "react";
import type { DataService } from "@/types";
import { useProjectValidation } from "@/hooks/useProjectValidation";
import ServiceSelector from "./ServiceSelector";

interface ProjectFormProps {
  onSubmit: (name: string, description: string, services: DataService[]) => void;
  initialName?: string;
  initialDescription?: string;
  initialServices?: DataService[];
}

export default function ProjectForm({
  onSubmit,
  initialName = "",
  initialDescription = "",
  initialServices = [],
}: ProjectFormProps) {
  const [name, setName] = useState(initialName);
  const [description, setDescription] = useState(initialDescription);
  const [services, setServices] = useState<DataService[]>(initialServices);

  const { state: validation, onNameChange } = useProjectValidation();

  // エラー復帰時にinitialNameが設定されていれば初期バリデーションを実行
  useEffect(() => {
    if (initialName) {
      onNameChange(initialName);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setName(value);
      onNameChange(value);
    },
    [onNameChange]
  );

  const canSubmit =
    name.trim().length > 0 &&
    validation.valid === true &&
    !validation.isValidating;

  const handleSubmit = useCallback(() => {
    if (!canSubmit) return;
    onSubmit(name.trim(), description.trim(), services);
  }, [canSubmit, name, description, services, onSubmit]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey && canSubmit) {
        e.preventDefault();
        handleSubmit();
      }
    },
    [canSubmit, handleSubmit]
  );

  return (
    <div className="space-y-6">
      {/* プロジェクト名 */}
      <div>
        <label htmlFor="project-name" className="block text-sm font-medium text-gray-700 mb-1">
          Project Name
        </label>
        <input
          id="project-name"
          type="text"
          value={name}
          onChange={handleNameChange}
          onKeyDown={handleKeyDown}
          placeholder="my-awesome-app"
          className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          autoFocus
        />
        {/* バリデーション結果 */}
        <div className="mt-1 min-h-[20px]">
          {validation.isValidating && (
            <span className="text-xs text-gray-400">Validating...</span>
          )}
          {!validation.isValidating && validation.valid === true && (
            <span className="text-xs text-green-600">
              Will be created at: {validation.path}
            </span>
          )}
          {!validation.isValidating && validation.valid === false && (
            <span className="text-xs text-red-600">{validation.error}</span>
          )}
        </div>
      </div>

      {/* 概要 */}
      <div>
        <label htmlFor="project-desc" className="block text-sm font-medium text-gray-700 mb-1">
          Description (optional)
        </label>
        <textarea
          id="project-desc"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Brief description of the project"
          rows={2}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
        />
      </div>

      {/* Data Services */}
      <ServiceSelector selected={services} onChange={setServices} />

      {/* 確認セクション */}
      <div className="border-t border-gray-200 pt-4">
        <h3 className="text-sm font-medium text-gray-700 mb-2">Summary</h3>
        <div className="bg-gray-50 rounded-lg p-4 text-sm space-y-1">
          <div>
            <span className="text-gray-500">Name:</span>{" "}
            <span className="text-gray-800 font-medium">{name || "-"}</span>
          </div>
          {description && (
            <div>
              <span className="text-gray-500">Description:</span>{" "}
              <span className="text-gray-800">{description}</span>
            </div>
          )}
          <div>
            <span className="text-gray-500">Services:</span>{" "}
            <span className="text-gray-800">
              {services.length > 0 ? services.join(", ") : "None"}
            </span>
          </div>
          {validation.valid && (
            <div>
              <span className="text-gray-500">Path:</span>{" "}
              <span className="text-gray-800 font-mono text-xs">{validation.path}</span>
            </div>
          )}
        </div>
      </div>

      {/* 送信ボタン */}
      <button
        onClick={handleSubmit}
        disabled={!canSubmit}
        className="w-full py-2.5 bg-blue-600 text-white rounded-lg font-medium text-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        Create Project
      </button>
    </div>
  );
}
