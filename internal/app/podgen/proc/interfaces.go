package proc

import (
	"context"

	"podgen/internal/app/podgen/podcast"
	"podgen/internal/storage"
)

// UploadResult holds the result of an S3 upload operation.
type UploadResult struct {
	Location string
}

// ObjectInfo holds metadata about an S3 object.
type ObjectInfo struct {
	Location string
	Size     int64
}

// EpisodeStore is an alias for storage.EpisodeStore for backward compatibility.
// New code should use storage.EpisodeStore directly.
type EpisodeStore = storage.EpisodeStore

// ObjectStorage defines the interface for S3-compatible object storage operations.
type ObjectStorage interface {
	DeleteEpisode(ctx context.Context, objectName string) error
	UploadEpisode(ctx context.Context, objectName, filePath string) (*UploadResult, error)
	UploadEpisodeWithProgress(ctx context.Context, objectName, filePath string, progress ProgressFunc) (*UploadResult, error)
	UploadImage(ctx context.Context, objectName, filePath string) (*UploadResult, error)
	UploadFeed(ctx context.Context, objectName, filePath string) (*UploadResult, error)
	GetObjectInfo(ctx context.Context, objectName string) (*ObjectInfo, error)
}

// FileScanner defines the interface for scanning podcast episode files.
type FileScanner interface {
	FindEpisodes(folderName string) ([]*podcast.Episode, error)
}

// ProgressReporter defines the interface for tracking upload/delete progress.
type ProgressReporter interface {
	StartFile(workerID int, filename string, totalSize int64)
	UpdateProgress(workerID int, uploaded, total int64)
	CompleteFile(workerID int, fileSize int64, err error)
	Finish()
	Reset(totalTasks int)
}
