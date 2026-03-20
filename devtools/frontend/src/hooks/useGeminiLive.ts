import { useState, useRef, useCallback, useEffect } from "react";
import type {
  ConnectionStatus,
  GeminiLiveConfig,
  GeminiLiveServerMessage,
  GeminiLiveSetupMessage,
} from "@/types/gemini";
import {
  isSetupComplete,
  isServerContent,
  isGeminiError,
} from "@/types/gemini";
import { fetchGeminiToken } from "@/lib/api";
import {
  float32ToInt16,
  arrayBufferToBase64,
  pcmToAudioBuffer,
} from "@/lib/audioProcessor";

// Gemini Live API の定数
// エフェメラルトークンを使用する場合は v1alpha と BidiGenerateContentConstrained を使用
const GEMINI_LIVE_WS_URL =
  "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContentConstrained";
const DEFAULT_MODEL = "models/gemini-2.5-flash-native-audio-preview-12-2025";
// Gemini Live API の音声フォーマット要件: 入力16kHz、出力24kHz
const INPUT_SAMPLE_RATE = 16000;
const OUTPUT_SAMPLE_RATE = 24000;

interface UseGeminiLiveReturn {
  connectionStatus: ConnectionStatus;
  isRecording: boolean;
  error: string | null;
  connect: () => Promise<void>;
  disconnect: () => void;
  startRecording: () => Promise<void>;
  stopRecording: () => void;
}

export function useGeminiLive(config?: GeminiLiveConfig): UseGeminiLiveReturn {
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("disconnected");
  const [isRecording, setIsRecording] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Refs for WebSocket and audio resources
  const wsRef = useRef<WebSocket | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const workletNodeRef = useRef<AudioWorkletNode | null>(null);
  const sourceNodeRef = useRef<MediaStreamAudioSourceNode | null>(null);
  const inputAudioContextRef = useRef<AudioContext | null>(null);

  // 音声出力用キューとフラグ
  const audioQueueRef = useRef<AudioBuffer[]>([]);
  const isPlayingRef = useRef(false);

  // 再帰呼び出し用のref
  const playNextAudioRef = useRef<() => void>(() => {});

  // 設定
  const model = config?.model || DEFAULT_MODEL;
  const systemInstruction = config?.systemInstruction;

  // 再帰呼び出し用の関数をrefに保存
  useEffect(() => {
    playNextAudioRef.current = () => {
      if (isPlayingRef.current) return;
      if (audioQueueRef.current.length === 0) return;

      const audioContext = audioContextRef.current;
      if (!audioContext || audioContext.state === "closed") return;

      isPlayingRef.current = true;
      const audioBuffer = audioQueueRef.current.shift();

      if (!audioBuffer) {
        isPlayingRef.current = false;
        return;
      }

      const source = audioContext.createBufferSource();
      source.buffer = audioBuffer;
      source.connect(audioContext.destination);

      source.onended = () => {
        isPlayingRef.current = false;
        // 再帰呼び出し
        playNextAudioRef.current();
      };

      source.start();
    };
  }, []);

  /**
   * キューから順次音声を再生
   */
  const playNextAudio = useCallback(() => {
    playNextAudioRef.current();
  }, []);

  /**
   * サーバーからのメッセージを処理
   */
  const handleServerMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const message: GeminiLiveServerMessage = JSON.parse(event.data);

        if (isSetupComplete(message)) {
          setConnectionStatus("connected");
          setError(null);
          return;
        }

        if (isServerContent(message)) {
          const serverContent = message.serverContent;

          // 音声データの処理
          if (serverContent.modelTurn?.parts) {
            for (const part of serverContent.modelTurn.parts) {
              if (part.inlineData?.data && part.inlineData.mimeType.includes("audio")) {
                // AudioContext を遅延作成（入力用 AudioContext と競合しないように）
                if (!audioContextRef.current || audioContextRef.current.state === "closed") {
                  audioContextRef.current = new AudioContext();
                }
                const audioContext = audioContextRef.current;
                const audioBuffer = pcmToAudioBuffer(
                  audioContext,
                  part.inlineData.data,
                  OUTPUT_SAMPLE_RATE
                );
                audioQueueRef.current.push(audioBuffer);
                playNextAudio();
              }
            }
          }
          return;
        }

        if (isGeminiError(message)) {
          setError(`Gemini API error: ${message.error.message}`);
          setConnectionStatus("error");
          return;
        }
      } catch {
        // JSON パースエラーは無視（バイナリメッセージの可能性）
      }
    },
    [playNextAudio]
  );

  /**
   * 内部用: 録音停止処理
   */
  const stopRecordingInternal = useCallback(() => {
    // WorkletNode を切断
    if (workletNodeRef.current) {
      workletNodeRef.current.disconnect();
      workletNodeRef.current = null;
    }

    // SourceNode を切断
    if (sourceNodeRef.current) {
      sourceNodeRef.current.disconnect();
      sourceNodeRef.current = null;
    }

    // 入力用 AudioContext を閉じる
    if (inputAudioContextRef.current && inputAudioContextRef.current.state !== "closed") {
      inputAudioContextRef.current.close();
      inputAudioContextRef.current = null;
    }

    // MediaStream を停止
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }

    setIsRecording(false);
  }, []);

  /**
   * WebSocket に接続
   */
  const connect = useCallback(async () => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    setConnectionStatus("connecting");
    setError(null);

    try {
      // エフェメラルトークンを取得
      const token = await fetchGeminiToken();

      // WebSocket 接続を確立（エフェメラルトークンは access_token パラメータで渡す）
      const wsUrl = `${GEMINI_LIVE_WS_URL}?access_token=${encodeURIComponent(token)}`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      // AudioContext は音声再生時に遅延作成する（入力用 AudioContext と競合しないように）

      ws.onopen = () => {
        // setup メッセージを送信（VAD設定を含む）
        const setupMessage: GeminiLiveSetupMessage = {
          setup: {
            model,
            generationConfig: {
              responseModalities: ["AUDIO"],
            },
            // VAD（Voice Activity Detection）設定
            realtimeInputConfig: {
              automaticActivityDetection: {
                disabled: false,
                startOfSpeechSensitivity: "START_SENSITIVITY_HIGH",
                endOfSpeechSensitivity: "END_SENSITIVITY_HIGH",
                silenceDurationMs: 500,
              },
            },
          },
        };

        if (systemInstruction) {
          setupMessage.setup.systemInstruction = {
            parts: [{ text: systemInstruction }],
          };
        }

        console.log("[GeminiLive] Sending setup message:", JSON.stringify(setupMessage));
        ws.send(JSON.stringify(setupMessage));
      };

      ws.onmessage = async (event) => {
        // Blob の場合はテキストに変換
        if (event.data instanceof Blob) {
          const text = await event.data.text();
          console.log("[GeminiLive] Received:", text.substring(0, 100));
          const textEvent = new MessageEvent("message", { data: text });
          handleServerMessage(textEvent);
        } else {
          console.log("[GeminiLive] Received:", String(event.data).substring(0, 100));
          handleServerMessage(event);
        }
      };

      ws.onerror = () => {
        setError("Failed to connect to Gemini Live API");
        setConnectionStatus("error");
      };

      ws.onclose = () => {
        setConnectionStatus((prev) => (prev === "error" ? prev : "disconnected"));
        wsRef.current = null;
      };
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Failed to get ephemeral token";
      setError(errorMessage);
      setConnectionStatus("error");
    }
  }, [model, systemInstruction, handleServerMessage]);

  /**
   * WebSocket を切断し、リソースを解放
   */
  const disconnect = useCallback(() => {
    // 録音を停止
    stopRecordingInternal();

    // WebSocket を閉じる
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    // 出力用 AudioContext を閉じる
    if (audioContextRef.current && audioContextRef.current.state !== "closed") {
      audioContextRef.current.close();
      audioContextRef.current = null;
    }

    // キューをクリア
    audioQueueRef.current = [];
    isPlayingRef.current = false;

    setConnectionStatus("disconnected");
    setError(null);
  }, [stopRecordingInternal]);

  /**
   * マイク入力を開始
   */
  const startRecording = useCallback(async () => {
    console.log("[GeminiLive] startRecording called");
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      setError("WebSocket is not connected");
      return;
    }

    if (isRecording) {
      return;
    }

    try {
      // マイクへのアクセスを取得
      console.log("[GeminiLive] Requesting microphone access...");
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
        },
      });
      streamRef.current = stream;

      // MediaStream の実際のサンプルレートを取得
      const audioTrack = stream.getAudioTracks()[0];
      const trackSettings = audioTrack.getSettings();
      console.log("[GeminiLive] Audio track settings:", trackSettings);

      // 入力用 AudioContext を作成（Gemini要件: 入力は16kHz）
      const inputAudioContext = new AudioContext({ sampleRate: INPUT_SAMPLE_RATE });
      inputAudioContextRef.current = inputAudioContext;

      // AudioContext が suspended の場合は resume
      if (inputAudioContext.state === "suspended") {
        console.log("[GeminiLive] AudioContext is suspended, resuming...");
        await inputAudioContext.resume();
      }
      console.log("[GeminiLive] AudioContext state:", inputAudioContext.state, "sampleRate:", inputAudioContext.sampleRate);

      // ソースノードを作成
      const source = inputAudioContext.createMediaStreamSource(stream);
      sourceNodeRef.current = source;

      // ScriptProcessorNode を使用（Google公式サンプルと同じ bufferSize: 1024）
      const bufferSize = 1024;
      const scriptProcessor = inputAudioContext.createScriptProcessor(bufferSize, 1, 1);
      let audioChunkCount = 0;

      scriptProcessor.onaudioprocess = (audioEvent) => {
        if (wsRef.current?.readyState !== WebSocket.OPEN) return;

        const inputData = audioEvent.inputBuffer.getChannelData(0);
        // Float32 を Int16 PCM に変換（Google公式サンプルと同じ処理）
        const int16Data = float32ToInt16(new Float32Array(inputData));

        // 音声データの最大値を確認（デバッグ用）
        if (audioChunkCount % 50 === 0) {
          const maxVal = Math.max(...Array.from(inputData).map(Math.abs));
          console.log(`[GeminiLive] Audio max amplitude: ${maxVal.toFixed(6)}`);
        }

        // Base64 エンコード
        const base64Data = arrayBufferToBase64(int16Data.buffer);

        // Gemini に送信（realtimeInput.audio 形式）
        const audioMessage = {
          realtimeInput: {
            audio: {
              data: base64Data,
              mimeType: "audio/pcm;rate=16000",
            },
          },
        };

        wsRef.current.send(JSON.stringify(audioMessage));
        audioChunkCount++;
        if (audioChunkCount % 50 === 1) {
          console.log(`[GeminiLive] Sent audio chunk #${audioChunkCount}, size: ${base64Data.length}`);
        }
      };

      // ノードを接続
      source.connect(scriptProcessor);
      scriptProcessor.connect(inputAudioContext.destination);

      console.log("[GeminiLive] Recording started, sampleRate:", inputAudioContext.sampleRate);
      setIsRecording(true);
      setError(null);
    } catch (err) {
      if (err instanceof Error && err.name === "NotAllowedError") {
        setError("Microphone permission denied");
      } else {
        setError(`Failed to start recording: ${err instanceof Error ? err.message : "Unknown error"}`);
      }
    }
  }, [isRecording]);

  /**
   * マイク入力を停止
   */
  const stopRecording = useCallback(() => {
    stopRecordingInternal();
  }, [stopRecordingInternal]);

  // コンポーネントのアンマウント時にリソースを解放
  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (audioContextRef.current && audioContextRef.current.state !== "closed") {
        audioContextRef.current.close();
      }
      if (inputAudioContextRef.current && inputAudioContextRef.current.state !== "closed") {
        inputAudioContextRef.current.close();
      }
      if (streamRef.current) {
        streamRef.current.getTracks().forEach((track) => track.stop());
      }
    };
  }, []);

  return {
    connectionStatus,
    isRecording,
    error,
    connect,
    disconnect,
    startRecording,
    stopRecording,
  };
}
