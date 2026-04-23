# Uploader IAM policy
resource "aws_iam_policy" "uploader" {
  name        = "video-review-uploader"
  description = "Allows the video-review CLI to upload encrypted assets"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "S3PutObjects"
        Effect = "Allow"
        Action = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.video_review.arn}/*"
      },
      {
        Sid    = "S3ListBucket"
        Effect = "Allow"
        Action = ["s3:ListBucket"]
        Resource = aws_s3_bucket.video_review.arn
      },
      {
        Sid    = "S3GetObjectForPresign"
        Effect = "Allow"
        Action = ["s3:GetObject"]
        Resource = "${aws_s3_bucket.video_review.arn}/*"
      },
      {
        Sid    = "KMSEncrypt"
        Effect = "Allow"
        Action = [
          "kms:Encrypt",
          "kms:GenerateDataKey",
          "kms:DescribeKey"
        ]
        Resource = aws_kms_key.video_review.arn
      }
    ]
  })
}

# Optionally attach the policy to an existing IAM user/role
resource "aws_iam_policy_attachment" "uploader" {
  count = var.uploader_iam_user_arn != "" ? 1 : 0

  name       = "video-review-uploader-attachment"
  policy_arn = aws_iam_policy.uploader.arn
  users      = []
  roles      = []
  groups     = []
}

output "uploader_policy_arn" {
  description = "ARN of the uploader IAM policy"
  value       = aws_iam_policy.uploader.arn
}
