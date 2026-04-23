# Architecture Decisions

## ADR-001: AES-256-GCM with Deterministic Nonces

**Date:** 2026-04-23
**Status:** Accepted

**Context:** We need symmetric encryption that works natively in both Go and the browser Web Crypto API, without a WASM dependency.

**Decision:** Use AES-256-GCM. Nonces are deterministic (prefix + chunk index) rather than random, because each asset has a fresh random CEK.

**Consequences:** (Key, nonce) pairs are unique per asset. Safe against nonce reuse. Enables reproducible test vectors. Ciphertext is authenticated — any modification is detected.

---

## ADR-002: Presigned URLs for Chunk Access (§7.3 Option A)

**Date:** 2026-04-23
**Status:** Accepted

**Context:** Chunk files in S3 must be accessible to the browser without requiring AWS credentials.

**Decision:** The uploader generates presigned GET URLs at upload time (default 48h expiry) and embeds them in the manifest. The manifest is publicly readable (as ciphertext) but the chunks require presigned URLs stored inside the encrypted manifest.

**Consequences:** Limits chunk access to 48h. The presigned URLs are inside the encrypted manifest, so S3 does not see them in plaintext. Simpler than Lambda@Edge auth. Access cannot be revoked before expiry; this is documented as a known limitation.

---

## ADR-003: No Build Step for Viewer

**Date:** 2026-04-23
**Status:** Accepted

**Context:** The viewer must be deployable as plain static files with no toolchain required.

**Decision:** Vanilla ES modules. No bundler, no transpiler, no framework.

**Consequences:** Larger payload than a bundled app for some dependency graphs, but since there are no dependencies this is fine. SRI hashes are computed by the deploy script rather than a build tool.

---

## ADR-004: Web Worker for Decryption

**Date:** 2026-04-23
**Status:** Accepted

**Context:** AES-GCM decryption of large chunks on the main thread would block the UI.

**Decision:** Offload all crypto.subtle calls to a dedicated `decrypt-worker.js`. CEK is imported once and cached in the worker.

**Consequences:** Decryption runs off the main thread. CEK bytes are transferred to the worker; the main thread zeros its copy. Worker is terminated after all chunks are decrypted.

---

## ADR-005: fMP4 for MSE Compatibility

**Date:** 2026-04-23
**Status:** Accepted

**Context:** The browser viewer uses Media Source Extensions for progressive playback. MSE requires an ISO BMFF (MP4) container with `moov` at the front and fragmented `moof+mdat` boxes.

**Decision:** ffmpeg flags `+frag_keyframe+empty_moov+default_base_moof+faststart` produce a valid fMP4. Fragment duration 2s aligns with typical keyframe interval.

**Consequences:** Clients can start playback before all chunks are downloaded. All target browsers support fMP4 via MSE.
