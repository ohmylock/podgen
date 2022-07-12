package proc

import (
	"context"

	log "github.com/go-pkgz/lgr"
	"github.com/minio/minio-go/v7"
)

// S3Store store
type S3Store struct {
	Client   *minio.Client
	Location string
	Bucket   string
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

// UploadFeed to s3 storage
func (s *S3Store) UploadFeed(ctx context.Context, objectName, filePath string) (*minio.UploadInfo, error) {
	return s.uploadFile(ctx, objectName, filePath, "plain/text")
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
	return &uploadInfo, nil
}
