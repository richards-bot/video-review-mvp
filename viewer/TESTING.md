# Viewer — Manual Test Matrix

This document describes the manual testing required to verify the viewer works
across all supported browsers and platforms.

## Browser Support Matrix

| Browser | OS | Version | Status |
|---------|----|---------| -------|
| Chrome | macOS | 100+ | Required |
| Chrome | Windows 10/11 | 100+ | Required |
| Chrome | Linux | 100+ | Required |
| Edge | Windows 10/11 | 100+ | Required |
| Firefox | macOS | 100+ | Required |
| Firefox | Windows 10/11 | 100+ | Required |
| Safari | macOS | 15+ | Required |
| Safari | iPadOS | 15+ | Required |
| Safari | iOS (iPhone) | any | Should show unsupported message |

## Test Cases

### TC-1: Valid URL — Full Playback

1. Upload a test video using the Go CLI.
2. Open the share URL in the target browser.
3. **Expected**: metadata displays, video begins playing within 5 seconds,
   playback completes without errors.

### TC-2: Invalid Fragment

1. Open the viewer with no fragment: `https://<host>/`
2. **Expected**: "This link appears invalid. Please check you have the complete URL."

### TC-3: Truncated Fragment

1. Open with only `#v=1&a=7f3a9c2e4b1d8f600123456789abcdef` (no key).
2. **Expected**: "This link appears invalid."

### TC-4: Wrong CEK

1. Open a valid share URL but flip one character in the `k=` parameter.
2. **Expected**: "The key in this link does not match the content."

### TC-5: Tampered Chunk

1. Manually overwrite one byte in `chunk_000000.enc` in S3.
2. Open the share URL.
3. **Expected**: "Content integrity check failed. This may indicate tampering."

### TC-6: Expired Presigned URLs

1. Use a share URL after the 48-hour expiry window.
2. **Expected**: "Content not found. The link may have expired or been withdrawn."
   (The manifest.enc is public so the manifest fetches OK; the first chunk's
   presigned URL returns 403/404.)

### TC-7: iPhone Detection

1. Open the share URL in Safari on an iPhone (iOS).
2. **Expected**: "iPhone is not supported. Please use iPad, Mac, or a desktop browser."

### TC-8: URL Fragment Cleared After Load

1. Open a valid share URL.
2. After the page loads, check the URL bar.
3. **Expected**: The fragment (`#...`) is no longer visible in the address bar.

### TC-9: Memory Management — Long Video

1. Upload a 10-minute video.
2. Play to completion in Chrome.
3. Open DevTools → Memory and observe that buffered video data is trimmed as
   playback progresses.
4. **Expected**: No `QuotaExceededError`; memory stays bounded.

## Source Codec Matrix

| Source format | Transcode result | Playback |
|---------------|-----------------|---------|
| .mp4 H.264 | fMP4 H.264 | Required |
| .mov ProRes | fMP4 H.264 | Required |
| .mp4 H.265 | fMP4 H.264 | Required (transcoded) |
| .avi DivX | fMP4 H.264 | Required (transcoded) |
| .mkv H.264 | fMP4 H.264 | Required |

## SRI Hash Update Process

After modifying `viewer.js` or `decrypt-worker.js`, update the SRI hashes
referenced in `index.html` (when used with a CDN or strict CSP that enforces
script integrity):

```bash
openssl dgst -sha384 -binary viewer.js | openssl base64 -A
openssl dgst -sha384 -binary decrypt-worker.js | openssl base64 -A
```

Then update the `integrity` attributes in `index.html` accordingly.

## Fixture-Based Unit Tests

The `viewer/fixtures/` directory contains a pre-encrypted asset for
deterministic testing. See `fixtures/fixture.json` for the CEK and expected
metadata. These fixtures are used by the Go fixture verification test in
`internal/crypto/aead_test.go`.

To run the browser-side fixture test, open `viewer/index.html` locally and
pass the fixture parameters via the URL fragment using the values in
`fixtures/fixture.json`.
