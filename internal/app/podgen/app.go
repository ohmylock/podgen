package podgen

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	log "github.com/go-pkgz/lgr"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"podgen/internal/app/podgen/proc"
	"podgen/internal/configs"
)

// App structure application
type App struct {
	config    *configs.Conf
	processor *proc.Processor
}

// NewApplication create application instance
func NewApplication(conf *configs.Conf, p *proc.Processor) (*App, error) {
	app := App{config: conf, processor: p}
	return &app, nil
}

// NewBoltDB create boltDb instance
func NewBoltDB(dbFile string) (*bolt.DB, error) {
	log.Printf("[INFO] bolt (persistent) store, %s", dbFile)
	if dbFile == "" {
		return nil, fmt.Errorf("empty db")
	}
	if err := os.MkdirAll(path.Dir(dbFile), 0o700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(dbFile, 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	if err != nil {
		return nil, err
	}

	return db, err
}

// NewS3Client create s3 client instance
func NewS3Client(endpoint, accessKeyID, secretAccessKey string, useSSL bool) (*minio.Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

// Update find and add to db new episodes of podcast
func (a *App) Update() {
	podcasts := a.findPodcasts()

	wg := sync.WaitGroup{}
	for i, p := range podcasts {
		wg.Add(1)
		go func(i string, p configs.Podcast) {
			countNew, err := a.updateFolder(p.Folder, i)
			if err != nil {
				wg.Done()
				return
			}
			if countNew > 0 {
				log.Printf("[INFO] found new %d episodes for %s", countNew, p.Title)
			}
			wg.Done()
		}(i, p)
	}
	wg.Wait()
}

// Upload episodes by podcasts
func (a *App) Upload(podcastIDs string) {
	podcasts := a.findPodcasts()

	var foundedPodcasts = make(map[string]configs.Podcast, len(podcasts))
	splitPodcastIDs := strings.Split(podcastIDs, ",")
	for podcastID, p := range podcasts {
		for _, rawPodcastID := range splitPodcastIDs {
			if podcastID != rawPodcastID {
				continue
			}
			foundedPodcasts[podcastID] = p
		}
	}

	wg := sync.WaitGroup{}
	for i, p := range foundedPodcasts {
		wg.Add(1)
		go func(i string, p configs.Podcast) {
			a.processor.UploadNewEpisodes(i, p.Folder, p.MaxSize)
			wg.Done()
		}(i, p)
	}
	wg.Wait()
}

// DeleteOldEpisodes delete old episodes by podcasts
func (a *App) DeleteOldEpisodes(podcastIDs string) {
	podcasts := a.findPodcasts()

	var foundedPodcasts = make(map[string]configs.Podcast, len(podcasts))
	splitPodcastIDs := strings.Split(podcastIDs, ",")
	for podcastID, p := range podcasts {
		for _, rawPodcastID := range splitPodcastIDs {
			if podcastID != rawPodcastID {
				continue
			}
			foundedPodcasts[podcastID] = p
		}
	}

	wg := sync.WaitGroup{}
	for i, p := range foundedPodcasts {
		wg.Add(1)
		go func(i string, p configs.Podcast) {
			err := a.processor.DeleteOldEpisodesByPodcast(i, p.Folder)
			if err != nil {
				log.Fatalf("[ERROR] can't delete old episodes by podcast %s, %v", i, err)
			}
			wg.Done()
		}(i, p)
	}
	wg.Wait()
}

func (a *App) findPodcasts() map[string]configs.Podcast {
	return a.config.Podcasts
}

func (a *App) updateFolder(folderName, podcastID string) (int64, error) {
	countNew, err := a.processor.Update(folderName, podcastID)
	if err != nil {
		return 0, err
	}

	return countNew, nil
}

// func (a *App) deleteOldEpisodes(p configs.Podcast) {
// 	a.processor.DeleteOldEpisodes(p.Folder)
// }
