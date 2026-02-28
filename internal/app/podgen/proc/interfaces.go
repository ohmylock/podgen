package proc

import (
	"context"

	"podgen/internal/app/podgen/podcast"
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

// EpisodeStore defines the interface for episode persistence operations.
type EpisodeStore interface {
	SaveEpisode(podcastID string, episode *podcast.Episode) error
	FindEpisodesByStatus(podcastID string, status podcast.Status) ([]*podcast.Episode, error)
	FindEpisodesBySession(podcastID, session string) ([]*podcast.Episode, error)
	FindEpisodesBySizeLimit(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error)
	GetEpisodeByFilename(podcastID, fileName string) (*podcast.Episode, error)
	GetLastEpisodeByNotStatus(podcastID string, status podcast.Status) (*podcast.Episode, error)
}

// ObjectStorage defines the interface for S3-compatible object storage operations.
type ObjectStorage interface {
	DeleteEpisode(ctx context.Context, objectName string) error
	UploadEpisode(ctx context.Context, objectName, filePath string) (*UploadResult, error)
	UploadImage(ctx context.Context, objectName, filePath string) (*UploadResult, error)
	UploadFeed(ctx context.Context, objectName, filePath string) (*UploadResult, error)
	GetObjectInfo(ctx context.Context, objectName string) (*ObjectInfo, error)
}

// FileScanner defines the interface for scanning podcast episode files.
type FileScanner interface {
	FindEpisodes(folderName string) ([]*podcast.Episode, error)
}
