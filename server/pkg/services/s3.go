package services

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// s3API is the AWS S3 operations subset the S3Service uses.
// Defining it as an interface allows test doubles to replace the real client.
type s3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// S3Service wraps the AWS S3 client with application-level helpers.
type S3Service struct {
	client     s3API
	bucket     string
	cdnBaseURL string
}

// NewS3Service initialises an S3Service using the default AWS credential chain
// (env vars AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY, then instance profile,
// etc.).
func NewS3Service(bucket, region, cdnBaseURL string) (*S3Service, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &S3Service{
		client:     s3.NewFromConfig(cfg),
		bucket:     bucket,
		cdnBaseURL: strings.TrimRight(cdnBaseURL, "/"),
	}, nil
}

// UploadImage uploads raw image bytes to S3 under the "images/" prefix and
// returns the S3 key and the public CloudFront CDN URL.
//
// Allowed extensions: .jpg / .jpeg, .png, .webp (max enforced by the caller).
func (s *S3Service) UploadImage(ctx context.Context, data []byte, originalName string) (s3Key string, cdnURL string, err error) {
	ext := strings.ToLower(filepath.Ext(originalName))

	var contentType string
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	default:
		return "", "", fmt.Errorf("unsupported file type: %s", ext)
	}

	key := fmt.Sprintf("images/%s%s", uuid.New().String(), ext)

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", "", fmt.Errorf("s3 put object: %w", err)
	}

	cdnURL = fmt.Sprintf("%s/%s", s.cdnBaseURL, key)
	return key, cdnURL, nil
}

// DeleteImage removes an object from S3 by its key.
func (s *S3Service) DeleteImage(ctx context.Context, s3Key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete object: %w", err)
	}
	return nil
}
