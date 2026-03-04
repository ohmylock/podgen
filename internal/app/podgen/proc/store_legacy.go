package proc

import (
	"encoding/json"
	"sync"

	"github.com/boltdb/bolt"
	log "github.com/go-pkgz/lgr"

	"podgen/internal/app/podgen/podcast"
	apperrors "podgen/internal/errors"
	"podgen/internal/storage"
)

// legacyMu provides write serialization for legacy DB access.
var legacyMu sync.Mutex

// saveLegacy implements SaveEpisode using the legacy DB field.
func (b *BoltDB) saveLegacy(podcastID string, episode *podcast.Episode) error {
	legacyMu.Lock()
	defer legacyMu.Unlock()
	return b.DB.Update(func(tx *bolt.Tx) error {
		key := []byte(episode.Filename)
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
	})
}

// findByStatusLegacy implements FindEpisodesByStatus using the legacy DB field.
func (b *BoltDB) findByStatusLegacy(podcastID string, filterStatus podcast.Status) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	err := b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			log.Printf("no bucket for %s", podcastID)
			return &apperrors.EpisodeError{PodcastID: podcastID, Op: "FindEpisodesByStatus", Err: apperrors.ErrNoBucket}
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

// findBySessionLegacy implements FindEpisodesBySession using the legacy DB field.
func (b *BoltDB) findBySessionLegacy(podcastID, session string) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	err := b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			return &apperrors.EpisodeError{PodcastID: podcastID, Op: "FindEpisodesBySession", Err: apperrors.ErrNoBucket}
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
		return nil
	})
	return result, err
}

// changeStatusLegacy implements ChangeStatusEpisodes using the legacy DB field.
func (b *BoltDB) changeStatusLegacy(podcastID string, fromStatus, toStatus podcast.Status) error {
	return b.DB.Batch(func(tx *bolt.Tx) error {
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

// findBySizeLimitLegacy implements FindEpisodesBySizeLimit using the legacy DB field.
func (b *BoltDB) findBySizeLimitLegacy(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	err := b.DB.View(func(tx *bolt.Tx) error {
		episodes, err := b.findByStatusInTx(tx, podcastID, status)
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

func (b *BoltDB) findByStatusInTx(tx *bolt.Tx, podcastID string, filterStatus podcast.Status) ([]*podcast.Episode, error) {
	var result []*podcast.Episode
	bucket := tx.Bucket([]byte(podcastID))
	if bucket == nil {
		log.Printf("no bucket for %s", podcastID)
		return nil, &apperrors.EpisodeError{PodcastID: podcastID, Op: "FindEpisodesByStatus", Err: apperrors.ErrNoBucket}
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

// getByFilenameLegacy implements GetEpisodeByFilename using the legacy DB field.
func (b *BoltDB) getByFilenameLegacy(podcastID, fileName string) (*podcast.Episode, error) {
	var episode *podcast.Episode
	err := b.DB.View(func(tx *bolt.Tx) error {
		key := []byte(fileName)
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			log.Printf("[WARN] no bucket for %s", podcastID)
			return &apperrors.EpisodeError{PodcastID: podcastID, Op: "GetEpisodeByFilename", Err: apperrors.ErrNoBucket}
		}
		item := bucket.Get(key)
		if item == nil {
			return storage.ErrNotFound
		}
		episode = &podcast.Episode{}
		if err := json.Unmarshal(item, episode); err != nil {
			log.Printf("[WARN] failed to unmarshal, %v", err)
			return err
		}
		if episode.Filename == "" {
			episode = nil
		}
		return nil
	})
	return episode, err
}

// getLastByStatusLegacy implements GetLastEpisodeByStatus using the legacy DB field.
//
//nolint:dupl // intentionally similar to getLastByNotStatusLegacy
func (b *BoltDB) getLastByStatusLegacy(podcastID string, status podcast.Status) (*podcast.Episode, error) {
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

// getLastByNotStatusLegacy implements GetLastEpisodeByNotStatus using the legacy DB field.
//
//nolint:dupl // intentionally similar to getLastByStatusLegacy
func (b *BoltDB) getLastByNotStatusLegacy(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	var result *podcast.Episode
	err := b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(podcastID))
		if bucket == nil {
			return &apperrors.EpisodeError{PodcastID: podcastID, Op: "GetLastEpisodeByNotStatus", Err: apperrors.ErrNoBucket}
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
			result = &item
			break
		}
		return nil
	})
	return result, err
}
