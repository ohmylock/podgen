// Package sqlite provides a SQLite-based implementation of the storage.Store interface.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/go-pkgz/lgr"
	_ "modernc.org/sqlite"

	"podgen/internal/app/podgen/podcast"
	apperrors "podgen/internal/errors"
	"podgen/internal/storage"
)

// Store implements storage.Store using SQLite with WAL mode.
type Store struct {
	db     *sql.DB
	dsn    string
	config storage.Config
}

// New creates a new SQLite store with the given configuration.
// The store must be opened with Open() before use.
func New(cfg storage.Config) *Store {
	return &Store{
		dsn:    cfg.DSN,
		config: cfg,
	}
}

// Open initializes the SQLite database connection with WAL mode.
func (s *Store) Open() error {
	if s.dsn == "" {
		return fmt.Errorf("empty db path: %w", storage.ErrInvalidConfig)
	}

	// Create parent directories if they don't exist
	if dir := filepath.Dir(s.dsn); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc", s.dsn)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Configure connection pool
	maxOpen := s.config.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := s.config.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)

	// Verify connection
	if err = db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// Enable WAL mode and other pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
	}
	for _, pragma := range pragmas {
		if _, err = db.Exec(pragma); err != nil {
			db.Close()
			return fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	s.db = db

	// Create schema
	if err = s.createSchema(); err != nil {
		s.db.Close()
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Printf("[INFO] SQLite store opened: %s (WAL mode enabled)", s.dsn)
	return nil
}

// createSchema creates the necessary tables and indexes.
func (s *Store) createSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS episodes (
			podcast_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			pub_date TEXT,
			size INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			location TEXT,
			session TEXT,
			title TEXT,
			artist TEXT,
			album TEXT,
			year TEXT,
			comment TEXT,
			duration TEXT,
			PRIMARY KEY (podcast_id, filename)
		);

		CREATE INDEX IF NOT EXISTS idx_episodes_status ON episodes(podcast_id, status);
		CREATE INDEX IF NOT EXISTS idx_episodes_session ON episodes(podcast_id, session);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close releases all database resources.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	log.Printf("[INFO] SQLite store closing: %s", s.dsn)
	err := s.db.Close()
	s.db = nil
	return err
}

// SaveEpisode persists an episode to the store.
func (s *Store) SaveEpisode(podcastID string, episode *podcast.Episode) error {
	if s.db == nil {
		return storage.ErrClosed
	}

	query := `
		INSERT INTO episodes (podcast_id, filename, pub_date, size, status, location, session, title, artist, album, year, comment, duration)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(podcast_id, filename) DO UPDATE SET
			pub_date = excluded.pub_date,
			size = excluded.size,
			status = excluded.status,
			location = excluded.location,
			session = excluded.session,
			title = excluded.title,
			artist = excluded.artist,
			album = excluded.album,
			year = excluded.year,
			comment = excluded.comment,
			duration = excluded.duration
	`

	_, err := s.db.Exec(query,
		podcastID,
		episode.Filename,
		episode.PubDate,
		episode.Size,
		episode.Status,
		episode.Location,
		episode.Session,
		episode.Title,
		episode.Artist,
		episode.Album,
		episode.Year,
		episode.Comment,
		episode.Duration,
	)
	if err != nil {
		return fmt.Errorf("failed to save episode: %w", err)
	}

	log.Printf("[INFO] save episode %s - %s", podcastID, episode.Filename)
	return nil
}

// FindEpisodesByStatus retrieves all episodes with the given status.
func (s *Store) FindEpisodesByStatus(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	// Check if podcast exists
	exists, err := s.podcastExists(podcastID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &apperrors.EpisodeError{PodcastID: podcastID, Op: "FindEpisodesByStatus", Err: apperrors.ErrNoBucket}
	}

	query := `
		SELECT filename, pub_date, size, status, location, session, title, artist, album, year, comment, duration
		FROM episodes
		WHERE podcast_id = ? AND status = ?
		ORDER BY filename
	`

	rows, err := s.db.Query(query, podcastID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes: %w", err)
	}
	defer rows.Close()

	return s.scanEpisodes(rows)
}

// FindEpisodesBySession retrieves all episodes for a given session.
func (s *Store) FindEpisodesBySession(podcastID, session string) ([]*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	// Check if podcast exists
	exists, err := s.podcastExists(podcastID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &apperrors.EpisodeError{PodcastID: podcastID, Op: "FindEpisodesBySession", Err: apperrors.ErrNoBucket}
	}

	query := `
		SELECT filename, pub_date, size, status, location, session, title, artist, album, year, comment, duration
		FROM episodes
		WHERE podcast_id = ? AND session = ?
		ORDER BY filename
	`

	rows, err := s.db.Query(query, podcastID, session)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes: %w", err)
	}
	defer rows.Close()

	return s.scanEpisodes(rows)
}

// FindEpisodesBySizeLimit retrieves episodes up to a total size limit.
func (s *Store) FindEpisodesBySizeLimit(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	// First get episodes by status
	episodes, err := s.FindEpisodesByStatus(podcastID, status)
	if err != nil {
		// Only swallow ErrNoBucket, propagate other errors
		var epErr *apperrors.EpisodeError
		if errors.As(err, &epErr) && errors.Is(epErr.Err, apperrors.ErrNoBucket) {
			return nil, nil
		}
		return nil, err
	}

	if sizeLimit <= 0 {
		return episodes, nil
	}

	// Apply size limit
	var result []*podcast.Episode
	var totalSize int64
	for _, ep := range episodes {
		if totalSize >= sizeLimit || (totalSize+ep.Size) >= sizeLimit {
			break
		}
		totalSize += ep.Size
		result = append(result, ep)
	}

	return result, nil
}

// GetEpisodeByFilename retrieves an episode by its filename.
func (s *Store) GetEpisodeByFilename(podcastID, fileName string) (*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	// Check if podcast exists
	exists, err := s.podcastExists(podcastID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &apperrors.EpisodeError{PodcastID: podcastID, Op: "GetEpisodeByFilename", Err: apperrors.ErrNoBucket}
	}

	query := `
		SELECT filename, pub_date, size, status, location, session, title, artist, album, year, comment, duration
		FROM episodes
		WHERE podcast_id = ? AND filename = ?
	`

	row := s.db.QueryRow(query, podcastID, fileName)
	episode, err := s.scanEpisode(row)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if episode.Filename == "" {
		return nil, storage.ErrNotFound
	}

	return episode, nil
}

// GetLastEpisodeByNotStatus retrieves the last episode that doesn't have the given status.
func (s *Store) GetLastEpisodeByNotStatus(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	// Check if podcast exists
	exists, err := s.podcastExists(podcastID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &apperrors.EpisodeError{PodcastID: podcastID, Op: "GetLastEpisodeByNotStatus", Err: apperrors.ErrNoBucket}
	}

	query := `
		SELECT filename, pub_date, size, status, location, session, title, artist, album, year, comment, duration
		FROM episodes
		WHERE podcast_id = ? AND status != ?
		ORDER BY filename DESC
		LIMIT 1
	`

	row := s.db.QueryRow(query, podcastID, status)
	episode, err := s.scanEpisode(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return episode, nil
}

// ListPodcasts returns all podcast IDs in the store.
func (s *Store) ListPodcasts() ([]string, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	query := `SELECT DISTINCT podcast_id FROM episodes ORDER BY podcast_id`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list podcasts: %w", err)
	}
	defer rows.Close()

	var podcasts []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan podcast id: %w", err)
		}
		podcasts = append(podcasts, id)
	}

	return podcasts, rows.Err()
}

// ListEpisodes returns all episodes for a podcast.
func (s *Store) ListEpisodes(podcastID string) ([]*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	query := `
		SELECT filename, pub_date, size, status, location, session, title, artist, album, year, comment, duration
		FROM episodes
		WHERE podcast_id = ?
		ORDER BY filename
	`

	rows, err := s.db.Query(query, podcastID)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes: %w", err)
	}
	defer rows.Close()

	return s.scanEpisodes(rows)
}

// podcastExists checks if any episodes exist for the given podcast.
func (s *Store) podcastExists(podcastID string) (bool, error) {
	var exists int
	err := s.db.QueryRow("SELECT 1 FROM episodes WHERE podcast_id = ? LIMIT 1", podcastID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// scanEpisode scans a single episode from a row.
func (s *Store) scanEpisode(row *sql.Row) (*podcast.Episode, error) {
	ep := &podcast.Episode{}
	err := row.Scan(
		&ep.Filename,
		&ep.PubDate,
		&ep.Size,
		&ep.Status,
		&ep.Location,
		&ep.Session,
		&ep.Title,
		&ep.Artist,
		&ep.Album,
		&ep.Year,
		&ep.Comment,
		&ep.Duration,
	)
	if err != nil {
		return nil, err
	}
	return ep, nil
}

// scanEpisodes scans multiple episodes from rows.
func (s *Store) scanEpisodes(rows *sql.Rows) ([]*podcast.Episode, error) {
	var episodes []*podcast.Episode
	for rows.Next() {
		ep := &podcast.Episode{}
		err := rows.Scan(
			&ep.Filename,
			&ep.PubDate,
			&ep.Size,
			&ep.Status,
			&ep.Location,
			&ep.Session,
			&ep.Title,
			&ep.Artist,
			&ep.Album,
			&ep.Year,
			&ep.Comment,
			&ep.Duration,
		)
		if err != nil {
			log.Printf("[WARN] failed to scan episode: %v", err)
			continue
		}
		episodes = append(episodes, ep)
	}
	return episodes, rows.Err()
}

// Verify Store implements storage.Store interface.
var _ storage.Store = (*Store)(nil)
