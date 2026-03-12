// Package bolt provides a BoltDB-based implementation of the storage.Store interface.
package bolt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
	log "github.com/go-pkgz/lgr"

	"podgen/internal/app/podgen/podcast"
	apperrors "podgen/internal/errors"
	"podgen/internal/storage"
)

// Store implements storage.Store using BoltDB.
type Store struct {
	db     *bolt.DB
	dsn    string
	config storage.Config
	mu     sync.Mutex
}

// New creates a new BoltDB store with the given configuration.
// The store must be opened with Open() before use.
func New(cfg storage.Config) *Store {
	return &Store{
		dsn:    cfg.DSN,
		config: cfg,
	}
}

// Open initializes the BoltDB database connection.
func (s *Store) Open() error {
	if s.dsn == "" {
		return fmt.Errorf("empty db path: %w", storage.ErrInvalidConfig)
	}

	if err := os.MkdirAll(filepath.Dir(s.dsn), 0o700); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := bolt.Open(s.dsn, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return fmt.Errorf("failed to open bolt database: %w", err)
	}

	s.db = db
	log.Printf("[INFO] BoltDB store opened: %s", s.dsn)
	return nil
}

// Close releases all database resources.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	log.Printf("[INFO] BoltDB store closing: %s", s.dsn)
	err := s.db.Close()
	s.db = nil
	return err
}

// WithWriteTx executes fn within a serialized write transaction.
func (s *Store) WithWriteTx(fn func(*bolt.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Update(fn)
}

// WithReadTx executes fn within a read-only transaction.
func (s *Store) WithReadTx(fn func(*bolt.Tx) error) error {
	return s.db.View(fn)
}

// SaveEpisode persists an episode to the store.
func (s *Store) SaveEpisode(podcastID string, episode *podcast.Episode) error {
	if s.db == nil {
		return storage.ErrClosed
	}

	return s.WithWriteTx(func(tx *bolt.Tx) error {
		return s.saveEpisode(tx, podcastID, episode)
	})
}

func (s *Store) saveEpisode(tx *bolt.Tx, podcastID string, episode *podcast.Episode) error {
	key, err := s.getEpisodeKey(episode)
	if err != nil {
		return err
	}

	bucket, err := tx.CreateBucketIfNotExists([]byte(podcastID))
	if err != nil {
		return err
	}

	jdata, err := json.Marshal(episode)
	if err != nil {
		return err
	}

	return bucket.Put(key, jdata)
}

// FindEpisodesByStatus retrieves all episodes with the given status.
func (s *Store) FindEpisodesByStatus(podcastID string, filterStatus podcast.Status) ([]*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	var result []*podcast.Episode
	err := s.WithReadTx(func(tx *bolt.Tx) error {
		var err error
		result, err = s.findEpisodesByStatus(tx, podcastID, filterStatus)
		return err
	})
	return result, err
}

func (s *Store) findEpisodesByStatus(tx *bolt.Tx, podcastID string, filterStatus podcast.Status) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	bucket := tx.Bucket([]byte(podcastID))
	if bucket == nil {
		// Return empty slice for new podcasts (no bucket yet), matching SQLite behavior
		return []*podcast.Episode{}, nil
	}

	c := bucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		item := podcast.Episode{}
		if err := json.Unmarshal(v, &item); err != nil {
			log.Printf("[WARN] failed to unmarshal, %v", err)
			continue
		}
		if item.Status != filterStatus {
			continue
		}
		result = append(result, &item)
	}

	return result, nil
}

// FindEpisodesBySession retrieves all episodes for a given session.
func (s *Store) FindEpisodesBySession(podcastID, session string) ([]*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	var result []*podcast.Episode
	err := s.WithReadTx(func(tx *bolt.Tx) error {
		var err error
		result, err = s.findEpisodesBySession(tx, podcastID, session)
		return err
	})
	return result, err
}

func (s *Store) findEpisodesBySession(tx *bolt.Tx, podcastID, session string) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	bucket := tx.Bucket([]byte(podcastID))
	if bucket == nil {
		// Return empty slice for new podcasts (no bucket yet), matching SQLite behavior
		return []*podcast.Episode{}, nil
	}

	c := bucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		item := podcast.Episode{}
		if err := json.Unmarshal(v, &item); err != nil {
			log.Printf("[WARN] failed to unmarshal, %v", err)
			continue
		}
		if item.Session != session {
			continue
		}
		result = append(result, &item)
	}

	return result, nil
}

// ChangeStatusEpisodes changes the status of all episodes matching fromStatus to toStatus.
func (s *Store) ChangeStatusEpisodes(podcastID string, fromStatus, toStatus podcast.Status) error {
	if s.db == nil {
		return storage.ErrClosed
	}

	return s.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			return &apperrors.EpisodeError{PodcastID: podcastID, Op: "ChangeStatusEpisodes", Err: apperrors.ErrNoBucket}
		}

		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			item := podcast.Episode{}
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal, %v", err)
				continue
			}
			if item.Status != fromStatus {
				continue
			}

			item.Status = toStatus
			jdata, jerr := json.Marshal(&item)
			if jerr != nil {
				return jerr
			}

			if err := bucket.Put(k, jdata); err != nil {
				return err
			}
		}
		return nil
	})
}

// FindEpisodesBySizeLimit retrieves episodes up to a total size limit.
// The limit is applied to the total podcast size (uploaded + new episodes).
func (s *Store) FindEpisodesBySizeLimit(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	var result []*podcast.Episode
	err := s.WithReadTx(func(tx *bolt.Tx) error {
		episodes, err := s.findEpisodesByStatus(tx, podcastID, status)
		if err != nil {
			log.Printf("[INFO] No episodes with status %d in podcast %s: %v", status, podcastID, err)
			return nil
		}

		// Get total size of already uploaded episodes
		uploadedSize := s.getUploadedSizeInTx(tx, podcastID)

		sizes := uploadedSize
		result = make([]*podcast.Episode, len(episodes))
		for i, episode := range episodes {
			if sizeLimit > 0 && (sizes >= sizeLimit || (sizes+episode.Size) > sizeLimit) {
				result = result[:i]
				return nil
			}
			sizes += episode.Size
			result[i] = episode
		}
		return nil
	})
	return result, err
}

// getUploadedSizeInTx returns the total size of uploaded episodes for a podcast.
func (s *Store) getUploadedSizeInTx(tx *bolt.Tx, podcastID string) int64 {
	episodes, err := s.findEpisodesByStatus(tx, podcastID, podcast.Uploaded)
	if err != nil {
		return 0
	}
	var totalSize int64
	for _, ep := range episodes {
		totalSize += ep.Size
	}
	return totalSize
}

// GetEpisodeByFilename retrieves an episode by its filename.
func (s *Store) GetEpisodeByFilename(podcastID, fileName string) (*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	var episode *podcast.Episode
	err := s.WithReadTx(func(tx *bolt.Tx) error {
		var err error
		episode, err = s.getEpisodeByFilenameInTx(tx, podcastID, fileName)
		return err
	})
	return episode, err
}

func (s *Store) getEpisodeByFilenameInTx(tx *bolt.Tx, podcastID, fileName string) (*podcast.Episode, error) {
	key, err := s.getEpisodeKeyByFilename(fileName)
	if err != nil {
		return nil, err
	}

	episode := &podcast.Episode{}
	bucket := tx.Bucket([]byte(podcastID))
	if bucket == nil {
		// No bucket means no episodes exist yet for this podcast - return ErrNotFound
		// to match SQLite behavior where missing rows return ErrNotFound
		return nil, storage.ErrNotFound
	}

	item := bucket.Get(key)
	if item == nil {
		return nil, storage.ErrNotFound
	}

	if err = json.Unmarshal(item, episode); err != nil {
		log.Printf("[WARN] failed to unmarshal, %v", err)
		return nil, err
	}

	if episode.Filename == "" {
		return nil, storage.ErrNotFound
	}

	return episode, nil
}

// GetLastEpisodeByStatus retrieves the last episode with the given status.
func (s *Store) GetLastEpisodeByStatus(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	var result *podcast.Episode
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			return &apperrors.EpisodeError{PodcastID: podcastID, Op: "GetLastEpisodeByStatus", Err: apperrors.ErrNoBucket}
		}

		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			item := podcast.Episode{}
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal, %v", err)
				continue
			}

			if item.Status != status {
				continue
			}

			result = &item
			break
		}
		return nil
	})

	return result, err
}

// GetLastEpisodeByNotStatus retrieves the last episode that doesn't have the given status.
func (s *Store) GetLastEpisodeByNotStatus(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	var result *podcast.Episode
	err := s.WithReadTx(func(tx *bolt.Tx) error {
		var err error
		result, err = s.getLastEpisodeByNotStatusInTx(tx, podcastID, status)
		return err
	})
	return result, err
}

func (s *Store) getLastEpisodeByNotStatusInTx(tx *bolt.Tx, podcastID string, status podcast.Status) (*podcast.Episode, error) {
	bucket := tx.Bucket([]byte(podcastID))
	if bucket == nil {
		return nil, &apperrors.EpisodeError{PodcastID: podcastID, Op: "GetLastEpisodeByNotStatus", Err: apperrors.ErrNoBucket}
	}

	c := bucket.Cursor()
	for k, v := c.Last(); k != nil; k, v = c.Prev() {
		item := podcast.Episode{}
		if err := json.Unmarshal(v, &item); err != nil {
			log.Printf("[WARN] failed to unmarshal, %v", err)
			continue
		}

		if item.Status == status {
			continue
		}

		return &item, nil
	}

	return nil, nil
}

// ListPodcasts returns all podcast IDs in the store.
func (s *Store) ListPodcasts() ([]string, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	var podcasts []string
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			podcasts = append(podcasts, string(name))
			return nil
		})
	})
	return podcasts, err
}

// ListEpisodes returns all episodes for a podcast.
func (s *Store) ListEpisodes(podcastID string) ([]*podcast.Episode, error) {
	if s.db == nil {
		return nil, storage.ErrClosed
	}

	var episodes []*podcast.Episode
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			item := podcast.Episode{}
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal, %v", err)
				continue
			}
			episodes = append(episodes, &item)
		}
		return nil
	})
	return episodes, err
}

// DB returns the underlying BoltDB instance for advanced operations.
// This is provided for backward compatibility and migration purposes.
func (s *Store) DB() *bolt.DB {
	return s.db
}

func (s *Store) getEpisodeKey(episode *podcast.Episode) ([]byte, error) {
	return s.getEpisodeKeyByFilename(episode.Filename)
}

func (s *Store) getEpisodeKeyByFilename(filename string) ([]byte, error) {
	return []byte(filename), nil
}

// Verify Store implements storage.Store interface.
var _ storage.Store = (*Store)(nil)
