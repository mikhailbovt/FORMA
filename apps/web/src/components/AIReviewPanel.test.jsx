import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AIReviewPanel } from "./AIReviewPanel.jsx";

const quality = {
  rubric_version: "forma-quality/1.0.0",
  quality: {
    score: 48,
    normalized_score: 80,
    maximum_score: 100,
    assessed_points: 60,
    unassessed_points: 40,
    ready: true,
    blockers: [],
    categories: [{ id: "essentials", label: "Essentials and focus", maximum_points: 15, assessed_points: 15, earned_points: 13, status: "warn" }],
  },
  ats_hygiene: { status: "pass", findings: [] },
  findings: [{ rule_id: "clarity.summary", status: "warn", message: "Tighten the summary.", earned_points: 2, possible_points: 4, evidence: [{ path: "$.summary", actual: "96 words", expected: "30–80 words" }] }],
};

describe("AIReviewPanel", () => {
  it("runs deterministic review without requiring an AI provider", async () => {
    const user = userEvent.setup();
    const onRun = vi.fn();
    render(<AIReviewPanel session={{ configured: false }} status="idle" review={null} quality={null} onRun={onRun} onConfigure={() => {}} />);
    expect(screen.getByText(/stable score, with or without AI/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Run resume review" }));
    expect(onRun).toHaveBeenCalledTimes(1);
  });

  it("keeps prior results visible but unmistakably busy during a rerun", () => {
    render(<AIReviewPanel session={{ configured: false }} status="loading" phase="rules" review={null} quality={quality} onRun={() => {}} onConfigure={() => {}} onApply={() => {}} onDismiss={() => {}} />);
    expect(screen.getByRole("complementary", { name: "Resume review results" })).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("status", { name: /Checking Forma rules/i })).toBeInTheDocument();
    expect(screen.getByText("80")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Reviewing/i })).toBeDisabled();
  });

  it("separates a stable score from long AI suggestion cards", () => {
    const summary = "The resume has strong product engineering experience, measurable outcomes, experimentation, instrumentation, dashboards, and data pipeline work.";
    render(<AIReviewPanel
      session={{ configured: true, provider: "openai", model: "gpt-test" }}
      status="complete"
      quality={{ ...quality, quality: { ...quality.quality, score: 81, normalized_score: 81, assessed_points: 100, unassessed_points: 0 } }}
      review={{ summary, warnings: [], suggestions: [
        { id: "one", section: "summary", title: "Align the headline", reason: "Use a more specific target.", proposed: "Data Scientist | Product Analytics & Engineering" },
        { id: "two", section: "experience", title: "Quantify the impact", reason: "Explain the analytical result.", proposed: "Reduced review time after instrumenting the workflow." },
      ] }}
      onRun={() => {}}
      onConfigure={() => {}}
      onApply={() => {}}
      onDismiss={() => {}}
    />);

    expect(screen.getByText(summary)).toBeInTheDocument();
    expect(screen.getAllByRole("article")).toHaveLength(2);
    expect(screen.getByText("Data Scientist | Product Analytics & Engineering")).toBeInTheDocument();
  });

  it("does not mislabel unassessed deterministic rules as semantic points", () => {
    render(<AIReviewPanel
      session={{ configured: false }}
      status="complete"
      quality={{
        ...quality,
        quality: { ...quality.quality, score: 5, normalized_score: 24, assessed_points: 21, unassessed_points: 79 },
        semantic: { maximum_points: 40, assessed_points: 0, earned_points: 0, unassessed_points: 40, criteria: [] },
      }}
      review={null}
      onRun={() => {}}
      onConfigure={() => {}}
    />);

    expect(screen.getByLabelText("Normalized resume score: 24 out of 100")).toBeInTheDocument();
    expect(screen.getByText("5 of 21 applicable points earned · 40 await optional AI review · 39 not applicable.")).toBeInTheDocument();
    expect(screen.queryByText("79 semantic points are not evaluated yet.")).not.toBeInTheDocument();
  });

  it("keeps the headline denominator fixed at 100 and explains score coverage", () => {
    render(<AIReviewPanel
      session={{ configured: true }}
      status="complete"
      quality={{
        ...quality,
        quality: {
          ...quality.quality,
          score: 51,
          normalized_score: 58,
          assessed_points: 88,
          unassessed_points: 12,
          categories: [{ id: "consistency", label: "Consistency and chronology", maximum_points: 8, assessed_points: 0, earned_points: 0, unassessed_points: 8, status: "unassessed" }],
        },
        semantic: { maximum_points: 40, assessed_points: 40, earned_points: 13, unassessed_points: 0, criteria: [
          { rule_id: "semantic.impact_strength", maximum_points: 12, status: "pass" },
          { rule_id: "semantic.clarity_specificity", maximum_points: 10, status: "partial" },
          { rule_id: "semantic.target_relevance", maximum_points: 10, status: "pass" },
          { rule_id: "semantic.voice_coherence", maximum_points: 8, status: "fail" },
        ] },
      }}
      review={{ summary: "Review complete.", suggestions: [], warnings: [] }}
      onRun={() => {}}
      onConfigure={() => {}}
    />);

    expect(screen.getByLabelText("Normalized resume score: 58 out of 100")).toHaveTextContent("58/100");
    expect(screen.getByText("51 of 88 applicable points earned · 12 not applicable.")).toBeInTheDocument();
    expect(screen.getByText("8 not applicable")).toBeInTheDocument();
    expect(screen.getByText("N/A")).toBeInTheDocument();
    expect(screen.getByText("Scored with Forma's versioned resume rubric.")).toBeInTheDocument();
    expect(screen.queryByText(/Connect AI/i)).not.toBeInTheDocument();
  });

  it("does not ask to reconnect AI when a completed review only has a not-applicable criterion", () => {
    render(<AIReviewPanel
      session={{ configured: false }}
      status="complete"
      quality={{
        ...quality,
        quality: { ...quality.quality, score: 45, normalized_score: 50, assessed_points: 90, unassessed_points: 10 },
        semantic: {
          maximum_points: 40,
          assessed_points: 30,
          earned_points: 20,
          unassessed_points: 10,
          criteria: [
            { rule_id: "semantic.impact_strength", maximum_points: 12, status: "pass" },
            { rule_id: "semantic.clarity_specificity", maximum_points: 10, status: "partial" },
            { rule_id: "semantic.target_relevance", maximum_points: 10, status: "not_applicable" },
            { rule_id: "semantic.voice_coherence", maximum_points: 8, status: "pass" },
          ],
        },
      }}
      review={null}
      onRun={() => {}}
      onConfigure={() => {}}
    />);

    expect(screen.getByText("Scored with Forma's versioned resume rubric.")).toBeInTheDocument();
    expect(screen.getByText("45 of 90 applicable points earned · 10 not applicable.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect AI" })).not.toBeInTheDocument();
  });
});
