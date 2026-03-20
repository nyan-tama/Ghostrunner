"use client";

import { useState, useCallback, useEffect, useSyncExternalStore } from "react";
import {
  LOCAL_STORAGE_KEY,
  LOCAL_STORAGE_HISTORY_KEY,
  MAX_PROJECT_HISTORY,
  DEFAULT_PROJECT_PATH,
} from "@/lib/constants";

function getStoredProjectPath(): string {
  if (typeof window === "undefined") {
    return DEFAULT_PROJECT_PATH;
  }
  return localStorage.getItem(LOCAL_STORAGE_KEY) || DEFAULT_PROJECT_PATH;
}

function getStoredHistory(): string[] {
  if (typeof window === "undefined") {
    return [];
  }
  try {
    const stored = localStorage.getItem(LOCAL_STORAGE_HISTORY_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch {
    return [];
  }
}

function subscribe(callback: () => void): () => void {
  window.addEventListener("storage", callback);
  return () => window.removeEventListener("storage", callback);
}

export function useSessionManagement() {
  const storedPath = useSyncExternalStore(
    subscribe,
    getStoredProjectPath,
    () => DEFAULT_PROJECT_PATH
  );

  const [projectPath, setProjectPathState] = useState<string>(storedPath);
  const [projectHistory, setProjectHistory] = useState<string[]>([]);

  useEffect(() => {
    setProjectHistory(getStoredHistory());
  }, []);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [totalCost, setTotalCost] = useState<number>(0);

  const setProjectPath = useCallback((path: string) => {
    setProjectPathState(path);
    localStorage.setItem(LOCAL_STORAGE_KEY, path);
  }, []);

  const addToHistory = useCallback((path: string) => {
    if (!path.trim()) return;

    setProjectHistory((prev) => {
      const filtered = prev.filter((p) => p !== path);
      const updated = [path, ...filtered].slice(0, MAX_PROJECT_HISTORY);
      localStorage.setItem(LOCAL_STORAGE_HISTORY_KEY, JSON.stringify(updated));
      return updated;
    });
  }, []);

  const resetSession = useCallback(() => {
    setSessionId(null);
    setTotalCost(0);
  }, []);

  const addCost = useCallback((cost: number) => {
    setTotalCost((prev) => prev + cost);
  }, []);

  return {
    projectPath,
    setProjectPath,
    projectHistory,
    addToHistory,
    sessionId,
    setSessionId,
    totalCost,
    addCost,
    resetSession,
  };
}
