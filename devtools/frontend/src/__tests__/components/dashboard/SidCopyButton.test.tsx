import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act, fireEvent, waitFor } from "@testing-library/react";

import SidCopyButton from "@/components/dashboard/SidCopyButton";

describe("SidCopyButton", () => {
  let writeTextMock: ReturnType<typeof vi.fn>;
  let originalClipboard: PropertyDescriptor | undefined;

  beforeEach(() => {
    writeTextMock = vi.fn().mockResolvedValue(undefined);
    originalClipboard = Object.getOwnPropertyDescriptor(navigator, "clipboard");
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: writeTextMock },
    });
  });

  afterEach(() => {
    if (originalClipboard) {
      Object.defineProperty(navigator, "clipboard", originalClipboard);
    } else {
      try {
        delete (navigator as unknown as { clipboard?: unknown }).clipboard;
      } catch {
        // ignore
      }
    }
    vi.restoreAllMocks();
  });

  it("クリックで clipboard にコピーされる", async () => {
    render(<SidCopyButton sessionId="abc-123" />);

    fireEvent.click(screen.getByRole("button"));

    await waitFor(() => {
      expect(writeTextMock).toHaveBeenCalledWith("abc-123");
    });
  });

  it("コピー成功後にラベルが「Copied」に変化する", async () => {
    render(<SidCopyButton sessionId="abc-123" />);

    fireEvent.click(screen.getByRole("button"));

    await waitFor(() => {
      expect(screen.getByRole("button")).toHaveTextContent("Copied");
    });
  });

  it("2 秒後にラベルが「SID」に戻る", async () => {
    vi.useFakeTimers();
    try {
      render(<SidCopyButton sessionId="abc-123" />);

      // click すると writeText() の Promise が microtask に積まれる
      await act(async () => {
        fireEvent.click(screen.getByRole("button"));
        // writeText の Promise（microtask）を flush して setCopied(true) を反映させる
        // タイマーは進めずに microtask だけを処理する
        await Promise.resolve();
        await Promise.resolve();
      });

      // この時点で "Copied" になっている（2秒タイマーはまだ進んでいない）
      expect(screen.getByRole("button")).toHaveTextContent("Copied");

      // 2 秒の setTimeout を明示的に進めて "SID" に戻る
      await act(async () => {
        vi.advanceTimersByTime(2000);
      });

      expect(screen.getByRole("button")).toHaveTextContent("SID");
    } finally {
      vi.useRealTimers();
    }
  });

  it("sessionId が null の場合は disabled", () => {
    render(<SidCopyButton sessionId={null} />);
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("clipboard API 失敗時は execCommand にフォールバック", async () => {
    writeTextMock.mockRejectedValueOnce(new Error("not allowed"));
    const execMock = vi.fn().mockReturnValue(true);
    const originalExec = document.execCommand;
    (document as unknown as { execCommand: typeof execMock }).execCommand = execMock;

    try {
      render(<SidCopyButton sessionId="abc-123" />);

      fireEvent.click(screen.getByRole("button"));

      await waitFor(() => {
        expect(execMock).toHaveBeenCalledWith("copy");
      });
      expect(writeTextMock).toHaveBeenCalled();
      // フォールバック成功でもラベルは Copied
      await waitFor(() => {
        expect(screen.getByRole("button")).toHaveTextContent("Copied");
      });
    } finally {
      document.execCommand = originalExec;
    }
  });
});
