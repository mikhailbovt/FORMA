import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeSampleResume } from "../data/sampleResume.js";
import { SectionEditorPanel } from "./SectionEditorPanel.jsx";

function Harness({ section }) {
  const [document, setDocument] = useState(makeSampleResume().document);
  return <>
    <SectionEditorPanel section={section} document={document} onChange={setDocument} onOpenReview={vi.fn()} />
    <output data-testid="document-state">{JSON.stringify(document)}</output>
  </>;
}

describe("SectionEditorPanel", () => {
  it("edits contact fields and profile links directly", async () => {
    const user = userEvent.setup();
    render(<Harness section="basics" />);

    const website = screen.getByLabelText("Website");
    await user.clear(website);
    await user.type(website, "mikhail.dev");
    const email = screen.getByLabelText("Email");
    await user.clear(email);
    await user.type(email, "mikhail@example.com");

    expect(screen.getByTestId("document-state")).toHaveTextContent("mikhail.dev");
    expect(screen.getByTestId("document-state")).toHaveTextContent("mikhail@example.com");
    expect(screen.getByRole("button", { name: "Add link" })).toBeInTheDocument();
  });

  it("can edit a role beyond the first experience item", async () => {
    const user = userEvent.setup();
    render(<Harness section="experience" />);

    await user.click(screen.getByRole("button", { name: /Software Engineer.*Compass/i }));
    const companies = screen.getAllByLabelText("Company");
    await user.clear(companies[1]);
    await user.type(companies[1], "Northstar Labs");

    expect(screen.getByTestId("document-state")).toHaveTextContent("Northstar Labs");
  });

  it("rejects unsupported profile-photo files before they reach the resume", async () => {
    const user = userEvent.setup({ applyAccept: false });
    render(<Harness section="basics" />);

    const input = screen.getByLabelText("Add photo");
    await user.upload(input, new File(["not an image"], "resume.txt", { type: "text/plain" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Choose a JPG, PNG, or WebP image.");
    expect(screen.getByTestId("document-state")).not.toHaveTextContent("data:image");
  });
});
