import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// Mock SpeechSynthesisUtterance
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

function createMockSynthesis(
  voices: Partial<SpeechSynthesisVoice>[] = [],
  overrides?: { speaking?: boolean; pending?: boolean }
) {
  const listeners = new Map<string, Set<EventListener>>();
  return {
    speak: vi.fn(),
    cancel: vi.fn(),
    getVoices: vi.fn(() => voices as SpeechSynthesisVoice[]),
    speaking: overrides?.speaking ?? false,
    pending: overrides?.pending ?? false,
    addEventListener: vi.fn((type: string, cb: EventListener) => {
      if (!listeners.has(type)) listeners.set(type, new Set());
      listeners.get(type)!.add(cb);
    }),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn((event: Event) => {
      listeners.get(event.type)?.forEach((cb) => cb(event));
      return true;
    }),
    _listeners: listeners,
  };
}

describe("webSpeech", () => {
  let mockSynthesis: ReturnType<typeof createMockSynthesis>;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.resetModules();
    vi.stubGlobal("SpeechSynthesisUtterance", MockUtterance);
    mockSynthesis = createMockSynthesis();
    vi.stubGlobal("speechSynthesis", mockSynthesis);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  // --- primeWebSpeech ---

  it("primeWebSpeech: selects voice via voiceschanged event", async () => {
    // Start with no voices
    mockSynthesis.getVoices.mockReturnValue([]);
    const { primeWebSpeech } = await import("@/lib/tts/webSpeech");

    primeWebSpeech();

    // The prime utterance should have lang fallback (no voice yet)
    const firstUtterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(firstUtterance.voice).toBeNull();
    expect(firstUtterance.lang).toBe("ja-JP");

    // Now fire voiceschanged with a ja voice
    const jaVoice = { lang: "ja-JP", name: "Japanese" } as SpeechSynthesisVoice;
    mockSynthesis.getVoices.mockReturnValue([jaVoice]);
    mockSynthesis.dispatchEvent(new Event("voiceschanged"));

    // Reset speaking/pending so second prime can proceed
    mockSynthesis.speaking = false;
    mockSynthesis.pending = false;
    mockSynthesis.speak.mockClear();

    // Second prime should use the discovered voice
    vi.resetModules();
    // Re-import would reset module state; instead call prime again on same module
    primeWebSpeech();

    const secondUtterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(secondUtterance.voice).toBe(jaVoice);
  });

  it("primeWebSpeech: selects voice immediately when available", async () => {
    const jaVoice = { lang: "ja-JP", name: "Japanese" } as SpeechSynthesisVoice;
    mockSynthesis = createMockSynthesis([jaVoice]);
    vi.stubGlobal("speechSynthesis", mockSynthesis);

    const { primeWebSpeech } = await import("@/lib/tts/webSpeech");
    primeWebSpeech();

    expect(mockSynthesis.speak).toHaveBeenCalledTimes(1);
    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.voice).toBe(jaVoice);
    expect(utterance.volume).toBe(0);
    expect(utterance.rate).toBe(10);
  });

  // --- speakWithWebSpeech ---

  it("speakWithWebSpeech: sets utterance.voice when ja-JP voice present", async () => {
    const jaVoice = { lang: "ja-JP", name: "Japanese" } as SpeechSynthesisVoice;
    mockSynthesis = createMockSynthesis([jaVoice]);
    vi.stubGlobal("speechSynthesis", mockSynthesis);

    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");
    speakWithWebSpeech("test", {});

    // Advance past the 50ms delay
    vi.advanceTimersByTime(50);

    expect(mockSynthesis.speak).toHaveBeenCalledTimes(1);
    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.voice).toBe(jaVoice);
  });

  it("speakWithWebSpeech: sets utterance.lang=ja-JP when no ja voice", async () => {
    mockSynthesis = createMockSynthesis([
      { lang: "en-US", name: "English" } as SpeechSynthesisVoice,
    ]);
    vi.stubGlobal("speechSynthesis", mockSynthesis);

    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");
    speakWithWebSpeech("test", {});
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.voice).toBeNull();
    expect(utterance.lang).toBe("ja-JP");
  });

  it("speakWithWebSpeech: cancel then 50ms wait before speak", async () => {
    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");
    speakWithWebSpeech("test", {});

    // cancel is called synchronously
    expect(mockSynthesis.cancel).toHaveBeenCalled();

    // At 49ms: speak should NOT have been called
    vi.advanceTimersByTime(49);
    expect(mockSynthesis.speak).not.toHaveBeenCalled();

    // At 50ms: speak fires
    vi.advanceTimersByTime(1);
    expect(mockSynthesis.speak).toHaveBeenCalledTimes(1);
  });

  it("speakWithWebSpeech: fires onStart callback", async () => {
    const onStart = vi.fn();
    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");
    speakWithWebSpeech("test", { onStart });
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    utterance.onstart?.();
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it("speakWithWebSpeech: fires onEnd callback", async () => {
    const onEnd = vi.fn();
    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");
    speakWithWebSpeech("test", { onEnd });
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    utterance.onend?.();
    expect(onEnd).toHaveBeenCalledTimes(1);
  });

  it("speakWithWebSpeech: onerror 'interrupted' calls onEnd, not onError", async () => {
    const onEnd = vi.fn();
    const onError = vi.fn();
    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");
    speakWithWebSpeech("test", { onEnd, onError });
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    utterance.onerror?.({ error: "interrupted" });
    expect(onEnd).toHaveBeenCalledTimes(1);
    expect(onError).not.toHaveBeenCalled();
  });

  it("speakWithWebSpeech: onerror other calls onError", async () => {
    const onError = vi.fn();
    const { speakWithWebSpeech } = await import("@/lib/tts/webSpeech");
    speakWithWebSpeech("test", { onError });
    vi.advanceTimersByTime(50);

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    const errorEvent = { error: "synthesis-failed" };
    utterance.onerror?.(errorEvent);
    expect(onError).toHaveBeenCalledTimes(1);
  });

  // --- Unsupported browser ---

  it("unsupported browser (no speechSynthesis): no-op, no crash", async () => {
    vi.stubGlobal("speechSynthesis", undefined);

    const { speakWithWebSpeech, cancelWebSpeech, primeWebSpeech } =
      await import("@/lib/tts/webSpeech");

    // None of these should throw
    expect(() => primeWebSpeech()).not.toThrow();
    expect(() => speakWithWebSpeech("test", {})).not.toThrow();
    expect(() => cancelWebSpeech()).not.toThrow();
  });
});
