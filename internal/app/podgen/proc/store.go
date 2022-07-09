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
func (b *BoltDB) SaveEpisode(podcastName string, episode podcast.Episode) (bool, error) {
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
		bucket, e := tx.CreateBucketIfNotExists([]byte(podcastName))
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

		log.Printf("[INFO] save episode %s - %s - %s - %d", string(key), podcastName, episode.Filename, episode.Size)
		e = bucket.Put(key, jdata)
		if e != nil {
			return e
		}

		created = true
		return e
	})

	return created, err
}
