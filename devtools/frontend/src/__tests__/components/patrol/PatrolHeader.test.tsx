import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import PatrolHeader from "@/components/patrol/PatrolHeader";

describe("PatrolHeader", () => {
  it("shows start button when not running", () => {
    render(
      <PatrolHeader
        isRunning={false}
        isPolling={false}
        isLoading={false}
        connectionStatus="connected"
        onStart={vi.fn()}
        onStop={vi.fn()}
        onTogglePolling={vi.fn()}
      />
    );

    expect(screen.getByText("巡回開始")).toBeInTheDocument();
    expect(screen.queryByText("巡回停止")).not.toBeInTheDocument();
  });

  it("shows stop button when running", () => {
    render(
      <PatrolHeader
        isRunning={true}
        isPolling={false}
        isLoading={false}
        connectionStatus="connected"
        onStart={vi.fn()}
        onStop={vi.fn()}
        onTogglePolling={vi.fn()}
      />
    );

    expect(screen.getByText("巡回停止")).toBeInTheDocument();
    expect(screen.queryByText("巡回開始")).not.toBeInTheDocument();
  });

  it("calls onStart when start button is clicked", async () => {
    const onStart = vi.fn();
    render(
      <PatrolHeader
        isRunning={false}
        isPolling={false}
        isLoading={false}
        connectionStatus="connected"
        onStart={onStart}
        onStop={vi.fn()}
        onTogglePolling={vi.fn()}
      />
    );

    await userEvent.click(screen.getByText("巡回開始"));
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it("calls onStop when stop button is clicked", async () => {
    const onStop = vi.fn();
    render(
      <PatrolHeader
        isRunning={true}
        isPolling={false}
        isLoading={false}
        connectionStatus="connected"
        onStart={vi.fn()}
        onStop={onStop}
        onTogglePolling={vi.fn()}
      />
    );

    await userEvent.click(screen.getByText("巡回停止"));
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("disables button and shows loading text when isLoading is true", () => {
    render(
      <PatrolHeader
        isRunning={false}
        isPolling={false}
        isLoading={true}
        connectionStatus="connected"
        onStart={vi.fn()}
        onStop={vi.fn()}
        onTogglePolling={vi.fn()}
      />
    );

    const button = screen.getByText("処理中...");
    expect(button).toBeDisabled();
  });

  it("calls onTogglePolling when polling checkbox is changed", async () => {
    const onTogglePolling = vi.fn();
    render(
      <PatrolHeader
        isRunning={false}
        isPolling={false}
        isLoading={false}
        connectionStatus="connected"
        onStart={vi.fn()}
        onStop={vi.fn()}
        onTogglePolling={onTogglePolling}
      />
    );

    await userEvent.click(screen.getByText("ポーリング"));
    expect(onTogglePolling).toHaveBeenCalledTimes(1);
  });

  it("shows polling checkbox as checked when isPolling is true", () => {
    render(
      <PatrolHeader
        isRunning={false}
        isPolling={true}
        isLoading={false}
        connectionStatus="connected"
        onStart={vi.fn()}
        onStop={vi.fn()}
        onTogglePolling={vi.fn()}
      />
    );

    const checkbox = screen.getByRole("checkbox");
    expect(checkbox).toBeChecked();
  });

  it("displays connection status indicator for 'connected'", () => {
    render(
      <PatrolHeader
        isRunning={false}
        isPolling={false}
        isLoading={false}
        connectionStatus="connected"
        onStart={vi.fn()}
        onStop={vi.fn()}
        onTogglePolling={vi.fn()}
      />
    );

    expect(screen.getByText("接続済み")).toBeInTheDocument();
  });

  it("displays connection status indicator for 'connecting'", () => {
    render(
      <PatrolHeader
        isRunning={false}
        isPolling={false}
        isLoading={false}
        connectionStatus="connecting"
        onStart={vi.fn()}
        onStop={vi.fn()}
        onTogglePolling={vi.fn()}
      />
    );

    expect(screen.getByText("接続中...")).toBeInTheDocument();
  });

  it("displays connection status indicator for 'disconnected'", () => {
    render(
      <PatrolHeader
        isRunning={false}
        isPolling={false}
        isLoading={false}
        connectionStatus="disconnected"
        onStart={vi.fn()}
        onStop={vi.fn()}
        onTogglePolling={vi.fn()}
      />
    );

    expect(screen.getByText("切断")).toBeInTheDocument();
  });
});
