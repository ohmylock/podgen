// Package factory provides a factory function for creating storage backends.
package factory

import (
	"fmt"
	"strings"

	"podgen/internal/storage"
	"podgen/internal/storage/bolt"
	"podgen/internal/storage/sqlite"
)

// New creates a new Store instance based on the provided configuration.
// It supports sqlite, bolt, and postgres (not yet implemented) backends.
// The store is not opened - call Open() before use.
func New(cfg storage.Config) (storage.Store, error) {
	switch cfg.Type {
	case storage.TypeSQLite:
		return sqlite.New(cfg), nil
	case storage.TypeBolt:
		return bolt.New(cfg), nil
	case storage.TypePostgres:
		return nil, fmt.Errorf("postgres backend: %w", ErrNotImplemented)
	default:
		return nil, fmt.Errorf("unknown storage type %q: %w", cfg.Type, storage.ErrInvalidConfig)
	}
}

// NewFromStrings creates a new Store from string type and DSN.
// This is a convenience function for use with configuration systems.
// Returns an error if the storage type is unknown.
func NewFromStrings(storageType, dsn string) (storage.Store, error) {
	st, err := ParseStorageType(storageType)
	if err != nil {
		return nil, err
	}
	cfg := storage.Config{
		Type:         st,
		DSN:          dsn,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}
	return New(cfg)
}

// ParseStorageType converts a string to StorageType.
// Returns an error for unknown storage types to prevent silent failures.
func ParseStorageType(s string) (storage.StorageType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sqlite", "sqlite3", "":
		return storage.TypeSQLite, nil
	case "bolt", "boltdb":
		return storage.TypeBolt, nil
	case "postgres", "postgresql", "pg":
		return storage.TypePostgres, nil
	default:
		return "", fmt.Errorf("unknown storage type %q: valid types are sqlite, bolt, postgres", s)
	}
}

// ErrNotImplemented is returned when a storage backend is not yet implemented.
var ErrNotImplemented = fmt.Errorf("not implemented")
