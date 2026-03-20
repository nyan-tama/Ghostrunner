import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import CreateComplete from "@/components/create/CreateComplete";

// openInVSCode をモック
vi.mock("@/lib/createApi", () => ({
  openInVSCode: vi.fn(),
}));

import { openInVSCode } from "@/lib/createApi";

const mockOpenInVSCode = vi.mocked(openInVSCode);

const defaultProject = {
  name: "my-app",
  path: "/home/user/projects/my-app",
};

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("CreateComplete", () => {
  it("displays project name and path", () => {
    render(
      <CreateComplete project={defaultProject} onCreateAnother={vi.fn()} />
    );

    expect(screen.getByText("my-app is ready for development")).toBeInTheDocument();
    expect(screen.getByText("/home/user/projects/my-app")).toBeInTheDocument();
  });

  it("displays 'Project Created' heading", () => {
    render(
      <CreateComplete project={defaultProject} onCreateAnother={vi.fn()} />
    );

    expect(screen.getByText("Project Created")).toBeInTheDocument();
  });

  it("renders 'Open in VS Code' and 'Create Another' buttons", () => {
    render(
      <CreateComplete project={defaultProject} onCreateAnother={vi.fn()} />
    );

    expect(screen.getByText("Open in VS Code")).toBeInTheDocument();
    expect(screen.getByText("Create Another")).toBeInTheDocument();
  });

  it("calls onCreateAnother when 'Create Another' is clicked", async () => {
    const onCreateAnother = vi.fn();
    render(
      <CreateComplete project={defaultProject} onCreateAnother={onCreateAnother} />
    );

    await userEvent.click(screen.getByText("Create Another"));

    expect(onCreateAnother).toHaveBeenCalledOnce();
  });

  it("calls openInVSCode when 'Open in VS Code' is clicked", async () => {
    mockOpenInVSCode.mockResolvedValue({ success: true, message: "Opened" });
    render(
      <CreateComplete project={defaultProject} onCreateAnother={vi.fn()} />
    );

    await userEvent.click(screen.getByText("Open in VS Code"));

    expect(mockOpenInVSCode).toHaveBeenCalledWith("/home/user/projects/my-app");
  });

  it("shows error message when openInVSCode fails", async () => {
    mockOpenInVSCode.mockRejectedValue(new Error("VS Code not found"));
    render(
      <CreateComplete project={defaultProject} onCreateAnother={vi.fn()} />
    );

    await userEvent.click(screen.getByText("Open in VS Code"));

    expect(await screen.findByText("VS Code not found")).toBeInTheDocument();
  });

  it("shows error message when openInVSCode returns success: false", async () => {
    mockOpenInVSCode.mockResolvedValue({
      success: false,
      message: "Could not launch editor",
    });
    render(
      <CreateComplete project={defaultProject} onCreateAnother={vi.fn()} />
    );

    await userEvent.click(screen.getByText("Open in VS Code"));

    expect(
      await screen.findByText("Could not launch editor")
    ).toBeInTheDocument();
  });
});
