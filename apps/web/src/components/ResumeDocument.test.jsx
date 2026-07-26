import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeSampleResume } from "../data/sampleResume.js";
import { ResumeDocument } from "./ResumeDocument.jsx";

function renderDocument(previewMode, { selectable = false, document = makeSampleResume().document, onSelectSection = vi.fn() } = {}) {
  render(
    <ResumeDocument
      document={document}
      selectedSection="experience"
      activeBullet={{ itemId: "exp-sentry", index: 0 }}
      previewMode={previewMode}
      selectable={selectable}
      onSelectSection={onSelectSection}
      onChange={vi.fn()}
      onRewrite={vi.fn()}
      onShorten={vi.fn()}
      onAddMetric={vi.fn()}
      onAddItem={vi.fn()}
      onActiveBullet={vi.fn()}
    />,
  );
}

describe("ResumeDocument preview", () => {
  it("removes editing semantics and writing tools in preview mode", () => {
    renderDocument(true);
    expect(screen.queryByRole("textbox", { name: "Full name" })).not.toBeInTheDocument();
    expect(screen.queryByRole("toolbar", { name: "AI writing actions" })).not.toBeInTheDocument();
    expect(screen.getByText("Alex Morgan")).toBeInTheDocument();
  });

  it("keeps inline fields editable in editor mode", () => {
    renderDocument(false);
    expect(screen.getByRole("textbox", { name: "Full name" })).toBeInTheDocument();
  });

  it("keeps split preview read-only while allowing section selection", async () => {
    const user = userEvent.setup();
    const onSelectSection = vi.fn();
    renderDocument(true, { selectable: true, onSelectSection });

    await user.click(screen.getByText("Alex Morgan"));
    expect(onSelectSection).toHaveBeenCalledWith("basics");
    expect(screen.queryByRole("textbox", { name: "Full name" })).not.toBeInTheDocument();
  });

  it("renders an optional profile photo", () => {
    const document = makeSampleResume().document;
    document.basics.photo_url = "data:image/jpeg;base64,cGhvdG8=";
    renderDocument(true, { document });
    expect(screen.getByRole("img", { name: /Alex Morgan profile/i })).toHaveAttribute("src", document.basics.photo_url);
  });
});
