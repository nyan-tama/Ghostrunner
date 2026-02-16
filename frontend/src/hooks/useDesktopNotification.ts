"use client";

import { useEffect, useRef, useCallback } from "react";

/**
 * macOS / ブラウザのデスクトップ通知を送るフック。
 * 初回マウント時に通知許可をリクエストする。
 */
export function useDesktopNotification() {
  const permissionRef = useRef<NotificationPermission>("default");

  useEffect(() => {
    if (typeof window === "undefined" || !("Notification" in window)) return;
    permissionRef.current = Notification.permission;
    if (Notification.permission === "default") {
      Notification.requestPermission().then((perm) => {
        permissionRef.current = perm;
      });
    }
  }, []);

  const notify = useCallback((title: string, body?: string) => {
    if (typeof window === "undefined" || !("Notification" in window)) return;
    if (permissionRef.current !== "granted") return;
    // タブがバックグラウンドの時のみ通知（フォアグラウンドなら画面で気づける）
    if (!document.hidden) return;
    new Notification(title, { body: body || undefined });
  }, []);

  return { notify };
}
