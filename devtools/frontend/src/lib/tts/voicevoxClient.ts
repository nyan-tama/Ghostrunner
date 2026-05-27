import { TTSError } from "@/lib/tts/errors";

// バックエンド (devtools backend :8888) への TTS リクエストクライアント。
// dev では NEXT_PUBLIC_API_BASE 未設定 → 相対 URL → Next.js rewrites で 8888 に転送。
// 本番では NEXT_PUBLIC_API_BASE を直接叩く (既存 chatApi.ts / constants.ts と同流儀)。
const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "";

interface RequestTTSParams {
  text: string;
  signal?: AbortSignal;
}

// `/api/tts` に POST し、音声 Blob を返す。
// 失敗時 (HTTP エラー / Content-Type 不正 / 空 Body) は TTSError を throw する。
// AbortSignal による abort 時は fetch がそのまま AbortError を投げるため、
// この関数は AbortError を捕捉せずそのまま伝播させる (呼び出し側で `error.name === "AbortError"` で判定)。
export async function requestTTS(params: RequestTTSParams): Promise<Blob> {
  const { text, signal } = params;

  let response: Response;
  try {
    response = await fetch(`${API_BASE}/api/tts`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ text }),
      signal,
    });
  } catch (err) {
    // AbortError は意図的キャンセル。そのまま投げ直して useTTS 側で判定させる。
    if (err instanceof Error && err.name === "AbortError") {
      throw err;
    }
    throw new TTSError("TTS リクエストの送信に失敗しました", {
      reason: "network_error",
      cause: err,
    });
  }

  if (!response.ok) {
    throw new TTSError(
      `TTS リクエストが失敗しました (HTTP ${response.status})`,
      {
        reason: "http_error",
        status: response.status,
        statusText: response.statusText,
      }
    );
  }

  // Content-Type の大文字小文字差を吸収。ヘッダ欠落も明示的に弾く。
  const contentType = response.headers.get("Content-Type");
  if (!contentType) {
    throw new TTSError("レスポンスに Content-Type ヘッダがありません", {
      reason: "missing_content_type",
    });
  }
  if (!contentType.toLowerCase().startsWith("audio/")) {
    throw new TTSError(
      `想定外の Content-Type: ${contentType}`,
      {
        reason: "invalid_content_type",
      }
    );
  }

  const blob = await response.blob();
  if (blob.size === 0) {
    throw new TTSError("レスポンスボディが空です", {
      reason: "empty_body",
    });
  }

  return blob;
}
