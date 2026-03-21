import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ProjectDeleteList from "@/components/ProjectDeleteList";
import type { ProjectInfo } from "@/types";

const sampleProjects: ProjectInfo[] = [
  { name: "project-alpha", path: "/home/user/projects/alpha" },
  { name: "project-beta", path: "/home/user/projects/beta" },
];

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("ProjectDeleteList", () => {
  it("displays project names and paths", () => {
    render(
      <ProjectDeleteList
        projects={sampleProjects}
        onDelete={vi.fn()}
        deletingPath={null}
      />
    );

    expect(screen.getByText("project-alpha")).toBeInTheDocument();
    expect(screen.getByText("/home/user/projects/alpha")).toBeInTheDocument();
    expect(screen.getByText("project-beta")).toBeInTheDocument();
    expect(screen.getByText("/home/user/projects/beta")).toBeInTheDocument();
  });

  it("calls onDelete with project path when delete button is clicked", async () => {
    const onDelete = vi.fn();
    render(
      <ProjectDeleteList
        projects={sampleProjects}
        onDelete={onDelete}
        deletingPath={null}
      />
    );

    const deleteButtons = screen.getAllByText("削除");
    await userEvent.click(deleteButtons[0]);

    expect(onDelete).toHaveBeenCalledWith("/home/user/projects/alpha");
  });

  it("disables the button for the project being deleted", () => {
    render(
      <ProjectDeleteList
        projects={sampleProjects}
        onDelete={vi.fn()}
        deletingPath="/home/user/projects/alpha"
      />
    );

    const deletingButton = screen.getByText("削除中...");
    expect(deletingButton).toBeDisabled();

    const normalButton = screen.getByText("削除");
    expect(normalButton).not.toBeDisabled();
  });
});
