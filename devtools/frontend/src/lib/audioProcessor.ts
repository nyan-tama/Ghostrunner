// 音声フォーマット変換ユーティリティ
// Gemini Live API 用の音声処理関数群

// 入力音声: 16kHz, 16-bit PCM, モノラル
// 出力音声: 24kHz, 16-bit PCM, モノラル

/**
 * Float32Array をダウンサンプリングする
 * マイク入力（通常44100Hz or 48000Hz）を16kHzに変換
 */
export function downsample(
  buffer: Float32Array,
  inputSampleRate: number,
  outputSampleRate: number
): Float32Array {
  if (inputSampleRate === outputSampleRate) {
    return buffer;
  }

  const ratio = inputSampleRate / outputSampleRate;
  const newLength = Math.floor(buffer.length / ratio);
  const result = new Float32Array(newLength);

  for (let i = 0; i < newLength; i++) {
    const srcIndex = Math.floor(i * ratio);
    result[i] = buffer[srcIndex];
  }

  return result;
}

/**
 * Float32Array (-1.0 ~ 1.0) を Int16Array (-32768 ~ 32767) に変換
 * Gemini Live API は 16-bit PCM を要求する
 */
export function float32ToInt16(float32Array: Float32Array): Int16Array {
  const int16Array = new Int16Array(float32Array.length);

  for (let i = 0; i < float32Array.length; i++) {
    // クリッピング: -1.0 ~ 1.0 の範囲に収める
    const sample = Math.max(-1, Math.min(1, float32Array[i]));
    // -32768 ~ 32767 の範囲に変換
    int16Array[i] = sample < 0 ? sample * 0x8000 : sample * 0x7fff;
  }

  return int16Array;
}

/**
 * Int16Array を Float32Array に変換
 * 音声再生用（AudioContext は Float32 を使用）
 */
export function int16ToFloat32(int16Array: Int16Array): Float32Array {
  const float32Array = new Float32Array(int16Array.length);

  for (let i = 0; i < int16Array.length; i++) {
    // -32768 ~ 32767 を -1.0 ~ 1.0 に変換
    float32Array[i] = int16Array[i] / 0x8000;
  }

  return float32Array;
}

/**
 * ArrayBuffer を Base64 文字列に変換
 * WebSocket で音声データを送信するために使用
 */
export function arrayBufferToBase64(buffer: ArrayBufferLike): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";

  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }

  return btoa(binary);
}

/**
 * Base64 文字列を ArrayBuffer に変換
 * 受信した音声データをデコードするために使用
 */
export function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);

  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }

  return bytes.buffer;
}

/**
 * Base64 PCM データを AudioBuffer に変換
 * 24kHz, 16-bit PCM, モノラルを想定
 */
export function pcmToAudioBuffer(
  audioContext: AudioContext,
  base64Data: string,
  sampleRate: number = 24000
): AudioBuffer {
  const arrayBuffer = base64ToArrayBuffer(base64Data);
  const int16Array = new Int16Array(arrayBuffer);
  const float32Array = int16ToFloat32(int16Array);

  const audioBuffer = audioContext.createBuffer(1, float32Array.length, sampleRate);
  audioBuffer.getChannelData(0).set(float32Array);

  return audioBuffer;
}

// AudioWorklet 関連の型定義
export interface AudioWorkletMessage {
  type: "audio";
  audio: Float32Array;
}

export interface AudioWorkletInitMessage {
  type: "init";
  sampleRate: number;
}
