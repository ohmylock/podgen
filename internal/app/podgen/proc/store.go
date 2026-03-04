package proc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/podcast"
	apperrors "podgen/internal/errors"
)

// BoltDB store
type BoltDB struct {
	DB *bolt.DB
	mu sync.Mutex
}

// WithWriteTx executes fn within a serialized write transaction.
func (b *BoltDB) WithWriteTx(fn func(*bolt.Tx) error) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.DB.Update(fn)
}

// WithReadTx executes fn within a read-only transaction.
func (b *BoltDB) WithReadTx(fn func(*bolt.Tx) error) error {
	return b.DB.View(fn)
}

// SaveEpisode save episodes to podcast bucket in bolt db
func (b *BoltDB) SaveEpisode(podcastID string, episode *podcast.Episode) error {
	return b.WithWriteTx(func(tx *bolt.Tx) error {
		return b.saveEpisode(tx, podcastID, episode)
	})
}

func (b *BoltDB) saveEpisode(tx *bolt.Tx, podcastID string, episode *podcast.Episode) error {
	key, err := b.getEpisodeKey(episode)
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

	log.Printf("[INFO] save episode %s - %s", podcastID, episode.Filename)
	return bucket.Put(key, jdata)
}

// FindEpisodesByStatus get episodes from store by status
func (b *BoltDB) FindEpisodesByStatus(podcastID string, filterStatus podcast.Status) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	err := b.WithReadTx(func(tx *bolt.Tx) error {
		var err error
		result, err = b.findEpisodesByStatus(tx, podcastID, filterStatus)
		return err
	})
	return result, err
}

func (b *BoltDB) findEpisodesByStatus(tx *bolt.Tx, podcastID string, filterStatus podcast.Status) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	bucket := tx.Bucket([]byte(podcastID))
	if bucket == nil {
		log.Printf("no bucket for %s", podcastID)
		return nil, errors.New("no bucket")
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

// FindEpisodesBySession get episodes from store by session
func (b *BoltDB) FindEpisodesBySession(podcastID, session string) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	err := b.WithReadTx(func(tx *bolt.Tx) error {
		var err error
		result, err = b.findEpisodesBySession(tx, podcastID, session)
		return err
	})
	return result, err
}

func (b *BoltDB) findEpisodesBySession(tx *bolt.Tx, podcastID, session string) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	bucket := tx.Bucket([]byte(podcastID))
	if bucket == nil {
		return nil, &apperrors.EpisodeError{PodcastID: podcastID, Op: "FindEpisodesBySession", Err: apperrors.ErrNoBucket}
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

// ChangeStatusEpisodes change status of episodes in store
func (b *BoltDB) ChangeStatusEpisodes(podcastID string, fromStatus, toStatus podcast.Status) error {
	err := b.DB.Batch(func(tx *bolt.Tx) error {
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

	return err
}

// FindEpisodesBySizeLimit get list of episodes with total size limit
func (b *BoltDB) FindEpisodesBySizeLimit(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	err := b.WithReadTx(func(tx *bolt.Tx) error {
		episodes, err := b.findEpisodesByStatus(tx, podcastID, status)
		if err != nil {
			log.Printf("[INFO] No episodes with status %d in podcast %s: %v", status, podcastID, err)
			return nil
		}
		var sizes int64
		result = make([]*podcast.Episode, len(episodes))
		for i, episode := range episodes {
			if sizeLimit > 0 && (sizes >= sizeLimit || (sizes+episode.Size) >= sizeLimit) {
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

// GetEpisodeByFilename get episode by filename from store
func (b *BoltDB) GetEpisodeByFilename(podcastID, fileName string) (*podcast.Episode, error) {
	var episode *podcast.Episode
	err := b.WithReadTx(func(tx *bolt.Tx) error {
		var err error
		episode, err = b.getEpisodeByFilenameInTx(tx, podcastID, fileName)
		return err
	})
	return episode, err
}

func (b *BoltDB) getEpisodeByFilenameInTx(tx *bolt.Tx, podcastID, fileName string) (*podcast.Episode, error) {
	key, err := b.getEpisodeKeyByFilename(fileName)
	if err != nil {
		return nil, err
	}

	episode := &podcast.Episode{}
	bucket := tx.Bucket([]byte(podcastID))
	if bucket == nil {
		log.Printf("[WARN] no bucket for %s", podcastID)
		return nil, fmt.Errorf("no bucket for %s", podcastID)
	}

	item := bucket.Get(key)
	if item == nil {
		return nil, errors.New("no episode found")
	}

	if err = json.Unmarshal(item, episode); err != nil {
		log.Printf("[WARN] failed to unmarshal, %v", err)
		return nil, err
	}

	if episode.Filename == "" {
		return nil, nil
	}

	return episode, nil
}

// GetLastEpisodeByStatus get last episode from store by status
func (b *BoltDB) GetLastEpisodeByStatus(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	var result *podcast.Episode
	err := b.DB.View(func(tx *bolt.Tx) error {
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

// GetLastEpisodeByNotStatus get last episode from store by not status
func (b *BoltDB) GetLastEpisodeByNotStatus(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	var result *podcast.Episode
	err := b.WithReadTx(func(tx *bolt.Tx) error {
		var err error
		result, err = b.getLastEpisodeByNotStatusInTx(tx, podcastID, status)
		return err
	})
	return result, err
}

func (b *BoltDB) getLastEpisodeByNotStatusInTx(tx *bolt.Tx, podcastID string, status podcast.Status) (*podcast.Episode, error) {
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

func (b *BoltDB) getEpisodeKey(episode *podcast.Episode) ([]byte, error) {
	return b.getEpisodeKeyByFilename(episode.Filename)
}

func (b *BoltDB) getEpisodeKeyByFilename(filename string) ([]byte, error) {
	return []byte(filename), nil
}
