package proc

import (
	"context"
	"fmt"
	"strings"

	log "github.com/go-pkgz/lgr"
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
		log.Fatalf("[ERROR] can't check exists bucket %s, %v", s.Bucket, errBucketExists)
	}
	if !exists {
		return nil
	}
	err := s.Client.RemoveObject(ctx, s.Bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}
	return nil
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
		log.Fatalf("[ERROR] can't check exists bucket %s, %v", s.Bucket, errBucketExists)
	}

	if !exists {
		err := s.Client.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: s.Location})
		if err != nil {
			log.Fatalf("[ERROR] can't create bucket %s, %v", s.Bucket, errBucketExists)
		}
	}
	uploadInfo, err := s.Client.FPutObject(ctx, s.Bucket, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, err
	}

	if uploadInfo.Location == "" {
		objectInfo, err := s.GetObjectInfo(ctx, objectName)
		if err != nil {
			log.Fatalf("[ERROR] can't get file location %s in bucket %s, %v", objectName, s.Bucket, err)
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
