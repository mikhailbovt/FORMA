# Contributing to Forma

Thank you for helping make private, high-quality resume tooling available to
everyone. Contributions of code, documentation, tests, accessibility fixes,
templates, translations, and careful bug reports are welcome.

By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Before you begin

- Search existing issues and pull requests before opening a duplicate.
- Discuss broad architecture, persistent-schema changes, new dependencies, or
  new provider protocols before investing in a large implementation.
- Never include real resumes, personal contact details, provider keys, session
  cookies, or raw production prompts in an issue, fixture, screenshot, or test.
- Report vulnerabilities privately according to [SECURITY.md](SECURITY.md).

## Development setup

The easiest production-shaped environment is Docker Compose:

```bash
cp .env.example .env
docker compose up --build -d
docker compose ps
```

PowerShell users can replace the first command with:

```powershell
Copy-Item .env.example .env
```

For host-based development, install the Go version declared in
`apps/api/go.mod` and a current Node.js LTS release with npm.

```bash
npm --prefix apps/web ci
cd apps/api && go test ./...
```

Run `make help` when GNU Make is available. All Make targets are wrappers around
ordinary `docker compose`, `npm`, and `go` commands, so Make is optional on
Windows.

## Repository guide

- `apps/web`: editor UI, preview, browser behavior, and web runtime proxy.
- `apps/api`: Go HTTP API, PostgreSQL migrations, validation, and AI adapters.
- `docs`: architecture, privacy behavior, product contracts, and project policies.
- `.github/workflows`: required continuous-integration checks.

Read [ARCHITECTURE.md](ARCHITECTURE.md) before changing service
boundaries, data persistence, AI transfers, or credential handling.

## Working on a change

1. Create a focused branch from the current default branch.
2. Keep the change small enough to review and separate unrelated cleanup.
3. Add tests that fail without the change and use synthetic data only.
4. Update user-facing docs when behavior, configuration, privacy, or data flow
   changes.
5. Run the checks below before opening a pull request.

Backend code should remain gofmt-formatted, propagate request cancellation,
bound external calls, avoid logging request bodies or secrets, and wrap errors
with useful context. Provider adapters must validate model output locally even
when the provider advertises schema enforcement.

Frontend changes should preserve keyboard operation, visible focus, semantic
controls, sufficient contrast, responsive behavior, and the explicit boundary
between ordinary editing and AI actions. Do not trigger a provider request from
autosave, field input, template selection, preview, or export.

## Required checks

```bash
docker compose config --quiet

npm --prefix apps/web ci
npm --prefix apps/web run build
npm --prefix apps/web run test --if-present
npm --prefix apps/web run test:sites --if-present

cd apps/api
test -z "$(find . -type f -name '*.go' -exec gofmt -l {} +)"
go vet ./...
go test ./...
go build -o ../../bin/forma-api ./cmd/api
```

On PowerShell, inspect `gofmt -l .`; it should print nothing. CI runs the same
frontend, backend, Compose, and container-build gates on every pull request.

## Database migrations

- Add a new forward migration; do not rewrite a migration that may already have
  been applied.
- Keep startup safe when upgrading an existing local volume.
- Include rollback or recovery notes for destructive and long-running changes.
- Test both a fresh database and an upgrade from the previous schema when the
  migration changes stored data.

## Adding an AI provider

A provider pull request must document and test:

- its canonical provider ID and authentication method;
- default and custom base-URL behavior, including server-side request-forgery
  protections;
- model-name handling without relying on a permanently current catalog;
- request timeout, retry, rate-limit, and cancellation behavior;
- structured-output validation and bounded repair behavior;
- redaction of keys, headers, prompts, resume data, and provider responses;
- what data leaves the machine and whether a fully local mode exists.

Update the provider matrix in [README.md](../README.md) and the data flow in
[PRIVACY.md](PRIVACY.md) when needed.

## Pull requests

A short pull-request template is filled in automatically. Issue templates for
bugs and feature requests are available from GitHub's **New issue** page; concise,
reproducible reports are more useful than long speculative designs.

A good pull request includes:

- a concise problem statement and the chosen behavior;
- screenshots or a short recording for visible UI changes, using synthetic data;
- test evidence and any intentionally untested boundary;
- migration, privacy, security, and compatibility impact;
- documentation updates and a clear follow-up list, if any.

Reviewers may ask for a change to be split when independent concerns make risk
or rollback difficult to understand. Maintainers can close inactive work after
giving reasonable notice; the contributor retains credit and can reopen the
discussion later.

## Licensing

By submitting a contribution, you agree that it may be distributed under the
repository's [Apache License 2.0](../LICENSE). Only submit work you have the right
to license.
