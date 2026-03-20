"use client";

import dynamic from "next/dynamic";

// ブラウザ API（WebSocket, AudioContext, getUserMedia）を使用するため
// SSR を無効化して動的にインポート
const GeminiLiveClient = dynamic(
  () => import("@/components/GeminiLiveClient"),
  { ssr: false }
);

export default function GeminiLivePage() {
  return <GeminiLiveClient />;
}
