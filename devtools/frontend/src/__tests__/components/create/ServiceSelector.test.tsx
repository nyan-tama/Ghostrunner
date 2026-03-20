import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ServiceSelector from "@/components/create/ServiceSelector";
import type { DataService } from "@/types";

describe("ServiceSelector", () => {
  it("renders three checkboxes with labels", () => {
    render(<ServiceSelector selected={[]} onChange={vi.fn()} />);

    expect(screen.getByText("PostgreSQL + GORM")).toBeInTheDocument();
    expect(screen.getByText("Cloudflare R2 / MinIO")).toBeInTheDocument();
    expect(screen.getByText("Redis")).toBeInTheDocument();

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
  });

  it("checks the boxes matching selected services", () => {
    render(
      <ServiceSelector selected={["database", "cache"]} onChange={vi.fn()} />
    );

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes[0]).toBeChecked(); // database
    expect(checkboxes[1]).not.toBeChecked(); // storage
    expect(checkboxes[2]).toBeChecked(); // cache
  });

  it("calls onChange with added service when unchecked box is clicked", async () => {
    const onChange = vi.fn();
    render(<ServiceSelector selected={[]} onChange={onChange} />);

    const checkboxes = screen.getAllByRole("checkbox");
    await userEvent.click(checkboxes[0]); // database

    expect(onChange).toHaveBeenCalledWith(["database"]);
  });

  it("calls onChange with removed service when checked box is clicked", async () => {
    const onChange = vi.fn();
    const selected: DataService[] = ["database", "storage"];
    render(<ServiceSelector selected={selected} onChange={onChange} />);

    const checkboxes = screen.getAllByRole("checkbox");
    await userEvent.click(checkboxes[0]); // uncheck database

    expect(onChange).toHaveBeenCalledWith(["storage"]);
  });

  it("renders descriptions for each service", () => {
    render(<ServiceSelector selected={[]} onChange={vi.fn()} />);

    expect(screen.getByText("Database with migration support")).toBeInTheDocument();
    expect(screen.getByText("Object storage for files")).toBeInTheDocument();
    expect(screen.getByText("In-memory cache and session store")).toBeInTheDocument();
  });
});
