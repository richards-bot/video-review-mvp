# Spec: Secure Video Review MVP

**Status:** Implemented (v1.0)
**Spec source:** This file (`openspec/specs/secure-video-review-mvp.md`)
**Full technical detail:** [`SPEC.md`](../../SPEC.md) (898 lines, authoritative)
**Beads issues:** `video-review-mvp-06p`, `video-review-mvp-cy2`, `video-review-mvp-hm3`

---

## Problem

Journalists and legal teams routinely need to share sensitive video with external reviewers (lawyers, editors, investigators). Current options are all bad: email is too large; consumer cloud storage trusts the provider; enterprise DRM requires reviewer accounts. The team needs a minimal, auditable, zero-trust solution.

## Solution

Two-component system:

1. **Go CLI uploader** — transcodes video to fMP4, splits into chunks, encrypts each chunk with AES-256-GCM, uploads to S3, prints a share URL.
2. **Static browser viewer** — fetches encrypted chunks, decrypts in a Web Worker, feeds the MSE pipeline, plays back in a `<video>` element.

The encryption key travels only in the `#fragment` of the share URL, which browsers never send to any server.

## Acceptance Criteria

All criteria are verified green as of commit `3159679`.

| # | Criterion | Status |
|---|-----------|--------|
| AC-01 | End-to-end smoke test using pre-encrypted fixture passes without ffmpeg | ✅ |
| AC-02 | Chunk integrity — tampered bytes cause decryption error, not silent corruption | ✅ |
| AC-03 | CLI logs each pipeline stage to stderr; does not log the CEK | ✅ |
| AC-04 | Go test suite passes (`go test ./...`) with no skips on non-ffmpeg code | ✅ |
| AC-05 | Viewer JS test suite passes (`npm test` in `viewer/`) — 47 tests | ✅ |
| AC-06 | `go vet ./...` and `gofmt -l` produce no output | ✅ |
| AC-07 | `viewer/index.html` carries SRI `integrity=` on all `<script>` tags | ✅ |
| AC-08 | No secrets in source; `.env.example` documents required variables | ✅ |
| AC-09 | Terraform creates S3 bucket with SSE-S3 and correct CORS for viewer origin | ✅ |
| AC-10 | Share URL fragment never appears in S3 server access logs (browser design) | ✅ (by design) |

## Assumptions

- Reviewers use a modern evergreen browser (Chrome 100+, Firefox 100+, Safari 15+) with Web Crypto API and MSE support.
- The uploader has `ffmpeg`/`ffprobe` installed locally — these are NOT in the CI/dev environment.
- AWS credentials are provided via the standard credential chain; no IAM role or KMS key is strictly required for the MVP (SSE-S3 is sufficient).
- The share URL is transmitted out-of-band (Signal, encrypted email) — the tool does not solve distribution.
- Chunk size of 8 MB is suitable for typical network conditions; no adaptive bitrate is needed for MVP.

## Risks and Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| CEK leaks via HTTP Referer header | Low | Viewer served over HTTPS; fragment not included in Referer by spec |
| S3 presigned URL expiry vs. long review sessions | Medium | Default 7-day expiry; v1.1 will add configurable TTL |
| Browser memory exhaustion for large files | Low | Chunks limited to 8 MB; MSE pipeline releases appended buffers |
| ffmpeg codec compat with unusual input formats | Medium | CLI validates output with ffprobe before upload; unsupported inputs fail early |
| Nonce reuse if same (CEK, chunk_index) used twice | Very Low | asset_id is random per upload; new CEK per upload; deterministic nonce is safe |

## Non-Goals (v1.0)

- Public-key key wrapping (PKI, signal ratchet, etc.)
- Reviewer identity, SSO, or access control lists
- Forensic or visible watermarking
- Time-coded comments or annotations
- Seeking / random access within a video
- Adaptive bitrate streaming
- Per-reviewer access revocation
- Server-side audit logging beyond S3 access logs

These are tracked for v1.1 in `SPEC.md §12`.

## Implementation Notes

See `SPEC.md` for:
- `§3` — Cryptographic design (nonce construction, HKDF usage, key size rationale)
- `§4` — Data formats (S3 layout, chunk binary format, manifest JSON schema, share URL encoding)
- `§5` — Go CLI implementation (package structure, pipeline stages, concurrency model)
- `§6` — Browser viewer (Web Worker, MSE loop, memory management, CSP headers)
- `§7` — S3 configuration (bucket policy, CORS rules, IAM least-privilege)
- `§9` — Acceptance criteria (authoritative list)
- `§10` — Security properties and non-properties
- `docs/DECISIONS.md` — ADR-001 through ADR-005

## Change History

| Date | Bead | Summary |
|------|------|---------|
| 2026-04-23 | `video-review-mvp-06p` | Full MVP implemented — Go CLI, viewer, Terraform, tests, fixtures |
| 2026-04-23 | `video-review-mvp-cy2` | Vitest harness added, 47 viewer tests, gofmt drift fixed |
| 2026-04-23 | `video-review-mvp-hm3` | SRI hash added to `viewer/index.html` for SPEC compliance |
