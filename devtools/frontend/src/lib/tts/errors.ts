// TTS フォールバック発生の理由を一意に識別する union。
// useTTS 側ではどの reason でも一律に Web Speech フォールバックへ降格させるが、
// 観測 (テスト・将来のログ) のために原因を保持しておく。
export type TTSFallbackReason =
  | "http_error"
  | "missing_content_type"
  | "invalid_content_type"
  | "empty_body"
  | "audio_error"
  | "network_error"
  | "play_rejected";

interface TTSErrorOptions {
  reason: TTSFallbackReason;
  status?: number;
  statusText?: string;
  cause?: unknown;
}

// VOICEVOX 経路の失敗 (HTTP / Content-Type / Body サイズ / audio.onerror 等) を表す Error。
// types/ 配下は interface/type のみで class を export しない流儀のため、
// class は lib/tts/errors.ts に分離している。
export class TTSError extends Error {
  readonly reason: TTSFallbackReason;
  readonly status?: number;
  readonly statusText?: string;

  constructor(message: string, options: TTSErrorOptions) {
    super(message);
    this.name = "TTSError";
    this.reason = options.reason;
    this.status = options.status;
    this.statusText = options.statusText;
    if (options.cause !== undefined) {
      // Error.cause は ES2022 標準。Node18+/モダンブラウザで利用可能。
      (this as Error & { cause?: unknown }).cause = options.cause;
    }
  }
}
