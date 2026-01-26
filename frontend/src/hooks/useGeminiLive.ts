import { useState, useRef, useCallback, useEffect } from "react";
import type {
  ConnectionStatus,
  GeminiLiveConfig,
  GeminiLiveServerMessage,
  GeminiLiveSetupMessage,
  GeminiLiveRealtimeInput,
} from "@/types/gemini";
import {
  isSetupComplete,
  isServerContent,
  isGeminiError,
} from "@/types/gemini";
import { fetchGeminiToken } from "@/lib/api";
import {
  downsample,
  float32ToInt16,
  arrayBufferToBase64,
  pcmToAudioBuffer,
} from "@/lib/audioProcessor";

// Gemini Live API の定数
const GEMINI_LIVE_WS_URL =
  "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent";
const DEFAULT_MODEL = "models/gemini-2.0-flash-live-001";
const INPUT_SAMPLE_RATE = 16000; // Gemini が要求する入力サンプルレート
const OUTPUT_SAMPLE_RATE = 24000; // Gemini が出力するサンプルレート
const AUDIO_CHUNK_SIZE = 4096; // AudioWorklet から送信するチャンクサイズ

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
                const audioContext = audioContextRef.current;
                if (audioContext && audioContext.state !== "closed") {
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

      // WebSocket 接続を確立
      const wsUrl = `${GEMINI_LIVE_WS_URL}?key=${token}`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      // AudioContext を作成（音声再生用）
      audioContextRef.current = new AudioContext({ sampleRate: OUTPUT_SAMPLE_RATE });

      ws.onopen = () => {
        // setup メッセージを送信
        const setupMessage: GeminiLiveSetupMessage = {
          setup: {
            model,
            generationConfig: {
              responseModalities: ["AUDIO"],
            },
          },
        };

        if (systemInstruction) {
          setupMessage.setup.systemInstruction = {
            parts: [{ text: systemInstruction }],
          };
        }

        ws.send(JSON.stringify(setupMessage));
      };

      ws.onmessage = handleServerMessage;

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
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      setError("WebSocket is not connected");
      return;
    }

    if (isRecording) {
      return;
    }

    try {
      // マイクへのアクセスを取得
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          sampleRate: { ideal: 48000 },
          channelCount: { exact: 1 },
          echoCancellation: true,
          noiseSuppression: true,
        },
      });
      streamRef.current = stream;

      // 入力用 AudioContext を作成
      const inputAudioContext = new AudioContext();
      inputAudioContextRef.current = inputAudioContext;

      // AudioWorklet を登録
      await inputAudioContext.audioWorklet.addModule("/audio-worklet-processor.js");

      // ソースノードを作成
      const source = inputAudioContext.createMediaStreamSource(stream);
      sourceNodeRef.current = source;

      // WorkletNode を作成
      const workletNode = new AudioWorkletNode(inputAudioContext, "audio-processor", {
        processorOptions: {
          chunkSize: AUDIO_CHUNK_SIZE,
        },
      });
      workletNodeRef.current = workletNode;

      // WorkletNode からのメッセージを処理
      workletNode.port.onmessage = (event) => {
        if (event.data.type === "audio" && wsRef.current?.readyState === WebSocket.OPEN) {
          const float32Data = event.data.audio as Float32Array;

          // ダウンサンプリング
          const downsampled = downsample(
            float32Data,
            inputAudioContext.sampleRate,
            INPUT_SAMPLE_RATE
          );

          // Int16 PCM に変換
          const int16Data = float32ToInt16(downsampled);

          // Base64 エンコード
          const base64Data = arrayBufferToBase64(int16Data.buffer);

          // Gemini に送信
          const realtimeInput: GeminiLiveRealtimeInput = {
            realtimeInput: {
              mediaChunks: [
                {
                  mimeType: "audio/pcm;rate=16000",
                  data: base64Data,
                },
              ],
            },
          };

          wsRef.current.send(JSON.stringify(realtimeInput));
        }
      };

      // ノードを接続
      source.connect(workletNode);
      workletNode.connect(inputAudioContext.destination);

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
