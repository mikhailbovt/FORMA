# Forma resume rubric

Forma separates measurable document hygiene from semantic editorial judgment.
The result is a transparent resume-health score, not a probability of passing an
ATS, receiving an interview, or being hired.

## Score model

Rubric version `forma-quality/1.0.0` has 100 fixed points:

| Layer | Points | Evaluated by |
| --- | ---: | --- |
| Deterministic checks | 60 | Forma's Go API; always available |
| Semantic content checks | 40 | An optional configured model or local Ollama, constrained by the rubric |

Without AI, Forma reports earned points against the deterministic rules that
apply to the current content, up to 60 points, and marks all 40 semantic points
as unassessed. Language- or content-dependent rules that do not apply are also
explicitly unassessed. The UI presents the API's normalized score on a stable
100-point scale and always shows coverage beside it: how many rubric points
were assessed, how many await optional AI review, and how many did not apply to
the current content. A score with pending semantic points is therefore a
provisional view of the evidence checked so far, not a fabricated AI opinion.

### Deterministic categories

| Category | Points | Examples |
| --- | ---: | --- |
| Essentials and focus | 15 | identity, usable contact method, headline, substantive section |
| Structure and completeness | 10 | populated visible entries, no placeholder-only sections |
| Evidence signals | 15 | action, context, result markers, duplicates, optional verified metrics |
| Clarity mechanics | 12 | summary and bullet length, pronouns, repeated openings, punctuation |
| Consistency and chronology | 8 | date validity, current-role state, ordering, consistent ranges |

### Semantic criteria

The model cannot choose criteria or weights. It may return one verdict for each
fixed rule:

| Rule | Points |
| --- | ---: |
| `semantic.impact_strength` | 12 |
| `semantic.clarity_specificity` | 10 |
| `semantic.target_relevance` | 10 |
| `semantic.voice_coherence` | 8 |

Each verdict must be `pass`, `partial`, `fail`, or `not_applicable`, include an
exact quote from the sanitized resume, provide a rationale, and meet the fixed
confidence threshold. Go validates the quote and maps the accepted verdict to
full, half, or zero points. Unknown rules, duplicate verdicts, invented quotes,
and low-confidence answers remain unassessed and cannot affect the score.

Target relevance is not scored unless the user supplied a target role or job
description.

## Separate signals

- **Ready to send** is a blocker-based boolean. A missing contact method or a
  resume without substantive content cannot be hidden by an otherwise decent
  number.
- **ATS hygiene** is `pass`, `warn`, or `fail` for machine-readable structure,
  dates, contacts, control characters, and duplicate evidence. It is explicitly
  not an ATS acceptance guarantee.
- **AI suggestions** are editorial proposals. They are shown separately and do
  not silently mutate the document.

## Reproducibility

- Rule IDs, weights, thresholds, and output order are versioned.
- The source digest is calculated from the normalized structured document.
- The same normalized document, context, language pack, and rubric version
  produce the same deterministic result.
- A scoring behavior change requires a rubric version change and golden fixture
  updates.
- Unsupported language-specific checks become `not_applicable`, not failures.

## Intentional non-rules

Forma does not universally penalize a photograph, a second page, an employment
gap, parallel roles, missing education, missing certifications, or the absence
of a number in every bullet. Those choices depend on country, seniority, role,
and document type. Metrics are valuable only when they are true.

The rubric is grounded in recurring guidance from the
[Harvard College resume guide](https://careerservices.fas.harvard.edu/resources/create-a-strong-resume/),
[MIT resume checklist](https://capd.mit.edu/resources/resume-checklist/), and
[Yale guidance for impactful bullets](https://ocs.yale.edu/resources/writing-impactful-resume-bullets/):
specific active language, relevant content, clear organization, consistent
formatting, and evidence of contribution and results.
