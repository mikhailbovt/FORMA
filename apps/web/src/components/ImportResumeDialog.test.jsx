import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ImportResumeDialog, isLinkedInProfileURL } from "./ImportResumeDialog.jsx";

describe("ImportResumeDialog", () => {
  it("validates LinkedIn profile URLs without attempting to fetch them", async () => {
    const user = userEvent.setup();
    render(<ImportResumeDialog open onClose={() => {}} onPreview={vi.fn()} onApply={vi.fn()} />);

    await user.click(screen.getByRole("tab", { name: /LinkedIn/i }));
    const input = screen.getByRole("textbox", { name: /LinkedIn profile URL/i });
    await user.type(input, "https://example.com/in/not-linkedin");
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("button", { name: "Preview import" })).toBeDisabled();
    expect(isLinkedInProfileURL("https://www.linkedin.com/in/ada-lovelace")).toBe(true);
  });

  it("previews a file before applying a safe merge", async () => {
    const user = userEvent.setup();
    const onPreview = vi.fn().mockResolvedValue({
      parser: { id: "json_resume", version: "1" },
      candidate: { title: "Ada CV", document: { basics: { name: "Ada Lovelace" }, experience: [], projects: [], education: [], skills: [] } },
      warnings: [{ code: "review_import", message: "Review imported dates." }],
    });
    const onApply = vi.fn();
    const { container } = render(<ImportResumeDialog open onClose={() => {}} onPreview={onPreview} onApply={onApply} />);
    const file = new File(["{}"], "resume.json", { type: "application/json" });
    await user.upload(container.querySelector('input[type="file"]'), file);
    await user.click(screen.getByRole("button", { name: "Preview import" }));

    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getByText("Review imported dates.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Apply import" }));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({ mode: "merge" }));
  });
});
