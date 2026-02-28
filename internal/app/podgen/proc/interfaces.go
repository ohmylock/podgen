package proc

import (
	"context"

	"github.com/boltdb/bolt"
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
	SaveEpisode(tx *bolt.Tx, podcastID string, episode *podcast.Episode) error
	FindEpisodesByStatus(tx *bolt.Tx, podcastID string, status podcast.Status) ([]*podcast.Episode, error)
	FindEpisodesBySession(tx *bolt.Tx, podcastID, session string) ([]*podcast.Episode, error)
	FindEpisodesBySizeLimit(tx *bolt.Tx, podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error)
	GetEpisodeByFilename(tx *bolt.Tx, podcastID, fileName string) (*podcast.Episode, error)
	GetLastEpisodeByNotStatus(tx *bolt.Tx, podcastID string, status podcast.Status) (*podcast.Episode, error)
	CreateTransaction() (*bolt.Tx, error)
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
