import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import CreateProgress from "@/components/create/CreateProgress";
import type { CreateStep } from "@/types";

function makeSteps(overrides: Partial<Record<string, CreateStep["status"]>> = {}): CreateStep[] {
  const defaults: CreateStep[] = [
    { id: "step1", label: "Step One", status: "pending" },
    { id: "step2", label: "Step Two", status: "pending" },
    { id: "step3", label: "Step Three", status: "pending" },
  ];
  return defaults.map((s) => ({
    ...s,
    status: overrides[s.id] ?? s.status,
  }));
}

describe("CreateProgress", () => {
  it("renders all step labels", () => {
    const steps = makeSteps();
    render(<CreateProgress steps={steps} progress={0} />);

    expect(screen.getByText("Step One")).toBeInTheDocument();
    expect(screen.getByText("Step Two")).toBeInTheDocument();
    expect(screen.getByText("Step Three")).toBeInTheDocument();
  });

  it("displays progress percentage text", () => {
    render(<CreateProgress steps={makeSteps()} progress={45} />);

    expect(screen.getByText("45%")).toBeInTheDocument();
    expect(screen.getByText("Progress")).toBeInTheDocument();
  });

  it("sets progress bar width style", () => {
    const { container } = render(
      <CreateProgress steps={makeSteps()} progress={70} />
    );

    const bar = container.querySelector(".bg-blue-500");
    expect(bar).toBeInTheDocument();
    expect(bar).toHaveStyle({ width: "70%" });
  });

  it("applies green color for done steps", () => {
    const steps = makeSteps({ step1: "done" });
    render(<CreateProgress steps={steps} progress={30} />);

    const stepOne = screen.getByText("Step One");
    expect(stepOne.className).toContain("text-green-700");
  });

  it("applies blue color and font-medium for active steps", () => {
    const steps = makeSteps({ step2: "active" });
    render(<CreateProgress steps={steps} progress={50} />);

    const stepTwo = screen.getByText("Step Two");
    expect(stepTwo.className).toContain("text-blue-700");
    expect(stepTwo.className).toContain("font-medium");
  });

  it("applies red color for error steps", () => {
    const steps = makeSteps({ step3: "error" });
    render(<CreateProgress steps={steps} progress={60} />);

    const stepThree = screen.getByText("Step Three");
    expect(stepThree.className).toContain("text-red-700");
  });

  it("applies gray color for pending steps", () => {
    const steps = makeSteps();
    render(<CreateProgress steps={steps} progress={0} />);

    const stepOne = screen.getByText("Step One");
    expect(stepOne.className).toContain("text-gray-400");
  });

  it("renders done icon (svg with green class) for done steps", () => {
    const steps = makeSteps({ step1: "done" });
    const { container } = render(
      <CreateProgress steps={steps} progress={30} />
    );

    const greenSvg = container.querySelector("svg.text-green-500");
    expect(greenSvg).toBeInTheDocument();
  });

  it("renders error icon (svg with red class) for error steps", () => {
    const steps = makeSteps({ step1: "error" });
    const { container } = render(
      <CreateProgress steps={steps} progress={30} />
    );

    const redSvg = container.querySelector("svg.text-red-500");
    expect(redSvg).toBeInTheDocument();
  });

  it("renders spinner for active steps", () => {
    const steps = makeSteps({ step1: "active" });
    const { container } = render(
      <CreateProgress steps={steps} progress={30} />
    );

    const spinner = container.querySelector(".animate-spin");
    expect(spinner).toBeInTheDocument();
  });

  it("renders circle border for pending steps", () => {
    const steps = makeSteps();
    const { container } = render(
      <CreateProgress steps={steps} progress={0} />
    );

    const circles = container.querySelectorAll(".border-gray-300");
    expect(circles.length).toBe(3);
  });
});
