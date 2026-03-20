import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ProjectCard from "@/components/patrol/ProjectCard";
import type { PatrolProject, PatrolProjectState } from "@/types/patrol";

// AnswerForm をモック
vi.mock("@/components/patrol/AnswerForm", () => ({
  default: ({ projectPath, onSubmit }: { projectPath: string; onSubmit: (path: string, answer: string) => void }) => (
    <div data-testid="answer-form">
      <button onClick={() => onSubmit(projectPath, "test-answer")}>mock-submit</button>
    </div>
  ),
}));

const baseProject: PatrolProject = {
  path: "/home/user/my-project",
  name: "My Project",
};

const makeState = (overrides: Partial<PatrolProjectState> = {}): PatrolProjectState => ({
  project_path: "/home/user/my-project",
  status: "idle",
  recent_commits: [],
  pending_tasks: 0,
  ...overrides,
});

describe("ProjectCard", () => {
  it("renders project name and path", () => {
    render(
      <ProjectCard
        project={baseProject}
        state={undefined}
        onRemove={vi.fn()}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    expect(screen.getByText("My Project")).toBeInTheDocument();
    expect(screen.getByText("/home/user/my-project")).toBeInTheDocument();
  });

  it("shows 'unknown' badge when state is undefined", () => {
    render(
      <ProjectCard
        project={baseProject}
        state={undefined}
        onRemove={vi.fn()}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    expect(screen.getByText("不明")).toBeInTheDocument();
  });

  it("shows correct status badge for each status", () => {
    const statuses: Array<{ status: PatrolProjectState["status"]; label: string }> = [
      { status: "idle", label: "待機中" },
      { status: "running", label: "実行中" },
      { status: "waiting_approval", label: "承認待ち" },
      { status: "queued", label: "キュー待ち" },
      { status: "completed", label: "完了" },
      { status: "error", label: "エラー" },
    ];

    for (const { status, label } of statuses) {
      const { unmount } = render(
        <ProjectCard
          project={baseProject}
          state={makeState({ status })}
          onRemove={vi.fn()}
          onAnswer={vi.fn()}
          isAnswerSubmitting={false}
        />
      );

      expect(screen.getByText(label)).toBeInTheDocument();
      unmount();
    }
  });

  it("displays recent commits when available", () => {
    const state = makeState({
      recent_commits: ["fix: bug fix", "feat: new feature"],
    });

    render(
      <ProjectCard
        project={baseProject}
        state={state}
        onRemove={vi.fn()}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    expect(screen.getByText("最近のコミット")).toBeInTheDocument();
    expect(screen.getByText("fix: bug fix")).toBeInTheDocument();
    expect(screen.getByText("feat: new feature")).toBeInTheDocument();
  });

  it("does not display commits section when no commits", () => {
    render(
      <ProjectCard
        project={baseProject}
        state={makeState({ recent_commits: [] })}
        onRemove={vi.fn()}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    expect(screen.queryByText("最近のコミット")).not.toBeInTheDocument();
  });

  it("displays pending tasks count when > 0", () => {
    render(
      <ProjectCard
        project={baseProject}
        state={makeState({ pending_tasks: 5 })}
        onRemove={vi.fn()}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    expect(screen.getByText("5")).toBeInTheDocument();
  });

  it("displays error message when present", () => {
    render(
      <ProjectCard
        project={baseProject}
        state={makeState({ status: "error", error: "Something went wrong" })}
        onRemove={vi.fn()}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });

  it("shows AnswerForm when status is waiting_approval and question exists", () => {
    const state = makeState({
      status: "waiting_approval",
      question: {
        question: "Continue?",
        header: "Confirm",
        options: [],
        multiSelect: false,
      },
    });

    render(
      <ProjectCard
        project={baseProject}
        state={state}
        onRemove={vi.fn()}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    expect(screen.getByTestId("answer-form")).toBeInTheDocument();
  });

  it("does not show AnswerForm when status is not waiting_approval", () => {
    const state = makeState({
      status: "running",
      question: {
        question: "Continue?",
        header: "Confirm",
        options: [],
        multiSelect: false,
      },
    });

    render(
      <ProjectCard
        project={baseProject}
        state={state}
        onRemove={vi.fn()}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    expect(screen.queryByTestId("answer-form")).not.toBeInTheDocument();
  });

  it("calls onRemove with project path when remove button is clicked", async () => {
    const onRemove = vi.fn();
    render(
      <ProjectCard
        project={baseProject}
        state={makeState()}
        onRemove={onRemove}
        onAnswer={vi.fn()}
        isAnswerSubmitting={false}
      />
    );

    await userEvent.click(screen.getByText("解除"));
    expect(onRemove).toHaveBeenCalledWith("/home/user/my-project");
  });
});
