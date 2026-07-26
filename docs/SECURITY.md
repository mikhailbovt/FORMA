# Security policy

Forma handles personal resume content and short-lived AI-provider credentials.
Please report security and privacy defects privately so users have time to
upgrade before details become public.

## Support status

FORMA is an open-source project provided as-is. No branch, version or release
currently has a guaranteed security-support window, response time or fix
commitment. Security reports and focused patches are welcome, but availability
to review or ship them is not promised.

| Version | Support status |
| --- | --- |
| Default branch | Community reports and patches welcome; no guaranteed fixes |
| Older snapshots and forks | No guaranteed fixes |

## Report a vulnerability

Use the repository's **Security** tab and choose **Report a vulnerability** to
open a private GitHub Security Advisory. Do not open a public issue. If private
advisories are unavailable, contact the repository owner through a private
channel listed on their GitHub profile and include only enough information to
establish a secure reporting channel.

Include, when available:

- affected revision, component, and configuration;
- impact and the attacker's required access;
- minimal reproduction using synthetic resume data;
- relevant logs with credentials, cookies, prompts, and personal data removed;
- a proposed mitigation or patch, if you have one.

Do not send real resumes or live provider keys. If a key was exposed during
testing, revoke it before sending the report.

Reports are reviewed as availability permits. No acknowledgement, assessment,
fix, release or disclosure timeline is guaranteed. When a project maintainer
engages with a report, coordinated disclosure timing can be agreed with the
reporter based on severity, exploitability and fix availability.

## High-value report areas

Reports are especially useful when they demonstrate:

- provider-key persistence, logging, disclosure, or cross-session access;
- resume or job-description content sent without an explicit AI action;
- failure to remove intended identity and contact fields before provider calls;
- server-side request forgery through a custom provider base URL;
- session fixation, cookie theft, cross-site request forgery, or unsafe CORS;
- SQL injection, broken object authorization, or cross-user data access;
- stored or reflected script execution through resume or model output;
- unbounded request bodies, provider responses, retries, or resource use;
- exposed API/database ports or unsafe default deployment behavior;
- vulnerable dependency paths with a practical impact on Forma.

Automated scanner output without a reachable vulnerable path is still welcome
as a lead, but may be treated as hardening rather than a confirmed
vulnerability.

## Safe-harbor intent

For good-faith research performed to improve Forma's security, maintainers ask
that you:

- test only systems and data you own or are authorized to use;
- avoid privacy violations, service disruption, persistence, and data loss;
- stop after confirming the minimum evidence needed;
- keep findings confidential while a fix is prepared;
- comply with applicable law.

The project does not authorize testing third-party AI providers, GitHub, Docker,
or infrastructure that is not operated by Forma maintainers. No bug bounty or
payment is implied.

## Secret exposure

Provider keys are designed to live only in API process memory for a bounded TTL.
They do not belong in `.env`, source files, database rows, logs, screenshots, or
issues. Treat any committed or publicly pasted key as compromised: revoke it at
the provider, replace it, remove it from active content, and assess repository
history and logs rather than assuming deletion made it safe.

For deployment and data-flow details, see
[ARCHITECTURE.md](ARCHITECTURE.md) and
[PRIVACY.md](PRIVACY.md).
