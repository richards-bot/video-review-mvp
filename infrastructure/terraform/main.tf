terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
  default     = "eu-west-2"
}

variable "bucket_name" {
  description = "Name of the S3 bucket for encrypted video storage"
  type        = string
}

variable "viewer_origin" {
  description = "HTTPS origin of the viewer (e.g. https://review.example.com)"
  type        = string
}

variable "uploader_iam_user_arn" {
  description = "ARN of the IAM user or role used by the Go CLI uploader"
  type        = string
  default     = ""
}
