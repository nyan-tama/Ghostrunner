import {
  describe,
  it,
  expect,
  vi,
  beforeEach,
  afterEach,
  type MockInstance,
} from "vitest";
import { renderHook, act } from "@testing-library/react";

// /api/tts クライアントと webSpeech ヘルパーをモジュールモック化する。
// (useTTS のフォールバック分岐ロジックを単体で検証するため)
const mockRequestTTS = vi.fn<(...args: unknown[]) => Promise<Blob>>();
const mockSpeakWithWebSpeech = vi.fn();
const mockCancelWebSpeech = vi.fn();
const mockPrimeWebSpeech = vi.fn();

vi.mock("@/lib/tts/elevenlabsClient", () => ({
  requestTTS: (...args: unknown[]) => mockRequestTTS(...args),
}));

vi.mock("@/lib/tts/webSpeech", () => ({
  speakWithWebSpeech: (...args: unknown[]) => mockSpeakWithWebSpeech(...args),
  cancelWebSpeech: (...args: unknown[]) => mockCancelWebSpeech(...args),
  primeWebSpeech: (...args: unknown[]) => mockPrimeWebSpeech(...args),
}));

// -----------------------------------------------------------------------------
// HTMLAudioElement の挙動を制御するためのモック
// -----------------------------------------------------------------------------

interface MockAudio {
  play: ReturnType<typeof vi.fn>;
  pause: ReturnType<typeof vi.fn>;
  setAttribute: ReturnType<typeof vi.fn>;
  removeAttribute: ReturnType<typeof vi.fn>;
  dispatchEvent: (event: Event) => boolean;
  onplaying: (() => void) | null;
  onended: (() => void) | null;
  onerror: (() => void) | null;
  src: string;
  muted: boolean;
  preload: string;
  currentTime: number;
  // 計測用: removeAttribute と play の呼出順序など
  _opLog: string[];
}

function createMockAudio(playImpl?: () => Promise<void>): MockAudio {
  const audio: MockAudio = {
    play: vi.fn(playImpl ?? (() => Promise.resolve())),
    pause: vi.fn(() => {
      audio._opLog.push("pause");
    }),
    setAttribute: vi.fn(),
    removeAttribute: vi.fn((_name: string) => {
      audio._opLog.push("removeAttribute");
    }),
    dispatchEvent: (event: Event) => {
      if (event.type === "playing" && audio.onplaying) audio.onplaying();
      if (event.type === "ended" && audio.onended) audio.onended();
      if (event.type === "error" && audio.onerror) audio.onerror();
      return true;
    },
    onplaying: null,
    onended: null,
    onerror: null,
    src: "",
    muted: false,
    preload: "",
    currentTime: 0,
    _opLog: [],
  };
  return audio;
}

// -----------------------------------------------------------------------------
// テスト本体
// -----------------------------------------------------------------------------

describe("useTTS", () => {
  let createdAudios: MockAudio[];
  // 次に new Audio() で返すインスタンスを生成するファクトリ。
  // テストごとに play() の Promise 制御等で上書きする。
  let audioFactory: () => MockAudio;
  let createObjectURLSpy: MockInstance;
  let revokeObjectURLSpy: MockInstance;

  beforeEach(() => {
    localStorage.clear();
    mockRequestTTS.mockReset();
    mockSpeakWithWebSpeech.mockReset();
    mockCancelWebSpeech.mockReset();
    mockPrimeWebSpeech.mockReset();

    createdAudios = [];
    audioFactory = () => createMockAudio();

    // new Audio() をコンストラクタ関数で差し替える。
    // vi.spyOn(globalThis, "Audio").mockImplementation(fn) はコンストラクタ化されないため
    // vi.stubGlobal でコンストラクタ関数自体を上書きする。
    function MockAudioCtor(this: unknown) {
      const a = audioFactory();
      createdAudios.push(a);
      // new で呼ばれた時に this を返す形にする (本物の Audio() コンストラクタと挙動を合わせる)
      Object.assign(this as object, a);
      return a as unknown as object;
    }
    vi.stubGlobal("Audio", MockAudioCtor as unknown as typeof Audio);

    // URL.createObjectURL / revokeObjectURL は jsdom でも実装あるが
    // 呼出を観測するため spy をかける
    let counter = 0;
    createObjectURLSpy = vi
      .spyOn(globalThis.URL, "createObjectURL")
      .mockImplementation(() => `blob:mock-${counter++}`);
    revokeObjectURLSpy = vi
      .spyOn(globalThis.URL, "revokeObjectURL")
      .mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  // --- 公開 API シェイプ ----------------------------------------------------

  it("exposes expected public API shape", async () => {
    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());
    expect(typeof result.current.speak).toBe("function");
    expect(typeof result.current.cancel).toBe("function");
    expect(typeof result.current.setEnabled).toBe("function");
    expect(typeof result.current.prime).toBe("function");
    expect(typeof result.current.enabled).toBe("boolean");
    expect(typeof result.current.isSpeaking).toBe("boolean");
    expect(result.current.error).toBeNull();
  });

  // --- enabled / localStorage / SSR ----------------------------------------

  it("enabled=false makes speak a no-op (no fetch, no webSpeech)", async () => {
    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    // enabled defaults to false
    act(() => {
      result.current.speak("hello");
    });

    await Promise.resolve();
    expect(mockRequestTTS).not.toHaveBeenCalled();
    expect(mockSpeakWithWebSpeech).not.toHaveBeenCalled();
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

  it("is SSR-safe: hook works in jsdom without crashing", async () => {
    // jsdom 環境では window が存在するため、useTTS 内部の typeof window === "undefined" 分岐は
    // ここでは到達不能。renderHook が throw しないことだけ確認 (型コンパイル含む構造的安全性)。
    const { useTTS } = await import("@/hooks/useTTS");
    expect(() => renderHook(() => useTTS())).not.toThrow();
  });

  // --- 正常系: ElevenLabs 成功 ---------------------------------------------

  it("speak success: fetches /api/tts, plays audio, isSpeaking toggles on events", async () => {
    const blob = new Blob(["audio"], { type: "audio/mpeg" });
    mockRequestTTS.mockResolvedValue(blob);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });

    // requestTTS が呼ばれている (await を解消)
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mockRequestTTS).toHaveBeenCalledTimes(1);

    // play されている (speak 用 Audio = createdAudios[0])
    const audio = createdAudios[0];
    expect(audio.play).toHaveBeenCalled();

    // playing イベントで isSpeaking=true
    act(() => {
      audio.dispatchEvent(new Event("playing"));
    });
    expect(result.current.isSpeaking).toBe(true);

    // ended イベントで isSpeaking=false + revokeObjectURL
    act(() => {
      audio.dispatchEvent(new Event("ended"));
    });
    expect(result.current.isSpeaking).toBe(false);
    expect(revokeObjectURLSpy).toHaveBeenCalled();
  });

  // --- フォールバック分岐: 7 経路 ------------------------------------------

  async function fallbackHelper(rejectVal: unknown) {
    mockRequestTTS.mockRejectedValue(rejectVal);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });

    // microtask flush
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    return result;
  }

  it("fallback: network error -> webSpeech invoked, error set", async () => {
    const networkErr = new TypeError("network down");
    const result = await fallbackHelper(networkErr);
    expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
    expect(result.current.error).toBeTruthy();
  });

  it("fallback: 4xx (TTSError wrapped) -> webSpeech invoked", async () => {
    // useTTS 視点では requestTTS が throw すれば理由は何でもよい
    const err = Object.assign(new Error("400"), { name: "TTSError" });
    const result = await fallbackHelper(err);
    expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
    expect(result.current.error).toBeTruthy();
  });

  it("fallback: 5xx -> webSpeech invoked", async () => {
    const err = Object.assign(new Error("503"), { name: "TTSError" });
    const result = await fallbackHelper(err);
    expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
    expect(result.current.error).toBeTruthy();
  });

  it("fallback: invalid Content-Type -> webSpeech invoked", async () => {
    const err = Object.assign(new Error("bad ct"), { name: "TTSError" });
    const result = await fallbackHelper(err);
    expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
    expect(result.current.error).toBeTruthy();
  });

  it("fallback: empty Blob -> webSpeech invoked", async () => {
    const err = Object.assign(new Error("empty"), { name: "TTSError" });
    const result = await fallbackHelper(err);
    expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
    expect(result.current.error).toBeTruthy();
  });

  it("fallback: audio.onerror -> webSpeech invoked, Blob URL revoked", async () => {
    const blob = new Blob(["x"], { type: "audio/mpeg" });
    mockRequestTTS.mockResolvedValue(blob);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    const audio = createdAudios[0];
    revokeObjectURLSpy.mockClear();

    act(() => {
      audio.dispatchEvent(new Event("error"));
    });

    expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
    expect(revokeObjectURLSpy).toHaveBeenCalled();
    expect(result.current.error).toBeTruthy();
  });

  it("fallback: audio.play() reject -> webSpeech invoked", async () => {
    const blob = new Blob(["x"], { type: "audio/mpeg" });
    mockRequestTTS.mockResolvedValue(blob);

    // audio.play() を reject させたい場合、factory を差し替える
    audioFactory = () => createMockAudio(() => Promise.reject(new Error("autoplay")));

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mockSpeakWithWebSpeech).toHaveBeenCalledTimes(1);
    expect(result.current.error).toBeTruthy();
  });

  it("AbortError from requestTTS does NOT trigger fallback", async () => {
    const abortErr = new Error("aborted");
    abortErr.name = "AbortError";
    mockRequestTTS.mockRejectedValue(abortErr);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mockSpeakWithWebSpeech).not.toHaveBeenCalled();
    expect(result.current.error).toBeNull();
  });

  // --- error クリアのタイミング (playing イベント) -------------------------

  it("error is cleared on `playing` event (not on play() call)", async () => {
    // 1 回目: フォールバックで error セット
    const networkErr = new TypeError("net");
    mockRequestTTS.mockRejectedValueOnce(networkErr);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(result.current.error).toBeTruthy();

    // 2 回目: 成功
    const blob = new Blob(["x"], { type: "audio/mpeg" });
    mockRequestTTS.mockResolvedValueOnce(blob);

    act(() => {
      result.current.speak("again");
    });
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    // play は呼ばれているが、まだ playing イベント未発火なので error は残ったまま
    // (`play` メソッド呼出と `playing` イベントを区別する仕様)
    expect(result.current.error).toBeTruthy();

    // playing イベントで初めてクリア
    const audio = createdAudios[createdAudios.length - 1];
    act(() => {
      audio.dispatchEvent(new Event("playing"));
    });
    expect(result.current.error).toBeNull();
  });

  // --- cancel / setEnabled(false) / unmount -------------------------------

  it("cancel() pauses audio and aborts in-flight fetch", async () => {
    let resolveBlob: (b: Blob) => void = () => {};
    mockRequestTTS.mockImplementation((...args: unknown[]) => {
      const params = args[0] as { signal?: AbortSignal };
      return new Promise<Blob>((resolve, reject) => {
        resolveBlob = resolve;
        params.signal?.addEventListener("abort", () => {
          const err = new Error("abort");
          err.name = "AbortError";
          reject(err);
        });
      });
    });

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });
    // fetch 進行中
    act(() => {
      result.current.cancel();
    });

    // pause が呼ばれている
    const audio = createdAudios[0];
    expect(audio.pause).toHaveBeenCalled();

    // 抑制: resolveBlob() を呼んでも、その後 play が新たに呼ばれない (signal.aborted で早期 return)。
    // 注意: setEnabled(true) が内部で prime() を呼び silentMp3 を play している (autoplay unlock)。
    //       そのため play 呼出数の "増分" を見る。
    const playCountBefore = audio.play.mock.calls.length;
    resolveBlob(new Blob(["x"], { type: "audio/mpeg" }));
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(audio.play.mock.calls.length).toBe(playCountBefore);
  });

  it("setEnabled(false) aborts in-flight fetch and pauses audio", async () => {
    let signalRef: AbortSignal | undefined;
    mockRequestTTS.mockImplementation((...args: unknown[]) => {
      const params = args[0] as { signal?: AbortSignal };
      signalRef = params.signal;
      return new Promise<Blob>(() => {}); // 永遠 pending
    });

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });
    expect(signalRef?.aborted).toBe(false);

    act(() => {
      result.current.setEnabled(false);
    });

    expect(signalRef?.aborted).toBe(true);
    const audio = createdAudios[0];
    expect(audio.pause).toHaveBeenCalled();
  });

  it("cancel() during fallback also calls cancelWebSpeech", async () => {
    // フォールバック状態を作る
    const networkErr = new TypeError("net");
    mockRequestTTS.mockRejectedValue(networkErr);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(mockSpeakWithWebSpeech).toHaveBeenCalled();

    // フォールバック中に cancel
    act(() => {
      result.current.cancel();
    });
    expect(mockCancelWebSpeech).toHaveBeenCalled();
  });

  it("cancel() without active fallback does NOT call cancelWebSpeech", async () => {
    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.cancel();
    });

    expect(mockCancelWebSpeech).not.toHaveBeenCalled();
  });

  it("unsupported browser + ElevenLabs failure -> error set, no throw", async () => {
    // ElevenLabs failure
    mockRequestTTS.mockRejectedValue(new TypeError("net"));
    // Web Speech 呼出は webSpeech.ts 側で no-op になる前提 (mock では何もしない)

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    expect(() => {
      act(() => {
        result.current.speak("hello");
      });
    }).not.toThrow();
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(result.current.error).toBeTruthy();
  });

  // --- prime() の挙動 ------------------------------------------------------

  it("prime(): audio.play().then(pause) pattern", async () => {
    // play 解決順序を制御するため Promise を分解
    let resolvePlay: () => void = () => {};
    const playPromise = new Promise<void>((r) => {
      resolvePlay = r;
    });
    audioFactory = () => createMockAudio(() => playPromise);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.prime();
    });

    const audio = createdAudios[0];
    expect(audio.muted).toBe(true);
    expect(audio.src).toContain("data:audio/mpeg");
    expect(audio.play).toHaveBeenCalled();
    expect(audio.pause).not.toHaveBeenCalled();

    // play() 解決後に pause + muted=false に戻る
    resolvePlay();
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(audio.pause).toHaveBeenCalled();
    expect(audio.muted).toBe(false);
    expect(audio.currentTime).toBe(0);
  });

  it("prime(): nullifies audio handlers and revokes lingering Blob URL", async () => {
    // 一度 speak 成功させてハンドラと Blob URL を残す
    const blob = new Blob(["x"], { type: "audio/mpeg" });
    mockRequestTTS.mockResolvedValueOnce(blob);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    const audio = createdAudios[0];
    expect(audio.onplaying).not.toBeNull();
    expect(audio.onended).not.toBeNull();
    expect(audio.onerror).not.toBeNull();

    revokeObjectURLSpy.mockClear();
    act(() => {
      result.current.prime();
    });

    expect(audio.onplaying).toBeNull();
    expect(audio.onended).toBeNull();
    expect(audio.onerror).toBeNull();
    expect(revokeObjectURLSpy).toHaveBeenCalled();
  });

  // --- stopAll でハンドラ null 化 -----------------------------------------

  it("stopAll (via cancel) nullifies audio handlers", async () => {
    const blob = new Blob(["x"], { type: "audio/mpeg" });
    mockRequestTTS.mockResolvedValueOnce(blob);

    const { useTTS } = await import("@/hooks/useTTS");
    const { result } = renderHook(() => useTTS());

    act(() => {
      result.current.setEnabled(true);
    });
    act(() => {
      result.current.speak("hello");
    });
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    const audio = createdAudios[0];
    expect(audio.onplaying).not.toBeNull();

    act(() => {
      result.current.cancel();
    });

    expect(audio.onplaying).toBeNull();
    expect(audio.onended).toBeNull();
    expect(audio.onerror).toBeNull();
  });
});
