package proc

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/podcast"
)

// BoltDB store
type BoltDB struct {
	DB *bolt.DB
}

// SaveEpisode save episodes to podcast bucket in bolt db
func (b *BoltDB) SaveEpisode(podcastID string, episode podcast.Episode) (bool, error) {
	var created bool
	key, err := func() ([]byte, error) {
		h := sha256.New()
		if _, err := h.Write([]byte(episode.Filename)); err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintf("%x-%d", h.Sum(nil), episode.Size)), nil
	}()

	if err != nil {
		return created, err
	}

	err = b.DB.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(podcastID))
		if e != nil {
			return e
		}

		if bucket.Get(key) != nil {
			return nil
		}

		jdata, jerr := json.Marshal(&episode)
		if jerr != nil {
			return jerr
		}

		log.Printf("[INFO] save episode %s - %s - %s - %d", string(key), podcastID, episode.Filename, episode.Size)
		e = bucket.Put(key, jdata)
		if e != nil {
			return e
		}

		created = true
		return e
	})

	return created, err
}

// FindEpisodesByStatus get episodes from store by status
func (b *BoltDB) FindEpisodesByStatus(podcastID string, filterStatus podcast.Status) ([]podcast.Episode, error) {
	var result []podcast.Episode
	err := b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", podcastID)
		}

		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			item := podcast.Episode{}
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal, %v", err)
				continue
			}
			if item.Status != filterStatus {
				continue
			}
			result = append(result, item)
		}
		return nil
	})

	return result, err
}

// ChangeStatusEpisodes change status of episodes in store
func (b *BoltDB) ChangeStatusEpisodes(podcastID string, fromStatus podcast.Status, toStatus podcast.Status) error {
	err := b.DB.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", podcastID)
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
