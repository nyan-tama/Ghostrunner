// OpenAI Realtime API 関連の型定義

// 接続状態
export type OpenAIConnectionStatus = "disconnected" | "connecting" | "connected" | "error";

// 設定
export interface OpenAIRealtimeConfig {
  model?: string;
  voice?: string;
  instructions?: string;
  modalities?: string[];
}

// クライアント -> サーバー（送信用）
export interface OpenAISessionUpdate {
  type: "session.update";
  session: {
    modalities?: string[];
    instructions?: string;
    voice?: string;
  };
}

export interface OpenAIInputAudioBufferAppend {
  type: "input_audio_buffer.append";
  audio: string; // Base64 encoded PCM
}

// サーバー -> クライアント（受信用）
export interface OpenAISessionCreatedEvent {
  type: "session.created";
  session: {
    id: string;
    model: string;
  };
}

export interface OpenAISessionUpdatedEvent {
  type: "session.updated";
  session: {
    id: string;
    model: string;
  };
}

export interface OpenAIResponseAudioDeltaEvent {
  type: "response.audio.delta";
  delta: string; // Base64 encoded PCM
  response_id: string;
  item_id: string;
}

export interface OpenAIResponseAudioDoneEvent {
  type: "response.audio.done";
  response_id: string;
  item_id: string;
}

export interface OpenAIErrorEvent {
  type: "error";
  error: {
    message: string;
    type?: string;
    code?: string;
  };
}

export type OpenAIServerEvent =
  | OpenAISessionCreatedEvent
  | OpenAISessionUpdatedEvent
  | OpenAIResponseAudioDeltaEvent
  | OpenAIResponseAudioDoneEvent
  | OpenAIErrorEvent;

// バックエンドAPI連携
export interface OpenAITokenResponse {
  success: boolean;
  token?: string;
  expireTime?: string;
  error?: string;
}

// 型ガード関数
export function isSessionCreated(msg: OpenAIServerEvent): msg is OpenAISessionCreatedEvent {
  return msg.type === "session.created";
}

export function isSessionUpdated(msg: OpenAIServerEvent): msg is OpenAISessionUpdatedEvent {
  return msg.type === "session.updated";
}

export function isResponseAudioDelta(msg: OpenAIServerEvent): msg is OpenAIResponseAudioDeltaEvent {
  return msg.type === "response.audio.delta";
}

export function isResponseAudioDone(msg: OpenAIServerEvent): msg is OpenAIResponseAudioDoneEvent {
  return msg.type === "response.audio.done";
}

export function isOpenAIError(msg: OpenAIServerEvent): msg is OpenAIErrorEvent {
  return msg.type === "error";
}
