import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import SessionPicker from "@/components/dashboard/SessionPicker";
import type { ChatSession } from "@/types/chat";

const sessions: ChatSession[] = [
  { id: "abcdef0123", title: "Plan review", timestamp: new Date().toISOString() },
  { id: "1234567890", title: "Refactor", timestamp: new Date().toISOString() },
];

describe("SessionPicker", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("session 一覧が表示される", async () => {
    const user = userEvent.setup();
    render(
      <SessionPicker
        sessions={sessions}
        currentSessionId={null}
        onSwitch={vi.fn()}
        onNewSession={vi.fn()}
      />
    );

    // ドロップダウンを開く
    await user.click(screen.getByRole("button", { expanded: false }));

    expect(screen.getByText("Plan review")).toBeInTheDocument();
    expect(screen.getByText("Refactor")).toBeInTheDocument();
  });

  it("session 選択で onSwitch が呼ばれる", async () => {
    const user = userEvent.setup();
    const onSwitch = vi.fn();

    render(
      <SessionPicker
        sessions={sessions}
        currentSessionId={null}
        onSwitch={onSwitch}
        onNewSession={vi.fn()}
      />
    );

    await user.click(screen.getByRole("button", { expanded: false }));
    await user.click(screen.getByText("Refactor"));

    expect(onSwitch).toHaveBeenCalledWith("1234567890");
  });

  it("新規 session ボタンで onNewSession が呼ばれる", async () => {
    const user = userEvent.setup();
    const onNewSession = vi.fn();

    render(
      <SessionPicker
        sessions={sessions}
        currentSessionId={null}
        onSwitch={vi.fn()}
        onNewSession={onNewSession}
      />
    );

    await user.click(screen.getByRole("button", { expanded: false }));
    await user.click(screen.getByText("+ 新規 session"));

    expect(onNewSession).toHaveBeenCalledTimes(1);
  });

  it("current session が aria-selected でハイライトされる", async () => {
    const user = userEvent.setup();

    render(
      <SessionPicker
        sessions={sessions}
        currentSessionId="abcdef0123"
        onSwitch={vi.fn()}
        onNewSession={vi.fn()}
      />
    );

    await user.click(screen.getByRole("button", { expanded: false }));

    const selected = screen.getByRole("option", { selected: true });
    expect(selected).toHaveTextContent("Plan review");
  });

  it("sessions が空の場合は『セッションなし』表示", async () => {
    const user = userEvent.setup();

    render(
      <SessionPicker
        sessions={[]}
        currentSessionId={null}
        onSwitch={vi.fn()}
        onNewSession={vi.fn()}
      />
    );

    await user.click(screen.getByRole("button", { expanded: false }));

    expect(screen.getByText("セッションなし")).toBeInTheDocument();
  });

  it("ドロップダウン展開時に onOpen が呼ばれる", async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();

    render(
      <SessionPicker
        sessions={sessions}
        currentSessionId={null}
        onSwitch={vi.fn()}
        onNewSession={vi.fn()}
        onOpen={onOpen}
      />
    );

    await user.click(screen.getByRole("button", { expanded: false }));
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("disabled 時は操作不可（ドロップダウンが開かない）", async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();

    render(
      <SessionPicker
        sessions={sessions}
        currentSessionId={null}
        onSwitch={vi.fn()}
        onNewSession={vi.fn()}
        onOpen={onOpen}
        disabled
      />
    );

    const trigger = screen.getByRole("button");
    expect(trigger).toBeDisabled();

    // disabled だがクリックを試行（変化しない）
    await user.click(trigger);
    expect(onOpen).not.toHaveBeenCalled();
    expect(screen.queryByText("Plan review")).not.toBeInTheDocument();
  });
});
