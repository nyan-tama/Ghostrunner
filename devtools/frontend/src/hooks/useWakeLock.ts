"use client";

import { useEffect, useRef, useState, useCallback } from "react";

const STORAGE_KEY = "ghostrunner_wake_lock_enabled";

/**
 * Screen Wake Lock API を使ってスマホのスリープを抑制するフック。
 *
 * - enabled を true にすると画面が消えなくなる（バッテリー消費増に注意）
 * - タブ切替 (visibilitychange) で自動解放されるため、復帰時に再取得する
 * - localStorage に ON/OFF 状態を永続化する
 * - Wake Lock API 非対応ブラウザでは isSupported=false を返すだけで何もしない
 */
export function useWakeLock() {
  const [isSupported, setIsSupported] = useState(false);
  const [enabled, setEnabled] = useState(false);
  const [isActive, setIsActive] = useState(false);
  const wakeLockRef = useRef<WakeLockSentinel | null>(null);

  // 初期化: API 対応チェック + localStorage から復元
  useEffect(() => {
    const supported = "wakeLock" in navigator;
    setIsSupported(supported);

    if (supported) {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored === "true") {
        setEnabled(true);
      }
    }
  }, []);

  // Wake Lock の取得
  const requestWakeLock = useCallback(async () => {
    if (!("wakeLock" in navigator)) return;

    try {
      const sentinel = await navigator.wakeLock.request("screen");
      wakeLockRef.current = sentinel;
      setIsActive(true);

      sentinel.addEventListener("release", () => {
        wakeLockRef.current = null;
        setIsActive(false);
      });
    } catch {
      // バッテリー低下時など、OS が拒否する場合がある
      setIsActive(false);
    }
  }, []);

  // Wake Lock の解放
  const releaseWakeLock = useCallback(async () => {
    if (wakeLockRef.current) {
      await wakeLockRef.current.release();
      wakeLockRef.current = null;
      setIsActive(false);
    }
  }, []);

  // enabled 切り替え時に取得/解放
  useEffect(() => {
    if (enabled) {
      requestWakeLock();
    } else {
      releaseWakeLock();
    }
    return () => {
      releaseWakeLock();
    };
  }, [enabled, requestWakeLock, releaseWakeLock]);

  // visibilitychange: タブ復帰時に再取得
  useEffect(() => {
    if (!enabled) return;

    function handleVisibilityChange() {
      if (document.visibilityState === "visible" && enabled) {
        requestWakeLock();
      }
    }

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [enabled, requestWakeLock]);

  // トグル関数
  const toggle = useCallback(() => {
    const next = !enabled;
    setEnabled(next);
    localStorage.setItem(STORAGE_KEY, String(next));
  }, [enabled]);

  return {
    /** Wake Lock API がこのブラウザで使えるか */
    isSupported,
    /** ユーザーが ON にしているか */
    enabled,
    /** 実際に Wake Lock が有効か（OS が拒否すると false になりうる） */
    isActive,
    /** ON/OFF を切り替える */
    toggle,
  };
}
