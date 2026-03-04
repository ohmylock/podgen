// Package storage provides a universal storage API with support for multiple database backends.
package storage

import (
	"errors"

	"podgen/internal/app/podgen/podcast"
)

// StorageType represents the type of storage backend.
type StorageType string

const (
	// TypeSQLite represents SQLite database backend.
	TypeSQLite StorageType = "sqlite"
	// TypeBolt represents BoltDB database backend.
	TypeBolt StorageType = "bolt"
	// TypePostgres represents PostgreSQL database backend.
	TypePostgres StorageType = "postgres"
)

// Config holds the storage configuration.
type Config struct {
	// Type specifies the storage backend type (sqlite, bolt, postgres).
	Type StorageType
	// DSN is the data source name or file path for the database.
	DSN string
	// MaxOpenConns sets the maximum number of open connections (for SQL backends).
	MaxOpenConns int
	// MaxIdleConns sets the maximum number of idle connections (for SQL backends).
	MaxIdleConns int
}

// DefaultConfig returns a Config with sensible defaults (SQLite with WAL mode).
func DefaultConfig(storagePath string) Config {
	return Config{
		Type:         TypeSQLite,
		DSN:          storagePath,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}
}

// Common errors for storage operations.
var (
	ErrNoBucket      = errors.New("no bucket/table found")
	ErrNotFound      = errors.New("episode not found")
	ErrInvalidConfig = errors.New("invalid storage configuration")
	ErrClosed        = errors.New("storage is closed")
)

// EpisodeStore defines the interface for episode persistence operations.
// This is the core interface that all storage backends must implement.
type EpisodeStore interface {
	// SaveEpisode persists an episode to the store.
	SaveEpisode(podcastID string, episode *podcast.Episode) error

	// FindEpisodesByStatus retrieves all episodes with the given status.
	FindEpisodesByStatus(podcastID string, status podcast.Status) ([]*podcast.Episode, error)

	// FindEpisodesBySession retrieves all episodes for a given session.
	FindEpisodesBySession(podcastID, session string) ([]*podcast.Episode, error)

	// FindEpisodesBySizeLimit retrieves episodes up to a total size limit.
	FindEpisodesBySizeLimit(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error)

	// GetEpisodeByFilename retrieves an episode by its filename.
	GetEpisodeByFilename(podcastID, fileName string) (*podcast.Episode, error)

	// GetLastEpisodeByNotStatus retrieves the last episode that doesn't have the given status.
	GetLastEpisodeByNotStatus(podcastID string, status podcast.Status) (*podcast.Episode, error)
}

// Store is the main storage interface that wraps EpisodeStore with lifecycle methods.
type Store interface {
	EpisodeStore

	// Open initializes the storage connection.
	Open() error

	// Close releases all storage resources.
	Close() error

	// ListPodcasts returns all podcast IDs in the store.
	ListPodcasts() ([]string, error)

	// ListEpisodes returns all episodes for a podcast.
	ListEpisodes(podcastID string) ([]*podcast.Episode, error)
}

// EpisodeIterator is a function type for iterating over episodes during migration.
type EpisodeIterator func(podcastID string, episode *podcast.Episode) error
