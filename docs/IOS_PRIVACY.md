# FORMA for iOS privacy policy

Effective date: July 30, 2026

FORMA for iOS is a local-first resume builder for iPhone and iPad. It does not
require a FORMA account and does not include advertising, analytics, tracking,
or a developer-operated cloud service.

This policy describes the native iOS app. The self-hosted web edition has a
different storage architecture, documented in [PRIVACY.md](PRIVACY.md).

## Data stored on your device

FORMA stores resumes, optional profile photos, app preferences, and the local
resume library on your device. API keys for AI providers are stored in Apple
Keychain and are not included in resume exports.

Typing, saving, importing, previewing, deterministic resume scoring, and
exporting do not contact an AI provider. Deleting the app removes its local app
data according to iOS behavior. Data already exported to Files or another app
remains wherever you saved or shared it.

## Optional AI features

AI Review and Rewrite are optional. When you explicitly start one of those
actions and confirm the disclosure, FORMA sends the sanitized resume content
and any job description you supplied directly to the provider you selected.
The resume owner's name and direct contact fields are removed before sending.

Sanitization reduces direct identifiers; it does not guarantee anonymity.
Employer names, schools, dates, uncommon projects, and free-form text may still
identify a person. Review sensitive content before using a hosted provider.

FORMA supports hosted providers, custom OpenAI-compatible endpoints, and
Ollama on a reachable local network. Those services are outside FORMA's trust
boundary. Their availability, processing locations, retention, model-training
controls, account terms, and privacy practices are governed by their own
policies. A custom or remote endpoint receives whatever content you choose to
send to it.

## Imports and exports

Files are accessed only after you select them. FORMA can import FORMA JSON,
JSON Resume, editable DOCX, text-layer PDF, and an official LinkedIn
data-export ZIP. Import parsing happens locally. FORMA does not fetch or scrape
LinkedIn profile pages. Always verify imported details before using a resume.

Exports are written to a location or destination you choose. API keys are not
included in exported PDF, DOCX, or JSON files.

## Optional Support purchase

The one-time Support purchase is a voluntary tip to the developer. Apple
processes the StoreKit transaction under Apple's terms. FORMA has no payment
backend and does not receive your full payment-card details. The purchase
unlocks no feature and creates no support or development commitment.

## Your choices

You can use all core resume features without configuring an AI provider or
making a purchase. You control whether to:

- add or remove local resume data and photos;
- import or export files;
- store or delete a provider API key;
- confirm or cancel each optional transfer to an AI provider; and
- make the optional Support purchase.

To remove information retained by a hosted AI provider, use that provider's
account and privacy controls. FORMA cannot delete records held by third parties.

## Children

FORMA is a general productivity app and is not directed to children. It does
not knowingly operate an account system or collect children's data for the
developer.

## Changes and contact

Material changes to this policy will be published in this repository with a
new effective date. For a privacy question that does not contain personal
resume data, open a [GitHub issue](https://github.com/mikhailbovt/FORMA/issues).
Do not post resumes, API keys, or other sensitive information publicly.

Report a privacy or security vulnerability privately through the repository's
Security tab as described in [SECURITY.md](SECURITY.md).
