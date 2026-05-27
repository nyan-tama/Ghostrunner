import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// SpeechSynthesisUtterance mock
class MockUtterance {
  text: string;
  voice: SpeechSynthesisVoice | null = null;
  lang = "";
  volume = 1;
  rate = 1;
  onstart: (() => void) | null = null;
  onend: (() => void) | null = null;
  onerror: ((ev: { error: string }) => void) | null = null;

  constructor(text: string) {
    this.text = text;
  }
}

interface MockSynthesis {
  speak: ReturnType<typeof vi.fn>;
  cancel: ReturnType<typeof vi.fn>;
  getVoices: ReturnType<typeof vi.fn>;
  addEventListener: ReturnType<typeof vi.fn>;
  removeEventListener: ReturnType<typeof vi.fn>;
  // 関数として呼び出すため明示的に call signature 付き型にする
  dispatchEvent: ((event: Event) => boolean) & {
    mock: ReturnType<typeof vi.fn>["mock"];
  };
  speaking: boolean;
  pending: boolean;
  _listeners: Map<string, Set<EventListener>>;
}

function createMockSynthesis(
  voices: Partial<SpeechSynthesisVoice>[] = []
): MockSynthesis {
  const listeners = new Map<string, Set<EventListener>>();
  return {
    speak: vi.fn(),
    cancel: vi.fn(),
    getVoices: vi.fn(() => voices as SpeechSynthesisVoice[]),
    addEventListener: vi.fn((type: string, cb: EventListener) => {
      if (!listeners.has(type)) listeners.set(type, new Set());
      listeners.get(type)!.add(cb);
    }),
    removeEventListener: vi.fn((type: string, cb: EventListener) => {
      listeners.get(type)?.delete(cb);
    }),
    dispatchEvent: vi.fn((event: Event) => {
      listeners.get(event.type)?.forEach((cb) => cb(event));
      return true;
    }),
    speaking: false,
    pending: false,
    _listeners: listeners,
  };
}

describe("webSpeech", () => {
  let mockSynthesis: MockSynthesis;

  beforeEach(() => {
    vi.resetModules();
    vi.useFakeTimers();
    mockSynthesis = createMockSynthesis();
    vi.stubGlobal("SpeechSynthesisUtterance", MockUtterance);
    vi.stubGlobal("speechSynthesis", mockSynthesis);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("speakWithWebSpeech: calls cancel then waits 50ms before speak", async () => {
    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");

    speakWithWebSpeech("hello", {});

    // cancel should fire immediately
    expect(mockSynthesis.cancel).toHaveBeenCalled();

    // At 49ms: speak should NOT yet be called
    vi.advanceTimersByTime(49);
    expect(mockSynthesis.speak).not.toHaveBeenCalled();

    // At 50ms: speak should fire
    vi.advanceTimersByTime(1);
    expect(mockSynthesis.speak).toHaveBeenCalledTimes(1);
  });

  it("voiceschanged event re-evaluates ja-JP voice", async () => {
    // 初回は空
    mockSynthesis.getVoices.mockReturnValue([]);

    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");

    // 一度 speak を呼ぶことで voiceschanged リスナーが登録される
    speakWithWebSpeech("init", {});
    vi.advanceTimersByTime(50);

    expect(mockSynthesis.addEventListener).toHaveBeenCalledWith(
      "voiceschanged",
      expect.any(Function)
    );

    // 次に ja-JP voice を追加して voiceschanged 発火
    const jaVoice = { lang: "ja-JP", name: "Japanese" } as SpeechSynthesisVoice;
    mockSynthesis.getVoices.mockReturnValue([jaVoice]);
    mockSynthesis.dispatchEvent(new Event("voiceschanged"));

    // 次の speak で voice がセットされること
    speakWithWebSpeech("test", {});
    vi.advanceTimersByTime(50);

    const lastCall = mockSynthesis.speak.mock.calls.at(-1);
    const utterance = lastCall![0] as MockUtterance;
    expect(utterance.voice).toBe(jaVoice);
  });

  it("selects ja-JP voice when available", async () => {
    const jaVoice = { lang: "ja-JP", name: "Japanese" } as SpeechSynthesisVoice;
    mockSynthesis.getVoices.mockReturnValue([
      { lang: "en-US", name: "English" } as SpeechSynthesisVoice,
      jaVoice,
    ]);

    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");

    speakWithWebSpeech("test", {});
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.voice).toBe(jaVoice);
  });

  it("falls back to lang=ja-JP when no ja voice is available", async () => {
    mockSynthesis.getVoices.mockReturnValue([
      { lang: "en-US", name: "English" } as SpeechSynthesisVoice,
    ]);

    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");

    speakWithWebSpeech("test", {});
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.voice).toBeNull();
    expect(utterance.lang).toBe("ja-JP");
  });

  it("invokes onStart/onEnd/onError callbacks via utterance events", async () => {
    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");

    const onStart = vi.fn();
    const onEnd = vi.fn();
    const onError = vi.fn();

    speakWithWebSpeech("test", { onStart, onEnd, onError });
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;

    utterance.onstart?.();
    expect(onStart).toHaveBeenCalledTimes(1);

    utterance.onend?.();
    expect(onEnd).toHaveBeenCalledTimes(1);

    utterance.onerror?.({ error: "synthesis-failed" });
    expect(onError).toHaveBeenCalledTimes(1);
  });

  it("treats 'interrupted' error as onEnd, not onError", async () => {
    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");

    const onEnd = vi.fn();
    const onError = vi.fn();

    speakWithWebSpeech("test", { onEnd, onError });
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    utterance.onerror?.({ error: "interrupted" });

    expect(onEnd).toHaveBeenCalledTimes(1);
    expect(onError).not.toHaveBeenCalled();
  });

  it("cancelWebSpeech: clears pending speak timeout and calls speechSynthesis.cancel", async () => {
    const { speakWithWebSpeech, cancelWebSpeech } = await import(
      "@/lib/tts/webSpeech"
    );

    speakWithWebSpeech("hello", {});
    // タイマー進行前にキャンセル
    cancelWebSpeech();
    // 既存呼出と合わせて cancel が呼ばれている (speak の冒頭 + cancelWebSpeech)
    expect(mockSynthesis.cancel).toHaveBeenCalled();

    // 50ms 進めても speak されない (clearTimeout 効いている)
    vi.advanceTimersByTime(100);
    expect(mockSynthesis.speak).not.toHaveBeenCalled();
  });

  it("primeWebSpeech: speaks silent utterance (volume=0, rate=10, ' ')", async () => {
    const { primeWebSpeech } = await import("@/lib/tts/webSpeech");

    primeWebSpeech();

    expect(mockSynthesis.speak).toHaveBeenCalledTimes(1);
    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.text).toBe(" ");
    expect(utterance.volume).toBe(0);
    expect(utterance.rate).toBe(10);
  });

  it("primeWebSpeech: no-op when already speaking", async () => {
    mockSynthesis.speaking = true;

    const { primeWebSpeech } = await import("@/lib/tts/webSpeech");

    primeWebSpeech();

    expect(mockSynthesis.speak).not.toHaveBeenCalled();
  });

  it("no-op when speechSynthesis is undefined (SSR / unsupported)", async () => {
    vi.stubGlobal("speechSynthesis", undefined);

    const { speakWithWebSpeech, cancelWebSpeech, primeWebSpeech } =
      await import("@/lib/tts/webSpeech");

    // should not throw
    expect(() => speakWithWebSpeech("test", {})).not.toThrow();
    expect(() => cancelWebSpeech()).not.toThrow();
    expect(() => primeWebSpeech()).not.toThrow();
  });
});
