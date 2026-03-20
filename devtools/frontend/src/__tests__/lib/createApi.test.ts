import { describe, it, expect, vi, beforeEach } from "vitest";
import { validateProjectName, createProjectStream, openInVSCode } from "@/lib/createApi";

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("validateProjectName", () => {
  it("returns validation result on success", async () => {
    const mockResponse = { valid: true, path: "/home/user/projects/my-app" };
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    });

    const result = await validateProjectName("my-app");

    expect(global.fetch).toHaveBeenCalledWith(
      "/api/projects/validate?name=my-app",
      { signal: undefined }
    );
    expect(result).toEqual(mockResponse);
  });

  it("encodes special characters in the name", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ valid: true, path: "/tmp/a b" }),
    });

    await validateProjectName("a b");

    expect(global.fetch).toHaveBeenCalledWith(
      "/api/projects/validate?name=a%20b",
      { signal: undefined }
    );
  });

  it("throws error when response is not ok", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ error: "Name already exists" }),
    });

    await expect(validateProjectName("existing")).rejects.toThrow(
      "Name already exists"
    );
  });

  it("throws fallback message when error field is missing", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({}),
    });

    await expect(validateProjectName("bad")).rejects.toThrow(
      "Validation request failed"
    );
  });

  it("passes AbortSignal to fetch", async () => {
    const controller = new AbortController();
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ valid: true, path: "/tmp/x" }),
    });

    await validateProjectName("x", controller.signal);

    expect(global.fetch).toHaveBeenCalledWith(
      expect.any(String),
      { signal: controller.signal }
    );
  });
});

describe("createProjectStream", () => {
  it("returns the Response object from fetch", async () => {
    const mockResponse = { ok: true, body: {} } as unknown as Response;
    global.fetch = vi.fn().mockResolvedValue(mockResponse);

    const result = await createProjectStream({
      name: "test-project",
      description: "A test",
      services: ["database"],
    });

    expect(result).toBe(mockResponse);
    expect(global.fetch).toHaveBeenCalledWith(
      "/api/projects/create/stream",
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: "test-project",
          description: "A test",
          services: ["database"],
        }),
        signal: undefined,
      }
    );
  });

  it("passes AbortSignal to fetch", async () => {
    const controller = new AbortController();
    global.fetch = vi.fn().mockResolvedValue({ ok: true } as Response);

    await createProjectStream(
      { name: "p", description: "", services: [] },
      controller.signal
    );

    expect(global.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ signal: controller.signal })
    );
  });
});

describe("openInVSCode", () => {
  it("returns success response", async () => {
    const mockResponse = { success: true, message: "Opened" };
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    });

    const result = await openInVSCode("/home/user/projects/my-app");

    expect(global.fetch).toHaveBeenCalledWith("/api/projects/open", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: "/home/user/projects/my-app" }),
    });
    expect(result).toEqual(mockResponse);
  });

  it("throws error when response is not ok", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ error: "VS Code not found" }),
    });

    await expect(openInVSCode("/tmp/x")).rejects.toThrow("VS Code not found");
  });

  it("throws fallback message when error field is missing", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({}),
    });

    await expect(openInVSCode("/tmp/x")).rejects.toThrow(
      "Failed to open project"
    );
  });
});
