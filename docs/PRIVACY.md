# Privacy model

Forma is local-first: its default Docker Compose stack stores resume data on the
operator's machine and publishes only the web interface. Local-first does not
mean that every AI option is offline. Choosing a hosted provider explicitly
sends a sanitized subset of content to that provider.

This document describes Forma's application behavior. It is not a promise about
an AI provider's retention, training, regional processing, or account policy.

## Data inventory

| Data | Where it lives | How long | Sent externally |
| --- | --- | --- | --- |
| Resume fields | PostgreSQL named Docker volume | Until the user deletes it or removes the volume | Only sanitized content during explicit AI actions |
| Imported resume file and preview | API memory and browser state | Request lifetime and until the preview is dismissed or applied | No; import parsing performs no provider or profile-page request |
| Provider, model, optional base URL | API process memory | Until TTL expiry or API restart | Used to select the requested endpoint |
| Provider API key | API process memory, tied to an HttpOnly session cookie | Until TTL expiry or API restart | Sent only to the selected provider for authentication |
| AI suggestions | Returned to the browser | Controlled by the browser/application state | Originates from the selected provider |
| Exported files | User-selected download location | Controlled by the user | No, unless the user uploads them elsewhere |
| Application logs | Container logging driver | Controlled by the Docker host | No by Forma; operators control log forwarding |

Forma does not require a provider key in `.env`, and provider keys must never be
persisted in PostgreSQL or application logs.

## When data leaves the stack

A model request is made only after the user clicks an explicit **Review** or
**Rewrite** control. Saving, autosaving, switching templates, previewing, and
exporting do not call an AI provider. The built-in 60-point document-quality
check also runs locally in the Go API and needs no provider key.

Before dispatch, the API recursively strips direct identity and contact fields,
including the resume owner's name and contact values. It sends the remaining
resume context required for the requested operation. A target role or job
description is included only when the user supplied it to that action.

Sanitization reduces direct identifiers; it is not guaranteed anonymization.
Employer names, schools, uncommon projects, dates, free-form text, and other
facts may still identify a person. Remove sensitive details before using a
hosted provider when that risk matters.

## Resume and LinkedIn imports

Uploaded Forma JSON, JSON Resume, DOCX, text-layer PDF, and official LinkedIn
data-export ZIP files are parsed in API memory to produce a preview. Forma does
not save the original file, write a temporary upload, persist the preview, or
send it to an AI provider. The user must explicitly apply the preview and save
the resulting structured resume before it reaches PostgreSQL.

Forma never fetches or scrapes a pasted LinkedIn profile URL. The browser can
add a validated `linkedin.com/in/...` URL as a profile link, but resume content
comes only from a file the user uploads. Imported mappings are conservative;
verify the preview because dates, employers, and other personal claims may be
misclassified in unstructured PDF or DOCX files.

File, text, archive-entry, expanded-size, and compression-ratio limits reduce
resource-exhaustion and archive-traversal risk. They do not make an untrusted
document harmless outside Forma; keep the source file protected and avoid
uploading data-export categories unrelated to the resume.

## Provider credentials

The browser submits `provider`, `model`, `api_key`, and an optional `base_url` to
`PUT /api/v1/ai/session`. The API stores them in a TTL-limited in-memory vault
and associates the vault entry with an opaque HttpOnly, SameSite cookie. Later
AI request bodies do not contain the key.

The session ends when its TTL expires or the API process restarts. Closing a
browser tab alone may not immediately erase an active server-side session. If a
key may have appeared in a screenshot, issue, log, shell history, or committed
file, revoke it at the provider and create a replacement.

For a remote HTTPS deployment, set `COOKIE_SECURE=true`, allow only the exact
web origin, and terminate TLS at a trusted reverse proxy. The default local
Compose setup uses HTTP on loopback and therefore disables secure cookies.

## Hosted providers and Ollama

Hosted services process content outside Forma's trust boundary. Before using
one, review its current data controls, retention terms, model-improvement
settings, regional endpoints, and organizational policy. A custom base URL may
point to any compatible service; the operator is responsible for trusting that
endpoint and its TLS configuration.

Ollama can keep inference local when its endpoint resolves to software running
on the same trusted machine or network. A remote Ollama endpoint is still an
external transfer.

The semantic part of a review is constrained to four fixed criteria. The model
does not choose score weights or a final score; the Go API verifies evidence
against the sanitized source before it can affect the score. This improves
repeatability but does not turn the score into a hiring or ATS guarantee.

## Logs and diagnostics

Safe logs may contain request timing, provider identifier, status category, and
an internal correlation ID. They must not contain:

- provider API keys or authorization headers;
- AI-session cookie values;
- raw resume, job-description, prompt, or model-response content;
- database credentials or full connection strings.

When reporting a bug, reproduce it with synthetic resume data and redact headers
and screenshots. Maintainers should request the minimum diagnostic data needed.

## Delete or reset local data

Stop containers while keeping resume data:

```bash
docker compose down
```

Permanently delete the Compose-managed PostgreSQL volume and all resumes in it:

```bash
docker compose down --volumes
```

The second command is destructive. Export or back up anything you need first.
Files already downloaded through the browser and external provider records are
not deleted by removing the Docker volume; delete those at their respective
locations.

## Backups

A database backup contains personal resume data and should be treated as
sensitive. Encrypt backups, restrict access, test restoration, and define a
retention period. Provider keys are never part of a legitimate Forma database
backup because they are not persisted there.

## No hidden telemetry

The default application architecture does not require analytics, advertising,
tracking pixels, or a hosted Forma account. A downstream deployment that adds
telemetry or remote logging must disclose it and update this document before
collecting data.

## Operator responsibilities

Docker host administrators can inspect containers, memory, volumes, and logs.
Protect the host account, keep Docker and dependencies patched, avoid exposing
port 3000 to untrusted networks, and use HTTPS plus authentication if deploying
beyond a single-user machine.

Report a suspected privacy or security defect through the private process in
[SECURITY.md](SECURITY.md), not in a public issue containing real data.
