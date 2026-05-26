import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

// SpeechSynthesisUtterance mock
class MockUtterance {
  text: string;
  voice: SpeechSynthesisVoice | null = null;
  lang = "";
  onstart: (() => void) | null = null;
  onend: (() => void) | null = null;
  onerror: ((ev: { error: string }) => void) | null = null;

  constructor(text: string) {
    this.text = text;
  }
}

vi.stubGlobal("SpeechSynthesisUtterance", MockUtterance);

// speechSynthesis mock
function createMockSynthesis(voices: Partial<SpeechSynthesisVoice>[] = []) {
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
    _listeners: listeners,
  };
}

describe("useTTS", () => {
  let mockSynthesis: ReturnType<typeof createMockSynthesis>;

  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
    mockSynthesis = createMockSynthesis();
    vi.stubGlobal("speechSynthesis", mockSynthesis);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  // Helper: import fresh module each time (since we mutate globals)
  async function importAndRender(
    opts?: Parameters<typeof import("@/hooks/useTTS")>[0]
  ) {
    const { useTTS } = await import("@/hooks/useTTS");
    return renderHook(() => useTTS());
  }

  it("speak calls cancel() then waits 50ms before speechSynthesis.speak", async () => {
    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    // Enable TTS
    act(() => {
      result.current.setEnabled(true);
    });

    act(() => {
      result.current.speak("hello");
    });

    // cancel should have been called
    expect(mockSynthesis.cancel).toHaveBeenCalled();

    // At 49ms: speak should NOT have been called
    act(() => {
      vi.advanceTimersByTime(49);
    });
    expect(mockSynthesis.speak).not.toHaveBeenCalled();

    // At 50ms: speak should fire
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(mockSynthesis.speak).toHaveBeenCalledTimes(1);
  });

  it("voiceschanged event triggers voice re-evaluation", async () => {
    // Start with no voices
    mockSynthesis.getVoices.mockReturnValue([]);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });

    // Now add a ja-JP voice and fire voiceschanged
    const jaVoice = { lang: "ja-JP", name: "Japanese" } as SpeechSynthesisVoice;
    mockSynthesis.getVoices.mockReturnValue([jaVoice]);

    act(() => {
      mockSynthesis.dispatchEvent(new Event("voiceschanged"));
    });

    // speak and check utterance has voice set
    act(() => {
      result.current.speak("test");
    });
    act(() => {
      vi.advanceTimersByTime(50);
    });

    expect(mockSynthesis.speak).toHaveBeenCalledTimes(1);
    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.voice).toBe(jaVoice);
  });

  it("selects ja-JP voice when available", async () => {
    const jaVoice = { lang: "ja-JP", name: "Japanese" } as SpeechSynthesisVoice;
    mockSynthesis.getVoices.mockReturnValue([
      { lang: "en-US", name: "English" } as SpeechSynthesisVoice,
      jaVoice,
    ]);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("test");
    });
    act(() => {
      vi.advanceTimersByTime(50);
    });

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.voice).toBe(jaVoice);
  });

  it("falls back to lang=ja-JP when no ja voice is available", async () => {
    mockSynthesis.getVoices.mockReturnValue([
      { lang: "en-US", name: "English" } as SpeechSynthesisVoice,
    ]);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("test");
    });
    act(() => {
      vi.advanceTimersByTime(50);
    });

    const utterance = mockSynthesis.speak.mock.calls[0][0] as MockUtterance;
    expect(utterance.voice).toBeNull();
    expect(utterance.lang).toBe("ja-JP");
  });

  it("enabled=false prevents speak from calling speechSynthesis.speak", async () => {
    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    // enabled defaults to false
    act(() => {
      result.current.speak("test");
    });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    expect(mockSynthesis.speak).not.toHaveBeenCalled();
  });

  it("persists enabled state to localStorage", async () => {
    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });

    expect(localStorage.getItem("ghostrunner_tts_enabled")).toBe("true");

    act(() => {
      result.current.setEnabled(false);
    });

    expect(localStorage.getItem("ghostrunner_tts_enabled")).toBe("false");
  });

  it("is SSR safe - no crash when speechSynthesis is undefined", async () => {
    // Remove speechSynthesis
    vi.stubGlobal("speechSynthesis", undefined);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    // Should not throw
    act(() => {
      result.current.speak("test");
    });
    act(() => {
      vi.advanceTimersByTime(100);
    });

    // error should be set for unsupported browser
    expect(result.current.error).toBeTruthy();
  });

  it("unsupported browser: speak is no-op and sets error", async () => {
    vi.stubGlobal("speechSynthesis", undefined);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    // setEnabled still works but speak sets error
    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("test");
    });

    expect(result.current.error).toContain("対応していない");
  });

  it("cancel() calls speechSynthesis.cancel and sets isSpeaking to false", async () => {
    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.cancel();
    });

    expect(mockSynthesis.cancel).toHaveBeenCalled();
    expect(result.current.isSpeaking).toBe(false);
  });
});
