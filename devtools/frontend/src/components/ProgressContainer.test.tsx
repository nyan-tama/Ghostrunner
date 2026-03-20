import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import ProgressContainer from "./ProgressContainer";

// OutputTextをモックして検証可能にする
vi.mock("./OutputText", () => ({
  default: ({ text }: { text: string }) => (
    <div data-testid="output-text">{text}</div>
  ),
}));

// 子コンポーネントのモック: テスト対象外の部分を最小限に
vi.mock("./LoadingIndicator", () => ({
  default: ({ visible, text }: { visible: boolean; text: string }) =>
    visible ? <div data-testid="loading-indicator">{text}</div> : null,
}));

vi.mock("./EventList", () => ({
  default: () => <div data-testid="event-list" />,
}));

vi.mock("./QuestionSection", () => ({
  default: ({ visible }: { visible: boolean }) =>
    visible ? <div data-testid="question-section" /> : null,
}));

vi.mock("./PlanApproval", () => ({
  default: ({ visible }: { visible: boolean }) =>
    visible ? <div data-testid="plan-approval" /> : null,
}));

vi.mock("./ContinueSession", () => ({
  default: ({ visible }: { visible: boolean }) =>
    visible ? <div data-testid="continue-session" /> : null,
}));

const defaultProps = {
  visible: true,
  prompt: "test prompt",
  events: [],
  loadingText: "Loading...",
  isLoading: false,
  questions: [],
  showQuestions: false,
  currentQuestionIndex: 0,
  showPlanApproval: false,
  resultOutput: "",
  resultType: null as "success" | "error" | null,
  sessionId: null as string | null,
  totalCost: 0,
  onAnswer: vi.fn(),
  onApprove: vi.fn(),
  onReject: vi.fn(),
  onAbort: vi.fn(),
  canAbort: false,
};

describe("ProgressContainer", () => {
  it("renders nothing when visible is false", () => {
    const { container } = render(
      <ProgressContainer {...defaultProps} visible={false} />
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders container when visible is true", () => {
    const { container } = render(
      <ProgressContainer {...defaultProps} visible={true} />
    );
    expect(container.firstChild).not.toBeNull();
  });

  it("renders OutputText when resultOutput and resultType are both present", () => {
    render(
      <ProgressContainer
        {...defaultProps}
        resultOutput="Command completed successfully"
        resultType="success"
      />
    );

    const outputText = screen.getByTestId("output-text");
    expect(outputText).toBeInTheDocument();
    expect(outputText).toHaveTextContent("Command completed successfully");
  });

  it("applies green background for success resultType", () => {
    const { container } = render(
      <ProgressContainer
        {...defaultProps}
        resultOutput="Success output"
        resultType="success"
      />
    );

    const resultArea = container.querySelector(".bg-green-100");
    expect(resultArea).toBeInTheDocument();
  });

  it("applies red background for error resultType", () => {
    const { container } = render(
      <ProgressContainer
        {...defaultProps}
        resultOutput="Error output"
        resultType="error"
      />
    );

    const resultArea = container.querySelector(".bg-red-100");
    expect(resultArea).toBeInTheDocument();
  });

  it("does not render result area when resultOutput is empty", () => {
    render(
      <ProgressContainer
        {...defaultProps}
        resultOutput=""
        resultType="success"
      />
    );

    expect(screen.queryByTestId("output-text")).not.toBeInTheDocument();
  });

  it("does not render result area when resultType is null", () => {
    render(
      <ProgressContainer
        {...defaultProps}
        resultOutput="Some output"
        resultType={null}
      />
    );

    expect(screen.queryByTestId("output-text")).not.toBeInTheDocument();
  });

  it("renders the prompt text", () => {
    render(
      <ProgressContainer {...defaultProps} prompt="Run build command" />
    );

    expect(screen.getByText("Run build command")).toBeInTheDocument();
  });
});
