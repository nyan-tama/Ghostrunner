import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import EventItem from "./EventItem";
import type { DisplayEvent } from "@/types";

// OutputTextをモックして、渡されたtextの検証を容易にする
vi.mock("./OutputText", () => ({
  default: ({ text }: { text: string }) => (
    <div data-testid="output-text">{text}</div>
  ),
}));

function createEvent(overrides: Partial<DisplayEvent> = {}): DisplayEvent {
  return {
    id: "test-1",
    type: "text",
    title: "Test Title",
    ...overrides,
  };
}

describe("EventItem", () => {
  it("renders OutputText when event.fullText is present", () => {
    const event = createEvent({
      fullText: "This is **full** text content",
    });
    render(<EventItem event={event} />);

    const outputText = screen.getByTestId("output-text");
    expect(outputText).toBeInTheDocument();
    expect(outputText).toHaveTextContent("This is **full** text content");
  });

  it("renders plain text detail when event.fullText is absent but event.detail is present", () => {
    const event = createEvent({
      detail: "plain detail text",
    });
    render(<EventItem event={event} />);

    expect(screen.queryByTestId("output-text")).not.toBeInTheDocument();
    expect(screen.getByText("plain detail text")).toBeInTheDocument();
  });

  it("renders nothing for detail section when both fullText and detail are absent", () => {
    const event = createEvent();
    const { container } = render(<EventItem event={event} />);

    expect(screen.queryByTestId("output-text")).not.toBeInTheDocument();
    // font-mono class is used for the detail div
    const detailDiv = container.querySelector(".font-mono");
    expect(detailDiv).not.toBeInTheDocument();
  });

  it("does not have show more/less toggle (no collapsing feature)", () => {
    const event = createEvent({
      fullText: "Some long text content that might have been collapsible before",
    });
    render(<EventItem event={event} />);

    expect(screen.queryByText(/show more/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/show less/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/more/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/less/i)).not.toBeInTheDocument();
  });

  it("renders purple box when event.type is task and detail exists", () => {
    const event = createEvent({
      type: "task",
      detail: "Task detail content",
    });
    const { container } = render(<EventItem event={event} />);

    const purpleBox = container.querySelector(".bg-purple-50");
    expect(purpleBox).toBeInTheDocument();
    expect(purpleBox!.classList.contains("border-purple-200")).toBe(true);
    // detail is shown in both the plain text area and inside the purple box
    const matches = screen.getAllByText("Task detail content");
    expect(matches.length).toBeGreaterThanOrEqual(1);
    // Verify purple box contains the detail text
    expect(purpleBox!.textContent).toContain("Task detail content");
  });

  it("does not render purple box when event.type is not task", () => {
    const event = createEvent({
      type: "tool",
      detail: "Tool detail content",
    });
    const { container } = render(<EventItem event={event} />);

    const purpleBox = container.querySelector(".bg-purple-50");
    expect(purpleBox).not.toBeInTheDocument();
  });

  it("renders purple dot for task type", () => {
    const event = createEvent({ type: "task" });
    const { container } = render(<EventItem event={event} />);

    const dot = container.querySelector(".bg-purple-500");
    expect(dot).toBeInTheDocument();
  });

  it("renders blue dot for tool type", () => {
    const event = createEvent({ type: "tool" });
    const { container } = render(<EventItem event={event} />);

    const dot = container.querySelector(".bg-blue-500");
    expect(dot).toBeInTheDocument();
  });

  it("renders event title", () => {
    const event = createEvent({ title: "My Event Title" });
    render(<EventItem event={event} />);

    expect(screen.getByText("My Event Title")).toBeInTheDocument();
  });

  it("prefers fullText over detail for the main content area", () => {
    const event = createEvent({
      fullText: "Full text content",
      detail: "Detail content",
    });
    render(<EventItem event={event} />);

    // OutputText should show fullText
    const outputText = screen.getByTestId("output-text");
    expect(outputText).toHaveTextContent("Full text content");

    // The plain detail div (font-mono) should not render since fullText is present
    // But detail still shows in the purple box area if type is not "task"
    // Since this is type "text", no purple box either
  });
});
