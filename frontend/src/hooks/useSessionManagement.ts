"use client";

import { useState, useCallback, useSyncExternalStore } from "react";
import { LOCAL_STORAGE_KEY, DEFAULT_PROJECT_PATH } from "@/lib/constants";

function getStoredProjectPath(): string {
  if (typeof window === "undefined") {
    return DEFAULT_PROJECT_PATH;
  }
  return localStorage.getItem(LOCAL_STORAGE_KEY) || DEFAULT_PROJECT_PATH;
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
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [totalCost, setTotalCost] = useState<number>(0);

  const setProjectPath = useCallback((path: string) => {
    setProjectPathState(path);
    localStorage.setItem(LOCAL_STORAGE_KEY, path);
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
    sessionId,
    setSessionId,
    totalCost,
    addCost,
    resetSession,
  };
}
