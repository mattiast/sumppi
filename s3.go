package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Client struct {
	client *s3.Client
}

func NewS3Client(ctx context.Context) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &S3Client{
		client: s3.NewFromConfig(cfg),
	}, nil
}

func (s *S3Client) UploadRSSContent(ctx context.Context, rssContent, s3Path string) error {
	// Parse S3 path (s3://bucket/key)
	bucket, key, err := parseS3Path(s3Path)
	if err != nil {
		return fmt.Errorf("failed to parse S3 path: %w", err)
	}

	// Upload to S3 directly from memory
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(rssContent),
		ContentType: aws.String("application/rss+xml"),
		ACL:         types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func parseS3Path(s3Path string) (bucket, key string, err error) {
	if !strings.HasPrefix(s3Path, "s3://") {
		return "", "", fmt.Errorf("invalid S3 path: must start with s3://")
	}

	path := strings.TrimPrefix(s3Path, "s3://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid S3 path: must be in format s3://bucket/key")
	}

	return parts[0], parts[1], nil
}

func generateS3URL(s3Path string) (string, error) {
	bucket, key, err := parseS3Path(s3Path)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key), nil
}
