resource "aws_kms_key" "video_review" {
  description             = "CMK for video-review S3 server-side encryption"
  enable_key_rotation     = true
  deletion_window_in_days = 30

  tags = {
    Project = "video-review-mvp"
  }
}

resource "aws_kms_alias" "video_review" {
  name          = "alias/video-review-mvp"
  target_key_id = aws_kms_key.video_review.key_id
}

output "kms_key_arn" {
  description = "ARN of the KMS CMK"
  value       = aws_kms_key.video_review.arn
}

output "kms_key_id" {
  description = "Key ID of the KMS CMK"
  value       = aws_kms_key.video_review.key_id
}
