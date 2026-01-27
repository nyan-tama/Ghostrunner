"use client";

import dynamic from "next/dynamic";

// ブラウザ API（WebSocket, AudioContext, getUserMedia）を使用するため
// SSR を無効化して動的にインポート
const OpenAIRealtimeClient = dynamic(
  () => import("@/components/OpenAIRealtimeClient"),
  { ssr: false }
);

export default function OpenAIRealtimePage() {
  return <OpenAIRealtimeClient />;
}
