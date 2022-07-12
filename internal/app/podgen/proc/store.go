package proc

import (
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
func (b *BoltDB) SaveEpisode(podcastID string, episode *podcast.Episode) error {
	key, err := b.getEpisodeKey(episode)

	if err != nil {
		return err
	}

	err = b.DB.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(podcastID))
		if e != nil {
			return e
		}

		jdata, jerr := json.Marshal(episode)
		if jerr != nil {
			return jerr
		}

		log.Printf("[INFO] save episode %s - %s - %s - %d", string(key), podcastID, episode.Filename, episode.Size)
		e = bucket.Put(key, jdata)
		if e != nil {
			return e
		}

		return e
	})

	return err
}

// FindEpisodesByStatus get episodes from store by status
func (b *BoltDB) FindEpisodesByStatus(podcastID string, filterStatus podcast.Status) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	err := b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			log.Fatalf("no bucket for %s", podcastID)
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
		return nil
	})

	return result, err
}

// ChangeStatusEpisodes change status of episodes in store
func (b *BoltDB) ChangeStatusEpisodes(podcastID string, fromStatus, toStatus podcast.Status) error {
	err := b.DB.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			log.Fatalf("no bucket for %s", podcastID)
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

// ChangeEpisodeStatus change status of episodes in store
func (b *BoltDB) ChangeEpisodeStatus(podcastID string, episode *podcast.Episode, status podcast.Status) error {
	episode.Status = status
	return b.SaveEpisode(podcastID, episode)
}

// FindEpisodesBySizeLimit get list of episodes with total size limit
func (b *BoltDB) FindEpisodesBySizeLimit(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
	episodes, err := b.FindEpisodesByStatus(podcastID, status)
	if err != nil {
		log.Printf("[INFO] No episodes in podcast %s", podcastID)
		return nil, nil
	}
	var sizes int64
	var result = make([]*podcast.Episode, len(episodes))
	for i, episode := range episodes {
		if sizeLimit > 0 && (sizes >= sizeLimit || (sizes+episode.Size) >= sizeLimit) {
			return result[:i], nil
		}
		sizes += episode.Size
		result[i] = episode
	}

	return result, nil
}

// GetEpisodeByFilename get episode by filename from store
func (b *BoltDB) GetEpisodeByFilename(podcastID, fileName string) (*podcast.Episode, error) {
	key, err := b.getEpisodeKeyByFilename(fileName)
	if err != nil {
		return nil, err
	}

	episode := &podcast.Episode{}
	err = b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			log.Printf("[WARN] no bucket for %s", podcastID)
			return fmt.Errorf("no bucket for %s", podcastID)
		}

		item := bucket.Get(key)
		if item == nil {
			return nil
		}

		if err = json.Unmarshal(item, episode); err != nil {
			log.Printf("[WARN] failed to unmarshal, %v", err)
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	if episode.Filename == "" {
		return nil, nil
	}

	return episode, nil
}

func (b *BoltDB) getEpisodeKey(episode *podcast.Episode) ([]byte, error) {
	return b.getEpisodeKeyByFilename(episode.Filename)
}

func (b *BoltDB) getEpisodeKeyByFilename(filename string) ([]byte, error) {
	return []byte(filename), nil
}
