# Importing resumes

Forma imports into a preview first. No uploaded file is saved to disk or written
to PostgreSQL until the user explicitly applies the normalized candidate resume.

## Supported formats

| Format | Strategy |
| --- | --- |
| Forma JSON | Lossless import of Forma's own export envelope |
| JSON Resume | Deterministic mapping into Forma's canonical schema |
| LinkedIn data ZIP | Allowlisted profile, positions, education, skills, projects, certifications, and languages CSV files |
| DOCX | In-memory OOXML text extraction followed by conservative heading-based mapping |
| Text-layer PDF | In-memory text extraction followed by conservative heading-based mapping |

Scanned PDFs are rejected with an actionable warning rather than silently
producing an empty or fabricated resume. OCR is intentionally outside the first
import version.

## LinkedIn

Forma does not fetch or scrape a pasted LinkedIn profile URL. Public profile
HTML is incomplete, authenticated pages are not a stable integration surface,
and LinkedIn restricts automated profile scraping. A pasted
`linkedin.com/in/...` URL is instead added to the resume's contact links.

For resume content, upload either:

1. the official LinkedIn account data ZIP; or
2. a text-based **Save profile as PDF** export.

LinkedIn documents the official
[data download](https://www.linkedin.com/help/linkedin/answer/a1339364) and
[profile PDF export](https://www.linkedin.com/help/linkedin/answer/a541960).

## Preview and apply modes

Every preview includes the parser ID and version, a SHA-256 source digest,
field mappings with source locators, and stable warnings.

- **Merge safely** fills empty scalar fields and appends new unique entries. It
  does not overwrite existing facts.
- **Replace content** replaces resume content while preserving the selected
  template, page size, and local presentation settings.

PDF and DOCX mappings are heuristic. The UI always reminds the user to verify
names, dates, employers, and claims before export or AI review.

## Safety limits

- File content is processed in memory and capped independently from the HTTP
  multipart envelope.
- ZIP entry names are allowlisted and normalized; path traversal is rejected.
- Entry count, per-entry bytes, total expanded bytes, and compression ratio are
  bounded to resist archive bombs.
- Messages, contacts, connections, identity files, and unrelated LinkedIn
  export categories are ignored.
- Import parsing never invokes an AI provider. Only normalized, user-approved
  resume text can later enter the explicit review flow.

