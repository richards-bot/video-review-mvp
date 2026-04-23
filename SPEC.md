# Secure Video Review MVP — Technical Specification

Version: 1.0  
Status: Implementation-ready  
Audience: Coding agent implementing the system

-----

## 1. Purpose and Scope

Build a two-component system for sharing legally-sensitive video between an uploader and one or more reviewers, such that the storage backend (AWS S3) never sees plaintext content and holds no decryption keys.

In scope:

- Go CLI for the uploader: transcode to proxy, encrypt, upload to S3
- Static HTML/JS web viewer: fetch encrypted chunks from S3, decrypt in browser, stream to `<video>` element
- Out-of-band key sharing via URL fragment
- Deterministic chunk encryption with authenticated ciphertext
- Fragmented MP4 (fMP4) for Media Source Extensions compatibility

Explicitly out of scope for MVP:

- Public-key key wrapping (keys are shared manually via URL fragment)
- Reviewer identity or SSO
- Forensic or visible watermarking
- Time-coded comments or annotations
- Seeking, random access, adaptive bitrate
- Multi-reviewer per-user attribution
- Access revocation
- Server-side audit logging (S3 access logs are considered sufficient)

These are v1.1+ concerns. The MVP validates the cryptographic and streaming core.

-----

## 2. Architecture Overview

```text
┌───────────────────┐          ┌─────────────────┐          ┌──────────────┐
│ Go CLI Uploader   │          │ Browser Viewer  │          │ S3           │
│                   │          │ (static page)   │          │              │
│ 1. ffmpeg → fMP4  │          │                 │          │ <asset_id>/  │
│ 2. chunk          │          │ 1. parse URL    │          │ manifest.enc │
│ 3. encrypt chunks │──PUT──▶  │ 2. fetch manifest◀─GET─    │ chunk_000000 │
│ 4. encrypt        │          │ 3. decrypt      │          │ chunk_000001 │
│    manifest       │          │ 4. fetch chunks │◀─GET──   │ ...          │
│ 5. print URL      │          │ 5. decrypt      │          │              │
│                   │          │ 6. MSE append   │          │              │
│                   │          │ 7. <video>      │          │              │
└───────────────────┘          └─────────────────┘          └──────────────┘

Share URL carries asset_id + CEK in fragment (#)
```

-----

## 3. Cryptographic Design

### 3.1 Primitives

- Symmetric cipher: AES-256-GCM
- Key derivation: HKDF-SHA256 (only used for nonce derivation — see below)
- Random source: crypto/rand (Go), crypto.getRandomValues (browser)

AES-256-GCM chosen because Web Crypto API supports it natively (no WASM dependency for the viewer) and Go’s crypto/cipher implements it with hardware AES-NI acceleration.

### 3.2 Per-asset keys

For each upload:

```text
CEK = 32 random bytes (AES-256 key)
asset_id = 16 random bytes, hex-encoded to 32 chars (S3 prefix)
```

### 3.3 Chunk encryption

Each chunk is encrypted independently so the viewer can decrypt and play progressively.

Nonce construction (deterministic, NOT random):

```text
nonce = 12 bytes =
  4 bytes constant prefix: 0x00 0x00 0x00 0x00
  8 bytes big-endian uint64: chunk_index
```

Deterministic nonces are safe here because:

- Each asset has its own fresh random CEK
- Chunk indices are unique within an asset
- AES-GCM is secure as long as (key, nonce) pairs are never reused

AAD (Additional Authenticated Data):

```text
aad = asset_id_hex || "|" || chunk_index_ascii
```

Example: `7f3a9c2e4b1d8f600123456789abcdef|42`

AAD binds each ciphertext to its position in the asset. Swapping chunks or substituting chunks from a different asset fails authentication.

Encryption:

```text
ciphertext_chunk = AES-256-GCM.seal(
  key = CEK,
  nonce = nonce,
  plaintext = chunk_plaintext,
  aad = aad
)
```

Output is plaintext concatenated with 16-byte GCM tag. Go’s `cipher.AEAD.Seal` produces this natively; browser `crypto.subtle.encrypt` with AES-GCM produces the same format.

### 3.4 Manifest encryption

The manifest (see §4.3) is itself encrypted with the CEK using the same scheme, with:

```text
nonce = 12 bytes:
  4 bytes constant prefix: 0xFF 0xFF 0xFF 0xFF (distinct from chunk prefix)
  8 bytes zero
aad = asset_id_hex || "|manifest"
```

The constant-prefix distinction prevents any ambiguity between chunk nonces and the manifest nonce.

### 3.5 CEK encoding

The CEK is shared via URL fragment. Encode as URL-safe base64 without padding (RFC 4648 §5):

- 32 bytes → 43 characters
- Example: `qZ8xJK3mN7pLvR2sT5wY9aB4eF6hU0cXKpL8JhMnQrS`

-----

## 4. Data Formats

### 4.1 S3 layout

```text
s3://<bucket>/<asset_id>/
  manifest.enc
  chunk_000000.enc
  chunk_000001.enc
  ...
  chunk_NNNNNN.enc
```

Chunk filenames use zero-padded 6-digit decimal indices. Maximum supported chunk count: 999,999 (sufficient for ~1TB of proxy at 1 MiB chunks).

### 4.2 Chunk file format

Raw ciphertext bytes as produced by AES-GCM Seal. No header, no framing. File size = plaintext size + 16 bytes (GCM tag).

### 4.3 Manifest schema

The manifest is a JSON object, UTF-8 encoded, then encrypted and stored as `manifest.enc`.

```json
{
  "version": 1,
  "asset_id": "7f3a9c2e4b1d8f600123456789abcdef",
  "created_at": "2026-04-23T14:30:00Z",
  "created_by": "rich@rdpryce.com",
  "source_filename": "interview-raw.mov",
  "codec": {
    "video": "avc1.64001f",
    "audio": "mp4a.40.2",
    "mime": "video/mp4; codecs=\"avc1.64001f, mp4a.40.2\""
  },
  "duration_ms": 127500,
  "width": 1920,
  "height": 1080,
  "framerate": 25.0,
  "chunk_size": 1048576,
  "chunk_count": 42,
  "total_bytes": 43829012,
  "plaintext_sha256": "3b8f...hex..."
}
```

Field notes:

- `version`: protocol version; fail fast on unknown versions
- `codec.mime`: exact MIME type passed to `MediaSource.addSourceBuffer()`
- `chunk_size`: plaintext chunk size in bytes. All chunks are exactly this size EXCEPT the final chunk, which may be smaller
- `total_bytes`: total plaintext size, for progress reporting and final-chunk size calculation
- `plaintext_sha256`: SHA-256 of the full plaintext proxy before chunking, for end-to-end integrity verification
- `created_by`: informational; for MVP this is the value of `$USER` or a `--author` flag. Not cryptographically bound to the uploader’s identity

### 4.4 Share URL format

```text
https://<viewer-host>/#v=1&a=<asset_id>&k=<cek_b64url>&b=<bucket>&r=<region>
```

- Everything after `#` is the fragment, never sent to the viewer host server
- `v`: URL schema version (currently `1`)
- `a`: `asset_id` (32 hex chars)
- `k`: CEK, URL-safe base64, no padding (43 chars)
- `b`: S3 bucket name
- `r`: AWS region (e.g. `eu-west-2`)

Example:

```text
https://review.example.com/#v=1&a=7f3a9c2e4b1d8f600123456789abcdef&k=qZ8xJK3mN7pLvR2sT5wY9aB4eF6hU0cXKpL8JhMnQrS&b=video-review-mvp&r=eu-west-2
```

-----

## 5. Go CLI Uploader

### 5.1 Module and dependencies

```text
module github.com/guardian/video-review-mvp

go 1.22

require (
  github.com/aws/aws-sdk-go-v2 v1.x
  github.com/aws/aws-sdk-go-v2/config v1.x
  github.com/aws/aws-sdk-go-v2/service/s3 v1.x
  github.com/spf13/cobra v1.x
)
```

External binary dependency: `ffmpeg` and `ffprobe` must be on PATH. The CLI MUST detect this at startup and fail with a clear error if missing.

### 5.2 Command structure

```text
video-review upload <source-file> [flags]
```

Flags:

- `--bucket string` S3 bucket (required; can be set via `VIDEO_REVIEW_BUCKET` env)
- `--region string` AWS region (required; can be set via `VIDEO_REVIEW_REGION` env or `AWS_REGION`)
- `--viewer-url string` Viewer base URL (required; can be set via `VIDEO_REVIEW_VIEWER_URL` env)
  e.g. `https://review.example.com`
- `--author string` Email or identifier of uploader (default: `$USER`)
- `--chunk-size int` Chunk size in bytes (default: `1048576` = 1 MiB)
- `--crf int` ffmpeg x264 CRF quality (default: `23`)
- `--max-bitrate string` ffmpeg maxrate (default: `"5M"`)
- `--keep-proxy` Keep intermediate proxy file after upload (default: false)
- `--output-format string` `"url"` or `"json"` (default: `"url"`)
- `-v, --verbose` Verbose logging

AWS credentials use the standard AWS SDK credential chain (env vars, shared config, IAM role). The CLI MUST NOT accept credentials as flags.

### 5.3 Pipeline stages

The uploader runs these stages sequentially. Each stage MUST log progress if `--verbose`.

#### Stage 1: Validate inputs

- Check source file exists and is readable
- Check `ffmpeg` and `ffprobe` are available (`exec.LookPath`)
- Validate bucket, region, viewer URL are set
- Validate viewer URL is `https://` (reject `http://` except `http://localhost` for dev)

#### Stage 2: Probe source

- Run `ffprobe -v error -print_format json -show_format -show_streams <source>`
- Extract duration, width, height, framerate for manifest
- Log a summary

#### Stage 3: Transcode to fragmented MP4

- Create temp directory: `os.MkdirTemp("", "video-review-*")`
- Path: `<tmpdir>/proxy.mp4`
- Run:

```bash
ffmpeg -i <source> \
  -c:v libx264 -preset medium -crf <crf> \
  -maxrate <max-bitrate> -bufsize <2x max-bitrate> \
  -pix_fmt yuv420p \
  -profile:v main -level 4.0 \
  -c:a aac -b:a 128k -ac 2 \
  -movflags +frag_keyframe+empty_moov+default_base_moof+faststart \
  -frag_duration 2000000 \
  -f mp4 \
  -y \
  <tmpdir>/proxy.mp4
```

- Stream `ffmpeg` stderr to logger if verbose
- On non-zero exit, fail with captured stderr

#### Stage 4: Generate keys and identifiers

- `CEK := 32 random bytes from crypto/rand`
- `asset_id_bytes := 16 random bytes from crypto/rand`
- `asset_id := hex.EncodeToString(asset_id_bytes)`

#### Stage 5: Compute plaintext SHA-256

- Stream-hash the proxy file with `sha256.New()` as it’s read
- Store digest for manifest

#### Stage 6: Chunk and encrypt

- Open proxy file for reading
- Read `chunk_size` bytes at a time
- For each chunk:
  - Construct nonce (§3.3)
  - Construct AAD (§3.3)
  - Encrypt with AES-256-GCM
  - Upload to S3 at `<asset_id>/chunk_NNNNNN.enc` using `PutObject`
  - Track `chunk_count` and running byte count
- Final chunk may be shorter than `chunk_size`; this is valid

#### Stage 7: Build and encrypt manifest

- Construct manifest JSON (§4.3)
- Serialise with `json.Marshal` (compact, no indent)
- Encrypt with the manifest nonce/AAD (§3.4)
- Upload to S3 at `<asset_id>/manifest.enc`

#### Stage 8: Emit share URL

- Construct URL per §4.4
- Print to stdout
- If `--output-format=json`, print:

```json
{
  "asset_id": "...",
  "bucket": "...",
  "region": "...",
  "viewer_url": "https://...",
  "share_url": "https://.../#v=1&a=...&k=...&b=...&r=...",
  "chunk_count": 42,
  "duration_ms": 127500
}
```

#### Stage 9: Cleanup

- Remove temp directory unless `--keep-proxy`

### 5.4 Concurrency

Chunk uploads MAY be parallelised to 4 concurrent S3 uploads using a worker pool. Chunks are independent once encrypted. However, the encryption MUST happen in chunk-index order on a single goroutine to avoid nonce-counter races. Simplest correct implementation: encrypt sequentially, upload via buffered channel to worker pool.

Do not optimise further for MVP. Premature concurrency here is a source of bugs.

### 5.5 Error handling

- Any S3 upload failure aborts the run. Partial uploads are acceptable to leave in S3 (the asset is unreachable without the manifest, which is uploaded last).
- Log the `asset_id` on every failure so the user can identify orphans for later cleanup.
- Exit code `0` on success, `1` on any error.

### 5.6 Logging

Use `log/slog` with a text handler. Default level INFO, `--verbose` enables DEBUG.

INFO-level messages the CLI MUST emit:

- Source file validated (with duration, resolution)
- Transcode started / completed (with elapsed time, output size)
- Upload started (with chunk count, total bytes)
- Upload progress every 10% or every 5 seconds (whichever is less frequent)
- Upload complete (with share URL)

### 5.7 Code organisation

```text
cmd/
  video-review/
    main.go        (cobra root, wires subcommands)
    upload.go      (upload subcommand)
internal/
  crypto/
    aead.go        (nonce construction, encrypt/decrypt helpers)
    aead_test.go   (MUST include round-trip and cross-check test vectors)
  chunker/
    chunker.go     (streaming chunker reading from io.Reader)
  ffmpeg/
    probe.go       (ffprobe wrapper)
    transcode.go   (ffmpeg wrapper)
  manifest/
    manifest.go    (schema, serialisation)
  s3upload/
    uploader.go    (S3 put with retry)
  share/
    url.go         (URL construction, base64 encoding)
```

### 5.8 Test requirements

- `internal/crypto`: unit tests with fixed-CEK fixed-nonce test vectors proving output is deterministic and decryptable
- `internal/chunker`: test edge cases — empty file, file < `chunk_size`, file = N * `chunk_size` exactly, file = N * `chunk_size` + 1
- `internal/manifest`: round-trip JSON marshal/unmarshal
- Integration test (optional, gated behind `TEST_S3_BUCKET` env): full upload round-trip against a real bucket

-----

## 6. Browser Viewer

### 6.1 Distribution

Single static page. No backend. No build step for MVP. Deliverables:

```text
viewer/
  index.html             (complete page, no external framework)
  viewer.js              (main page logic, MSE + fetch orchestration)
  decrypt-worker.js      (Web Worker for AES-GCM decryption)
  styles.css             (minimal styling)
```

Deploy as plain files to S3 bucket fronted by CloudFront with:

- `Content-Security-Policy: default-src 'self'; connect-src 'self' https://*.s3.amazonaws.com https://*.s3.*.amazonaws.com; script-src 'self'; style-src 'self' 'unsafe-inline'; media-src blob:; worker-src 'self'`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload`

All scripts MUST be loaded with Subresource Integrity (SRI) hashes in the HTML.

### 6.2 Page flow

1. Page loads. Main script parses `window.location.hash`.
2. If fragment missing or malformed: render an error state explaining the URL appears invalid.
3. Parse `v`, `a`, `k`, `b`, `r` from fragment.
4. Validate: `v === "1"`, `a` is 32 hex chars, `k` decodes to 32 bytes, `b` is non-empty, `r` is non-empty.
5. Construct S3 base URL: `https://${b}.s3.${r}.amazonaws.com/${a}/`
6. Show “Loading…” state.
7. Fetch `manifest.enc` via `fetch()`.
8. Decrypt manifest in the Web Worker.
9. Validate manifest: `version === 1`, `asset_id === a`, reasonable `chunk_count` and `chunk_size`.
10. Display asset metadata (duration, resolution, created_at, created_by) in the UI header.
11. Set up `MediaSource`, attach to `<video>` element, wait for `sourceopen`.
12. Add `SourceBuffer` with `manifest.codec.mime`.
13. Begin chunk fetch loop (§6.5).
14. Clear fragment from URL bar: `history.replaceState(null, '', window.location.pathname)` — this prevents the key remaining visible in the address bar after load.

### 6.3 UI (minimal)

```text
┌────────────────────────────────────────────┐
│ Secure Video Review                        │
├────────────────────────────────────────────┤
│ interview-raw.mov                          │
│ 1920×1080 · 2:07 · 42 MB                   │
│ Shared by rich@rdpryce.com                 │
│ 2026-04-23 14:30 UTC                       │
├────────────────────────────────────────────┤
│                                            │
│ [ VIDEO ELEMENT WITH CONTROLS ]            │
│                                            │
├────────────────────────────────────────────┤
│ Loading: ████████████░░░░░░ 12 / 42         │
└────────────────────────────────────────────┘
```

Use `<video controls>` with browser-native controls. Do not attempt custom download-prevention for MVP — this is out of scope.

### 6.4 Decryption Web Worker

`decrypt-worker.js` receives messages of shape:

```js
{ type: 'decrypt', assetId, chunkIndex, ciphertext, cek }
// or
{ type: 'decrypt-manifest', assetId, ciphertext, cek }
```

Responds with:

```js
{ type: 'result', chunkIndex, plaintext }
// or
{ type: 'error', chunkIndex, message }
```

The worker imports the CEK once per session and caches it:

```js
let cachedKey = null;
async function getKey(cekBytes) {
  if (cachedKey) return cachedKey;
  cachedKey = await crypto.subtle.importKey(
    'raw', cekBytes, { name: 'AES-GCM' }, false, ['decrypt']
  );
  return cachedKey;
}
```

Nonce construction in JS (match §3.3 exactly):

```js
function chunkNonce(chunkIndex) {
  const nonce = new Uint8Array(12);
  // bytes 0-3 are already 0x00
  const view = new DataView(nonce.buffer);
  view.setBigUint64(4, BigInt(chunkIndex), false); // big-endian
  return nonce;
}

function manifestNonce() {
  const nonce = new Uint8Array(12);
  nonce[0] = 0xFF; nonce[1] = 0xFF; nonce[2] = 0xFF; nonce[3] = 0xFF;
  return nonce;
}
```

AAD construction:

```js
function chunkAAD(assetId, chunkIndex) {
  return new TextEncoder().encode(`${assetId}|${chunkIndex}`);
}
function manifestAAD(assetId) {
  return new TextEncoder().encode(`${assetId}|manifest`);
}
```

Decryption:

```js
const plaintext = await crypto.subtle.decrypt(
  { name: 'AES-GCM', iv: nonce, additionalData: aad },
  key,
  ciphertext
);
```

Transfer `ArrayBuffer`s between worker and main thread using the transferable objects `postMessage(msg, [buffer])` to avoid copying.

### 6.5 Chunk fetch and MSE append loop

Main thread pseudocode:

```js
const queue = new AsyncQueue(); // bounded to ~3 chunks ahead
let nextAppendIndex = 0;

// Producer: fetch + decrypt
(async () => {
  for (let i = 0; i < manifest.chunk_count; i++) {
    const url = `${s3Base}chunk_${String(i).padStart(6, '0')}.enc`;
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(`chunk ${i}: HTTP ${resp.status}`);
    const ciphertext = await resp.arrayBuffer();
    const plaintext = await decryptInWorker(assetId, i, ciphertext, cek);
    await queue.put({ index: i, bytes: plaintext });
    updateProgress(i + 1, manifest.chunk_count);
  }
  queue.close();
})();

// Consumer: append to SourceBuffer in order
(async () => {
  while (true) {
    const item = await queue.get();
    if (item === null) break; // queue closed
    await appendBuffer(sourceBuffer, item.bytes);
  }
  mediaSource.endOfStream();
})();
```

`appendBuffer` helper must await `updateend`:

```js
function appendBuffer(sb, bytes) {
  return new Promise((resolve, reject) => {
    const onEnd = () => { cleanup(); resolve(); };
    const onErr = (e) => { cleanup(); reject(e); };
    const cleanup = () => {
      sb.removeEventListener('updateend', onEnd);
      sb.removeEventListener('error', onErr);
    };
    sb.addEventListener('updateend', onEnd);
    sb.addEventListener('error', onErr);
    sb.appendBuffer(bytes);
  });
}
```

### 6.6 Memory management

Long videos accumulate buffered data. On every 10th chunk appended:

```js
if (video.currentTime > 30 && !sb.updating) {
  sb.remove(0, video.currentTime - 30);
  await new Promise(r => sb.addEventListener('updateend', r, { once: true }));
}
```

This caps retained buffer at ~30 seconds of played video.

### 6.7 Error handling

The viewer MUST display a clear error state for each failure mode:

| Failure | User-visible message |
|--------------------------|----------------------------------------------------------------------------------------------------------------|
| Missing/malformed fragment | “This link appears invalid. Please check you have the complete URL.” |
| Manifest fetch 404 | “Content not found. The link may have expired or been withdrawn.” |
| Manifest fetch other error | “Unable to reach storage. Check your connection and try again.” |
| Manifest decryption fails | “The key in this link does not match the content. Please verify the link.” |
| Chunk decryption fails | “Content integrity check failed. This may indicate tampering. Do not trust this content.” |
| MSE unsupported | “This browser does not support the required video playback features. Please try Chrome, Firefox, or Safari 15+.” |
| Codec unsupported | “Your browser cannot play this video format.” |

Never display raw error strings from underlying APIs to the user.

### 6.8 Browser support matrix

MUST work on:

- Chrome/Edge 100+ (desktop)
- Firefox 100+ (desktop)
- Safari 15+ (macOS desktop, iPadOS)

Known limitation: iOS Safari on iPhone has incomplete MSE support. The viewer MUST detect this and show: “iPhone is not supported. Please use iPad, Mac, or a desktop browser.”

Feature detection at page load:

```js
if (!('MediaSource' in window)) { /* show unsupported */ }
if (!MediaSource.isTypeSupported(manifest.codec.mime)) { /* show unsupported */ }
if (!crypto.subtle || !crypto.subtle.decrypt) { /* show unsupported */ }
```

### 6.9 Security headers (set by hosting layer, documented here for reference)

See §6.1.

### 6.10 Test requirements

- Unit tests (Jest or Vitest) for fragment parsing, nonce construction, AAD construction, CEK base64 decoding
- A test fixture: a small pre-encrypted asset (manifest + 3 chunks) with a known CEK, committed to the repo. Test that the viewer can decrypt and validate the fixture
- Manual test matrix documented in `viewer/TESTING.md` listing browsers × operating systems × source codec combinations to verify

-----

## 7. S3 Configuration

The deployment bucket needs the following configuration. These are operational prerequisites, not part of the code deliverable, but the coding agent MUST produce a Terraform module or CloudFormation template that applies them.

### 7.1 Bucket policy

- Block all public access
- Object Ownership: BucketOwnerEnforced
- Default encryption: SSE-KMS with a customer-managed CMK
- Versioning: Enabled
- Lifecycle: objects in `*/chunk_*.enc` and `*/manifest.enc` expire 30 days after creation (MVP retention; production would differ)

### 7.2 CORS configuration

```json
{
  "CORSRules": [
    {
      "AllowedOrigins": ["https://<viewer-host>"],
      "AllowedMethods": ["GET", "HEAD"],
      "AllowedHeaders": ["Range", "If-Modified-Since", "If-None-Match"],
      "ExposeHeaders": ["Content-Length", "Content-Range", "ETag"],
      "MaxAgeSeconds": 3600
    }
  ]
}
```

### 7.3 IAM

Two IAM entities:

Uploader policy (attached to the IAM user/role the Go CLI uses):

- `s3:PutObject` on `arn:aws:s3:::<bucket>/*`
- `kms:Encrypt`, `kms:GenerateDataKey` on the CMK
- `s3:ListBucket` on `arn:aws:s3:::<bucket>` (for existence checks)

Viewer access — the objects are private but must be readable by unauthenticated browsers. Two options:

Option A (MVP-preferred): generate presigned URLs for each chunk and embed the manifest with signed URLs at upload time. Valid for e.g. 48 hours. Simpler, but URLs are time-limited.

Option B: public-read ACLs on objects. Unacceptable; rejected.

Option C (production): a Lambda@Edge or API Gateway endpoint that authenticates the request (e.g. checks a short-lived JWT) and redirects to a presigned URL. Out of scope for MVP.

For MVP: use Option A. The uploader generates presigned URLs at upload time (expiry default `48h`, configurable via `--expiry-hours`), and embeds the full list into a **public URL index** file uploaded at `<asset_id>/urls.json`. This file is itself encrypted with the CEK so S3 doesn’t learn the presigned URLs either. Modify §6.5 accordingly: the viewer first decrypts `urls.json`, then uses the presigned URLs listed within to fetch chunks.

Wait — this creates a chicken-and-egg: if `urls.json` needs a presigned URL to fetch, but the viewer doesn’t have one yet, it can’t start. Resolution: the manifest location is the one unauthenticated entry point. The viewer constructs the manifest URL from the bucket/region in the fragment and fetches it directly. The manifest (plaintext after decryption) contains presigned URLs for all chunks.

Updated manifest schema (replaces §4.3’s chunk-naming convention for MVP):

```json
{
  "version": 1,
  "asset_id": "...",
  "created_at": "...",
  "created_by": "...",
  "source_filename": "...",
  "codec": {
    "video": "...",
    "audio": "...",
    "mime": "..."
  },
  "duration_ms": 127500,
  "width": 1920,
  "height": 1080,
  "framerate": 25.0,
  "chunk_size": 1048576,
  "chunk_count": 42,
  "total_bytes": 43829012,
  "plaintext_sha256": "3b8f...hex...",
  "chunks": [
    { "index": 0, "url": "https://...presigned...", "size": 1048576 },
    { "index": 1, "url": "https://...presigned...", "size": 1048576 }
  ],
  "expires_at": "2026-04-25T14:30:00Z"
}
```

And: the manifest itself needs an unauthenticated fetch path. Simplest: make `manifest.enc` the sole bucket-policy-allowed public object per asset, OR use CloudFront with a public distribution fronting only `*/manifest.enc`. For MVP, configure the bucket policy to allow `s3:GetObject` on `*/manifest.enc` only (not chunks), which limits exposure to the ciphertext manifest — useless without the CEK.

Final access model for MVP:

- `manifest.enc` — publicly readable (as ciphertext); URL constructed by viewer from fragment
- `chunk_*.enc` — private; accessed via presigned URLs embedded in the decrypted manifest

Corresponding bucket policy (example):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowPublicManifestRead",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::<bucket>/*/manifest.enc"
    }
  ]
}
```

-----

## 8. Repository Layout

```text
.
├── README.md
├── SPEC.md (this document)
├── go.mod
├── go.sum
├── cmd/
│   └── video-review/
│       ├── main.go
│       └── upload.go
├── internal/
│   ├── crypto/
│   ├── chunker/
│   ├── ffmpeg/
│   ├── manifest/
│   ├── s3upload/
│   └── share/
├── viewer/
│   ├── index.html
│   ├── viewer.js
│   ├── decrypt-worker.js
│   ├── styles.css
│   ├── TESTING.md
│   └── fixtures/
│       ├── fixture-manifest.enc
│       ├── fixture-chunk-000000.enc
│       ├── fixture-chunk-000001.enc
│       ├── fixture-chunk-000002.enc
│       └── fixture.json (CEK + expected metadata for tests)
├── infrastructure/
│   └── terraform/
│       ├── main.tf
│       ├── bucket.tf
│       ├── kms.tf
│       └── iam.tf
└── scripts/
    ├── build.sh (builds Go binary for darwin/linux/windows)
    └── deploy-viewer.sh (syncs viewer/ to S3 + invalidates CloudFront)
```

-----

## 9. Acceptance Criteria

The MVP is complete when all of the following are demonstrable:

1. A user runs `video-review upload interview.mov --bucket ... --region ... --viewer-url ...` and receives a share URL on stdout within a time proportional to source duration (roughly 1× realtime for the `ffmpeg` stage on a modern laptop).
2. Opening the share URL in Chrome, Firefox, and Safari on desktop begins video playback within 5 seconds for a proxy under 50 MB.
3. Playback is smooth to completion for videos up to 10 minutes in length.
4. Modifying any single byte of any chunk in S3 causes the viewer to display the integrity-failure error and halt playback.
5. Modifying the `k` parameter in the URL fragment causes the viewer to display the key-mismatch error.
6. Removing or truncating the URL fragment causes the viewer to display the invalid-link error.
7. The S3 bucket access logs show only opaque ciphertext being served — no filename, comment, or content metadata appears in any server-side log.
8. The Go CLI and viewer share a fixture test: the CLI can produce a fixture, the viewer can decrypt that fixture, and a Go test can decrypt fixtures produced by a separate browser-side encrypt-and-save utility (if built).
9. Unit test coverage in `internal/crypto` is 100%; in other internal packages ≥ 80%.
10. `go vet`, `staticcheck`, and `gofmt -l` produce no findings.

-----

## 10. Security Properties and Non-properties

### 10.1 What this MVP provides

- Confidentiality of content at rest in S3: S3 stores only AES-256-GCM ciphertext under keys S3 never sees.
- Integrity of content: any modification of any chunk or the manifest is detected and rejected.
- No server-side trust required for secrecy: AWS operators with full S3 access cannot decrypt content.
- No network-path trust required for secrecy: TLS protects the key in transit via URL fragment, and the fragment is never sent to servers in HTTP requests.

### 10.2 What this MVP explicitly does NOT provide

- Protection against a compromised viewer host. If the static HTML/JS is modified by an attacker with write access to the viewer origin, they can exfiltrate the CEK. Mitigations (SRI, CSP, hash publication) raise the bar but don’t eliminate the risk. Zero-knowledge in the browser is unavoidably weaker than zero-knowledge in a signed native client.
- Protection against a compromised reviewer device. Plaintext exists in browser memory during playback. A compromised browser can record it.
- Leak attribution. There is no watermarking. If a reviewer records their screen, there is no technical means to identify them as the source.
- Access revocation. Once the CEK is shared, it cannot be un-shared. Presigned URL expiry limits the window but does not invalidate the key itself.
- Reviewer identity assurance. Anyone with the URL can view. The URL IS the credential.
- Defence against legal compulsion of the sender or a reviewer to produce the CEK.

These limitations MUST be documented in the `README.md` and acknowledged in the UI via a small “About security” link on the viewer page.

-----

## 11. Implementation Notes for the Agent

### 11.1 Order of work

Recommended build order:

1. `internal/crypto` with round-trip tests — establish the cryptographic contract
2. Matching JS decryption in isolation — prove Go-encrypt/JS-decrypt interop with hardcoded test vectors
3. `internal/chunker` and `internal/manifest`
4. `internal/s3upload`
5. `internal/ffmpeg`
6. Wire into `cmd/video-review/upload.go`
7. Static viewer: fragment parsing, manifest fetch, one-shot full decrypt-and-blob playback (pre-MSE) to validate end-to-end
8. Migrate viewer to MSE streaming playback
9. Error states and UI polish
10. Terraform and deploy scripts

Step 2 is the highest-leverage validation: if Go-encrypted bytes don’t decrypt correctly in the browser, nothing else matters. Do it early with fixed test vectors on both sides.

### 11.2 Don’ts

- Don’t introduce a JavaScript build step, bundler, or framework for the viewer MVP. Vanilla ES modules only.
- Don’t implement any form of CEK escrow, backup, or recovery. The CEK is strictly ephemeral.
- Don’t log the CEK anywhere — not in stdout, not in debug logs, not in manifest metadata.
- Don’t write plaintext video to disk anywhere on the viewer side. Memory only.
- Don’t use `Math.random()` for anything cryptographic. Only `crypto.getRandomValues` and `crypto/rand`.
- Don’t implement custom crypto primitives. Use Go stdlib `crypto/cipher` and Web Crypto `subtle`.
- Don’t copy CEK bytes into strings (they become unclearable in JS). Work with `Uint8Array` throughout and zero the array after key import where possible.

### 11.3 Things to verify interactively during development

- That the fMP4 output of `ffmpeg` actually plays via MSE. If you see `QuotaExceededError` or silent failures on `appendBuffer`, the `movflags` are wrong.
- That Go’s AES-GCM ciphertext format (`plaintext || tag`) matches what Web Crypto `subtle.decrypt` expects. They do — but verify with a hardcoded test vector before building anything else.
- That the manifest MIME string matches the actual codec profile `ffmpeg` produced. `ffprobe -v error -show_entries stream=codec_tag_string,profile -of default=nw=1 proxy.mp4` gives you what to put in the manifest.

-----

## 12. Out-of-scope — v1.1 Roadmap (for context only)

These are NOT part of the MVP but shape architectural choices:

- Public-key CEK wrapping: replace URL-fragment CEK with per-reviewer X25519 wrapped CEKs
- SSO-authenticated reviewer identity: replace share-URL-as-credential with authenticated access
- Visible watermarking: overlay reviewer identity on `<video>` via canvas compositing
- Time-coded encrypted comments: separate comment stream encrypted under a distinct key
- Server-side audit log: append-only, cryptographically chained

The MVP’s data formats (manifest schema, chunk layout, URL format) should be designed so these can be added without a breaking change. Where MVP choices preclude a future feature, the spec should call it out — none are currently known to conflict.

-----

## 13. Definition of Done

Ship when:

- All acceptance criteria in §9 pass
- `README.md` documents: installation, usage example, security properties/non-properties (§10), known browser limitations
- Terraform applies cleanly to a fresh AWS account and produces a working environment
- A recorded demo exists showing upload → share → view round-trip
- Code review by at least one other engineer confirms the cryptographic contract between uploader and viewer is correctly implemented

End of specification.
