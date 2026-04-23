resource "aws_s3_bucket" "video_review" {
  bucket = var.bucket_name

  tags = {
    Project = "video-review-mvp"
  }
}

# Block all public access (enforced at bucket level)
resource "aws_s3_bucket_public_access_block" "video_review" {
  bucket = aws_s3_bucket.video_review.id

  block_public_acls       = true
  block_public_policy     = false # we add a specific policy below for manifest.enc
  ignore_public_acls      = true
  restrict_public_buckets = false
}

# Bucket ownership — BucketOwnerEnforced disables ACLs
resource "aws_s3_bucket_ownership_controls" "video_review" {
  bucket = aws_s3_bucket.video_review.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

# Versioning
resource "aws_s3_bucket_versioning" "video_review" {
  bucket = aws_s3_bucket.video_review.id

  versioning_configuration {
    status = "Enabled"
  }
}

# Default SSE-KMS encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "video_review" {
  bucket = aws_s3_bucket.video_review.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.video_review.arn
    }
    bucket_key_enabled = true
  }
}

# Lifecycle: expire chunks and manifests after 30 days
resource "aws_s3_bucket_lifecycle_configuration" "video_review" {
  bucket = aws_s3_bucket.video_review.id

  rule {
    id     = "expire-encrypted-assets"
    status = "Enabled"

    filter {
      prefix = ""
    }

    expiration {
      days = 30
    }

    noncurrent_version_expiration {
      noncurrent_days = 7
    }
  }
}

# CORS for browser viewer access
resource "aws_s3_bucket_cors_configuration" "video_review" {
  bucket = aws_s3_bucket.video_review.id

  cors_rule {
    allowed_origins = [var.viewer_origin]
    allowed_methods = ["GET", "HEAD"]
    allowed_headers = ["Range", "If-Modified-Since", "If-None-Match"]
    expose_headers  = ["Content-Length", "Content-Range", "ETag"]
    max_age_seconds = 3600
  }
}

# Bucket policy: allow public read of manifest.enc objects only
resource "aws_s3_bucket_policy" "video_review" {
  bucket = aws_s3_bucket.video_review.id

  # Depends on the public access block disabling block_public_policy=false
  depends_on = [aws_s3_bucket_public_access_block.video_review]

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowPublicManifestRead"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.video_review.arn}/*/manifest.enc"
      }
    ]
  })
}

output "bucket_name" {
  description = "S3 bucket name"
  value       = aws_s3_bucket.video_review.id
}

output "bucket_arn" {
  description = "S3 bucket ARN"
  value       = aws_s3_bucket.video_review.arn
}
