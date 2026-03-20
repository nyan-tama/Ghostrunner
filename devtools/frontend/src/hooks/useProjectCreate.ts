"use client";

import { useState, useCallback, useRef } from "react";
import type {
  CreatePhase,
  CreateProgressEvent,
  CreateStep,
  CreatedProject,
  DataService,
} from "@/types";
import { createProjectStream } from "@/lib/createApi";

const STEPS: readonly { id: string; label: string }[] = [
  { id: "template_copy", label: "Copy template files" },
  { id: "placeholder_replace", label: "Replace placeholders" },
  { id: "env_create", label: "Create .env file" },
  { id: "dependency_install", label: "Install dependencies" },
  { id: "claude_assets", label: "Copy Claude assets" },
  { id: "claude_md", label: "Generate CLAUDE.md" },
  { id: "devtools_link", label: "Register with devtools" },
  { id: "git_init", label: "Initialize git repository" },
  { id: "server_start", label: "Start development server" },
  { id: "health_check", label: "Health check" },
] as const;

function buildInitialSteps(): CreateStep[] {
  return STEPS.map((s) => ({ id: s.id, label: s.label, status: "pending" as const }));
}

interface UseProjectCreateReturn {
  phase: CreatePhase;
  steps: CreateStep[];
  progress: number;
  errorMessage: string;
  createdProject: CreatedProject | null;
  startCreate: (name: string, description: string, services: DataService[]) => void;
  resetToForm: () => void;
}

export function useProjectCreate(): UseProjectCreateReturn {
  const [phase, setPhase] = useState<CreatePhase>("form");
  const [steps, setSteps] = useState<CreateStep[]>(buildInitialSteps);
  const [progress, setProgress] = useState(0);
  const [errorMessage, setErrorMessage] = useState("");
  const [createdProject, setCreatedProject] = useState<CreatedProject | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const updateStepStatus = useCallback(
    (stepId: string, status: CreateStep["status"]) => {
      setSteps((prev) =>
        prev.map((s) => (s.id === stepId ? { ...s, status } : s))
      );
    },
    []
  );

  const handleEvent = useCallback(
    (event: CreateProgressEvent, name: string) => {
      setProgress(event.progress);

      switch (event.type) {
        case "progress":
          // 現在のステップをactiveに、完了したステップをdoneに
          setSteps((prev) =>
            prev.map((s) => {
              if (s.id === event.step) {
                return { ...s, status: "active" as const };
              }
              // 既にactiveだったステップをdoneに変更
              if (s.status === "active" && s.id !== event.step) {
                return { ...s, status: "done" as const };
              }
              return s;
            })
          );
          break;

        case "complete":
          // 全ステップをdoneに
          setSteps((prev) => prev.map((s) => ({ ...s, status: "done" as const })));
          setCreatedProject({
            name,
            path: event.path || "",
          });
          setPhase("complete");
          break;

        case "error":
          updateStepStatus(event.step, "error");
          setErrorMessage(event.error || event.message);
          setPhase("error");
          break;
      }
    },
    [updateStepStatus]
  );

  const processSSEResponse = useCallback(
    async (response: Response, name: string) => {
      if (!response.ok) {
        try {
          const data = await response.json();
          setErrorMessage(data.error || "Failed to create project");
        } catch {
          setErrorMessage(`Failed to create project (HTTP ${response.status})`);
        }
        setPhase("error");
        return;
      }

      const reader = response.body?.getReader();
      if (!reader) {
        setErrorMessage("Failed to get response reader");
        setPhase("error");
        return;
      }

      const decoder = new TextDecoder();
      let buffer = "";

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() || "";

          for (const line of lines) {
            if (line.startsWith("data: ")) {
              const data = line.slice(6);
              if (data) {
                try {
                  const event: CreateProgressEvent = JSON.parse(data);
                  handleEvent(event, name);
                } catch {
                  // JSONパースエラーは無視
                }
              }
            }
          }
        }
      } catch (error) {
        if (error instanceof Error && error.name === "AbortError") {
          return;
        }
        const message = error instanceof Error ? error.message : "Stream error";
        setErrorMessage(`Connection lost: ${message}`);
        setPhase("error");
      }
    },
    [handleEvent]
  );

  const startCreate = useCallback(
    (name: string, description: string, services: DataService[]) => {
      // 既存の接続を中断
      abortRef.current?.abort();

      const controller = new AbortController();
      abortRef.current = controller;

      // 状態をリセットして作成開始
      setSteps(buildInitialSteps());
      setProgress(0);
      setErrorMessage("");
      setCreatedProject(null);
      setPhase("creating");

      createProjectStream({ name, description, services }, controller.signal)
        .then((response) => processSSEResponse(response, name))
        .catch((error) => {
          if (error instanceof Error && error.name === "AbortError") {
            return;
          }
          setErrorMessage(
            "Failed to connect: " + (error instanceof Error ? error.message : "Unknown error")
          );
          setPhase("error");
        });
    },
    [processSSEResponse]
  );

  const resetToForm = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setPhase("form");
    setSteps(buildInitialSteps());
    setProgress(0);
    setErrorMessage("");
    setCreatedProject(null);
  }, []);

  return {
    phase,
    steps,
    progress,
    errorMessage,
    createdProject,
    startCreate,
    resetToForm,
  };
}
