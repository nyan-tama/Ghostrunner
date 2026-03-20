import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ProjectForm from "@/components/create/ProjectForm";

const mockOnNameChange = vi.fn();

// useProjectValidation をモック
vi.mock("@/hooks/useProjectValidation", () => ({
  useProjectValidation: () => ({
    state: mockValidationState,
    onNameChange: mockOnNameChange,
  }),
}));

// ServiceSelector をモックして簡略化
vi.mock("@/components/create/ServiceSelector", () => ({
  default: ({ selected, onChange }: { selected: string[]; onChange: (s: string[]) => void }) => (
    <div data-testid="service-selector">
      <span>{selected.join(",")}</span>
      <button onClick={() => onChange([...selected, "database"])}>add-db</button>
    </div>
  ),
}));

let mockValidationState = {
  isValidating: false,
  valid: null as boolean | null,
  path: "",
  error: "",
};

beforeEach(() => {
  vi.clearAllMocks();
  mockValidationState = {
    isValidating: false,
    valid: null,
    path: "",
    error: "",
  };
});

describe("ProjectForm", () => {
  it("renders project name input and submit button", () => {
    render(<ProjectForm onSubmit={vi.fn()} />);

    expect(screen.getByLabelText("Project Name")).toBeInTheDocument();
    expect(screen.getByText("Create Project")).toBeInTheDocument();
  });

  it("calls onNameChange when user types in the name input", async () => {
    render(<ProjectForm onSubmit={vi.fn()} />);

    const input = screen.getByLabelText("Project Name");
    await userEvent.type(input, "a");

    expect(mockOnNameChange).toHaveBeenCalledWith("a");
  });

  it("disables submit button when name is empty", () => {
    render(<ProjectForm onSubmit={vi.fn()} />);

    const button = screen.getByText("Create Project");
    expect(button).toBeDisabled();
  });

  it("disables submit button when validation is in progress", () => {
    mockValidationState = {
      isValidating: true,
      valid: null,
      path: "",
      error: "",
    };

    render(<ProjectForm onSubmit={vi.fn()} initialName="test" />);

    const button = screen.getByText("Create Project");
    expect(button).toBeDisabled();
  });

  it("disables submit button when validation result is false", () => {
    mockValidationState = {
      isValidating: false,
      valid: false,
      path: "",
      error: "Name already exists",
    };

    render(<ProjectForm onSubmit={vi.fn()} initialName="existing" />);

    const button = screen.getByText("Create Project");
    expect(button).toBeDisabled();
  });

  it("enables submit button when name is valid", () => {
    mockValidationState = {
      isValidating: false,
      valid: true,
      path: "/home/user/projects/my-app",
      error: "",
    };

    render(<ProjectForm onSubmit={vi.fn()} initialName="my-app" />);

    const button = screen.getByText("Create Project");
    expect(button).toBeEnabled();
  });

  it("calls onSubmit with trimmed name, description and services when submitted", async () => {
    mockValidationState = {
      isValidating: false,
      valid: true,
      path: "/tmp/test",
      error: "",
    };

    const onSubmit = vi.fn();
    render(
      <ProjectForm
        onSubmit={onSubmit}
        initialName="my-app"
        initialDescription="A great app"
      />
    );

    const button = screen.getByText("Create Project");
    await userEvent.click(button);

    expect(onSubmit).toHaveBeenCalledWith("my-app", "A great app", []);
  });

  it("shows 'Validating...' when isValidating is true", () => {
    mockValidationState = {
      isValidating: true,
      valid: null,
      path: "",
      error: "",
    };

    render(<ProjectForm onSubmit={vi.fn()} initialName="test" />);

    expect(screen.getByText("Validating...")).toBeInTheDocument();
  });

  it("shows validation path when valid is true", () => {
    mockValidationState = {
      isValidating: false,
      valid: true,
      path: "/home/user/projects/my-app",
      error: "",
    };

    render(<ProjectForm onSubmit={vi.fn()} initialName="my-app" />);

    expect(
      screen.getByText(/Will be created at:.*\/home\/user\/projects\/my-app/)
    ).toBeInTheDocument();
  });

  it("shows validation error when valid is false", () => {
    mockValidationState = {
      isValidating: false,
      valid: false,
      path: "",
      error: "Directory already exists",
    };

    render(<ProjectForm onSubmit={vi.fn()} initialName="bad" />);

    expect(screen.getByText("Directory already exists")).toBeInTheDocument();
  });
});
