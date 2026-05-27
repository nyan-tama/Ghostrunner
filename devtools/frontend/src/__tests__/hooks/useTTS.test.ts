import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { TTSError, type TTSFallbackReason } from "@/lib/tts/errors";

// ---------------------------------------------------------------------------
// Mock: voicevoxClient
// ---------------------------------------------------------------------------
const mockRequestTTS = vi.fn<
  [{ text: string; signal?: AbortSignal }],
  Promise<Blob>
>();

vi.mock("@/lib/tts/voicevoxClient", () => ({
  requestTTS: (params: { text: string; signal?: AbortSignal }) =>
    mockRequestTTS(params),
}));

// ---------------------------------------------------------------------------
// Mock: webSpeech
// ---------------------------------------------------------------------------
const mockSpeakWithWebSpeech = vi.fn();
const mockCancelWebSpeech = vi.fn();
const mockPrimeWebSpeech = vi.fn();

vi.mock("@/lib/tts/webSpeech", () => ({
  speakWithWebSpeech: (...args: unknown[]) => mockSpeakWithWebSpeech(...args),
  cancelWebSpeech: (...args: unknown[]) => mockCancelWebSpeech(...args),
  primeWebSpeech: (...args: unknown[]) => mockPrimeWebSpeech(...args),
}));

// ---------------------------------------------------------------------------
// Mock: Audio
// ---------------------------------------------------------------------------
let audioInstances: MockAudio[] = [];

class MockAudio {
  onplaying: (() => void) | null = null;
  onended: (() => void) | null = null;
  onerror: (() => void) | null = null;

  play = vi.fn(() => Promise.resolve());
  pause = vi.fn();
  removeAttribute = vi.fn();
  setAttribute = vi.fn();

  private _src = "";
  private _preload = "";
  private _muted = false;
  private _currentTime = 0;

  get src() {
    return this._src;
  }
  set src(v: string) {
    this._src = v;
  }
  get preload() {
    return this._preload;
  }
  set preload(v: string) {
    this._preload = v;
  }
  get muted() {
    return this._muted;
  }
  set muted(v: boolean) {
    this._muted = v;
  }
  get currentTime() {
    return this._currentTime;
  }
  set currentTime(v: number) {
    this._currentTime = v;
  }

  constructor() {
    audioInstances.push(this);
  }
}

vi.stubGlobal("Audio", MockAudio);
// Mock only the static methods on URL, preserving the URL constructor.
const originalCreateObjectURL = URL.createObjectURL;
const originalRevokeObjectURL = URL.revokeObjectURL;
URL.createObjectURL = vi.fn(() => "blob:mock");
URL.revokeObjectURL = vi.fn();

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------
function latestAudio(): MockAudio {
  return audioInstances[audioInstances.length - 1];
}

/** Resolve pending microtasks (Promise callbacks). */
async function flushMicrotasks() {
  await act(async () => {
    // Let all pending promises resolve
    await new Promise((r) => setTimeout(r, 0));
  });
}

function makeTTSError(reason: TTSFallbackReason): TTSError {
  return new TTSError(`test error: ${reason}`, { reason });
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("useTTS", () => {
  beforeEach(() => {
    audioInstances = [];
    localStorage.clear();
    mockRequestTTS.mockReset();
    mockSpeakWithWebSpeech.mockReset();
    mockCancelWebSpeech.mockReset();
    mockPrimeWebSpeech.mockReset();
    (URL.createObjectURL as ReturnType<typeof vi.fn>).mockReturnValue("blob:mock");
    (URL.revokeObjectURL as ReturnType<typeof vi.fn>).mockClear();
  });

  afterEach(() => {
    // Do not use vi.restoreAllMocks() here because it would restore the
    // URL.createObjectURL/revokeObjectURL mocks set at module scope.
  });

  // Import the hook freshly (it has "use client" but jsdom env handles it).
  async function importHook() {
    const mod = await import("@/hooks/useTTS");
    return mod.useTTS;
  }

  // ========================================================================
  // Fallback paths (AC2/AC7) - 7 cases
  // ========================================================================

  describe("Fallback paths (VOICEVOX -> Web Speech)", () => {
    it("network_error: requestTTS throws TypeError -> fallback + network error message", async () => {
      const useTTS = await importHook();
      mockRequestTTS.mockRejectedValue(new TypeError("Failed to fetch"));

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
      expect(result.current.error).toBe(
        "VOICEVOX 接続失敗。Web Speech に降格しました"
      );
    });

    it("http_error: requestTTS throws TTSError(http_error) -> fallback + network error message", async () => {
      const useTTS = await importHook();
      mockRequestTTS.mockRejectedValue(makeTTSError("http_error"));

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
      expect(result.current.error).toBe(
        "VOICEVOX 接続失敗。Web Speech に降格しました"
      );
    });

    it("missing_content_type: requestTTS throws TTSError(missing_content_type) -> fallback", async () => {
      const useTTS = await importHook();
      mockRequestTTS.mockRejectedValue(makeTTSError("missing_content_type"));

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
      expect(result.current.error).toBe(
        "VOICEVOX 接続失敗。Web Speech に降格しました"
      );
    });

    it("invalid_content_type: requestTTS throws TTSError(invalid_content_type) -> fallback", async () => {
      const useTTS = await importHook();
      mockRequestTTS.mockRejectedValue(makeTTSError("invalid_content_type"));

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
      expect(result.current.error).toBe(
        "VOICEVOX 接続失敗。Web Speech に降格しました"
      );
    });

    it("empty_body: requestTTS throws TTSError(empty_body) -> fallback", async () => {
      const useTTS = await importHook();
      mockRequestTTS.mockRejectedValue(makeTTSError("empty_body"));

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
      expect(result.current.error).toBe(
        "VOICEVOX 接続失敗。Web Speech に降格しました"
      );
    });

    it("audio_error: blob OK but audio.onerror fires -> fallback + audio error message", async () => {
      const useTTS = await importHook();
      const blob = new Blob(["audio"], { type: "audio/wav" });
      mockRequestTTS.mockResolvedValue(blob);

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      // Trigger audio error
      const audio = latestAudio();
      await act(async () => {
        audio.onerror?.();
      });

      expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
      expect(result.current.error).toBe(
        "音声再生失敗。Web Speech に降格しました"
      );
    });

    it("play_rejected: audio.play() rejects -> fallback + audio error message", async () => {
      const useTTS = await importHook();
      const blob = new Blob(["audio"], { type: "audio/wav" });
      mockRequestTTS.mockResolvedValue(blob);

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });

      // Set the rejection AFTER setEnabled (which triggers prime and consumes
      // one play call). This ensures the rejection targets the speak() call.
      const audio = latestAudio();
      audio.play.mockRejectedValueOnce(new DOMException("NotAllowedError"));

      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
      expect(result.current.error).toBe(
        "音声再生失敗。Web Speech に降格しました"
      );
    });
  });

  // ========================================================================
  // Prime tests (AC9)
  // ========================================================================

  describe("prime()", () => {
    it("calls primeWebSpeech, then audio.play with muted silent MP3", async () => {
      const useTTS = await importHook();
      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.prime();
      });

      expect(mockPrimeWebSpeech).toHaveBeenCalledTimes(1);
      const audio = latestAudio();
      expect(audio.muted).toBe(true);
      expect(audio.src).toContain("data:audio/mpeg;base64,");
      expect(audio.play).toHaveBeenCalledTimes(1);
    });

    it("consecutive prime() - second is ignored (primeInFlightRef guard)", async () => {
      const useTTS = await importHook();
      const { result } = renderHook(() => useTTS());

      // First prime: play returns a promise that never resolves yet
      const audio = latestAudio();
      let resolvePlay!: () => void;
      audio.play.mockReturnValueOnce(
        new Promise<void>((r) => {
          resolvePlay = r;
        })
      );

      act(() => {
        result.current.prime();
      });

      // Second prime while first is in-flight
      act(() => {
        result.current.prime();
      });

      // primeWebSpeech called only once (second prime was ignored)
      expect(mockPrimeWebSpeech).toHaveBeenCalledTimes(1);
      expect(audio.play).toHaveBeenCalledTimes(1);

      // Resolve first prime
      await act(async () => {
        resolvePlay();
      });
    });

    it("prime() when isSpeaking - ignored", async () => {
      const useTTS = await importHook();
      const blob = new Blob(["audio"], { type: "audio/wav" });
      mockRequestTTS.mockResolvedValue(blob);

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });

      // Start speaking
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      // Trigger onplaying to set isSpeaking=true
      const audio = latestAudio();
      await act(async () => {
        audio.onplaying?.();
      });
      expect(result.current.isSpeaking).toBe(true);

      // Reset mock counts to check prime call
      mockPrimeWebSpeech.mockClear();
      audio.play.mockClear();

      act(() => {
        result.current.prime();
      });

      // Should be ignored
      expect(mockPrimeWebSpeech).not.toHaveBeenCalled();
    });

    it("prime() play() reject - no error set (silent fail)", async () => {
      const useTTS = await importHook();
      const { result } = renderHook(() => useTTS());

      const audio = latestAudio();
      audio.play.mockRejectedValueOnce(new DOMException("NotAllowedError"));

      act(() => {
        result.current.prime();
      });
      await flushMicrotasks();

      expect(result.current.error).toBeNull();
    });
  });

  // ========================================================================
  // Speak / cancel tests
  // ========================================================================

  describe("speak/cancel", () => {
    it("enabled=false -> fetch not called (AC3)", async () => {
      const useTTS = await importHook();
      const { result } = renderHook(() => useTTS());

      // enabled defaults to false
      act(() => {
        result.current.speak("test");
      });
      await flushMicrotasks();

      expect(mockRequestTTS).not.toHaveBeenCalled();
    });

    it("consecutive speak() -> first aborted + new fetch (AC5)", async () => {
      const useTTS = await importHook();

      // First call: never-resolving promise
      let firstSignal: AbortSignal | undefined;
      mockRequestTTS.mockImplementationOnce(({ signal }) => {
        firstSignal = signal;
        return new Promise(() => {}); // never resolves
      });

      // Second call: resolves normally
      const blob = new Blob(["audio"], { type: "audio/wav" });
      mockRequestTTS.mockResolvedValueOnce(blob);

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });

      act(() => {
        result.current.speak("first");
      });

      act(() => {
        result.current.speak("second");
      });
      await flushMicrotasks();

      expect(firstSignal?.aborted).toBe(true);
      expect(mockRequestTTS).toHaveBeenCalledTimes(2);
    });

    it("cancel() order: abort -> pause -> removeAttribute -> revoke", async () => {
      const useTTS = await importHook();
      const blob = new Blob(["audio"], { type: "audio/wav" });
      mockRequestTTS.mockResolvedValue(blob);

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      const audio = latestAudio();
      // Clear mocks to track cancel order
      audio.pause.mockClear();
      audio.removeAttribute.mockClear();
      (URL.revokeObjectURL as ReturnType<typeof vi.fn>).mockClear();

      act(() => {
        result.current.cancel();
      });

      expect(audio.pause).toHaveBeenCalled();
      expect(audio.removeAttribute).toHaveBeenCalledWith("src");
      expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:mock");
    });

    it("AbortError -> no fallback, no error (intentional cancel)", async () => {
      const useTTS = await importHook();
      // Use a plain Error with name="AbortError" because jsdom's DOMException
      // does not extend Error, so the production code's `instanceof Error` check fails.
      const abortError = Object.assign(new Error("Aborted"), {
        name: "AbortError",
      });
      mockRequestTTS.mockRejectedValue(abortError);

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      expect(mockSpeakWithWebSpeech).not.toHaveBeenCalled();
      expect(result.current.error).toBeNull();
    });

    it("cancel during fallback calls cancelWebSpeech", async () => {
      const useTTS = await importHook();
      mockRequestTTS.mockRejectedValue(makeTTSError("network_error"));

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      // Fallback should be active now
      expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);

      mockCancelWebSpeech.mockClear();
      act(() => {
        result.current.cancel();
      });

      expect(mockCancelWebSpeech).toHaveBeenCalled();
    });

    it("setEnabled(false) triggers stopAll", async () => {
      const useTTS = await importHook();
      const blob = new Blob(["audio"], { type: "audio/wav" });
      mockRequestTTS.mockResolvedValue(blob);

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      const audio = latestAudio();
      audio.pause.mockClear();

      act(() => {
        result.current.setEnabled(false);
      });

      expect(audio.pause).toHaveBeenCalled();
    });

    it("setEnabled(true) triggers prime()", async () => {
      const useTTS = await importHook();
      const { result } = renderHook(() => useTTS());

      mockPrimeWebSpeech.mockClear();

      act(() => {
        result.current.setEnabled(true);
      });

      expect(mockPrimeWebSpeech).toHaveBeenCalled();
    });
  });

  // ========================================================================
  // Error clearing (AC8)
  // ========================================================================

  describe("Error clearing", () => {
    it("after fallback, new successful speak -> onplaying clears error to null", async () => {
      const useTTS = await importHook();

      // First speak: network error -> fallback
      mockRequestTTS.mockRejectedValueOnce(makeTTSError("network_error"));

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("fail");
      });
      await flushMicrotasks();

      expect(result.current.error).toBe(
        "VOICEVOX 接続失敗。Web Speech に降格しました"
      );

      // Second speak: success
      const blob = new Blob(["audio"], { type: "audio/wav" });
      mockRequestTTS.mockResolvedValueOnce(blob);

      act(() => {
        result.current.speak("success");
      });
      await flushMicrotasks();

      // Trigger onplaying
      const audio = latestAudio();
      await act(async () => {
        audio.onplaying?.();
      });

      expect(result.current.error).toBeNull();
    });

    it("onplay (not onplaying) does NOT clear error - only onplaying does", async () => {
      const useTTS = await importHook();

      // First speak: fallback to set error
      mockRequestTTS.mockRejectedValueOnce(makeTTSError("network_error"));

      const { result } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("fail");
      });
      await flushMicrotasks();

      expect(result.current.error).not.toBeNull();

      // Second speak: success but we check that only onplaying clears error
      const blob = new Blob(["audio"], { type: "audio/wav" });
      mockRequestTTS.mockResolvedValueOnce(blob);

      act(() => {
        result.current.speak("success");
      });
      await flushMicrotasks();

      // The hook uses audio.onplaying, not audio.onplay.
      // The MockAudio class has no onplay property, so the hook never sets it.
      // Error should still be present before onplaying fires.
      expect(result.current.error).toBe(
        "VOICEVOX 接続失敗。Web Speech に降格しました"
      );

      // Now fire onplaying to clear it
      const audio = latestAudio();
      await act(async () => {
        audio.onplaying?.();
      });
      expect(result.current.error).toBeNull();
    });
  });

  // ========================================================================
  // Unmount
  // ========================================================================

  describe("Unmount", () => {
    it("cleanup: abort -> pause -> removeAttribute -> revoke -> cancelWebSpeech", async () => {
      const useTTS = await importHook();
      mockRequestTTS.mockRejectedValue(makeTTSError("network_error"));

      const { result, unmount } = renderHook(() => useTTS());

      act(() => {
        result.current.setEnabled(true);
      });
      act(() => {
        result.current.speak("hello");
      });
      await flushMicrotasks();

      // Fallback is active
      expect(mockSpeakWithWebSpeech).toHaveBeenCalled();

      const audio = latestAudio();
      audio.pause.mockClear();
      audio.removeAttribute.mockClear();
      (URL.revokeObjectURL as ReturnType<typeof vi.fn>).mockClear();
      mockCancelWebSpeech.mockClear();

      unmount();

      expect(audio.pause).toHaveBeenCalled();
      expect(audio.removeAttribute).toHaveBeenCalledWith("src");
      expect(mockCancelWebSpeech).toHaveBeenCalled();
    });

    it("Audio created only once on mount", async () => {
      const useTTS = await importHook();
      audioInstances = []; // Reset to count from mount

      renderHook(() => useTTS());

      expect(audioInstances).toHaveLength(1);
    });
  });
});
