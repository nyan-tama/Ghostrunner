import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

describe("voicevoxClient - requestTTS", () => {
  const originalEnv = process.env.NEXT_PUBLIC_API_BASE;
  let fetchSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.resetModules();
    fetchSpy = vi.spyOn(global, "fetch");
    delete process.env.NEXT_PUBLIC_API_BASE;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    if (originalEnv !== undefined) {
      process.env.NEXT_PUBLIC_API_BASE = originalEnv;
    } else {
      delete process.env.NEXT_PUBLIC_API_BASE;
    }
  });

  function makeResponse(
    body: BodyInit | null,
    init?: ResponseInit & { headers?: Record<string, string> }
  ): Response {
    const headers = new Headers(init?.headers);
    return new Response(body, { ...init, headers });
  }

  function audioWavResponse(): Response {
    return makeResponse("audio-data", {
      status: 200,
      headers: { "Content-Type": "audio/wav" },
    });
  }

  async function loadRequestTTS() {
    const mod = await import("@/lib/tts/voicevoxClient");
    return mod.requestTTS;
  }

  // --- URL construction ---

  it("uses relative /api/tts when NEXT_PUBLIC_API_BASE is unset", async () => {
    delete process.env.NEXT_PUBLIC_API_BASE;
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(audioWavResponse());

    await requestTTS({ text: "hello" });

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const url = fetchSpy.mock.calls[0][0] as string;
    expect(url).toBe("/api/tts");
  });

  it("uses full URL when NEXT_PUBLIC_API_BASE is set", async () => {
    process.env.NEXT_PUBLIC_API_BASE = "http://localhost:8888";
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(audioWavResponse());

    await requestTTS({ text: "hello" });

    const url = fetchSpy.mock.calls[0][0] as string;
    expect(url).toBe("http://localhost:8888/api/tts");
  });

  // --- Request shape ---

  it("sends POST with Content-Type application/json", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(audioWavResponse());

    await requestTTS({ text: "test" });

    const init = fetchSpy.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("POST");
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe(
      "application/json"
    );
  });

  it("sends JSON body with text field", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(audioWavResponse());

    await requestTTS({ text: "hello world" });

    const init = fetchSpy.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(init.body as string)).toEqual({ text: "hello world" });
  });

  it("propagates AbortSignal to fetch", async () => {
    const requestTTS = await loadRequestTTS();
    const controller = new AbortController();
    fetchSpy.mockResolvedValue(audioWavResponse());

    await requestTTS({ text: "t", signal: controller.signal });

    const init = fetchSpy.mock.calls[0][1] as RequestInit;
    expect(init.signal).toBe(controller.signal);
  });

  // --- Success cases ---

  it("returns Blob on 200 with audio/wav", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(audioWavResponse());

    const result = await requestTTS({ text: "t" });
    expect(result.size).toBeGreaterThan(0);
  });

  it("accepts Content-Type audio/Wav (mixed case)", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(
      makeResponse("data", {
        status: 200,
        headers: { "Content-Type": "audio/Wav" },
      })
    );

    const result = await requestTTS({ text: "t" });
    expect(result.size).toBeGreaterThan(0);
  });

  it("accepts Content-Type Audio/wav (mixed case)", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(
      makeResponse("data", {
        status: 200,
        headers: { "Content-Type": "Audio/wav" },
      })
    );

    const result = await requestTTS({ text: "t" });
    expect(result.size).toBeGreaterThan(0);
  });

  it("accepts Content-Type AUDIO/WAV (all upper)", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(
      makeResponse("data", {
        status: 200,
        headers: { "Content-Type": "AUDIO/WAV" },
      })
    );

    const result = await requestTTS({ text: "t" });
    expect(result.size).toBeGreaterThan(0);
  });

  // --- Error: missing Content-Type ---

  it("throws TTSError(missing_content_type) when Content-Type is null", async () => {
    const requestTTS = await loadRequestTTS();
    // Response constructor with no Content-Type set explicitly.
    // Response() auto-adds Content-Type for string body, so use a custom approach.
    const res = new Response("data", { status: 200 });
    res.headers.delete("Content-Type");
    fetchSpy.mockResolvedValue(res);

    try {
      await requestTTS({ text: "t" });
      expect.unreachable("should have thrown");
    } catch (err) {
      expect((err as { name: string }).name).toBe("TTSError");
      expect((err as { reason: string }).reason).toBe("missing_content_type");
    }
  });

  it("throws TTSError(invalid_content_type) for text/html", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(
      makeResponse("<html>", {
        status: 200,
        headers: { "Content-Type": "text/html" },
      })
    );

    try {
      await requestTTS({ text: "t" });
      expect.unreachable("should have thrown");
    } catch (err) {
      expect((err as { name: string }).name).toBe("TTSError");
      expect((err as { reason: string }).reason).toBe("invalid_content_type");
    }
  });

  it("throws TTSError(empty_body) when blob size is 0", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(
      makeResponse("", {
        status: 200,
        headers: { "Content-Type": "audio/wav" },
      })
    );

    try {
      await requestTTS({ text: "t" });
      expect.unreachable("should have thrown");
    } catch (err) {
      expect((err as { name: string }).name).toBe("TTSError");
      expect((err as { reason: string }).reason).toBe("empty_body");
    }
  });

  // --- Error: HTTP errors ---

  it("throws TTSError(http_error) for 504 response", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(
      makeResponse(null, { status: 504, statusText: "Gateway Timeout" })
    );

    try {
      await requestTTS({ text: "t" });
      expect.unreachable("should have thrown");
    } catch (err) {
      expect((err as { name: string }).name).toBe("TTSError");
      expect((err as { reason: string }).reason).toBe("http_error");
      expect((err as { status: number }).status).toBe(504);
    }
  });

  it("throws TTSError(http_error) for 400 response", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockResolvedValue(
      makeResponse(null, { status: 400, statusText: "Bad Request" })
    );

    try {
      await requestTTS({ text: "t" });
      expect.unreachable("should have thrown");
    } catch (err) {
      expect((err as { name: string }).name).toBe("TTSError");
      expect((err as { reason: string }).reason).toBe("http_error");
      expect((err as { status: number }).status).toBe(400);
    }
  });

  // --- Error: AbortError re-thrown as-is ---

  it("re-throws AbortError as-is (not wrapped in TTSError)", async () => {
    const requestTTS = await loadRequestTTS();
    // Use a plain Error with name="AbortError" because jsdom's DOMException
    // does not extend Error, which matches the real fetch AbortError behavior
    // in Node/jsdom test environments.
    const abortError = Object.assign(new Error("The operation was aborted"), {
      name: "AbortError",
    });
    fetchSpy.mockRejectedValue(abortError);

    try {
      await requestTTS({ text: "t" });
      expect.unreachable("should have thrown");
    } catch (err) {
      expect((err as Error).name).toBe("AbortError");
      // Should NOT be wrapped in TTSError
      expect((err as { reason?: string }).reason).toBeUndefined();
    }
  });

  // --- Error: other fetch error ---

  it("wraps TypeError (network error) in TTSError with network_error reason", async () => {
    const requestTTS = await loadRequestTTS();
    fetchSpy.mockRejectedValue(new TypeError("Failed to fetch"));

    try {
      await requestTTS({ text: "t" });
      expect.unreachable("should have thrown");
    } catch (err) {
      expect((err as { name: string }).name).toBe("TTSError");
      expect((err as { reason: string }).reason).toBe("network_error");
    }
  });
});
