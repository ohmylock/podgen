package proc

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
)

// ProgressFunc is a callback for upload progress reporting.
type ProgressFunc func(uploaded, total int64)

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
		return fmt.Errorf("can't check exists bucket %s: %w", s.Bucket, errBucketExists)
	}
	if !exists {
		return nil
	}
	return s.Client.RemoveObject(ctx, s.Bucket, objectName, minio.RemoveObjectOptions{})
}

// UploadEpisode to s3 storage
func (s *S3Store) UploadEpisode(ctx context.Context, objectName, filePath string) (*UploadResult, error) {
	return s.uploadFile(ctx, objectName, filePath, nil)
}

// UploadEpisodeWithProgress uploads to s3 storage with progress callback
func (s *S3Store) UploadEpisodeWithProgress(ctx context.Context, objectName, filePath string, progress ProgressFunc) (*UploadResult, error) {
	return s.uploadFile(ctx, objectName, filePath, progress)
}

// UploadImage to s3 storage
func (s *S3Store) UploadImage(ctx context.Context, objectName, filePath string) (*UploadResult, error) {
	return s.uploadFile(ctx, objectName, filePath, nil)
}

// UploadFeed to s3 storage
func (s *S3Store) UploadFeed(ctx context.Context, objectName, filePath string) (*UploadResult, error) {
	return s.uploadFile(ctx, objectName, filePath, nil)
}

// detectContentType determines MIME type from file extension.
func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Custom mappings for common types
	customTypes := map[string]string{
		".mp3":  "audio/mpeg",
		".m4a":  "audio/mp4",
		".ogg":  "audio/ogg",
		".wav":  "audio/wav",
		".flac": "audio/flac",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".rss":  "application/rss+xml",
		".xml":  "application/xml",
		".json": "application/json",
		".html": "text/html",
		".txt":  "text/plain",
	}

	if ct, ok := customTypes[ext]; ok {
		return ct
	}

	// Fallback to mime package
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}

	return "application/octet-stream"
}

func (s *S3Store) uploadFile(ctx context.Context, objectName, filePath string, progress ProgressFunc) (*UploadResult, error) {
	exists, errBucketExists := s.Client.BucketExists(ctx, s.Bucket)
	if errBucketExists != nil {
		return nil, fmt.Errorf("can't check exists bucket %s: %w", s.Bucket, errBucketExists)
	}

	if !exists {
		if err := s.Client.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: s.Location}); err != nil {
			return nil, fmt.Errorf("can't create bucket %s: %w", s.Bucket, err)
		}
	}

	var uploadInfo minio.UploadInfo
	var err error

	contentType := detectContentType(filePath)
	opts := minio.PutObjectOptions{
		ContentType:        contentType,
		ContentDisposition: "inline",
	}

	if progress != nil {
		// Use PutObject with progress tracking
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("can't open file %s: %w", filePath, err)
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("can't stat file %s: %w", filePath, err)
		}

		reader := &progressReader{
			reader:   file,
			total:    stat.Size(),
			callback: progress,
		}

		uploadInfo, err = s.Client.PutObject(ctx, s.Bucket, objectName, reader, stat.Size(), opts)
		if err != nil {
			return nil, err
		}
	} else {
		// Use FPutObject for simple upload
		uploadInfo, err = s.Client.FPutObject(ctx, s.Bucket, objectName, filePath, opts)
		if err != nil {
			return nil, err
		}
	}

	location := uploadInfo.Location
	if location == "" {
		objectInfo, err := s.GetObjectInfo(ctx, objectName)
		if err != nil {
			return nil, fmt.Errorf("can't get file location %s in bucket %s: %w", objectName, s.Bucket, err)
		}
		location = objectInfo.Location
	}
	return &UploadResult{Location: location}, nil
}

// GetObjectInfo from object on s3 storage
func (s *S3Store) GetObjectInfo(ctx context.Context, objectName string) (*ObjectInfo, error) {
	statInfo, err := s.Client.StatObject(ctx, s.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	endpoint := s.Client.EndpointURL()

	objectInfo := ObjectInfo{
		Location: fmt.Sprintf("%s/%s/%s", strings.TrimRight(endpoint.String(), "/"), s.Bucket, statInfo.Key),
		Size:     statInfo.Size,
	}

	return &objectInfo, nil
}

// progressReader wraps an io.Reader to track upload progress.
type progressReader struct {
	reader   io.Reader
	total    int64
	uploaded int64
	callback ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.uploaded += int64(n)
		if pr.callback != nil {
			pr.callback(pr.uploaded, pr.total)
		}
	}
	return n, err
}
