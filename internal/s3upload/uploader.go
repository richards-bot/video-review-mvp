// Package s3upload handles uploading encrypted assets to S3.
package s3upload

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps the S3 service for asset uploads.
type Client struct {
	s3     *s3.Client
	bucket string
	log    *slog.Logger
}

// New creates a new upload client.
func New(s3Client *s3.Client, bucket string, log *slog.Logger) *Client {
	return &Client{s3: s3Client, bucket: bucket, log: log}
}

// PutObject uploads data to S3 at bucket/key with retry on transient errors.
func (c *Client) PutObject(ctx context.Context, key string, data []byte) error {
	const maxRetries = 3
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(data),
		})
		if err == nil {
			return nil
		}
		lastErr = err
		c.log.Warn("S3 put failed, retrying",
			"key", key,
			"attempt", attempt,
			"error", err,
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		}
	}
	return fmt.Errorf("s3 put %s: %w", key, lastErr)
}

// PresignGetObject generates a presigned GET URL for the given key.
func PresignGetObject(ctx context.Context, presignClient *s3.PresignClient, bucket, key string, expiry time.Duration) (string, error) {
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign %s: %w", key, err)
	}
	return req.URL, nil
}
