import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeSampleResume } from "../data/sampleResume.js";
import { SectionRail } from "./SectionRail.jsx";

describe("SectionRail", () => {
  it("selects resume sections and keeps settings visible", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const onOpenSettings = vi.fn();
    render(<SectionRail document={makeSampleResume().document} selected="basics" onSelect={onSelect} onAdd={vi.fn()} onMove={vi.fn()} onToggle={vi.fn()} onOpenReview={vi.fn()} onOpenTemplates={vi.fn()} onOpenSettings={onOpenSettings} />);

    await user.click(screen.getByRole("button", { name: "Summary" }));
    await user.click(screen.getByRole("button", { name: "AI settings" }));

    expect(onSelect).toHaveBeenCalledWith("summary");
    expect(onOpenSettings).toHaveBeenCalledTimes(1);
  });
});
