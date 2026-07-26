import { describe, expect, it } from "vitest";
import { makeSampleResume } from "../data/sampleResume.js";
import { addArrayItem, applyImportedResume, applyReviewSuggestion, applyReviewSuggestionWithResult, fileSafeName, formatPeriod, moveSection, normalizeDocument, resumeReadiness, toggleSection } from "./resume.js";

describe("resume utilities", () => {
  it("formats partial dates without inventing a day", () => {
    expect(formatPeriod("2022-04", "", true)).toMatch(/Apr 2022 — Present/);
    expect(formatPeriod("2019", "2021")).toBe("2019 — 2021");
  });

  it("moves and hides sections without mutating input", () => {
    const document = makeSampleResume().document;
    const moved = moveSection(document, "experience", -1);
    expect(moved.section_order).not.toEqual(document.section_order);
    expect(document.section_order[2]).toBe("experience");
    const hidden = toggleSection(document, "projects");
    expect(hidden.hidden_sections).toContain("projects");
    expect(document.hidden_sections).not.toContain("projects");
  });

  it("adds valid section items and normalizes imported arrays", () => {
    const document = normalizeDocument({ basics: { full_name: "Ada" }, experience: "broken" });
    expect(document.experience).toEqual([]);
    expect(addArrayItem(document, "experience").experience[0].company).toBe("New company");
  });

  it("applies a structured suggestion only to the addressed fact", () => {
    const document = makeSampleResume().document;
    const original = document.experience[0].highlights[0];
    const next = applyReviewSuggestion(document, { section: "experience", item_id: "exp-sentry", field: "highlights", index: 0, original, proposed: "Verified replacement." });
    expect(next.experience[0].highlights[0]).toBe("Verified replacement.");
    expect(document.experience[0].highlights[0]).not.toBe("Verified replacement.");
  });

  it("applies a canonical backend suggestion by matching its original text", () => {
    const document = makeSampleResume().document;
    const original = document.experience[0].highlights[0];
    const next = applyReviewSuggestion(document, {
      section: "experience",
      original,
      proposed: "A verified replacement.",
    });

    expect(next.experience[0].highlights[0]).toBe("A verified replacement.");
    expect(document.experience[0].highlights[0]).toBe(original);
  });

  it("applies canonical suggestions across section field aliases", () => {
    const document = makeSampleResume().document;
    const project = applyReviewSuggestionWithResult(document, {
      section: "project",
      original: document.projects[0].description,
      replacement: "Open-source incident review workspace for engineering teams.",
    });
    expect(project.applied).toBe(true);
    expect(project.document.projects[0].description).toContain("engineering teams");

    const headline = applyReviewSuggestionWithResult(document, {
      section: "header",
      original: document.basics.headline,
      replacement: "Senior Product Engineer",
    });
    expect(headline.applied).toBe(true);
    expect(headline.document.basics.headline).toBe("Senior Product Engineer");
    expect(headline.document.basics.email).toBe(document.basics.email);
  });

  it("replaces an exact sentence without discarding the surrounding summary", () => {
    const document = makeSampleResume().document;
    const original = "I ship reliable, measurable features";
    const result = applyReviewSuggestionWithResult(document, {
      section: "summary",
      original,
      proposed: "I deliver reliable, measurable features",
    });
    expect(result.applied).toBe(true);
    expect(result.document.summary).toContain("Product engineer with 6+ years");
    expect(result.document.summary).toContain("I deliver reliable, measurable features");
  });

  it("keeps stale and ambiguous suggestions visible instead of mutating the wrong fact", () => {
    const document = makeSampleResume().document;
    const stale = applyReviewSuggestionWithResult(document, {
      section: "experience",
      original: "This text no longer exists.",
      proposed: "Replacement.",
    });
    expect(stale.applied).toBe(false);
    expect(stale.reason).toBe("source_not_found");
    expect(stale.document).toBe(document);

    const duplicate = structuredClone(document);
    duplicate.experience[1].highlights[0] = duplicate.experience[0].highlights[0];
    const ambiguous = applyReviewSuggestionWithResult(duplicate, {
      section: "experience",
      original: duplicate.experience[0].highlights[0],
      proposed: "A replacement that must not guess.",
    });
    expect(ambiguous.applied).toBe(false);
    expect(ambiguous.reason).toBe("ambiguous_source");
    expect(ambiguous.document).toBe(duplicate);
  });

  it("rejects empty, unsupported, and contact-targeted AI changes", () => {
    const document = makeSampleResume().document;
    expect(applyReviewSuggestionWithResult(document, { section: "summary", original: "", proposed: "New" }).reason).toBe("invalid_suggestion");
    expect(applyReviewSuggestionWithResult(document, { section: "unknown", original: "Old", proposed: "New" }).reason).toBe("unsupported_target");
    expect(applyReviewSuggestionWithResult(document, { section: "basics", field: "email", original: document.basics.email, proposed: "other@example.com" }).applied).toBe(false);
  });

  it("reports deterministic readiness and safe filenames", () => {
    const result = resumeReadiness(makeSampleResume().document);
    expect(result.passed).toBeGreaterThanOrEqual(3);
    expect(fileSafeName("Alex / Product CV")).toBe("alex-product-cv.json");
  });

  it("merges imported content without overwriting existing facts or adding duplicates", () => {
    const current = makeSampleResume();
    const imported = {
      title: "Imported profile",
      document: normalizeDocument({
        basics: { full_name: "Imported name", email: "imported@example.com", profiles: [] },
        summary: "Imported summary",
        experience: [
          structuredClone(current.document.experience[0]),
          { id: "imported", company: "New Co", position: "Engineer", start_date: "2024", current: true, highlights: ["Built a service."] },
        ],
      }),
    };
    const result = applyImportedResume(current, imported, { mode: "merge", linkedinURL: "https://www.linkedin.com/in/alex" });

    expect(result.title).toBe(current.title);
    expect(result.document.basics.full_name).toBe(current.document.basics.full_name);
    expect(result.document.basics.profiles.some((profile) => profile.url.includes("linkedin.com/in/alex"))).toBe(true);
    expect(result.document.experience.filter((item) => item.company === current.document.experience[0].company)).toHaveLength(1);
    expect(result.document.experience.some((item) => item.company === "New Co")).toBe(true);
  });

  it("replaces imported content while preserving presentation settings", () => {
    const current = makeSampleResume();
    current.document.template = "portrait";
    current.document.paper_size = "letter";
    const imported = { title: "Imported", document: normalizeDocument({ basics: { full_name: "Grace Hopper" }, summary: "Compiler pioneer." }) };
    const result = applyImportedResume(current, imported, { mode: "replace" });

    expect(result.document.basics.full_name).toBe("Grace Hopper");
    expect(result.document.summary).toBe("Compiler pioneer.");
    expect(result.document.template).toBe("portrait");
    expect(result.document.paper_size).toBe("letter");
  });
});
