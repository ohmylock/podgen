// Package podgen main
package podgen

import (
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"strings"
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

	for i, p := range podcasts {
		countNew, err := a.processor.Update(p.Folder, i)
		if err != nil {
			log.Printf("[ERROR] can't update folder %s, %v", p.Folder, err)
			continue
		}
		if countNew > 0 {
			log.Printf("[INFO] found new %d episodes for %s", countNew, p.Title)
		}
	}
}

// UploadEpisodes by podcasts to s3 storage
func (a *App) UploadEpisodes(podcastIDs string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	session, err := a.makeSessionString()
	if err != nil {
		log.Printf("[ERROR] can't make session string, %v", err)
		return
	}

	log.Printf("[INFO] Start session: %s", session)

	for i, p := range podcasts {
		if err := a.processor.UploadNewEpisodes(session, i, p.Folder, p.MaxSize); err != nil {
			log.Printf("[ERROR] can't upload new episodes for %s, %v", i, err)
		}
	}
}

// DeleteOldEpisodes delete old episodes by podcasts
func (a *App) DeleteOldEpisodes(podcastIDs string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	for i, p := range podcasts {
		if !p.DeleteOldEpisodes {
			continue
		}

		err := a.processor.DeleteOldEpisodesByPodcast(i, p.Folder)
		if err != nil {
			log.Printf("[ERROR] can't delete old episodes by podcast %s, %v", i, err)
		}
	}
}

// GenerateFeed for podcasts
func (a *App) GenerateFeed(podcastIDs string, podcastImages map[string]string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	for i, p := range podcasts {
		podcastImageURL := podcastImages[i]

		feedFilename, err := a.processor.GenerateFeed(i, p, podcastImageURL)
		if err != nil {
			log.Printf("[ERROR] can't generate feed for %s, %v", i, err)
			continue
		}
		uploadInfo := a.processor.UploadFeed(p.Folder, feedFilename)
		if uploadInfo != nil {
			log.Printf("Feed url %s", uploadInfo.Location)
		}
	}
}

// UploadPodcastImage by podcast to s3 storage
func (a *App) UploadPodcastImage(podcastIDs string) map[string]string {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	result := make(map[string]string, len(podcasts))
	for i, p := range podcasts {
		imageURL, err := a.processor.UploadPodcastImage(i, p.Folder, podcastDefaultImage)
		if err != nil {
			log.Printf("[ERROR] can't upload podcast image %s, %v", podcastDefaultImage, err)
			continue
		}
		result[i] = imageURL
	}

	return result
}

// GetPodcastImages by podcast from s3 storage
func (a *App) GetPodcastImages(podcastIDs string) map[string]string {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	result := make(map[string]string, len(podcasts))
	for i, p := range podcasts {
		imageURL := a.processor.GetPodcastImage(p.Folder, podcastDefaultImage)
		result[i] = imageURL
	}

	return result
}

// FindPodcasts get list podcast from config file
func (a *App) FindPodcasts() map[string]configs.Podcast {
	return a.config.Podcasts
}

// RollbackEpisodes rollback last episode by podcasts
func (a *App) RollbackEpisodes(podcastIDs string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	for i := range podcasts {
		err := a.processor.RollbackLastEpisodes(i)
		if err != nil {
			log.Printf("[ERROR] can't rollback episode by podcast %s, %v", i, err)
		}
	}
}

// RollbackEpisodesBySession rollback episodes by podcasts and session
func (a *App) RollbackEpisodesBySession(podcastIDs, session string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	for i := range podcasts {
		err := a.processor.RollbackEpisodesOfSession(i, session)
		if err != nil {
			log.Printf("[ERROR] can't rollback episode by podcast %s, %v", i, err)
		}
	}
}

func (a *App) filterPodcastsByPodcastIDs(podcastIDs string) map[string]configs.Podcast {
	podcasts := a.FindPodcasts()
	result := make(map[string]configs.Podcast, len(podcasts))
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

// makeSessionString generate session string
func (a *App) makeSessionString() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Printf("[ERROR] can't generate session string, %v", err)
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
