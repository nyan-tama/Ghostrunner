import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ProjectRegister from "@/components/patrol/ProjectRegister";

// fetchProjects モック
const mockFetchProjects = vi.fn();

vi.mock("@/lib/api", () => ({
  fetchProjects: (...args: unknown[]) => mockFetchProjects(...args),
}));

describe("ProjectRegister", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchProjects.mockResolvedValue({
      success: true,
      projects: [
        { name: "Project A", path: "/proj/a" },
        { name: "Project B", path: "/proj/b" },
        { name: "Project C", path: "/proj/c" },
      ],
    });
  });

  it("renders the header and add button", () => {
    render(<ProjectRegister registeredPaths={[]} onRegister={vi.fn()} />);

    expect(screen.getByText("巡回対象プロジェクト")).toBeInTheDocument();
    expect(screen.getByText("追加")).toBeInTheDocument();
  });

  it("does not fetch projects until add button is clicked", () => {
    render(<ProjectRegister registeredPaths={[]} onRegister={vi.fn()} />);

    expect(mockFetchProjects).not.toHaveBeenCalled();
  });

  it("fetches and displays available projects when add button clicked", async () => {
    render(<ProjectRegister registeredPaths={[]} onRegister={vi.fn()} />);

    await userEvent.click(screen.getByText("追加"));

    await waitFor(() => {
      expect(screen.getByText("Project A")).toBeInTheDocument();
    });

    expect(screen.getByText("Project B")).toBeInTheDocument();
    expect(screen.getByText("Project C")).toBeInTheDocument();
  });

  it("filters out already registered projects", async () => {
    render(
      <ProjectRegister
        registeredPaths={["/proj/a", "/proj/c"]}
        onRegister={vi.fn()}
      />
    );

    await userEvent.click(screen.getByText("追加"));

    await waitFor(() => {
      expect(screen.getByText("Project B")).toBeInTheDocument();
    });

    expect(screen.queryByText("Project A")).not.toBeInTheDocument();
    expect(screen.queryByText("Project C")).not.toBeInTheDocument();
  });

  it("shows empty message when all projects are registered", async () => {
    render(
      <ProjectRegister
        registeredPaths={["/proj/a", "/proj/b", "/proj/c"]}
        onRegister={vi.fn()}
      />
    );

    await userEvent.click(screen.getByText("追加"));

    await waitFor(() => {
      expect(screen.getByText("追加可能なプロジェクトがありません")).toBeInTheDocument();
    });
  });

  it("calls onRegister and closes panel when a project is selected", async () => {
    const onRegister = vi.fn();
    render(
      <ProjectRegister registeredPaths={[]} onRegister={onRegister} />
    );

    await userEvent.click(screen.getByText("追加"));

    await waitFor(() => {
      expect(screen.getByText("Project B")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText("Project B"));

    expect(onRegister).toHaveBeenCalledWith("/proj/b");
    // Panel should be closed after selection
    expect(screen.queryByText("Project A")).not.toBeInTheDocument();
  });

  it("toggles panel open/close with the button", async () => {
    render(<ProjectRegister registeredPaths={[]} onRegister={vi.fn()} />);

    // Open
    await userEvent.click(screen.getByText("追加"));
    await waitFor(() => {
      expect(screen.getByText("Project A")).toBeInTheDocument();
    });
    expect(screen.getByText("閉じる")).toBeInTheDocument();

    // Close
    await userEvent.click(screen.getByText("閉じる"));
    expect(screen.queryByText("Project A")).not.toBeInTheDocument();
    expect(screen.getByText("追加")).toBeInTheDocument();
  });
});
