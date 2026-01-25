"use client";

import { useState, useCallback } from "react";
import type { FileInfo } from "@/types";
import { fetchFiles } from "@/lib/api";
import { DEV_FOLDERS } from "@/lib/constants";

export function useFileSelector() {
  const [files, setFiles] = useState<Record<string, FileInfo[]>>({});
  const [selectedFile, setSelectedFile] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(false);

  const [error, setError] = useState<string | null>(null);

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

  const getGroupedFiles = useCallback(() => {
    return DEV_FOLDERS.map((folder) => ({
      folder,
      files: files[folder] || [],
    })).filter((group) => group.files.length > 0);
  }, [files]);

  return {
    files,
    selectedFile,
    setSelectedFile,
    loadFiles,
    getGroupedFiles,
    isLoading,
    error,
  };
}
