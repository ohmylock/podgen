package proc

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
)

// S3Store store
type S3Store struct {
	Client   *minio.Client
	Location string
	Bucket   string
}

// ObjectInfo struct of object in s3 storage
type ObjectInfo struct {
	minio.ObjectInfo
	Location string
}

// DeleteEpisode from s3 storage
func (s *S3Store) DeleteEpisode(ctx context.Context, objectName string) error {
	exists, errBucketExists := s.Client.BucketExists(ctx, s.Bucket)
	if errBucketExists != nil {
		return fmt.Errorf("can't check exists bucket %s: %w", s.Bucket, errBucketExists)
	}
	if !exists {
		return nil
	}
	return s.Client.RemoveObject(ctx, s.Bucket, objectName, minio.RemoveObjectOptions{})
}

// UploadEpisode to s3 storage
func (s *S3Store) UploadEpisode(ctx context.Context, objectName, filePath string) (*minio.UploadInfo, error) {
	return s.uploadFile(ctx, objectName, filePath, "audio/mp3")
}

// UploadImage to s3 storage
func (s *S3Store) UploadImage(ctx context.Context, objectName, filePath string) (*minio.UploadInfo, error) {
	return s.uploadFile(ctx, objectName, filePath, "image/png")
}

// UploadFeed to s3 storage
func (s *S3Store) UploadFeed(ctx context.Context, objectName, filePath string) (*minio.UploadInfo, error) {
	return s.uploadFile(ctx, objectName, filePath, "application/rss+xml")
}

func (s *S3Store) uploadFile(ctx context.Context, objectName, filePath, contentType string) (*minio.UploadInfo, error) {
	exists, errBucketExists := s.Client.BucketExists(ctx, s.Bucket)
	if errBucketExists != nil {
		return nil, fmt.Errorf("can't check exists bucket %s: %w", s.Bucket, errBucketExists)
	}

	if !exists {
		if err := s.Client.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: s.Location}); err != nil {
			return nil, fmt.Errorf("can't create bucket %s: %w", s.Bucket, err)
		}
	}
	uploadInfo, err := s.Client.FPutObject(ctx, s.Bucket, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, err
	}

	if uploadInfo.Location == "" {
		objectInfo, err := s.GetObjectInfo(ctx, objectName)
		if err != nil {
			return nil, fmt.Errorf("can't get file location %s in bucket %s: %w", objectName, s.Bucket, err)
		}
		uploadInfo.Location = objectInfo.Location
	}
	return &uploadInfo, nil
}

// GetObjectInfo from object on s3 storage
func (s *S3Store) GetObjectInfo(ctx context.Context, objectName string) (*ObjectInfo, error) {
	statInfo, err := s.Client.StatObject(ctx, s.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	endpoint := s.Client.EndpointURL()

	objectInfo := ObjectInfo{
		ObjectInfo: statInfo,
		Location:   fmt.Sprintf("%s/%s/%s", strings.TrimRight(endpoint.String(), "/"), s.Bucket, statInfo.Key),
	}

	return &objectInfo, nil
}
