# Forma API contract

Base path: `/api/v1`. JSON requests require `Content-Type: application/json`.
The default JSON request-envelope limit is 2 MiB. Import preview has an
independent 16 MiB multipart envelope and tighter file/parser limits. Timestamps
are UTC RFC 3339 values and IDs are UUIDs.

## Resume document

Create and replace requests have this shape:

```json
{
  "title": "Product engineer — general",
  "document": {
    "version": 1,
    "basics": {
      "name": "Alex Morgan",
      "headline": "Product Engineer",
      "email": "alex@example.com",
      "phone": "+1 555 0100",
      "location": "Austin, TX",
      "website": "https://example.com",
      "links": [{ "label": "GitHub", "url": "https://github.com/example" }]
    },
    "summary": "Product engineer building reliable SaaS products.",
    "experience": [
      {
        "id": "exp-1",
        "company": "Sentry Labs",
        "position": "Product Engineer",
        "location": "Austin, TX",
        "start_date": "2022-04",
        "end_date": "",
        "current": true,
        "summary": "Built product and platform features.",
        "highlights": ["Reduced false alerts by 35%."]
      }
    ],
    "projects": [],
    "education": [],
    "skills": [{ "id": "skill-1", "name": "Backend", "keywords": ["Go", "PostgreSQL"] }],
    "portfolio": [],
    "certifications": [],
    "languages": [{ "id": "lang-1", "name": "English", "fluency": "Professional" }],
    "custom_sections": [],
    "order": ["summary", "experience", "projects", "education", "skills"],
    "hidden_sections": [],
    "template": "forma",
    "page_size": "A4",
    "language": "en"
  }
}
```

`version` defaults to `1`; `template` to `forma`; `page_size` to `A4`; and `language` to `en`. `page_size` accepts `A4` or `LETTER`. The document supports experience, projects, education, grouped skills, portfolio, certifications, languages, custom sections, explicit section order and visibility, template, page size, and output language. Each document is stored as one JSONB value.

### Resume routes

- `GET /resumes?limit=50&offset=0` returns `data.items`, `data.limit`, and `data.offset`. Limit range is 1–100.
- `POST /resumes` returns `201`, the created resume, and `Location: /api/v1/resumes/{id}`.
- `GET /resumes/{id}` returns one resume or `404 resume_not_found`.
- `PUT /resumes/{id}` replaces the complete title/document and returns the updated resume.
- `DELETE /resumes/{id}` returns `204`.
- `POST /resumes/{id}/duplicate` returns `201` and a copy with a new UUID and ` copy` title suffix.

Stored responses add `id`, `created_at`, and `updated_at` around the create/replace shape.

## Resume import preview

`POST /imports/preview` accepts one `multipart/form-data` field named `file`.
The file may be a Forma JSON export, JSON Resume, DOCX, text-layer PDF, or an
official LinkedIn data-export ZIP. The file limit is 12 MiB inside a 16 MiB
multipart envelope.

```text
Content-Disposition: form-data; name="file"; filename="resume.json"
```

The response is a preview only; the API does not persist it:

```json
{
  "data": {
    "candidate": { "version": 1, "basics": {}, "experience": [] },
    "parser": { "id": "json-resume", "version": "1.0.0" },
    "source_sha256": "…",
    "mappings": [],
    "warnings": []
  }
}
```

Uploads are parsed in memory. ZIP traversal, excessive entry counts, oversized
expansion, and suspicious compression ratios are rejected. JSON requests that
contain only a LinkedIn URL return `422 url_import_not_supported`: Forma never
fetches or scrapes LinkedIn profile pages. The browser may keep a validated
profile URL as a contact link and asks the user to upload their own ZIP or PDF.

## Resume quality

`POST /quality/evaluate` accepts `{ "resume": <resume document> }` and does not
require an AI session. It returns the versioned deterministic portion of the
rubric: up to 60 assessed points, with the optional 40 semantic points marked
unassessed.

```json
{
  "data": {
    "rubric_version": "forma-quality/1.0.0",
    "source_digest": "…",
    "language": "en",
    "quality": {
      "score": 47,
      "maximum_score": 100,
      "assessed_points": 60,
      "unassessed_points": 40,
      "normalized_score": 78,
      "ready": true,
      "blockers": [],
      "categories": []
    },
    "ats_hygiene": { "status": "pass", "findings": [] },
    "semantic": {
      "maximum_points": 40,
      "assessed_points": 0,
      "earned_points": 0,
      "unassessed_points": 40,
      "ignored_count": 0,
      "criteria": []
    },
    "findings": []
  }
}
```

This is a document-quality rubric, not an ATS acceptance, interview, or hiring
probability.

## AI session

`PUT /ai/session`:

```json
{
  "provider": "openai",
  "model": "gpt-5.6-terra",
  "api_key": "provider-secret",
  "base_url": "https://api.openai.com/v1"
}
```

`model` is arbitrary editable text. `base_url` may be omitted for a preset and must be an HTTP(S) URL without embedded credentials, query, or fragment. `api_key` is required for hosted preset providers and optional for `custom`/`ollama`.

The response is redacted:

```json
{
  "data": {
    "configured": true,
    "provider": "openai",
    "model": "gpt-5.6-terra",
    "has_api_key": true,
    "base_url": "https://api.openai.com/v1",
    "expires_at": "2026-07-26T12:30:00Z"
  }
}
```

The response sets `forma_ai_session`, an opaque HttpOnly SameSite cookie. Browser calls must include credentials. `GET /ai/session` returns the same redacted view; `DELETE /ai/session` destroys the server-side value and expires the cookie.

## AI review

`POST /ai/review` is the explicit full-review action:

```json
{
  "resume": {
    "basics": { "name": "Alex Morgan", "email": "alex@example.com", "headline": "Product Engineer" },
    "summary": "Builds reliable products.",
    "experience": []
  },
  "target_role": "Senior Product Engineer",
  "job_description": "Optional job text selected by the user.",
  "focus": "Clarity and measurable outcomes"
}
```

Response:

```json
{
  "data": {
    "quality": {
      "rubric_version": "forma-quality/1.0.0",
      "quality": {
        "score": 75,
        "maximum_score": 100,
        "assessed_points": 100,
        "unassessed_points": 0
      }
    },
    "ai": {
      "summary": "Clear foundation; strengthen outcome-oriented evidence.",
      "assessments": [
        {
          "rule_id": "semantic.impact_strength",
          "verdict": "partial",
          "evidence": "Builds reliable products.",
          "confidence": 0.9,
          "reason": "The action is clear but the outcome is broad."
        }
      ],
      "suggestions": [
        {
          "id": "summary-1",
          "section": "summary",
          "severity": "medium",
          "title": "Make the scope concrete",
          "reason": "The current line is broad.",
          "original": "Builds reliable products.",
          "replacement": "Product engineer building reliable SaaS products."
        }
      ],
      "warnings": []
    }
  }
}
```

The example assessment list above is abbreviated. The AI must return exactly
the four fixed semantic rule IDs. It does not return
or control a score. The Go rubric accepts only known, non-duplicate assessments
above the confidence threshold whose evidence is an exact quote from the
sanitized source; target relevance remains unassessed without a supplied target
role or job description. Severity is `low`, `medium`, or `high`; there are at
most 25 suggestions. Suggestions are not persisted or applied automatically.

## AI rewrite

`POST /ai/rewrite` operates on a selected non-personal passage:

```json
{
  "text": "Built alerting workflow and made it faster.",
  "section": "experience",
  "instruction": "rewrite",
  "context": { "position": "Product Engineer", "highlights": ["Reduced false alerts by 35%."] },
  "target_role": "Senior Product Engineer"
}
```

Response:

```json
{
  "data": {
    "rewritten_text": "Built an alerting workflow that reduced false alerts by 35%.",
    "explanation": "Leads with the action and uses the supplied outcome.",
    "warnings": []
  }
}
```

Sections `basics`, `header`, `contact`, and `personal` are rejected. Replacement text containing a new numeric claim is rejected even if the provider otherwise returns valid JSON.

## Status and error codes

Common codes:

| HTTP | Code | Meaning |
| --- | --- | --- |
| 400 | `invalid_json`, `invalid_id`, `invalid_query` | malformed request |
| 404 | `resume_not_found`, `not_found` | missing resource/route |
| 409 | `ai_not_configured` | no live volatile AI session |
| 413 | `body_too_large`, `upload_too_large`, import limit codes | body or uploaded content exceeded a limit |
| 415 | `unsupported_media_type` | JSON Content-Type missing/wrong |
| 422 | `validation_error`, `url_import_not_supported`, import parser codes | semantic validation, privacy rejection, or unusable import |
| 502 | `provider_error`, `invalid_ai_output` | provider/network/schema/safety failure |
| 503 | health response | PostgreSQL is unavailable |
| 504 | `provider_timeout` | provider exceeded the request timeout |
