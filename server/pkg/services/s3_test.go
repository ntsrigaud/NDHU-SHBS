package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ── mock ─────────────────────────────────────────────────────────────────────

type mockS3API struct {
	putErr error
	delErr error
}

func (m *mockS3API) PutObject(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, m.putErr
}

func (m *mockS3API) DeleteObject(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, m.delErr
}

func newMockS3Service(putErr, delErr error) *S3Service {
	return &S3Service{
		client:     &mockS3API{putErr: putErr, delErr: delErr},
		bucket:     "test-bucket",
		cdnBaseURL: "https://cdn.example.com",
	}
}

// ── NewS3Service ──────────────────────────────────────────────────────────────

func TestNewS3Service_Success(t *testing.T) {
	// AWS SDK loads credentials lazily — NewS3Service should not fail even
	// when no real credentials are present.
	svc, err := NewS3Service("bucket", "us-east-1", "https://cdn.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

// ── UploadImage ───────────────────────────────────────────────────────────────

func TestUploadImage_JPEG(t *testing.T) {
	svc := newMockS3Service(nil, nil)
	key, url, err := svc.UploadImage(context.Background(), []byte("data"), "photo.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(key, "images/") || !strings.HasSuffix(key, ".jpg") {
		t.Errorf("unexpected key: %s", key)
	}
	if !strings.HasPrefix(url, "https://cdn.example.com/images/") {
		t.Errorf("unexpected url: %s", url)
	}
}

func TestUploadImage_JpegExtension(t *testing.T) {
	svc := newMockS3Service(nil, nil)
	_, _, err := svc.UploadImage(context.Background(), []byte("data"), "photo.jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadImage_PNG(t *testing.T) {
	svc := newMockS3Service(nil, nil)
	_, _, err := svc.UploadImage(context.Background(), []byte("data"), "photo.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadImage_WebP(t *testing.T) {
	svc := newMockS3Service(nil, nil)
	_, _, err := svc.UploadImage(context.Background(), []byte("data"), "photo.webp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadImage_UnsupportedExtension(t *testing.T) {
	svc := newMockS3Service(nil, nil)
	_, _, err := svc.UploadImage(context.Background(), []byte("data"), "photo.gif")
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
	if !strings.Contains(err.Error(), "unsupported file type") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUploadImage_S3Error(t *testing.T) {
	svc := newMockS3Service(errors.New("internal s3 error"), nil)
	_, _, err := svc.UploadImage(context.Background(), []byte("data"), "photo.jpg")
	if err == nil {
		t.Fatal("expected error from S3")
	}
	if !strings.Contains(err.Error(), "s3 put object") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ── DeleteImage ───────────────────────────────────────────────────────────────

func TestDeleteImage_Success(t *testing.T) {
	svc := newMockS3Service(nil, nil)
	if err := svc.DeleteImage(context.Background(), "images/test.jpg"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteImage_S3Error(t *testing.T) {
	svc := newMockS3Service(nil, errors.New("delete failed"))
	err := svc.DeleteImage(context.Background(), "images/test.jpg")
	if err == nil {
		t.Fatal("expected error from S3")
	}
	if !strings.Contains(err.Error(), "s3 delete object") {
		t.Errorf("unexpected error message: %v", err)
	}
}
