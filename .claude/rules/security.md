# Security Rules

Security issues block commits.

## Pre-Commit Checklist

- No hardcoded secrets (API keys, passwords, tokens) — use environment variables
- All user inputs validated and sanitized
- SQL: parameterized queries only (never string interpolation)
- XSS: output encoding, CSP headers
- CSRF protection on state-changing endpoints
- Auth/authz on protected routes
- Rate limiting on public endpoints
- Error messages don't leak internals or PII
- Logging excludes secrets and PII

## If a Security Issue Is Found

1. STOP — don't commit or deploy
2. Fix the vulnerability immediately
3. Rotate any exposed secrets
4. Search codebase for similar issues
5. Document in DECISIONS.md

## Files Needing Extra Scrutiny

- `**/auth/**`, `**/api/**`, `**/*service*`, `**/*middleware*`
- `.env*`, `**/config/**`
