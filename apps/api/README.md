# Forma API

Local-first REST API for Forma. It stores resume documents in PostgreSQL and proxies explicit AI review/rewrite actions without persisting provider credentials.

## Run locally

Prerequisites: Go 1.25+ and PostgreSQL 16+.

```bash
go run ./cmd/api
```

The API listens on `http://localhost:8080`. Embedded SQL migrations run transactionally at startup. Startup fails if PostgreSQL is unavailable or a migration cannot be applied.

Default development connection:

```text
postgres://forma:forma@localhost:5432/forma?sslmode=disable
```

Configuration:

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | HTTP listen address |
| `DATABASE_URL` | local URL above | pgx PostgreSQL connection URL |
| `CORS_ORIGIN` | `http://localhost:3000` | exact browser origin allowed with credentials |
| `AI_SESSION_TTL` | `30m` | in-memory provider credential lifetime; minimum `1m` |
| `COOKIE_SECURE` | `false` | set `Secure` on the AI session cookie |
| `MAX_BODY_BYTES` | `2097152` | maximum non-import request body; minimum 1024 bytes |

Import preview uses its own fixed 16 MiB multipart-envelope limit and a 12 MiB
file limit so lowering the JSON limit cannot break supported uploads.

The production image is built from `Dockerfile`, listens on port 8080, runs as a non-root distroless user, and includes a `/forma-api healthcheck` command.

## API at a glance

All routes are under `/api/v1`:

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/health` | process/database readiness |
| `POST` | `/imports/preview` | parse one supported upload without persisting it |
| `POST` | `/quality/evaluate` | run the deterministic 60-point rubric without AI |
| `GET` | `/resumes?limit=50&offset=0` | resume list |
| `POST` | `/resumes` | create a resume |
| `GET` | `/resumes/{uuid}` | fetch a resume |
| `PUT` | `/resumes/{uuid}` | replace title and document |
| `DELETE` | `/resumes/{uuid}` | delete a resume |
| `POST` | `/resumes/{uuid}/duplicate` | copy a resume |
| `GET` | `/ai/providers` | provider metadata and editable model suggestions |
| `GET` | `/ai/session` | redacted current provider configuration |
| `PUT` | `/ai/session` | keep provider/model/key in volatile memory |
| `DELETE` | `/ai/session` | destroy the volatile AI session |
| `POST` | `/ai/review` | assess the fixed semantic rubric and return separate suggestions |
| `POST` | `/ai/rewrite` | explicitly rewrite selected non-personal text |

Successful resource responses use `{"data": ...}`. Errors use a stable envelope:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Resume is invalid",
    "fields": { "title": "is required" },
    "request_id": "..."
  }
}
```

See [API.md](API.md) for request and response contracts.

## AI provider adapters

Model IDs are suggestions, not enums: clients may send any non-empty model ID. Base URLs are editable for regional endpoints, gateways, and compatible servers.

| Provider ID | Protocol | Structured output |
| --- | --- | --- |
| `openai` | OpenAI Responses API | strict JSON Schema |
| `anthropic` | Anthropic Messages API | forced tool with input schema |
| `gemini` | Gemini native `generateContent` | JSON response schema |
| `deepseek` | OpenAI-compatible Chat Completions | JSON object + one repair retry |
| `qwen` | OpenAI-compatible Chat Completions | JSON object + one repair retry |
| `kimi` | OpenAI-compatible Chat Completions | JSON object + one repair retry |
| `glm` | OpenAI-compatible Chat Completions | JSON object + one repair retry |
| `custom` | OpenAI-compatible Chat Completions | JSON object + one repair retry |
| `ollama` | Ollama native `/api/chat` | JSON Schema format |

Provider presets are convenience defaults and may age; the user remains able to edit both `model` and `base_url`.

## Privacy and safety boundaries

- AI is invoked only by `POST /ai/review` or `POST /ai/rewrite`; saving a resume never invokes a provider.
- API keys live only in the process-memory TTL store. They are never written to PostgreSQL, returned by the API, placed in URLs, or included in application logs.
- The opaque session ID is carried by an `HttpOnly`, `SameSite=Strict` cookie scoped to `/api/v1/ai`.
- Personal name and contact fields are removed before resume/context JSON is sent out. Header/contact rewrites are rejected.
- Prompts instruct providers not to invent facts. Responses are decoded against a local schema and rejected when replacement text introduces a numeric claim absent from the submitted facts.
- Providers cannot set a numeric score. They return evidence for four fixed semantic criteria; the Go rubric validates that evidence and owns all weights and scoring.
- AI suggestions are returned for user review; the API never silently applies them to a stored resume.
- Import files are parsed in memory without provider calls, persistence, LinkedIn scraping, or other outbound requests.
- There is no telemetry in this service.

## Verification

```bash
go test ./...
go vet ./...
```

Tests cover deterministic and semantic scoring, import formats and archive limits,
prompt sanitization, invented-metric rejection, volatile session behavior, each
provider protocol envelope, the single repair retry, cookie redaction, resume
defaults, CORS, and health handling.
