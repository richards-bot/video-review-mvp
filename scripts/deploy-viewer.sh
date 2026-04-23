#!/usr/bin/env bash
# Deploy the static viewer to S3 and optionally invalidate CloudFront.
#
# Usage:
#   VIEWER_BUCKET=my-viewer-bucket ./scripts/deploy-viewer.sh
#   VIEWER_BUCKET=my-viewer-bucket CF_DISTRIBUTION_ID=EXAMPLEID ./scripts/deploy-viewer.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
VIEWER_DIR="${REPO_ROOT}/viewer"

: "${VIEWER_BUCKET:?Set VIEWER_BUCKET to the S3 bucket hosting the viewer}"
VIEWER_PREFIX="${VIEWER_PREFIX:-}"  # optional prefix within the bucket

# ── Compute SRI hashes ───────────────────────────────────────────────────────
compute_sri() {
  local file="$1"
  echo "sha384-$(openssl dgst -sha384 -binary "${file}" | openssl base64 -A)"
}

VIEWER_JS_SRI=$(compute_sri "${VIEWER_DIR}/viewer.js")
WORKER_JS_SRI=$(compute_sri "${VIEWER_DIR}/decrypt-worker.js")

echo "SRI viewer.js:          ${VIEWER_JS_SRI}"
echo "SRI decrypt-worker.js:  ${WORKER_JS_SRI}"

# ── Upload files ─────────────────────────────────────────────────────────────
S3_PREFIX="${VIEWER_BUCKET}${VIEWER_PREFIX:+/${VIEWER_PREFIX}}"

echo ""
echo "Uploading viewer to s3://${S3_PREFIX}/ ..."

aws s3 sync "${VIEWER_DIR}" "s3://${S3_PREFIX}/" \
  --exclude "fixtures/*" \
  --exclude "TESTING.md" \
  --content-type "text/html" \
  --include "*.html" \
  --cache-control "no-cache, no-store, must-revalidate"

aws s3 cp "${VIEWER_DIR}/viewer.js" "s3://${S3_PREFIX}/viewer.js" \
  --content-type "application/javascript" \
  --cache-control "public, max-age=31536000, immutable"

aws s3 cp "${VIEWER_DIR}/decrypt-worker.js" "s3://${S3_PREFIX}/decrypt-worker.js" \
  --content-type "application/javascript" \
  --cache-control "public, max-age=31536000, immutable"

aws s3 cp "${VIEWER_DIR}/styles.css" "s3://${S3_PREFIX}/styles.css" \
  --content-type "text/css" \
  --cache-control "public, max-age=31536000, immutable"

# ── CloudFront invalidation ──────────────────────────────────────────────────
if [[ -n "${CF_DISTRIBUTION_ID:-}" ]]; then
  echo ""
  echo "Invalidating CloudFront distribution ${CF_DISTRIBUTION_ID}..."
  aws cloudfront create-invalidation \
    --distribution-id "${CF_DISTRIBUTION_ID}" \
    --paths "/*"
  echo "Invalidation created."
fi

echo ""
echo "Viewer deployed."
echo ""
echo "Add these SRI hashes to index.html if serving with strict CSP:"
echo "  viewer.js:         ${VIEWER_JS_SRI}"
echo "  decrypt-worker.js: ${WORKER_JS_SRI}"
