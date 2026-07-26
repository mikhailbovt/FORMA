import {
  ArrowClockwise,
  Check,
  CheckCircle,
  FileText,
  ShieldCheck,
  Sparkle,
  Target,
  WarningCircle,
  XCircle,
} from "@phosphor-icons/react";
import { Button, Spinner } from "./ui.jsx";

const initialChecks = [
  { id: "rules", label: "60 deterministic points", icon: Target },
  { id: "ats", label: "ATS hygiene checks", icon: FileText },
  { id: "semantic", label: "40 optional semantic points", icon: ShieldCheck },
];

const phaseCopy = {
  rules: ["Checking Forma rules", "Dates, structure, evidence signals, and consistency are evaluated locally by the Go API."],
  ai: ["Waiting for your AI provider", "The provider is grading fixed semantic criteria and preparing independently applicable suggestions."],
  validate: ["Validating the review", "Forma is checking evidence quotes and rejecting unsupported scoring decisions."],
};

function StatusIcon({ status }) {
  if (status === "pass") return <CheckCircle size={16} weight="fill" />;
  if (status === "fail") return <XCircle size={16} weight="fill" />;
  return <WarningCircle size={16} weight={status === "warn" ? "fill" : "regular"} />;
}

function ReviewProgress({ phase = "rules" }) {
  const [title, description] = phaseCopy[phase] || phaseCopy.rules;
  return (
    <div className="review-progress" role="status" aria-live="polite">
      <Spinner label={title} />
      <div><strong>{title}</strong><p>{description}</p></div>
      <div className="review-progress__track" aria-hidden="true"><span /></div>
    </div>
  );
}

function getSemanticCoverage(evaluation) {
  const semantic = evaluation?.semantic;
  const criteria = semantic?.criteria || [];
  const pendingPoints = criteria.length > 0
    ? criteria.filter((criterion) => criterion.status === "unassessed").reduce((total, criterion) => total + (criterion.maximum_points || 0), 0)
    : semantic?.unassessed_points || 0;
  const notApplicablePoints = criteria
    .filter((criterion) => criterion.status === "not_applicable")
    .reduce((total, criterion) => total + (criterion.maximum_points || 0), 0);
  return { pendingPoints, notApplicablePoints };
}

function QualityScore({ evaluation }) {
  const quality = evaluation?.quality;
  if (!quality) return null;
  const maximumScore = quality.maximum_score || 100;
  const unassessedPoints = quality.unassessed_points || 0;
  const assessedPoints = quality.assessed_points ?? Math.max(0, maximumScore - unassessedPoints);
  const normalizedScore = quality.normalized_score ?? (assessedPoints > 0 ? Math.round((quality.score * maximumScore) / assessedPoints) : 0);
  const { pendingPoints: semanticPending, notApplicablePoints: semanticNotApplicable } = getSemanticCoverage(evaluation);
  const notApplicablePoints = Math.max(0, unassessedPoints - semanticPending);
  const coverageParts = [`${quality.score} of ${assessedPoints} applicable points earned`];
  if (semanticPending > 0) coverageParts.push(`${semanticPending} await optional AI review`);
  if (notApplicablePoints > 0) coverageParts.push(`${notApplicablePoints} not applicable`);
  const coverageCopy = `${coverageParts.join(" · ")}.`;
  const findingsByRule = new Map((evaluation.findings || []).map((finding) => [finding.rule_id, finding]));
  return (
    <section className="quality-card" aria-label="Forma resume score">
      <div className="quality-card__score">
        <span>Resume health</span>
        <strong aria-label={`Normalized resume score: ${normalizedScore} out of ${maximumScore}`}>{normalizedScore}<small>/{maximumScore}</small></strong>
        <p>{coverageCopy}</p>
      </div>
      <div className="quality-card__meta">
        <span className={`quality-ready quality-ready--${quality.ready ? "yes" : "no"}`}>
          {quality.ready ? <Check size={14} weight="bold" /> : <WarningCircle size={14} weight="fill" />}
          {quality.ready ? "Ready for a human fact check" : "Not ready to send"}
        </span>
        <small>{evaluation.rubric_version || "Forma rubric"} · same content, same deterministic score</small>
      </div>
      <div className="quality-categories">
        {(quality.categories || []).map((category) => {
          const semanticCoverage = [
            semanticPending > 0 ? `${semanticPending} pending` : "",
            semanticNotApplicable > 0 ? `${semanticNotApplicable} not applicable` : "",
          ].filter(Boolean).join(" · ");
          const coverageLabel = category.id === "semantic" ? semanticCoverage : category.unassessed_points > 0 ? `${category.unassessed_points} not applicable` : "";
          const scoreLabel = category.assessed_points > 0
            ? `${category.earned_points}/${category.assessed_points}`
            : category.id === "semantic" && semanticPending > 0 ? "Pending" : "N/A";
          return (
            <div className="quality-category" key={category.id}>
              <StatusIcon status={category.status} />
              <span>{category.label}{coverageLabel && <small>{coverageLabel}</small>}</span>
              <strong>{scoreLabel}</strong>
            </div>
          );
        })}
      </div>
      {(quality.blockers || []).length > 0 && <div className="quality-blockers">{quality.blockers.map((blocker) => <p key={blocker}><XCircle size={15} />{findingsByRule.get(blocker)?.message || blocker}</p>)}</div>}
    </section>
  );
}

function QualityFindings({ evaluation }) {
  const findings = (evaluation?.findings || []).filter((finding) => finding.status === "warn" || finding.status === "fail");
  if (!findings.length) return null;
  return (
    <section className="quality-findings" aria-label="Resume checklist findings">
      <div className="review-section-title"><span>Forma checklist</span><small>{findings.length} items to review</small></div>
      {findings.map((finding) => (
        <details className={`quality-finding quality-finding--${finding.status}`} key={finding.rule_id}>
          <summary><StatusIcon status={finding.status} /><span>{finding.message}</span><strong>{finding.earned_points}/{finding.possible_points}</strong></summary>
          <div>
            {(finding.evidence || []).map((evidence, index) => (
              <p key={`${evidence.path}-${index}`}><code>{evidence.path}</code>{evidence.actual && <span>{evidence.actual}</span>}{evidence.expected && <small>Expected: {evidence.expected}</small>}</p>
            ))}
            <small>Rule: {finding.rule_id}</small>
          </div>
        </details>
      ))}
    </section>
  );
}

function ATSResult({ evaluation }) {
  if (!evaluation?.ats_hygiene) return null;
  const ats = evaluation.ats_hygiene;
  return (
    <section className="ats-result">
      <div><FileText size={18} /><span>ATS hygiene</span><strong>{ats.status}</strong></div>
      {(ats.findings || []).slice(0, 3).map((finding) => <p key={finding.rule_id}><StatusIcon status={finding.status} />{finding.message}</p>)}
    </section>
  );
}

function AISuggestions({ session, review, semanticPending, onConfigure, onApply, onDismiss, loading }) {
  if (!session?.configured) {
    if (semanticPending === 0) return null;
    return (
      <section className="ai-upgrade">
        <Sparkle size={21} />
        <div><strong>Add semantic feedback</strong><p>Connect a provider or local Ollama to evaluate the remaining 40 points and receive rewrite suggestions.</p></div>
        <Button size="sm" variant="secondary" onClick={onConfigure}>Connect AI</Button>
      </section>
    );
  }
  if (!review) return null;
  return (
    <section className="ai-suggestions" aria-label="AI suggestions">
      <div className="review-section-title"><span>AI suggestions</span><small>{session.provider} · {session.model}</small></div>
      {review.summary && <p className="ai-suggestions__summary">{review.summary}</p>}
      <div className="review-list">
        {(review.suggestions || []).length === 0 ? (
          <div className="review-empty"><Check size={24} weight="bold" /><h3>No blocking wording issues</h3><p>Your draft is ready for a final human fact check.</p></div>
        ) : review.suggestions.map((suggestion) => (
          <article className="review-suggestion" key={suggestion.id || `${suggestion.section}-${suggestion.title}`}>
            <span className="review-suggestion__section">{suggestion.section}</span>
            <h3>{suggestion.title}</h3>
            <p>{suggestion.reason}</p>
            {suggestion.proposed && <div className="review-suggestion__proposal"><small>Proposed change</small><p>{suggestion.proposed}</p></div>}
            <div className="review-suggestion__actions"><Button size="sm" onClick={() => onApply(suggestion)} disabled={loading}>Apply change</Button><Button size="sm" variant="ghost" onClick={() => onDismiss(suggestion)} disabled={loading}>Keep original</Button></div>
          </article>
        ))}
      </div>
      {(review.warnings || []).map((warning, index) => <div className="ai-review-warning" key={`${warning}-${index}`}><WarningCircle size={16} />{warning}</div>)}
    </section>
  );
}

export function AIReviewPanel({ session, status, phase, error, review, quality, onRun, onConfigure, onApply, onDismiss }) {
  const loading = status === "loading";
  const hasResults = Boolean(quality || review);
  const { pendingPoints: semanticPending } = getSemanticCoverage(quality);
  const reviewSubtitle = semanticPending > 0
    ? session?.configured
      ? "Forma scored the available evidence; some semantic points remain unassessed."
      : "Deterministic checks are complete. Connect AI to assess the optional semantic points."
    : "Scored with Forma's versioned resume rubric.";

  if (!hasResults) {
    return (
      <aside className="ai-panel" aria-label="Resume review">
        <div className="ai-panel__heading">
          <h2>Resume review</h2>
          {session?.configured && <button type="button" onClick={onConfigure}>{session.provider} · {session.model}</button>}
        </div>
        <div className={`ai-panel__intro ${loading ? "is-loading" : ""}`}>
          {loading ? <Spinner label="Running resume review" /> : <Sparkle size={28} />}
          <h3>{loading ? "Reviewing your resume…" : "A stable score, with or without AI."}</h3>
          <p>{loading ? "Forma starts with versioned local rules. If AI is connected, semantic grading runs afterward." : "Check structure, evidence, clarity, chronology, ATS hygiene, and optional semantic quality."}</p>
          <Button className="ai-panel__run" onClick={onRun} disabled={loading}>{loading ? <><Spinner label="Running resume review" /> Reviewing</> : "Run resume review"}</Button>
          <small>AI never chooses weights or invents the final score.</small>
          {error && <div className="ai-panel__error" role="alert"><WarningCircle size={17} />{error}</div>}
        </div>
        {loading ? <ReviewProgress phase={phase} /> : <div className="ai-panel__checks">{initialChecks.map(({ id, label, icon: Icon }) => <div className="ai-check" key={id}><Icon size={20} /><span>{label}</span></div>)}</div>}
      </aside>
    );
  }

  return (
    <aside className={`ai-panel ai-panel--results ${loading ? "is-refreshing" : ""}`} aria-label="Resume review results" aria-busy={loading}>
      <div className="ai-panel__heading">
        <div><h2>Resume review</h2><p>{reviewSubtitle}</p></div>
        <button type="button" onClick={onRun} disabled={loading} className={loading ? "is-spinning" : ""}><ArrowClockwise size={15} /> {loading ? "Reviewing" : "Run again"}</button>
      </div>
      {loading && <ReviewProgress phase={phase} />}
      {error && <div className="ai-panel__error" role="alert"><WarningCircle size={17} />{error}</div>}
      <div className="review-results-body">
        <QualityScore evaluation={quality} />
        <QualityFindings evaluation={quality} />
        <ATSResult evaluation={quality} />
        <AISuggestions session={session} review={review} semanticPending={semanticPending} onConfigure={onConfigure} onApply={onApply} onDismiss={onDismiss} loading={loading} />
      </div>
    </aside>
  );
}
