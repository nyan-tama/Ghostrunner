import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import AnswerForm from "@/components/patrol/AnswerForm";
import type { Question } from "@/types";

const baseQuestion: Question = {
  question: "How to proceed?",
  header: "Approval Required",
  options: [
    { label: "Yes", description: "Continue with changes" },
    { label: "No", description: "Abort changes" },
  ],
  multiSelect: false,
};

describe("AnswerForm", () => {
  it("renders question header and text", () => {
    render(
      <AnswerForm
        projectPath="/proj/a"
        question={baseQuestion}
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    expect(screen.getByText("Approval Required")).toBeInTheDocument();
    expect(screen.getByText("How to proceed?")).toBeInTheDocument();
  });

  it("renders option buttons", () => {
    render(
      <AnswerForm
        projectPath="/proj/a"
        question={baseQuestion}
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    expect(screen.getByText("Yes")).toBeInTheDocument();
    expect(screen.getByText("Continue with changes")).toBeInTheDocument();
    expect(screen.getByText("No")).toBeInTheDocument();
    expect(screen.getByText("Abort changes")).toBeInTheDocument();
  });

  it("calls onSubmit with option label immediately for single-select", async () => {
    const onSubmit = vi.fn();
    render(
      <AnswerForm
        projectPath="/proj/a"
        question={baseQuestion}
        isSubmitting={false}
        onSubmit={onSubmit}
      />
    );

    await userEvent.click(screen.getByText("Yes"));

    expect(onSubmit).toHaveBeenCalledWith("/proj/a", "Yes");
  });

  it("supports multi-select: toggles selection and submits joined labels", async () => {
    const onSubmit = vi.fn();
    const multiQuestion: Question = {
      ...baseQuestion,
      multiSelect: true,
    };

    render(
      <AnswerForm
        projectPath="/proj/a"
        question={multiQuestion}
        isSubmitting={false}
        onSubmit={onSubmit}
      />
    );

    // Click two options
    await userEvent.click(screen.getByText("Yes"));
    await userEvent.click(screen.getByText("No"));

    // Click submit button
    await userEvent.click(screen.getByText("選択を送信"));

    expect(onSubmit).toHaveBeenCalledWith("/proj/a", "Yes, No");
  });

  it("submits free text input when typed and submit clicked", async () => {
    const onSubmit = vi.fn();
    render(
      <AnswerForm
        projectPath="/proj/a"
        question={baseQuestion}
        isSubmitting={false}
        onSubmit={onSubmit}
      />
    );

    const input = screen.getByPlaceholderText("自由テキストで回答...");
    await userEvent.type(input, "custom answer");
    await userEvent.click(screen.getByText("送信"));

    expect(onSubmit).toHaveBeenCalledWith("/proj/a", "custom answer");
  });

  it("submits free text on Enter key", async () => {
    const onSubmit = vi.fn();
    render(
      <AnswerForm
        projectPath="/proj/a"
        question={baseQuestion}
        isSubmitting={false}
        onSubmit={onSubmit}
      />
    );

    const input = screen.getByPlaceholderText("自由テキストで回答...");
    await userEvent.type(input, "enter answer{Enter}");

    expect(onSubmit).toHaveBeenCalledWith("/proj/a", "enter answer");
  });

  it("disables buttons and input when isSubmitting is true", () => {
    render(
      <AnswerForm
        projectPath="/proj/a"
        question={baseQuestion}
        isSubmitting={true}
        onSubmit={vi.fn()}
      />
    );

    expect(screen.getByPlaceholderText("自由テキストで回答...")).toBeDisabled();
    expect(screen.getByText("送信中...")).toBeDisabled();
    // Option buttons should also be disabled
    const yesButton = screen.getByText("Yes").closest("button")!;
    expect(yesButton).toBeDisabled();
  });

  it("shows default header 'Question' when header is empty", () => {
    const questionNoHeader: Question = {
      ...baseQuestion,
      header: "",
    };

    render(
      <AnswerForm
        projectPath="/proj/a"
        question={questionNoHeader}
        isSubmitting={false}
        onSubmit={vi.fn()}
      />
    );

    expect(screen.getByText("Question")).toBeInTheDocument();
  });

  it("does not submit when free text is empty and no multi-select", async () => {
    const onSubmit = vi.fn();
    render(
      <AnswerForm
        projectPath="/proj/a"
        question={baseQuestion}
        isSubmitting={false}
        onSubmit={onSubmit}
      />
    );

    await userEvent.click(screen.getByText("送信"));

    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("free text takes priority over multi-select options", async () => {
    const onSubmit = vi.fn();
    const multiQuestion: Question = {
      ...baseQuestion,
      multiSelect: true,
    };

    render(
      <AnswerForm
        projectPath="/proj/a"
        question={multiQuestion}
        isSubmitting={false}
        onSubmit={onSubmit}
      />
    );

    // Select an option
    await userEvent.click(screen.getByText("Yes"));

    // Type custom text
    const input = screen.getByPlaceholderText("自由テキストで回答...");
    await userEvent.type(input, "override");

    // Submit
    await userEvent.click(screen.getByText("選択を送信"));

    // Free text should take priority
    expect(onSubmit).toHaveBeenCalledWith("/proj/a", "override");
  });
});
