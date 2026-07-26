# Prototype Instructions

Run the local server yourself and open the preview in the browser available to this environment. Do not give the user server-start instructions when you can run it.

Before making substantial visual changes, use the Product Design plugin's `get-context` skill when the visual source is unclear or no longer matches the current goal. When the user gives durable prototype-specific design feedback, preferences, or decisions, record them in `AGENTS.md`.

Build app UI in `src/`. Keep `.openai/hosting.json`, `worker/index.js`, `scripts/prepare-sites-build.mjs`, and `tests/sites-worker.test.mjs` intact so the same local prototype can be handed to Sites. Before a Sites handoff, run `npm run build` and `npm run test:sites`; the build must leave `dist/client/index.html`, `dist/server/index.js`, and `dist/.openai/hosting.json`.

## Selected Forma direction

- Keep the approved monochrome split editor: navigation and structured editing on the left, an always-visible document preview on the right.
- Keep all left-rail section rows visually consistent. `Header` and `Summary` use semantic icons from the same family, size, stroke, spacing, and baseline as every other section.
- Keep templates, AI settings, import, and review discoverable from the left/editor workspace instead of hiding core actions in an overflow-only path.
- Full-document review is always explicit and opt-in. Its idle state has a dedicated run button; opening the panel never sends resume data to a provider.
- The local rubric and checklist work without an API key. AI feedback stays a separate optional layer, and reruns must expose a visible in-progress state.
- Inline AI actions may operate on selected text, while the left editor workspace owns the full-document review flow and keeps the preview visible.
