// Package podgen main
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

const podcastDefaultImage string = "podcast.png"

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
func (a *App) Update(podcastIDs string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	wg := sync.WaitGroup{}
	for i, p := range podcasts {
		wg.Add(1)
		go func(i string, p configs.Podcast) {
			defer wg.Done()
			countNew, err := a.updateFolder(p.Folder, i)
			if err != nil {
				return
			}
			if countNew > 0 {
				log.Printf("[INFO] found new %d episodes for %s", countNew, p.Title)
			}
		}(i, p)
	}
	wg.Wait()
}

// UploadEpisodes by podcasts to s3 storage
func (a *App) UploadEpisodes(podcastIDs string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	wg := sync.WaitGroup{}
	for i, p := range podcasts {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i string, p configs.Podcast) {
			defer wg.Done()
			a.processor.UploadNewEpisodes(i, p.Folder, p.MaxSize)
		}(&wg, i, p)
	}
	wg.Wait()
}

// DeleteOldEpisodes delete old episodes by podcasts
func (a *App) DeleteOldEpisodes(podcastIDs string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	wg := sync.WaitGroup{}
	for i, p := range podcasts {
		wg.Add(1)
		go func(i string, p configs.Podcast) {
			defer wg.Done()

			if !p.DeleteOldEpisodes {
				return
			}

			err := a.processor.DeleteOldEpisodesByPodcast(i, p.Folder)
			if err != nil {
				log.Fatalf("[ERROR] can't delete old episodes by podcast %s, %v", i, err)
			}
		}(i, p)
	}
	wg.Wait()
}

// GenerateFeed for podcasts
func (a *App) GenerateFeed(podcastIDs string, podcastImages map[string]string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	wg := sync.WaitGroup{}
	for i, p := range podcasts {
		wg.Add(1)
		go func(i string, p configs.Podcast) {
			defer wg.Done()

			podcastImageURL, ok := podcastImages[i]
			if !ok {
				podcastImageURL = ""
			}

			feedFilename, err := a.processor.GenerateFeed(i, p, podcastImageURL)
			if err != nil {
				log.Fatalf("%s", err)
			}
			uploadInfo := a.processor.UploadFeed(p.Folder, feedFilename)
			log.Printf("Feed url %s", uploadInfo.Location)
		}(i, p)
	}
	wg.Wait()
}

// UploadPodcastImage by podcast to s3 storage
func (a *App) UploadPodcastImage(podcastIDs string) map[string]string {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	var result = make(map[string]string, len(podcasts))
	wg := sync.WaitGroup{}
	for i, p := range podcasts {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i string, p configs.Podcast) {
			defer wg.Done()

			imageURL, err := a.processor.UploadPodcastImage(i, p.Folder, podcastDefaultImage)
			if err != nil {
				log.Printf("[ERROR] can't upload podcast image %s, %v", podcastDefaultImage, err)
				return
			}

			result[i] = imageURL
		}(&wg, i, p)
	}

	wg.Wait()

	return result
}

// GetPodcastImages by podcast from s3 storage
func (a *App) GetPodcastImages(podcastIDs string) map[string]string {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	var result = make(map[string]string, len(podcasts))
	wg := sync.WaitGroup{}
	for i, p := range podcasts {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i string, p configs.Podcast) {
			defer wg.Done()
			result[i] = a.processor.GetPodcastImage(p.Folder, podcastDefaultImage)
		}(&wg, i, p)
	}
	wg.Wait()

	return result
}

// FindPodcasts get list podcast from config file
func (a *App) FindPodcasts() map[string]configs.Podcast {
	return a.config.Podcasts
}

func (a *App) updateFolder(folderName, podcastID string) (int64, error) {
	countNew, err := a.processor.Update(folderName, podcastID)
	if err != nil {
		return 0, err
	}

	return countNew, nil
}

func (a *App) filterPodcastsByPodcastIDs(podcastIDs string) map[string]configs.Podcast {
	podcasts := a.FindPodcasts()
	var result = make(map[string]configs.Podcast, len(podcasts))
	splitPodcastIDs := strings.Split(podcastIDs, ",")
	for podcastID, p := range podcasts {
		for _, rawPodcastID := range splitPodcastIDs {
			if podcastID != strings.Trim(rawPodcastID, " ") {
				continue
			}
			result[podcastID] = p
		}
	}

	return result
}
