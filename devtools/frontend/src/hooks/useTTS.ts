"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { LOCAL_STORAGE_TTS_ENABLED_KEY } from "@/lib/constants";
import { requestTTS } from "@/lib/tts/voicevoxClient";
import { TTSError, type TTSFallbackReason } from "@/lib/tts/errors";
import {
  cancelWebSpeech,
  primeWebSpeech,
  speakWithWebSpeech,
} from "@/lib/tts/webSpeech";
import { SILENT_MP3_DATA_URL } from "@/lib/tts/silentMp3";

interface UseTTSReturn {
  speak: (text: string) => void;
  cancel: () => void;
  enabled: boolean;
  setEnabled: (v: boolean) => void;
  isSpeaking: boolean;
  error: string | null;
  // iOS Safari の autoplay 制約対策。ユーザージェスチャ（タップ/トグル）の
  // 同期コンテキストで呼ぶと、<audio> 要素の autoplay unlock が完了し、
  // 後続の非同期 audio.play() が UA に握りつぶされなくなる。
  // 送信ボタン・ショートカット・TTS ON 切替などから呼ぶ。
  prime: () => void;
}

// ネットワーク/サーバー側の問題 (reason 1-5)
const NETWORK_ERROR_MESSAGE =
  "VOICEVOX 接続失敗。Web Speech に降格しました";
// 再生クライアント側の問題 (reason 6-7: audio_error, play_rejected)
const AUDIO_ERROR_MESSAGE =
  "音声再生失敗。Web Speech に降格しました";

// reason からエラーメッセージを決定する。
// ネットワーク/サーバー側(1-5)と再生クライアント側(6-7)で原因が異なるため分割。
function fallbackMessageForReason(reason: TTSFallbackReason): string {
  if (reason === "audio_error" || reason === "play_rejected") {
    return AUDIO_ERROR_MESSAGE;
  }
  return NETWORK_ERROR_MESSAGE;
}

export function useTTS(): UseTTSReturn {
  const [enabled, setEnabledState] = useState(false);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // VOICEVOX 経路で使う <audio> 要素 (iOS Safari の unlock を維持するため 1 つを使い回す)。
  const audioRef = useRef<HTMLAudioElement | null>(null);
  // 進行中の /api/tts fetch をキャンセルするための AbortController。
  const abortRef = useRef<AbortController | null>(null);
  // 現在 audio.src に紐付いている Blob URL。次の speak / cancel / unmount で revoke する。
  const currentObjectUrlRef = useRef<string | null>(null);
  // 現在 Web Speech フォールバック中かを記録するフラグ。cancel() の挙動分岐に使う。
  const isFallbackActiveRef = useRef<boolean>(false);
  // prime() の in-flight ガード。連続タップでの重複発火防止 (冪等性、S-7)。
  const primeInFlightRef = useRef<boolean>(false);

  // <audio> 要素の初期化 (マウント時に 1 度だけ)。
  useEffect(() => {
    if (typeof window === "undefined") return;
    const audio = new Audio();
    // playsInline は TypeScript の HTMLAudioElement 型に存在しないが、
    // iOS Safari では <audio> でも有効な属性として認識される (UA 差吸収のための保険)。
    audio.setAttribute("playsinline", "");
    audio.preload = "auto";
    audioRef.current = audio;
  }, []);

  // localStorage から enabled を復元 (SSR セーフ)。
  useEffect(() => {
    if (typeof window === "undefined") return;
    const stored = localStorage.getItem(LOCAL_STORAGE_TTS_ENABLED_KEY);
    if (stored === "true") {
      setEnabledState(true);
    }
  }, []);

  // 現在の Blob URL を revoke してリファレンスをクリアする。
  const revokeCurrentObjectUrl = useCallback(() => {
    if (currentObjectUrlRef.current) {
      URL.revokeObjectURL(currentObjectUrlRef.current);
      currentObjectUrlRef.current = null;
    }
  }, []);

  // 進行中の再生 / fetch / フォールバックを停止する。
  // setEnabled(false) / cancel() / speak() 冒頭 / unmount から呼ばれる。
  const stopAll = useCallback(() => {
    // 1. 進行中の fetch を abort
    if (abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    // 2. audio を停止し src を切り離す。前回 speak() で設定した onXxx ハンドラ
    //    (handlePlaying / handleEnded / handleError) が残っていると、後続の
    //    prime() や speak() で誤発火する恐れがあるためここで null 化する。
    const audio = audioRef.current;
    if (audio) {
      audio.pause();
      audio.removeAttribute("src");
      audio.onplaying = null;
      audio.onended = null;
      audio.onerror = null;
    }
    // 3. Blob URL を解放
    revokeCurrentObjectUrl();
    // 4. フォールバック中なら Web Speech も止める
    if (isFallbackActiveRef.current) {
      cancelWebSpeech();
      isFallbackActiveRef.current = false;
    }
    setIsSpeaking(false);
  }, [revokeCurrentObjectUrl]);

  // Web Speech フォールバックを発火する。
  // 呼び出し前に stopAll などで前段の再生/フェッチを片付けておくこと。
  const triggerFallback = useCallback(
    (reason: TTSFallbackReason, text: string) => {
      setError(fallbackMessageForReason(reason));
      isFallbackActiveRef.current = true;
      speakWithWebSpeech(text, {
        onStart: () => setIsSpeaking(true),
        onEnd: () => setIsSpeaking(false),
        onError: () => setIsSpeaking(false),
      });
    },
    []
  );

  const speak = useCallback(
    (text: string) => {
      if (!enabled) return;
      if (typeof window === "undefined") return;

      // 既存再生 / 進行中 fetch / フォールバックを片付ける。
      stopAll();

      const audio = audioRef.current;
      if (!audio) {
        // <audio> 初期化前に呼ばれた稀ケース: Web Speech にフォールバックする。
        triggerFallback("audio_error", text);
        return;
      }

      const controller = new AbortController();
      abortRef.current = controller;

      // VOICEVOX から Blob を取得して再生する。失敗時は Web Speech 降格。
      void (async () => {
        let blob: Blob;
        try {
          blob = await requestTTS({ text, signal: controller.signal });
        } catch (err) {
          // 意図的な abort はフォールバックしない。
          if (err instanceof Error && err.name === "AbortError") {
            return;
          }
          // TTSError なら reason を取り出す、それ以外は network_error 扱い。
          const reason: TTSFallbackReason =
            err instanceof TTSError ? err.reason : "network_error";
          triggerFallback(reason, text);
          return;
        }

        // fetch 完了後に setEnabled(false) / cancel() が来ていれば、
        // controller が abort 済または abortRef が差し替わっている。
        if (controller.signal.aborted || abortRef.current !== controller) {
          return;
        }

        const objectUrl = URL.createObjectURL(blob);
        currentObjectUrlRef.current = objectUrl;

        // play 前に必ず pause する (前 src との衝突回避: UA 差対策)。
        audio.pause();
        audio.src = objectUrl;

        // playing で isSpeaking=true + error クリア (play メソッドではなく実再生開始)。
        const handlePlaying = () => {
          setIsSpeaking(true);
          setError(null);
        };
        const handleEnded = () => {
          setIsSpeaking(false);
          revokeCurrentObjectUrl();
        };
        const handleError = () => {
          setIsSpeaking(false);
          revokeCurrentObjectUrl();
          triggerFallback("audio_error", text);
        };
        // 同一 audio 要素を使い回すため、前回のハンドラを onXxx で上書きする
        // (addEventListener だと多重登録のリスクがある)。
        audio.onplaying = handlePlaying;
        audio.onended = handleEnded;
        audio.onerror = handleError;

        isFallbackActiveRef.current = false;

        try {
          await audio.play();
        } catch {
          // play 拒否 (autoplay block 等) もフォールバック対象。
          // revoke 後に audio.src が無効 Blob URL のまま残ると、後続で
          // audio.onerror が発火し triggerFallback が二重発火する恐れがある。
          // ハンドラを null 化し src も外してから revoke / フォールバックする。
          audio.onerror = null;
          audio.removeAttribute("src");
          revokeCurrentObjectUrl();
          triggerFallback("play_rejected", text);
        }
      })();
    },
    [enabled, stopAll, triggerFallback, revokeCurrentObjectUrl]
  );

  const cancel = useCallback(() => {
    stopAll();
  }, [stopAll]);

  // iOS Safari 用 unlock。ユーザージェスチャの同期スコープで呼ぶこと。
  // Web Speech 側 (フォールバック用) と <audio> 要素の両方を同時に prime する。
  const prime = useCallback(() => {
    if (typeof window === "undefined") return;
    // 再生中または prime 処理が進行中なら何もしない (冪等性ガード)。
    if (isSpeaking) return;
    if (primeInFlightRef.current) return;
    primeInFlightRef.current = true;

    // 同期スコープ内で先に Web Speech を prime (同期 API、ユーザージェスチャを消費しない)。
    primeWebSpeech();

    const audio = audioRef.current;
    if (!audio) {
      primeInFlightRef.current = false;
      return;
    }

    // 直前の speak() で設定された onXxx ハンドラが残っていると、
    // SILENT_MP3 のデコード失敗時に空文字 text の handleError が
    // triggerFallback を呼ぶ事故が起き得る。冒頭で必ず null 化する。
    audio.onplaying = null;
    audio.onended = null;
    audio.onerror = null;
    // 残留 Blob URL があれば先に revoke する (audio.src を上書きする前に解放)。
    revokeCurrentObjectUrl();

    // 同期スコープ内で連続実行。await を挟まない。
    audio.muted = true;
    audio.src = SILENT_MP3_DATA_URL;
    const playPromise = audio.play();
    if (playPromise && typeof playPromise.then === "function") {
      playPromise
        .then(() => {
          audio.pause();
          audio.currentTime = 0;
          audio.muted = false;
        })
        .catch(() => {
          audio.muted = false;
        })
        .finally(() => {
          primeInFlightRef.current = false;
        });
    } else {
      primeInFlightRef.current = false;
    }
  }, [isSpeaking, revokeCurrentObjectUrl]);

  const setEnabled = useCallback(
    (v: boolean) => {
      setEnabledState(v);
      if (typeof window !== "undefined") {
        localStorage.setItem(LOCAL_STORAGE_TTS_ENABLED_KEY, String(v));
      }
      if (v) {
        // ON 切替自体がユーザージェスチャなので、ここで unlock しておく。
        prime();
      } else {
        // OFF にしたら進行中の再生・fetch・フォールバックを全て止める。
        stopAll();
      }
    },
    [prime, stopAll]
  );

  // unmount 時の cleanup。順序は (1) abort → (2) audio.pause() →
  // (3) audio.removeAttribute("src") → (4) URL.revokeObjectURL の固定順序。
  // 逆順だと MEDIA_ERR_SRC_NOT_SUPPORTED 発火リスク。
  useEffect(() => {
    return () => {
      if (abortRef.current) {
        abortRef.current.abort();
        abortRef.current = null;
      }
      const audio = audioRef.current;
      if (audio) {
        audio.pause();
        audio.removeAttribute("src");
        audio.onplaying = null;
        audio.onended = null;
        audio.onerror = null;
      }
      if (currentObjectUrlRef.current) {
        URL.revokeObjectURL(currentObjectUrlRef.current);
        currentObjectUrlRef.current = null;
      }
      if (isFallbackActiveRef.current) {
        cancelWebSpeech();
        isFallbackActiveRef.current = false;
      }
    };
  }, []);

  return { speak, cancel, enabled, setEnabled, isSpeaking, error, prime };
}
