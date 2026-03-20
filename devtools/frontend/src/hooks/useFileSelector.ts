"use client";

import { useState, useCallback } from "react";
import type { FileInfo } from "@/types";
import { fetchFiles } from "@/lib/api";
import { DEV_FOLDERS } from "@/lib/constants";

export function useFileSelector() {
  const [files, setFiles] = useState<Record<string, FileInfo[]>>({});
  const [selectedFiles, setSelectedFiles] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const addSelectedFile = useCallback((file: string) => {
    setSelectedFiles((prev) => {
      if (prev.includes(file)) return prev; // 重複防止
      return [...prev, file];
    });
  }, []);

  const removeSelectedFile = useCallback((file: string) => {
    setSelectedFiles((prev) => prev.filter((f) => f !== file));
  }, []);

  const clearSelectedFiles = useCallback(() => {
    setSelectedFiles([]);
  }, []);

  // 内部でファイルを取得する関数（ローディング表示なし）
  const fetchFilesInternal = useCallback(async (project: string) => {
    if (!project.trim()) return;

    try {
      const data = await fetchFiles(project);
      if (data.success && data.files) {
        setFiles(data.files);
      }
      // リフレッシュ中はエラーを表示しない（初回ロード時のみ表示）
    } catch {
      // リフレッシュ中のエラーは無視
    }
  }, []);

  // 初回ロード用の関数（ローディング表示あり）
  const loadFiles = useCallback(async (project: string) => {
    if (!project.trim()) return;

    setIsLoading(true);
    setError(null);
    try {
      const data = await fetchFiles(project);
      if (data.success && data.files) {
        setFiles(data.files);
      } else if (data.error) {
        setError(data.error);
      }
    } catch {
      setError("Failed to load files");
    } finally {
      setIsLoading(false);
    }
  }, []);

  // ドロップダウンフォーカス時のリフレッシュ用関数（ローディング表示なし）
  const refreshFiles = useCallback(
    (project: string) => {
      fetchFilesInternal(project);
    },
    [fetchFilesInternal]
  );

  const getGroupedFiles = useCallback(() => {
    return DEV_FOLDERS.map((folder) => ({
      folder,
      files: files[folder] || [],
    })).filter((group) => group.files.length > 0);
  }, [files]);

  return {
    files,
    selectedFiles,
    addSelectedFile,
    removeSelectedFile,
    clearSelectedFiles,
    loadFiles,
    refreshFiles,
    getGroupedFiles,
    isLoading,
    error,
  };
}
