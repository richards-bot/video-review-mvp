# Secure Video Review MVP

Share legally-sensitive video with reviewers via a browser link. Content is encrypted client-side before upload — AWS S3 stores only ciphertext and never holds decryption keys.

## How It Works

1. **Upload**: Run the Go CLI. It transcodes the video to a browser-compatible proxy, encrypts it in chunks with AES-256-GCM, uploads to S3, and prints a share URL.
2. **Share**: Send the URL to reviewers. The decryption key is in the URL fragment — it is never sent to any server.
3. **View**: The browser fetches encrypted chunks via presigned URLs, decrypts in a Web Worker, and plays progressively via Media Source Extensions.

## Prerequisites

- Go 1.22+ (for building the CLI)
- ffmpeg and ffprobe on PATH (for transcoding)
- AWS credentials with the uploader IAM policy applied
- An S3 bucket and CloudFront distribution (see `infrastructure/terraform/`)

## Quick Start

### 1. Build the CLI

```bash
./scripts/build.sh
# Binary at dist/video-review-linux-amd64 (or darwin/windows)
```

### 2. Configure

```bash
export VIDEO_REVIEW_BUCKET=my-video-review-bucket
export VIDEO_REVIEW_REGION=eu-west-2
export VIDEO_REVIEW_VIEWER_URL=https://review.example.com
```

Or pass flags directly:

```bash
./dist/video-review-linux-amd64 upload interview.mov \
  --bucket my-video-review-bucket \
  --region eu-west-2 \
  --viewer-url https://review.example.com
```

### 3. Upload

```bash
video-review upload interview.mov
# → https://review.example.com/#v=1&a=7f3a9c...&k=qZ8x...&b=my-bucket&r=eu-west-2
```

Share the printed URL with reviewers.

### 4. Deploy the viewer

```bash
VIEWER_BUCKET=my-viewer-bucket CF_DISTRIBUTION_ID=EXAMPLEID ./scripts/deploy-viewer.sh
```

## Infrastructure

Provision the S3 bucket and IAM policies with Terraform:

```bash
cd infrastructure/terraform
terraform init
terraform apply \
  -var="bucket_name=my-video-review-bucket" \
  -var="viewer_origin=https://review.example.com"
```

## Running Tests

```bash
go test ./...
```

Tests in `internal/crypto/` verify round-trip and cross-check vectors. Tests in `internal/chunker/` cover edge cases. Integration tests requiring a real S3 bucket are gated behind `TEST_S3_BUCKET`.

## Security Properties

### What this provides

- **Confidentiality at rest**: S3 stores only AES-256-GCM ciphertext. AWS operators cannot decrypt content.
- **Integrity**: Any modification to a chunk or the manifest is detected and rejected by the viewer.
- **No server-side key trust**: The decryption key lives only in the URL fragment; it is never logged or stored server-side.

### What this does NOT provide

- **Protection against a compromised reviewer device**: Plaintext video exists in browser memory during playback.
- **Access revocation**: Presigned URLs expire (default 48h) but the CEK cannot be invalidated.
- **Reviewer identity assurance**: Anyone with the URL can view. The URL is the credential.
- **Watermarking or leak attribution**: No technical means to identify the source of a leaked recording.
- **Protection against a compromised viewer host**: If the HTML/JS is modified by an attacker, they can exfiltrate the CEK.
- **Legal compulsion protection**: A reviewer can be compelled to produce the CEK.

These limitations are documented in §10 of SPEC.md.

## Browser Support

| Browser | Minimum Version |
|---------|----------------|
| Chrome / Edge | 100+ |
| Firefox | 100+ |
| Safari | 15+ (macOS, iPadOS) |
| Safari (iPhone) | Not supported — MSE is incomplete |

## File Layout

```
cmd/video-review/     Go CLI (main + upload command)
internal/
  crypto/             AES-256-GCM helpers and test vectors
  chunker/            Streaming chunk reader
  ffmpeg/             ffprobe and ffmpeg wrappers
  manifest/           Manifest schema and serialisation
  s3upload/           S3 put with retry and presign
  share/              Share URL construction
viewer/               Static browser viewer (no build step)
infrastructure/
  terraform/          S3, KMS, IAM configuration
scripts/
  build.sh            Cross-platform Go binary build
  deploy-viewer.sh    Sync viewer to S3 + CloudFront invalidation
```

## License

Internal project — not for public distribution.
