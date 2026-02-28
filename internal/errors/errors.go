package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common storage conditions.
var (
	ErrNoBucket       = errors.New("no bucket")
	ErrEpisodeNotFound = errors.New("episode not found")
)

// EpisodeError provides structured context for episode-related failures.
type EpisodeError struct {
	PodcastID string
	Filename  string
	Op        string
	Err       error
}

// Error implements the error interface.
func (e *EpisodeError) Error() string {
	if e.Filename != "" {
		return fmt.Sprintf("op=%s podcast=%s file=%s: %v", e.Op, e.PodcastID, e.Filename, e.Err)
	}
	return fmt.Sprintf("op=%s podcast=%s: %v", e.Op, e.PodcastID, e.Err)
}

// Unwrap enables errors.Is and errors.As to inspect the underlying error.
func (e *EpisodeError) Unwrap() error {
	return e.Err
}
