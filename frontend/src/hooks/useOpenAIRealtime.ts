import { useState, useRef, useCallback, useEffect } from "react";
import type {
  OpenAIConnectionStatus,
  OpenAIRealtimeConfig,
  OpenAIServerEvent,
  OpenAIConversationItemCreate,
  OpenAIResponseCreate,
} from "@/types/openai";
import {
  isSessionCreated,
  isSessionUpdated,
  isResponseAudioDelta,
  isOpenAIError,
} from "@/types/openai";
import { fetchOpenAIRealtimeToken } from "@/lib/api";
import {
  float32ToInt16,
  arrayBufferToBase64,
  pcmToAudioBuffer,
} from "@/lib/audioProcessor";

// OpenAI Realtime API GA版の定数
const OPENAI_REALTIME_WS_URL = "wss://api.openai.com/v1/realtime";
const DEFAULT_MODEL = "gpt-realtime";
const DEFAULT_VOICE = "verse";
// OpenAI Realtime API の音声フォーマット要件: 入出力ともに24kHz
const INPUT_SAMPLE_RATE = 24000;
const OUTPUT_SAMPLE_RATE = 24000;

// 再接続の最大試行回数と指数バックオフ
const MAX_RECONNECT_ATTEMPTS = 3;
const INITIAL_RECONNECT_DELAY_MS = 1000;

interface UseOpenAIRealtimeReturn {
  connectionStatus: OpenAIConnectionStatus;
  isRecording: boolean;
  error: string | null;
  connect: () => Promise<void>;
  disconnect: () => void;
  startRecording: () => Promise<void>;
  stopRecording: () => void;
  sendText: (message: string) => void;
}

export function useOpenAIRealtime(config?: OpenAIRealtimeConfig): UseOpenAIRealtimeReturn {
  const [connectionStatus, setConnectionStatus] = useState<OpenAIConnectionStatus>("disconnected");
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

  // 自動再接続用の ref
  const reconnectAttemptRef = useRef(0);
  const shouldReconnectRef = useRef(false);
  const connectFnRef = useRef<(() => Promise<void>) | null>(null);

  // 設定
  const model = config?.model || DEFAULT_MODEL;
  const voice = config?.voice || DEFAULT_VOICE;
  const instructions = config?.instructions;

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
        const message: OpenAIServerEvent = JSON.parse(event.data);

        if (isSessionCreated(message)) {
          // GA版: session.update を送信してセッションを設定
          if (wsRef.current?.readyState === WebSocket.OPEN) {
            const sessionUpdate = {
              type: "session.update",
              session: {
                type: "realtime",
                instructions: instructions || "あなたは親切な音声アシスタントです。日本語で会話してください。フレンドリーで自然な口調で応答してください。",
                audio: {
                  input: {
                    format: { type: "audio/pcm", rate: INPUT_SAMPLE_RATE },
                    turn_detection: {
                      type: "server_vad",
                      threshold: 0.5,
                      prefix_padding_ms: 300,
                      silence_duration_ms: 500,
                    },
                  },
                  output: {
                    format: { type: "audio/pcm", rate: OUTPUT_SAMPLE_RATE },
                    voice: voice,
                  },
                },
              },
            };
            wsRef.current.send(JSON.stringify(sessionUpdate));
          }
          return;
        }

        if (isSessionUpdated(message)) {
          setConnectionStatus("connected");
          setError(null);
          return;
        }

        if (isResponseAudioDelta(message)) {
          // AudioContext を遅延作成（入力用 AudioContext と競合しないように）
          if (!audioContextRef.current || audioContextRef.current.state === "closed") {
            audioContextRef.current = new AudioContext();
          }
          const audioContext = audioContextRef.current;
          const audioBuffer = pcmToAudioBuffer(
            audioContext,
            message.delta,
            OUTPUT_SAMPLE_RATE
          );
          audioQueueRef.current.push(audioBuffer);
          playNextAudio();
          return;
        }

        if (isOpenAIError(message)) {
          setError(`OpenAI Realtime API error: ${message.error.message}`);
          setConnectionStatus("error");
          return;
        }
      } catch {
        // JSON パースエラーは無視（バイナリメッセージの可能性）
      }
    },
    [playNextAudio, instructions, voice]
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

    // 自動再接続を有効化
    shouldReconnectRef.current = true;

    setConnectionStatus("connecting");
    setError(null);

    try {
      // エフェメラルトークンを取得
      const token = await fetchOpenAIRealtimeToken(model, voice);

      // WebSocket 接続を確立（サブプロトコルでトークンを渡す）
      const wsUrl = `${OPENAI_REALTIME_WS_URL}?model=${encodeURIComponent(model)}`;
      const ws = new WebSocket(wsUrl, [
        "realtime",
        `openai-insecure-api-key.${token}`,
      ]);
      wsRef.current = ws;

      ws.onmessage = (event) => {
        handleServerMessage(event);
      };

      ws.onerror = () => {
        setError("Failed to connect to OpenAI Realtime API");
        setConnectionStatus("error");
      };

      ws.onclose = () => {
        wsRef.current = null;

        // 自動再接続: shouldReconnectRef が true かつ再接続試行回数が上限未満
        if (shouldReconnectRef.current && reconnectAttemptRef.current < MAX_RECONNECT_ATTEMPTS) {
          const delay = INITIAL_RECONNECT_DELAY_MS * Math.pow(2, reconnectAttemptRef.current);
          reconnectAttemptRef.current += 1;
          setConnectionStatus("connecting");

          setTimeout(() => {
            if (shouldReconnectRef.current && connectFnRef.current) {
              connectFnRef.current();
            }
          }, delay);
        } else {
          setConnectionStatus((prev) => (prev === "error" ? prev : "disconnected"));
        }
      };

      // 接続成功時に再接続カウンタをリセット
      ws.onopen = () => {
        reconnectAttemptRef.current = 0;
      };
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Failed to get ephemeral token";
      setError(errorMessage);
      setConnectionStatus("error");
    }
  }, [model, voice, handleServerMessage]);

  // connect 関数を ref に保存（自動再接続で使用）
  useEffect(() => {
    connectFnRef.current = connect;
  }, [connect]);

  /**
   * WebSocket を切断し、リソースを解放
   */
  const disconnect = useCallback(() => {
    // 意図的な切断なので自動再接続を無効化
    shouldReconnectRef.current = false;
    reconnectAttemptRef.current = 0;

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
          echoCancellation: true,
          noiseSuppression: true,
        },
      });
      streamRef.current = stream;

      // 入力用 AudioContext を作成（デフォルトサンプルレートを使用）
      const inputAudioContext = new AudioContext();
      inputAudioContextRef.current = inputAudioContext;
      const nativeSampleRate = inputAudioContext.sampleRate;

      // AudioContext が suspended の場合は resume
      if (inputAudioContext.state === "suspended") {
        await inputAudioContext.resume();
      }

      // ソースノードを作成
      const source = inputAudioContext.createMediaStreamSource(stream);
      sourceNodeRef.current = source;

      // ScriptProcessorNode を使用
      const bufferSize = 4096;
      const scriptProcessor = inputAudioContext.createScriptProcessor(bufferSize, 1, 1);

      // リサンプリング用の関数（ネイティブ -> 24kHz）
      const resampleTo24k = (inputData: Float32Array, fromRate: number): Float32Array => {
        const ratio = fromRate / INPUT_SAMPLE_RATE;
        const newLength = Math.floor(inputData.length / ratio);
        const result = new Float32Array(newLength);
        for (let i = 0; i < newLength; i++) {
          const srcIndex = Math.floor(i * ratio);
          result[i] = inputData[srcIndex];
        }
        return result;
      };

      scriptProcessor.onaudioprocess = (audioEvent) => {
        if (wsRef.current?.readyState !== WebSocket.OPEN) return;

        const inputData = audioEvent.inputBuffer.getChannelData(0);

        // リサンプリング（ネイティブ -> 24kHz）
        const resampledData = nativeSampleRate !== INPUT_SAMPLE_RATE
          ? resampleTo24k(inputData, nativeSampleRate)
          : new Float32Array(inputData);

        const int16Data = float32ToInt16(resampledData);
        const base64Data = arrayBufferToBase64(int16Data.buffer);

        wsRef.current.send(JSON.stringify({
          type: "input_audio_buffer.append",
          audio: base64Data,
        }));
      };

      // ノードを接続
      source.connect(scriptProcessor);
      scriptProcessor.connect(inputAudioContext.destination);

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

  /**
   * テキストメッセージを送信し、AIに音声で応答させる
   */
  const sendText = useCallback((message: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      return;
    }

    // conversation.item.create でユーザーメッセージを追加
    const conversationItem: OpenAIConversationItemCreate = {
      type: "conversation.item.create",
      item: {
        type: "message",
        role: "user",
        content: [{ type: "input_text", text: message }],
      },
    };
    wsRef.current.send(JSON.stringify(conversationItem));

    // response.create でAIの音声応答を要求
    const responseCreate: OpenAIResponseCreate = {
      type: "response.create",
    };
    wsRef.current.send(JSON.stringify(responseCreate));
  }, []);

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
    sendText,
  };
}
