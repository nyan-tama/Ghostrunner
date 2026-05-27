import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { requestTTS } from "@/lib/tts/elevenlabsClient";
import { TTSError } from "@/lib/tts/errors";

// 注意: jsdom + undici 環境の Response.blob() は new Blob() を body に渡すと
// 内部表現がそのまま文字列化されてサイズが安定しない。
// テストでは ArrayBuffer / 文字列 body を渡して、得られる Blob のサイズを直接コントロールする。

function audioResponse(
  body: BodyInit | null,
  init?: ResponseInit
): Response {
  return new Response(body, {
    status: 200,
    headers: { "Content-Type": "audio/mpeg" },
    ...init,
  });
}

function blobLike(value: unknown): boolean {
  // jsdom 環境では instanceof Blob が realm をまたいで失敗するため duck typing で判定する。
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as { size?: unknown }).size === "number" &&
    typeof (value as { type?: unknown }).type === "string" &&
    typeof (value as { arrayBuffer?: unknown }).arrayBuffer === "function"
  );
}

describe("requestTTS (elevenlabsClient)", () => {
  beforeEach(() => {
    // 念のため env をクリアして「相対 URL」モードで動かす
    vi.stubEnv("NEXT_PUBLIC_API_BASE", "");
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
  });

  it("200 + audio/mpeg returns Blob", async () => {
    const bytes = new Uint8Array([1, 2, 3, 4, 5]).buffer;
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(audioResponse(bytes));

    const result = await requestTTS({ text: "hello" });
    expect(blobLike(result)).toBe(true);
    expect(result.size).toBe(5);
    expect(result.type).toBe("audio/mpeg");
  });

  it("sends POST /api/tts with body {text}", async () => {
    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        audioResponse(new Uint8Array([1]).buffer)
      );

    await requestTTS({ text: "こんにちは" });

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [url, init] = fetchSpy.mock.calls[0];
    expect(url).toBe("/api/tts");
    expect(init?.method).toBe("POST");
    expect((init?.headers as Record<string, string>)["Content-Type"]).toBe(
      "application/json"
    );
    expect(init?.body).toBe(JSON.stringify({ text: "こんにちは" }));
  });

  it("network error -> TTSError(network_error) with cause", async () => {
    const cause = new TypeError("network down");
    vi.spyOn(globalThis, "fetch").mockRejectedValue(cause);

    let caught: unknown;
    try {
      await requestTTS({ text: "x" });
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(TTSError);
    expect((caught as TTSError).reason).toBe("network_error");
    expect((caught as Error & { cause?: unknown }).cause).toBe(cause);
  });

  it.each([400, 401, 429, 500, 502, 503, 504])(
    "non-ok %s -> TTSError(http_error)",
    async (status) => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
        new Response("err", { status })
      );

      let caught: unknown;
      try {
        await requestTTS({ text: "x" });
      } catch (err) {
        caught = err;
      }
      expect(caught).toBeInstanceOf(TTSError);
      expect((caught as TTSError).reason).toBe("http_error");
      expect((caught as TTSError).status).toBe(status);
    }
  );

  it("missing Content-Type header -> TTSError(missing_content_type)", async () => {
    const resp = new Response(new Uint8Array([1, 2, 3]).buffer, {
      status: 200,
    });
    // Response が ArrayBuffer 由来で Content-Type を自動セットしないか念のため削除
    resp.headers.delete("Content-Type");
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(resp);

    let caught: unknown;
    try {
      await requestTTS({ text: "x" });
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(TTSError);
    expect((caught as TTSError).reason).toBe("missing_content_type");
  });

  it("non-audio Content-Type -> TTSError(invalid_content_type)", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response('{"error":"x"}', {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );

    let caught: unknown;
    try {
      await requestTTS({ text: "x" });
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(TTSError);
    expect((caught as TTSError).reason).toBe("invalid_content_type");
  });

  it("Content-Type with uppercase (AUDIO/MPEG) is accepted (toLowerCase)", async () => {
    const bytes = new Uint8Array([7, 7, 7]).buffer;
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(bytes, {
        status: 200,
        headers: { "Content-Type": "AUDIO/MPEG" },
      })
    );

    const result = await requestTTS({ text: "x" });
    expect(blobLike(result)).toBe(true);
    expect(result.size).toBe(3);
  });

  it("empty Blob -> TTSError(empty_body)", async () => {
    // size 0 を確実にするため空文字列 body を使う
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("", {
        status: 200,
        headers: { "Content-Type": "audio/mpeg" },
      })
    );

    let caught: unknown;
    try {
      await requestTTS({ text: "x" });
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(TTSError);
    expect((caught as TTSError).reason).toBe("empty_body");
  });

  it("AbortSignal abort propagates AbortError (not wrapped)", async () => {
    const controller = new AbortController();
    // fetch を Abort 反応するように mock
    vi.spyOn(globalThis, "fetch").mockImplementationOnce(
      (_url, init) =>
        new Promise((_resolve, reject) => {
          const signal = (init as RequestInit | undefined)?.signal;
          if (signal) {
            signal.addEventListener("abort", () => {
              const err = new Error("aborted");
              err.name = "AbortError";
              reject(err);
            });
          }
        })
    );

    const promise = requestTTS({ text: "x", signal: controller.signal });
    controller.abort();

    let caught: unknown;
    try {
      await promise;
    } catch (err) {
      caught = err;
    }
    expect((caught as Error).name).toBe("AbortError");
    // TTSError でラップしないこと
    expect(caught).not.toBeInstanceOf(TTSError);
  });

  it("uses NEXT_PUBLIC_API_BASE when set", async () => {
    vi.stubEnv("NEXT_PUBLIC_API_BASE", "https://example.test");
    // Module cache を再評価させるため import を取り直す (API_BASE は import 時に確定するため)
    vi.resetModules();
    const { requestTTS: requestTTSReimported } = await import(
      "@/lib/tts/elevenlabsClient"
    );

    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        audioResponse(new Uint8Array([1]).buffer)
      );

    await requestTTSReimported({ text: "x" });

    const [url] = fetchSpy.mock.calls[0];
    expect(url).toBe("https://example.test/api/tts");
  });
});
