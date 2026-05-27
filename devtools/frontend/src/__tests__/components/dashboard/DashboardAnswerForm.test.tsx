import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

// Mock dashboardApi
const mockSubmitAnswer = vi.fn();
vi.mock("@/lib/dashboardApi", () => ({
  fetchDashboardState: vi.fn(),
  submitAnswer: (...args: unknown[]) => mockSubmitAnswer(...args),
}));

import DashboardAnswerForm from "@/components/dashboard/DashboardAnswerForm";
import type { UnansweredItem } from "@/types/dashboard";

const baseItem: UnansweredItem = {
  planPath: "/test/plan.md",
  lineStart: 10,
  lineEnd: 15,
  questionText: "Which approach do you prefer?",
  heading: "Design Decision",
};

describe("DashboardAnswerForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("builds Question from UnansweredItem with options:[], multiSelect:false", () => {
    const onAnswered = vi.fn();
    render(
      <DashboardAnswerForm
        projectPath="/test/project"
        item={baseItem}
        onAnswered={onAnswered}
      />
    );

    // The question text should be displayed
    expect(screen.getByText("Which approach do you prefer?")).toBeInTheDocument();

    // The heading should be displayed
    expect(screen.getByText("Design Decision")).toBeInTheDocument();

    // Since options is empty, no option buttons should exist
    // Only the submit button and input should be present
    expect(screen.getByPlaceholderText("自由テキストで回答...")).toBeInTheDocument();
  });

  it("submits free text input and triggers onSubmit", async () => {
    const user = userEvent.setup();
    const onAnswered = vi.fn();
    mockSubmitAnswer.mockResolvedValue({ success: true });

    render(
      <DashboardAnswerForm
        projectPath="/test/project"
        item={baseItem}
        onAnswered={onAnswered}
      />
    );

    const input = screen.getByPlaceholderText("自由テキストで回答...");
    await user.type(input, "Option A is better");

    const submitButton = screen.getByRole("button", { name: "送信" });
    await user.click(submitButton);

    expect(mockSubmitAnswer).toHaveBeenCalledWith({
      projectPath: "/test/project",
      planPath: "/test/plan.md",
      lineStart: 10,
      answer: "Option A is better",
    });
  });

  it("handles heading null/undefined without crashing", () => {
    const onAnswered = vi.fn();
    const itemNoHeading: UnansweredItem = {
      ...baseItem,
      heading: undefined as unknown as string,
    };

    // Should not throw
    render(
      <DashboardAnswerForm
        projectPath="/test/project"
        item={itemNoHeading}
        onAnswered={onAnswered}
      />
    );

    // The question should still render
    expect(screen.getByText("Which approach do you prefer?")).toBeInTheDocument();

    // The header should fallback to "Question" (AnswerForm default for empty header)
    expect(screen.getByText("Question")).toBeInTheDocument();
  });

  it("disables submit button when isSubmitting", async () => {
    const user = userEvent.setup();
    const onAnswered = vi.fn();

    // Make submitAnswer hang (never resolves)
    mockSubmitAnswer.mockReturnValue(new Promise(() => {}));

    render(
      <DashboardAnswerForm
        projectPath="/test/project"
        item={baseItem}
        onAnswered={onAnswered}
      />
    );

    const input = screen.getByPlaceholderText("自由テキストで回答...");
    await user.type(input, "answer text");

    const submitButton = screen.getByRole("button", { name: "送信" });
    await user.click(submitButton);

    // After click, the button should become disabled (showing "送信中...")
    expect(screen.getByRole("button", { name: "送信中..." })).toBeDisabled();
  });
});
