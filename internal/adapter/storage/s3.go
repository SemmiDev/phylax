package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/semmidev/phylax/internal/config"
)

type S3Storage struct {
	client   *s3.Client
	uploader *s3manager.Uploader
	bucket   string
	prefix   string
}

// NewS3 creates a new S3Storage instance using AWS SDK v2
func NewS3(cfg *appconfig.UploadTarget) (*S3Storage, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	uploader := s3manager.NewUploader(client)

	return &S3Storage{
		client:   client,
		uploader: uploader,
		bucket:   cfg.Bucket,
		prefix:   cfg.Prefix,
	}, nil
}

// Upload uploads a local file to S3
func (s *S3Storage) Upload(ctx context.Context, localPath string, remoteName string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	key := filepath.Join(s.prefix, remoteName)

	_, err = s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// List returns all files in the bucket with the given prefix
func (s *S3Storage) List(ctx context.Context) ([]string, error) {
	resp, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &s.bucket,
		Prefix: &s.prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	var files []string
	for _, obj := range resp.Contents {
		name := strings.TrimPrefix(*obj.Key, s.prefix)
		if name != "" {
			files = append(files, name)
		}
	}

	return files, nil
}

// Delete removes a file from S3
func (s *S3Storage) Delete(ctx context.Context, remoteName string) error {
	key := filepath.Join(s.prefix, remoteName)

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// GetOldFiles returns files older than a given time
func (s *S3Storage) GetOldFiles(ctx context.Context, cutoffTime time.Time) ([]string, error) {
	resp, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &s.bucket,
		Prefix: &s.prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	var oldFiles []string
	for _, obj := range resp.Contents {
		if obj.LastModified.Before(cutoffTime) {
			name := strings.TrimPrefix(*obj.Key, s.prefix)
			if name != "" {
				oldFiles = append(oldFiles, name)
			}
		}
	}

	return oldFiles, nil
}
