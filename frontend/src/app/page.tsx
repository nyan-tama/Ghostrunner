"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import type { StreamEvent, DisplayEvent, Question, ToolInput, RestartStatus, ImageData, ProjectInfo } from "@/types";
import {
  PLAN_APPROVAL_KEYWORDS,
  BACKEND_HEALTH_URL,
  LOCAL_STORAGE_GIT_WORKFLOW_KEY,
  GIT_WORKFLOW_INSTRUCTION,
} from "@/lib/constants";
import { executeCommandStream, continueSessionStream, fetchProjects } from "@/lib/api";
import { useSSEStream } from "@/hooks/useSSEStream";
import { useSessionManagement } from "@/hooks/useSessionManagement";
import { useFileSelector } from "@/hooks/useFileSelector";
import { useVoiceNotification } from "@/hooks/useVoiceNotification";
import { useDesktopNotification } from "@/hooks/useDesktopNotification";
import CommandForm from "@/components/CommandForm";
import ProgressContainer from "@/components/ProgressContainer";

function truncate(str: string | undefined, len: number): string {
  if (!str) return "";
  return str.length > len ? str.substring(0, len) + "..." : str;
}

function shortenPath(path: string | undefined): string {
  if (!path) return "";
  const parts = path.split("/");
  if (parts.length <= 4) return path;
  return ".../" + parts.slice(-3).join("/");
}

export default function Home() {
  const {
    projectPath,
    setProjectPath,
    projectHistory,
    addToHistory,
    sessionId,
    setSessionId,
    totalCost,
    addCost,
    resetSession,
  } = useSessionManagement();

  const {
    selectedFiles,
    addSelectedFile,
    removeSelectedFile,
    loadFiles,
    refreshFiles,
    getGroupedFiles,
  } = useFileSelector();

  const [projects, setProjects] = useState<ProjectInfo[]>([]);

  useEffect(() => {
    fetchProjects().then((result) => {
      if (result.success && result.projects) {
        setProjects(result.projects);
      }
    });
  }, []);

  const [command, setCommand] = useState("plan");
  const [args, setArgs] = useState("");
  const [images, setImages] = useState<ImageData[]>([]);
  const [gitWorkflow, setGitWorkflow] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem(LOCAL_STORAGE_GIT_WORKFLOW_KEY) === "true";
  });

  const handleGitWorkflowChange = useCallback((enabled: boolean) => {
    setGitWorkflow(enabled);
    localStorage.setItem(LOCAL_STORAGE_GIT_WORKFLOW_KEY, String(enabled));
  }, []);

  // 音声通知フック
  const {
    enabled: voiceNotificationEnabled,
    setEnabled: setVoiceNotificationEnabled,
    connectionStatus: voiceConnectionStatus,
    isRecording: voiceIsRecording,
    error: voiceError,
    notifyCompletion,
    notifyError,
    startRecording: voiceStartRecording,
    stopRecording: voiceStopRecording,
  } = useVoiceNotification();

  const { notify: desktopNotify } = useDesktopNotification();

  const handleVoiceNotificationChange = useCallback((enabled: boolean) => {
    setVoiceNotificationEnabled(enabled);
  }, [setVoiceNotificationEnabled]);

  const abortControllerRef = useRef<AbortController | null>(null);

  const [showProgress, setShowProgress] = useState(false);
  const [prompt, setPrompt] = useState("");
  const [events, setEvents] = useState<DisplayEvent[]>([]);
  const [loadingText, setLoadingText] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [showQuestions, setShowQuestions] = useState(false);
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [showPlanApproval, setShowPlanApproval] = useState(false);
  const [resultOutput, setResultOutput] = useState("");
  const [resultType, setResultType] = useState<"success" | "error" | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // サーバー再起動機能（開発環境のみ）
  const [restartStatus, setRestartStatus] = useState<RestartStatus>("idle");

  // 質問セットと同時にインデックスをリセット
  const setQuestionsWithReset = useCallback((newQuestions: Question[]) => {
    setQuestions(newQuestions);
    setCurrentQuestionIndex(0);
  }, []);

  const handleRestartServers = useCallback(async () => {
    setRestartStatus("restarting");

    // Fire-and-Forget: 両方のAPIを呼び出し
    fetch("/api/restart/backend", { method: "POST" }).catch(() => {});
    fetch("/api/restart/frontend", { method: "POST" }).catch(() => {});

    // ヘルスチェックポーリング（30秒間、1秒ごと）
    for (let i = 0; i < 30; i++) {
      try {
        const res = await fetch(BACKEND_HEALTH_URL);
        if (res.ok) {
          setRestartStatus("success");
          // 少し待ってからリロード（ユーザーに成功を見せるため）
          setTimeout(() => {
            window.location.reload();
          }, 500);
          return;
        }
      } catch {
        // エラーは無視してリトライ
      }
      await new Promise((r) => setTimeout(r, 1000));
    }

    // タイムアウト
    setRestartStatus("timeout");
  }, []);

  const addEvent = useCallback(
    (type: DisplayEvent["type"], title: string, detail?: string, fullText?: string) => {
      const id = `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
      setEvents((prev) => [...prev, { id, type, title, detail, fullText }]);
    },
    []
  );

  const handleToolUse = useCallback(
    (toolName: string, toolInput: ToolInput, message?: string) => {
      setLoadingText(message || `Using ${toolName}...`);

      switch (toolName) {
        case "Task": {
          const taskPrompt = toolInput.prompt || "";
          const taskType = toolInput.subagent_type || toolInput.description || "Task";
          addEvent("task", `Task: ${taskType}`, truncate(taskPrompt, 200));
          break;
        }
        case "Read": {
          const readPath = toolInput.file_path || "";
          const readOffset = toolInput.offset ? ` (offset: ${toolInput.offset})` : "";
          const readLimit = toolInput.limit ? ` (limit: ${toolInput.limit})` : "";
          addEvent("tool", "Read", `${shortenPath(readPath)}${readOffset}${readLimit}`);
          break;
        }
        case "Write": {
          const writePath = toolInput.file_path || "";
          const contentLen = toolInput.content ? toolInput.content.length : 0;
          addEvent("tool", "Write", `${shortenPath(writePath)} (${contentLen} chars)`);
          break;
        }
        case "Edit": {
          const editPath = toolInput.file_path || "";
          const oldLen = toolInput.old_string ? toolInput.old_string.length : 0;
          const newLen = toolInput.new_string ? toolInput.new_string.length : 0;
          addEvent("tool", "Edit", `${shortenPath(editPath)} (${oldLen} -> ${newLen} chars)`);
          break;
        }
        case "Glob": {
          const globPattern = toolInput.pattern || "";
          const globPath = toolInput.path ? ` in ${shortenPath(toolInput.path)}` : "";
          addEvent("tool", "Glob", `"${globPattern}"${globPath}`);
          break;
        }
        case "Grep": {
          const grepPattern = toolInput.pattern || "";
          const grepPath = toolInput.path ? ` in ${shortenPath(toolInput.path)}` : "";
          const grepGlob = toolInput.glob ? ` (${toolInput.glob})` : "";
          addEvent("tool", "Grep", `"${truncate(grepPattern, 50)}"${grepPath}${grepGlob}`);
          break;
        }
        case "Bash": {
          const cmd = toolInput.command || "";
          const desc = toolInput.description ? `[${toolInput.description}] ` : "";
          addEvent("tool", "Bash", `${desc}${truncate(cmd, 80)}`);
          break;
        }
        case "TodoWrite": {
          const todoCount = toolInput.todos ? toolInput.todos.length : 0;
          addEvent("tool", "TodoWrite", `${todoCount} items`);
          break;
        }
        case "WebFetch": {
          const fetchUrl = toolInput.url || "";
          addEvent("tool", "WebFetch", truncate(fetchUrl, 60));
          break;
        }
        case "WebSearch": {
          const searchQuery = toolInput.query || "";
          addEvent("tool", "WebSearch", `"${truncate(searchQuery, 50)}"`);
          break;
        }
        case "ExitPlanMode":
          addEvent("tool", "ExitPlanMode", "Requesting plan approval");
          break;
        case "EnterPlanMode":
          addEvent("tool", "EnterPlanMode", "Starting plan mode");
          break;
        case "AskUserQuestion":
          break;
        default: {
          const inputStr = JSON.stringify(toolInput);
          addEvent("tool", toolName, truncate(inputStr, 100));
        }
      }
    },
    [addEvent]
  );

  const handleStreamEvent = useCallback(
    (event: StreamEvent) => {
      if (event.session_id) {
        setSessionId(event.session_id);
      }

      switch (event.type) {
        case "init":
          addEvent("info", "Session started");
          setLoadingText("Starting...");
          setIsLoading(true);
          break;

        case "thinking":
          setLoadingText("Thinking...");
          setIsLoading(true);
          addEvent("info", "Thinking...");
          break;

        case "tool_use":
          if (event.tool_name) {
            handleToolUse(
              event.tool_name,
              (event.tool_input as ToolInput) || {},
              event.message
            );
          }
          break;

        case "text":
          if (event.message) {
            if (event.message.length > 200) {
              addEvent("text", "Output", undefined, event.message);
            } else {
              addEvent("text", event.message);
            }
          }
          break;

        case "question":
          setIsLoading(false);
          if (event.result?.questions) {
            setQuestionsWithReset(event.result.questions);
            setShowQuestions(true);
          }
          break;

        case "complete":
          setIsLoading(false);
          if (event.result) {
            if (event.result.cost_usd) {
              addCost(event.result.cost_usd);
            }
            if (event.result.questions && event.result.questions.length > 0) {
              setQuestionsWithReset(event.result.questions);
              setShowQuestions(true);
            } else {
              const output = event.result.output || "(completed)";
              setResultOutput(output);
              setResultType("success");

              // 通知: 完了
              notifyCompletion(output);
              desktopNotify("コマンド完了", truncate(output, 100));

              const needsApproval = PLAN_APPROVAL_KEYWORDS.some((keyword) =>
                output.includes(keyword)
              );
              if (needsApproval && event.result.session_id) {
                setShowPlanApproval(true);
              }
            }
          }
          break;

        case "error":
          setIsLoading(false);
          addEvent("error", event.message || "Error occurred");
          break;
      }
    },
    [addEvent, addCost, handleToolUse, setSessionId, setQuestionsWithReset, notifyCompletion, desktopNotify]
  );

  const handleError = useCallback((error: string) => {
    setIsLoading(false);
    setIsSubmitting(false);
    setResultOutput(error);
    setResultType("error");
    // 通知: エラー
    notifyError(error);
    desktopNotify("コマンドエラー", truncate(error, 100));
  }, [notifyError, desktopNotify]);

  const handleComplete = useCallback(() => {
    setIsLoading(false);
    setIsSubmitting(false);
  }, []);

  const { processStream } = useSSEStream({
    onEvent: handleStreamEvent,
    onError: handleError,
    onComplete: handleComplete,
  });

  const resetProgress = useCallback(() => {
    setEvents([]);
    setQuestions([]);
    setShowQuestions(false);
    setShowPlanApproval(false);
    setResultOutput("");
    setResultType(null);
    setLoadingText("Starting...");
    setIsLoading(true);
  }, []);

  const handleSubmit = useCallback(async () => {
    let combinedArgs = "";
    if (selectedFiles.length > 0 && args) {
      combinedArgs = selectedFiles.join(" ") + " " + args;
    } else if (selectedFiles.length > 0) {
      combinedArgs = selectedFiles.join(" ");
    } else {
      combinedArgs = args;
    }

    if (gitWorkflow) {
      combinedArgs += GIT_WORKFLOW_INSTRUCTION;
    }

    if (!projectPath || !combinedArgs) return;

    // 既存のAbortControllerがあればキャンセル
    abortControllerRef.current?.abort();

    // 新しいAbortControllerを作成
    const controller = new AbortController();
    abortControllerRef.current = controller;

    addToHistory(projectPath);
    resetSession();
    resetProgress();
    setShowProgress(true);
    setPrompt(`/${command} ${combinedArgs}`);
    setIsSubmitting(true);

    try {
      const response = await executeCommandStream(
        {
          project: projectPath,
          command,
          args: combinedArgs,
          images: images.length > 0 ? images : undefined,
        },
        controller.signal
      );
      await processStream(response);
    } catch (error) {
      // AbortError は handleAbort で処理済みなので無視
      if (error instanceof Error && error.name === "AbortError") {
        return;
      }
      handleError("Failed to connect: " + (error as Error).message);
    } finally {
      // 完了後にAbortControllerをクリア
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null;
      }
    }
  }, [
    projectPath,
    command,
    args,
    selectedFiles,
    images,
    gitWorkflow,
    addToHistory,
    resetSession,
    resetProgress,
    processStream,
    handleError,
  ]);

  const handleAnswer = useCallback(
    async (answer: string) => {
      if (!sessionId) {
        handleError("No active session");
        return;
      }

      // 既存のAbortControllerがあればキャンセル
      abortControllerRef.current?.abort();

      // 新しいAbortControllerを作成
      const controller = new AbortController();
      abortControllerRef.current = controller;

      setShowQuestions(false);
      setShowPlanApproval(false);
      setLoadingText("Continuing...");
      setIsLoading(true);
      setIsSubmitting(true);

      try {
        const response = await continueSessionStream(
          {
            project: projectPath,
            session_id: sessionId,
            answer,
          },
          controller.signal
        );
        await processStream(response);
      } catch (error) {
        // AbortError は handleAbort で処理済みなので無視
        if (error instanceof Error && error.name === "AbortError") {
          return;
        }
        handleError("Failed to connect: " + (error as Error).message);
      } finally {
        // 完了後にAbortControllerをクリア
        if (abortControllerRef.current === controller) {
          abortControllerRef.current = null;
        }
      }
    },
    [sessionId, projectPath, processStream, handleError]
  );

  // 逐次質問表示用: 最後の質問以外はインデックスをインクリメント、最後の質問のみバックエンドに送信
  const handleAnswerWithNext = useCallback(
    (answer: string) => {
      const isLastQuestion = currentQuestionIndex >= questions.length - 1;
      if (isLastQuestion) {
        handleAnswer(answer);
      } else {
        setCurrentQuestionIndex((prev) => prev + 1);
      }
    },
    [currentQuestionIndex, questions.length, handleAnswer]
  );

  const handleApprove = useCallback(() => {
    handleAnswer("yes, proceed with the plan");
  }, [handleAnswer]);

  const handleReject = useCallback(() => {
    handleAnswer("no, cancel the plan");
  }, [handleAnswer]);

  const handleAbort = useCallback(() => {
    // AbortController で接続を切断
    abortControllerRef.current?.abort();
    abortControllerRef.current = null;

    // 状態をリセット
    setIsLoading(false);
    setIsSubmitting(false);
    setLoadingText("");

    // 実行ログ（events）は保持し、中断イベントを追加
    addEvent("info", "Execution aborted");

    // 結果表示
    setResultOutput("Execution aborted by user");
    setResultType("error");

    // 質問・承認UIは非表示
    setShowQuestions(false);
    setShowPlanApproval(false);
    setQuestions([]);
  }, [addEvent]);

  return (
    <div className="max-w-[900px] mx-auto px-5 py-5 bg-gray-100 min-h-screen">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-gray-800 text-2xl font-bold">Ghost Runner</h1>
        <div className="flex gap-2 items-center">
          <a
            href={projectPath ? `/docs?project=${encodeURIComponent(projectPath)}` : "/docs"}
            className="px-3 py-1 text-xs bg-gray-100 text-gray-600 rounded hover:bg-gray-200 transition-colors"
            title="Documentation"
          >
            Docs
          </a>
          <a
            href="/gemini-live"
            className="px-3 py-1 text-xs bg-blue-100 text-blue-700 rounded hover:bg-blue-200 transition-colors"
            title="Gemini Live API interface"
          >
            Gemini Live
          </a>
          <a
            href="/openai-realtime"
            className="px-3 py-1 text-xs bg-green-100 text-green-700 rounded hover:bg-green-200 transition-colors"
            title="OpenAI Realtime API interface"
          >
            OpenAI Realtime
          </a>
          {process.env.NODE_ENV === "development" && (
            <button
              onClick={handleRestartServers}
              disabled={restartStatus === "restarting"}
              className="px-3 py-1 text-xs bg-gray-200 text-gray-600 rounded hover:bg-gray-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              title="Restart Backend and Frontend servers"
            >
              {restartStatus === "restarting"
                ? "Restarting..."
                : restartStatus === "timeout"
                  ? "Timeout - Reload manually"
                  : "Restart Servers"}
            </button>
          )}
        </div>
      </div>

      <CommandForm
        projectPath={projectPath}
        onProjectChange={setProjectPath}
        projects={projects}
        projectHistory={projectHistory}
        command={command}
        onCommandChange={setCommand}
        selectedFiles={selectedFiles}
        onAddFile={addSelectedFile}
        onRemoveFile={removeSelectedFile}
        args={args}
        onArgsChange={setArgs}
        images={images}
        onImagesChange={setImages}
        groupedFiles={getGroupedFiles()}
        onLoadFiles={loadFiles}
        onRefreshFiles={refreshFiles}
        onSubmit={handleSubmit}
        isSubmitting={isSubmitting}
        gitWorkflow={gitWorkflow}
        onGitWorkflowChange={handleGitWorkflowChange}
        voiceNotificationEnabled={voiceNotificationEnabled}
        onVoiceNotificationChange={handleVoiceNotificationChange}
        voiceConnectionStatus={voiceConnectionStatus}
        voiceIsRecording={voiceIsRecording}
        voiceError={voiceError}
        onVoiceStartRecording={voiceStartRecording}
        onVoiceStopRecording={voiceStopRecording}
      />

      <ProgressContainer
        visible={showProgress}
        prompt={prompt}
        events={events}
        loadingText={loadingText}
        isLoading={isLoading}
        questions={questions}
        showQuestions={showQuestions}
        currentQuestionIndex={currentQuestionIndex}
        showPlanApproval={showPlanApproval}
        resultOutput={resultOutput}
        resultType={resultType}
        sessionId={sessionId}
        totalCost={totalCost}
        onAnswer={handleAnswerWithNext}
        onApprove={handleApprove}
        onReject={handleReject}
        onAbort={handleAbort}
        canAbort={isLoading && !showQuestions && !showPlanApproval}
      />
    </div>
  );
}
