import { describe, it, expect, vi, beforeEach } from "vitest";
import { destroyProject } from "@/lib/api";

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("destroyProject", () => {
  it("sends POST request with correct URL, method, and body", async () => {
    const mockResponse = { success: true };
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    });

    const result = await destroyProject("/home/user/projects/my-app");

    expect(global.fetch).toHaveBeenCalledWith("/api/projects/destroy", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: "/home/user/projects/my-app" }),
    });
    expect(result).toEqual(mockResponse);
  });

  it("throws error with server message when response is not ok", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: () => Promise.resolve({ error: "Project not found" }),
    });

    await expect(destroyProject("/tmp/nonexistent")).rejects.toThrow(
      "Project not found"
    );
  });

  it("throws fallback message when error field is missing", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({}),
    });

    await expect(destroyProject("/tmp/bad")).rejects.toThrow(
      "Failed to destroy project (500)"
    );
  });
});
