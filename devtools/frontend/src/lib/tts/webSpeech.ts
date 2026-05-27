// Web Speech API を使った TTS フォールバック経路。
// VOICEVOX 経路が失敗した時のみ useTTS から呼ばれる。
//
// 既存 useTTS.ts (コミット 4d98554) の Web Speech 実装ロジックを純関数群に移植している:
//   - voiceschanged イベントで ja-JP の voice を選択
//   - speak の前に必ず cancel → 50ms 待機 (iOS Safari の cancel→speak 不発バグ対策)
//   - prime() の無音 utterance パターン (volume=0, rate=10, " ")
//
// 内部状態 (選択された voice、setTimeout ハンドル、voiceschanged リスナー) は
// モジュールスコープのクロージャで保持する。useTTS hook の責務には含めない。
// テスト時は vi.resetModules() で初期化する。

interface SpeakCallbacks {
  onStart?: () => void;
  onEnd?: () => void;
  onError?: (e: SpeechSynthesisErrorEvent) => void;
}

// モジュールスコープ状態
let voiceRef: SpeechSynthesisVoice | null = null;
let speakTimeoutId: ReturnType<typeof setTimeout> | null = null;
let voicesChangedListenerAttached = false;

// Web Speech API が利用可能かを SSR セーフに判定する。
function isWebSpeechAvailable(): boolean {
  return typeof window !== "undefined" && !!window.speechSynthesis;
}

function selectJapaneseVoice(): void {
  if (!isWebSpeechAvailable()) return;
  const voices = window.speechSynthesis.getVoices();
  const jaVoice = voices.find((v) => v.lang.startsWith("ja"));
  voiceRef = jaVoice ?? null;
}

// voiceschanged リスナーは多重登録防止のためモジュール内で 1 度だけアタッチする。
function ensureVoiceListener(): void {
  if (!isWebSpeechAvailable() || voicesChangedListenerAttached) return;
  selectJapaneseVoice();
  window.speechSynthesis.addEventListener("voiceschanged", selectJapaneseVoice);
  voicesChangedListenerAttached = true;
}

function applyVoice(utterance: SpeechSynthesisUtterance): void {
  if (voiceRef) {
    utterance.voice = voiceRef;
  } else {
    // ja-JP voice 未取得時のフォールバック。UA が lang から最適 voice を選ぶ。
    utterance.lang = "ja-JP";
  }
}

// Web Speech 経由でテキストを読み上げる。
// 未対応ブラウザ・SSR では no-op。callbacks.onError は Web Speech のエラー発生時のみ呼ぶ。
export function speakWithWebSpeech(
  text: string,
  callbacks: SpeakCallbacks
): void {
  if (!isWebSpeechAvailable()) {
    return;
  }
  ensureVoiceListener();

  // iOS Safari の cancel→speak 不発バグ対策: cancel 後に 50ms 待つ。
  cancelWebSpeech();

  speakTimeoutId = setTimeout(() => {
    speakTimeoutId = null;
    const utterance = new SpeechSynthesisUtterance(text);
    applyVoice(utterance);

    utterance.onstart = () => callbacks.onStart?.();
    utterance.onend = () => callbacks.onEnd?.();
    utterance.onerror = (ev) => {
      // "interrupted" は cancel による正常停止なので onError は呼ばず onEnd 相当の扱い。
      if (ev.error === "interrupted") {
        callbacks.onEnd?.();
        return;
      }
      callbacks.onError?.(ev);
    };

    window.speechSynthesis.speak(utterance);
  }, 50);
}

// Web Speech の発話を即停止する。pending の speak タイマーもキャンセルする。
export function cancelWebSpeech(): void {
  if (speakTimeoutId !== null) {
    clearTimeout(speakTimeoutId);
    speakTimeoutId = null;
  }
  if (!isWebSpeechAvailable()) {
    return;
  }
  window.speechSynthesis.cancel();
}

// iOS Safari 用 unlock。ユーザージェスチャの同期スコープから呼ぶこと。
// 無音 utterance を speak することで、以降の非同期 speak() が握りつぶされないようにする。
// 既に再生中 / pending の場合は何もしない (二重 prime 防止)。
export function primeWebSpeech(): void {
  if (!isWebSpeechAvailable()) return;
  ensureVoiceListener();
  if (window.speechSynthesis.speaking || window.speechSynthesis.pending) {
    return;
  }
  try {
    const u = new SpeechSynthesisUtterance(" ");
    u.volume = 0;
    u.rate = 10; // 最速で吐かせて即終了させる
    applyVoice(u);
    window.speechSynthesis.speak(u);
  } catch {
    // unlock 試行の失敗は黙って無視する。後続の speak で再試行される。
  }
}
