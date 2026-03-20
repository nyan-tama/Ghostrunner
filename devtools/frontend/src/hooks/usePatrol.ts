"use client";

import { useState, useCallback, useEffect } from "react";
import type {
  PatrolProject,
  PatrolProjectState,
  PatrolSSEEvent,
} from "@/types/patrol";
import {
  fetchPatrolProjects,
  fetchPatrolStates,
  registerPatrolProject,
  removePatrolProject,
  startPatrol as apiStartPatrol,
  stopPatrol as apiStopPatrol,
  sendPatrolAnswer as apiSendAnswer,
  startPolling as apiStartPolling,
  stopPolling as apiStopPolling,
} from "@/lib/patrolApi";
import { usePatrolSSE } from "@/hooks/usePatrolSSE";
import type { SSEConnectionStatus } from "@/hooks/usePatrolSSE";

interface UsePatrolReturn {
  projects: PatrolProject[];
  projectStates: Record<string, PatrolProjectState>;
  isRunning: boolean;
  isPolling: boolean;
  error: string | null;
  connectionStatus: SSEConnectionStatus;
  registerProject: (path: string) => Promise<void>;
  removeProject: (path: string) => Promise<void>;
  handleStartPatrol: () => Promise<void>;
  handleStopPatrol: () => Promise<void>;
  handleSendAnswer: (projectPath: string, answer: string) => Promise<void>;
  handleTogglePolling: () => Promise<void>;
}

export function usePatrol(): UsePatrolReturn {
  const [projects, setProjects] = useState<PatrolProject[]>([]);
  const [projectStates, setProjectStates] = useState<Record<string, PatrolProjectState>>({});
  const [isRunning, setIsRunning] = useState(false);
  const [isPolling, setIsPolling] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // SSEイベントハンドラ
  const handleSSEEvent = useCallback((event: PatrolSSEEvent) => {
    if (event.state) {
      setProjectStates((prev) => ({
        ...prev,
        [event.project_path]: event.state!,
      }));
    }

    switch (event.type) {
      case "project_started":
        setIsRunning(true);
        break;
      case "project_completed":
      case "project_error":
        // 個別プロジェクトの完了/エラー。巡回全体の停止ではない
        break;
      case "scan_completed":
        // スキャン完了時、全stateを更新
        if (event.state) {
          setProjectStates((prev) => ({
            ...prev,
            [event.project_path]: event.state!,
          }));
        }
        break;
    }
  }, []);

  const { connectionStatus } = usePatrolSSE({ onEvent: handleSSEEvent });

  // 初回マウント時にデータ取得
  useEffect(() => {
    const loadInitialData = async () => {
      try {
        const [projectsRes, statesRes] = await Promise.all([
          fetchPatrolProjects(),
          fetchPatrolStates(),
        ]);
        if (projectsRes.success && projectsRes.projects) {
          setProjects(projectsRes.projects);
        }
        if (statesRes.success && statesRes.states) {
          setProjectStates(statesRes.states);
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to load initial data";
        setError(message);
      }
    };
    loadInitialData();
  }, []);

  const registerProject = useCallback(async (path: string) => {
    try {
      setError(null);
      await registerPatrolProject(path);
      const res = await fetchPatrolProjects();
      if (res.success && res.projects) {
        setProjects(res.projects);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to register project";
      setError(message);
      throw err;
    }
  }, []);

  const removeProject = useCallback(async (path: string) => {
    try {
      setError(null);
      await removePatrolProject(path);
      setProjects((prev) => prev.filter((p) => p.path !== path));
      setProjectStates((prev) => {
        const next = { ...prev };
        delete next[path];
        return next;
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to remove project";
      setError(message);
      throw err;
    }
  }, []);

  const handleStartPatrol = useCallback(async () => {
    try {
      setError(null);
      await apiStartPatrol();
      setIsRunning(true);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to start patrol";
      setError(message);
      throw err;
    }
  }, []);

  const handleStopPatrol = useCallback(async () => {
    try {
      setError(null);
      await apiStopPatrol();
      setIsRunning(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to stop patrol";
      setError(message);
      throw err;
    }
  }, []);

  const handleSendAnswer = useCallback(async (projectPath: string, answer: string) => {
    try {
      setError(null);
      await apiSendAnswer(projectPath, answer);
      // 回答送信後、該当プロジェクトのstateを即座にrunningに更新
      setProjectStates((prev) => {
        const current = prev[projectPath];
        if (!current) return prev;
        return {
          ...prev,
          [projectPath]: {
            ...current,
            status: "running",
            question: undefined,
          },
        };
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to send answer";
      setError(message);
      throw err;
    }
  }, []);

  const handleTogglePolling = useCallback(async () => {
    try {
      setError(null);
      if (isPolling) {
        await apiStopPolling();
        setIsPolling(false);
      } else {
        await apiStartPolling();
        setIsPolling(true);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to toggle polling";
      setError(message);
      throw err;
    }
  }, [isPolling]);

  return {
    projects,
    projectStates,
    isRunning,
    isPolling,
    error,
    connectionStatus,
    registerProject,
    removeProject,
    handleStartPatrol,
    handleStopPatrol,
    handleSendAnswer,
    handleTogglePolling,
  };
}
