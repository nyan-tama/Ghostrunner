"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { LOCAL_STORAGE_TTS_ENABLED_KEY } from "@/lib/constants";

interface UseTTSReturn {
  speak: (text: string) => void;
  cancel: () => void;
  enabled: boolean;
  setEnabled: (v: boolean) => void;
  isSpeaking: boolean;
  error: string | null;
  // iOS Safari の autoplay 制約対策。ユーザージェスチャ（タップ/トグル）の
  // 同期コンテキストで呼ぶと、Web Speech のパイプが開き、後続の非同期 speak()
  // が握りつぶされなくなる。送信ボタン・ショートカット・TTS ON 切替などから呼ぶ。
  prime: () => void;
}

export function useTTS(): UseTTSReturn {
  const [enabled, setEnabledState] = useState(false);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const voiceRef = useRef<SpeechSynthesisVoice | null>(null);
  const speakTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // localStorage から復元（SSR セーフ）
  useEffect(() => {
    const stored = localStorage.getItem(LOCAL_STORAGE_TTS_ENABLED_KEY);
    if (stored === "true") {
      setEnabledState(true);
    }
  }, []);

  // voice 選択: getVoices + voiceschanged イベント購読
  useEffect(() => {
    if (typeof window === "undefined" || !window.speechSynthesis) {
      return;
    }

    function selectJapaneseVoice() {
      const voices = window.speechSynthesis.getVoices();
      const jaVoice = voices.find((v) => v.lang.startsWith("ja"));
      voiceRef.current = jaVoice ?? null;
    }

    selectJapaneseVoice();
    window.speechSynthesis.addEventListener("voiceschanged", selectJapaneseVoice);
    return () => {
      window.speechSynthesis.removeEventListener(
        "voiceschanged",
        selectJapaneseVoice
      );
    };
  }, []);

  const cancel = useCallback(() => {
    if (speakTimeoutRef.current) {
      clearTimeout(speakTimeoutRef.current);
      speakTimeoutRef.current = null;
    }
    if (typeof window !== "undefined" && window.speechSynthesis) {
      window.speechSynthesis.cancel();
    }
    setIsSpeaking(false);
  }, []);

  const speak = useCallback(
    (text: string) => {
      if (typeof window === "undefined" || !window.speechSynthesis) {
        setError("音声合成に対応していないブラウザです");
        return;
      }
      if (!enabled) return;

      setError(null);

      // iOS Safari の cancel -> speak 不発バグ対策: cancel 後に 50ms 待つ
      cancel();

      speakTimeoutRef.current = setTimeout(() => {
        speakTimeoutRef.current = null;
        const utterance = new SpeechSynthesisUtterance(text);

        if (voiceRef.current) {
          utterance.voice = voiceRef.current;
        } else {
          utterance.lang = "ja-JP";
        }

        utterance.onstart = () => setIsSpeaking(true);
        utterance.onend = () => setIsSpeaking(false);
        utterance.onerror = (ev) => {
          setIsSpeaking(false);
          // "interrupted" は cancel による正常停止なので無視
          if (ev.error !== "interrupted") {
            setError(`音声合成エラー: ${ev.error}`);
          }
        };

        window.speechSynthesis.speak(utterance);
      }, 50);
    },
    [enabled, cancel]
  );

  // iOS Safari 用 unlock。ユーザージェスチャの同期スコープで無音 utterance を speak し、
  // 以降の非同期 speak() が握りつぶされないようにパイプを開く。
  // 既に再生中の場合は不要なので何もしない。
  const prime = useCallback(() => {
    if (typeof window === "undefined" || !window.speechSynthesis) return;
    if (window.speechSynthesis.speaking || window.speechSynthesis.pending) return;
    try {
      const u = new SpeechSynthesisUtterance(" ");
      u.volume = 0;
      u.rate = 10; // 最速で吐かせて即終了させる
      if (voiceRef.current) {
        u.voice = voiceRef.current;
      } else {
        u.lang = "ja-JP";
      }
      window.speechSynthesis.speak(u);
    } catch {
      // unlock 試行の失敗は黙って無視（後続の speak で再試行される）
    }
  }, []);

  const setEnabled = useCallback(
    (v: boolean) => {
      setEnabledState(v);
      if (typeof window !== "undefined") {
        localStorage.setItem(LOCAL_STORAGE_TTS_ENABLED_KEY, String(v));
      }
      if (v) {
        // ON 切替自体がユーザージェスチャなので、ここで unlock しておく
        prime();
      } else if (typeof window !== "undefined" && window.speechSynthesis) {
        window.speechSynthesis.cancel();
        setIsSpeaking(false);
      }
    },
    [prime]
  );

  // クリーンアップ
  useEffect(() => {
    return () => {
      if (speakTimeoutRef.current) {
        clearTimeout(speakTimeoutRef.current);
      }
      if (typeof window !== "undefined" && window.speechSynthesis) {
        window.speechSynthesis.cancel();
      }
    };
  }, []);

  return { speak, cancel, enabled, setEnabled, isSpeaking, error, prime };
}
