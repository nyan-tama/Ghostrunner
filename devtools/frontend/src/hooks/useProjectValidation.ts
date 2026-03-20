"use client";

import { useState, useCallback, useRef } from "react";
import { validateProjectName } from "@/lib/createApi";

interface ValidationState {
  isValidating: boolean;
  valid: boolean | null;
  path: string;
  error: string;
}

const EMPTY_STATE: ValidationState = {
  isValidating: false,
  valid: null,
  path: "",
  error: "",
};

/**
 * 300msデバウンス付きプロジェクト名バリデーション
 *
 * nameの変更をonNameChangeコールバックで受け取り、デバウンス後にAPIを呼び出す。
 * タイマーとAbortControllerはuseRefで管理し、onNameChangeの参照安定性を保つ。
 */
export function useProjectValidation(): {
  state: ValidationState;
  onNameChange: (name: string) => void;
} {
  const [state, setState] = useState<ValidationState>(EMPTY_STATE);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const controllerRef = useRef<AbortController | null>(null);

  const onNameChange = useCallback((name: string) => {
    // 前回のタイマーとリクエストをキャンセル
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }
    if (controllerRef.current) {
      controllerRef.current.abort();
    }

    const trimmed = name.trim();
    if (!trimmed) {
      timerRef.current = null;
      controllerRef.current = null;
      setState(EMPTY_STATE);
      return;
    }

    setState((prev) => ({ ...prev, isValidating: true, valid: null, error: "" }));

    const controller = new AbortController();
    controllerRef.current = controller;
    timerRef.current = setTimeout(async () => {
      try {
        const result = await validateProjectName(trimmed, controller.signal);
        if (!controller.signal.aborted) {
          setState({
            isValidating: false,
            valid: result.valid,
            path: result.path,
            error: result.error || "",
          });
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setState({
            isValidating: false,
            valid: false,
            path: "",
            error: err instanceof Error ? err.message : "Validation failed",
          });
        }
      }
    }, 300);
  }, []);

  return { state, onNameChange };
}
