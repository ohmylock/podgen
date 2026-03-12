// Package podgen main
package podgen

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

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
func (a *App) Update(ctx context.Context, podcastIDs string) error {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	if len(podcasts) == 0 {
		log.Printf("[WARN] no podcasts found for IDs: %s", podcastIDs)
		return nil
	}

	var errs []error
	for i, p := range podcasts {
		log.Printf("[INFO] scanning podcast %s, folder: %s", i, p.Folder)
		countNew, err := a.processor.Update(ctx, p.Folder, i)
		if err != nil {
			log.Printf("[ERROR] can't update folder %s, %v", p.Folder, err)
			errs = append(errs, fmt.Errorf("update %s: %w", i, err))
			continue
		}
		if countNew > 0 {
			log.Printf("[INFO] found new %d episodes for %s", countNew, p.Title)
		} else {
			log.Printf("[INFO] no new episodes for %s", p.Title)
		}
	}
	return errors.Join(errs...)
}

// UploadEpisodes by podcasts to s3 storage
func (a *App) UploadEpisodes(ctx context.Context, podcastIDs string) error {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	if len(podcasts) == 0 {
		log.Printf("[WARN] no podcasts found for IDs: %s", podcastIDs)
		return nil
	}

	session, err := a.makeSessionString()
	if err != nil {
		log.Printf("[ERROR] can't make session string, %v", err)
		return fmt.Errorf("make session string: %w", err)
	}

	log.Printf("[INFO] Start session: %s", session)

	var errs []error
	for i, p := range podcasts {
		log.Printf("[INFO] uploading podcast %s, folder: %s, maxSize: %d", i, p.Folder, p.MaxSize)
		if err := a.processor.UploadNewEpisodes(ctx, session, i, p.Folder, p.MaxSize); err != nil {
			log.Printf("[ERROR] can't upload new episodes for %s, %v", i, err)
			errs = append(errs, fmt.Errorf("upload %s: %w", i, err))
		}
	}
	return errors.Join(errs...)
}

// DeleteOldEpisodes delete old episodes by podcasts
// If force is true, deletes for all podcasts regardless of delete_old_episodes config
func (a *App) DeleteOldEpisodes(ctx context.Context, podcastIDs string, force bool) error {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	var errs []error
	for i, p := range podcasts {
		if !force && !p.DeleteOldEpisodes {
			continue
		}

		err := a.processor.DeleteOldEpisodesByPodcast(ctx, i, p.Folder)
		if err != nil {
			log.Printf("[ERROR] can't delete old episodes by podcast %s, %v", i, err)
			errs = append(errs, fmt.Errorf("delete %s: %w", i, err))
		}
	}
	return errors.Join(errs...)
}

// GenerateFeed for podcasts
func (a *App) GenerateFeed(ctx context.Context, podcastIDs string, podcastImages map[string]string) error {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	var errs []error
	for i, p := range podcasts {
		podcastImageURL := podcastImages[i]

		feedFilename, err := a.processor.GenerateFeed(ctx, i, p, podcastImageURL)
		if err != nil {
			log.Printf("[ERROR] can't generate feed for %s, %v", i, err)
			errs = append(errs, fmt.Errorf("generate feed %s: %w", i, err))
			continue
		}
		uploadInfo, err := a.processor.UploadFeed(ctx, p.Folder, feedFilename)
		if err != nil {
			log.Printf("[ERROR] can't upload feed for %s, %v", i, err)
			errs = append(errs, fmt.Errorf("upload feed %s: %w", i, err))
			continue
		}
		if uploadInfo != nil {
			log.Printf("Feed url %s", uploadInfo.Location)
		}
	}
	return errors.Join(errs...)
}

// UploadPodcastImage by podcast to s3 storage.
// If forceRegenerate is true, artwork is regenerated even if an image already exists.
func (a *App) UploadPodcastImage(ctx context.Context, podcastIDs string, forceRegenerate bool) map[string]string {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	result := make(map[string]string, len(podcasts))
	for i, p := range podcasts {
		imageURL, err := a.processor.UploadPodcastImage(ctx, i, p.Folder, podcastDefaultImage, a.config.IsArtworkAutoGenerateEnabled(), forceRegenerate, p.Title)
		if err != nil {
			log.Printf("[ERROR] can't upload podcast image %s, %v", podcastDefaultImage, err)
			continue
		}
		result[i] = imageURL
	}

	return result
}

// GetPodcastImages by podcast from s3 storage
func (a *App) GetPodcastImages(ctx context.Context, podcastIDs string) map[string]string {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	result := make(map[string]string, len(podcasts))
	for i, p := range podcasts {
		imageURL := a.processor.GetPodcastImage(ctx, p.Folder, podcastDefaultImage)
		result[i] = imageURL
	}

	return result
}

// FindPodcasts get list podcast from config file
func (a *App) FindPodcasts() map[string]configs.Podcast {
	return a.config.Podcasts
}

// GetFeedURLs returns RSS feed URLs for specified podcasts
func (a *App) GetFeedURLs(podcastIDs string) map[string]string {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)
	result := make(map[string]string, len(podcasts))

	for id, p := range podcasts {
		url, err := a.processor.GetFeedURL(id, p.Folder,
			a.config.CloudStorage.EndPointURL,
			a.config.CloudStorage.Bucket)
		if err != nil {
			log.Printf("[ERROR] can't get feed URL for %s: %v", id, err)
			continue
		}
		result[id] = url
	}
	return result
}

// RollbackEpisodes rollback last episode by podcasts
func (a *App) RollbackEpisodes(ctx context.Context, podcastIDs string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	for i := range podcasts {
		err := a.processor.RollbackLastEpisodes(ctx, i)
		if err != nil {
			log.Printf("[ERROR] can't rollback episode by podcast %s, %v", i, err)
		}
	}
}

// RollbackEpisodesBySession rollback episodes by podcasts and session
func (a *App) RollbackEpisodesBySession(ctx context.Context, podcastIDs, session string) {
	podcasts := a.filterPodcastsByPodcastIDs(podcastIDs)

	for i := range podcasts {
		err := a.processor.RollbackEpisodesOfSession(ctx, i, session)
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
