// Gemini Live API 関連の型定義

// 接続状態
export type ConnectionStatus = "disconnected" | "connecting" | "connected" | "error";

// VAD設定用の型
export interface AutomaticActivityDetection {
  disabled?: boolean;
  startOfSpeechSensitivity?: "START_SENSITIVITY_LOW" | "START_SENSITIVITY_MEDIUM" | "START_SENSITIVITY_HIGH";
  endOfSpeechSensitivity?: "END_SENSITIVITY_LOW" | "END_SENSITIVITY_MEDIUM" | "END_SENSITIVITY_HIGH";
  prefixPaddingMs?: number;
  silenceDurationMs?: number;
}

// setup メッセージ用の型
export interface GeminiLiveSetupMessage {
  setup: {
    model: string;
    generationConfig: {
      responseModalities: string[];
    };
    systemInstruction?: {
      parts: { text: string }[];
    };
    realtimeInputConfig?: {
      automaticActivityDetection?: AutomaticActivityDetection;
    };
  };
}

// 音声入力送信用の型
export interface GeminiLiveRealtimeInput {
  realtimeInput: {
    mediaChunks: {
      mimeType: string;
      data: string; // Base64 encoded PCM
    }[];
  };
}

// サーバーからのメッセージ型（Union型）
export interface GeminiLiveSetupComplete {
  setupComplete: {
    model: string;
  };
}

export interface GeminiLiveServerContent {
  serverContent: {
    modelTurn?: {
      parts: {
        inlineData?: {
          data: string;
          mimeType: string;
        };
        text?: string;
      }[];
    };
    turnComplete?: boolean;
  };
}

export interface GeminiLiveToolCall {
  toolCall: {
    functionCalls: unknown[];
  };
}

export interface GeminiLiveError {
  error: {
    message: string;
    code?: number;
  };
}

export type GeminiLiveServerMessage =
  | GeminiLiveSetupComplete
  | GeminiLiveServerContent
  | GeminiLiveToolCall
  | GeminiLiveError;

// フック設定用の型
export interface GeminiLiveConfig {
  model?: string;
  systemInstruction?: string;
}

// バックエンドからのトークンレスポンス型
export interface GeminiTokenResponse {
  success: boolean;
  token?: string;
  expireTime?: string;
  error?: string;
}

// 型ガード関数
export function isSetupComplete(msg: GeminiLiveServerMessage): msg is GeminiLiveSetupComplete {
  return "setupComplete" in msg;
}

export function isServerContent(msg: GeminiLiveServerMessage): msg is GeminiLiveServerContent {
  return "serverContent" in msg;
}

export function isToolCall(msg: GeminiLiveServerMessage): msg is GeminiLiveToolCall {
  return "toolCall" in msg;
}

export function isGeminiError(msg: GeminiLiveServerMessage): msg is GeminiLiveError {
  return "error" in msg;
}
