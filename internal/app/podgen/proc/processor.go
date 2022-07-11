package proc

import (
	"context"
	"fmt"
	"sync"

	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/podcast"
)

// Processor is searcher of episode files and writer to store
type Processor struct {
	Storage  *BoltDB
	Files    *Files
	S3Client *S3Store
}

type UploadEpisode struct {
	PodcastID string
	Filename  string
}

// Update podcast files
func (p *Processor) Update(folderName, podcastID string) (int64, error) {
	var countNew int64
	episodes, err := p.Files.FindEpisodes(folderName)
	if err != nil {
		log.Fatalf("[ERROR] can't scan folder %s, %v", folderName, err)
		return 0, err
	}

	for _, episode := range episodes {
		created, e := p.Storage.SaveEpisode(podcastID, episode)
		if e != nil {
			log.Fatalf("[ERROR] can't add episode %s to %s, %v", episode.Filename, podcastID, e)
		}
		if created {
			countNew++
		}

	}

	return countNew, nil
}

func (p *Processor) DeleteOldEpisodes(podcastID string) (bool, error) {
	p.Storage.FindEpisodesByStatus(podcastID, podcast.New)

	// err = p.Storage.ChangeStatusEpisodes(podcastID, podcast.New, podcast.Deleted)
	// if err != nil {
	// 	if err != nil {
	// 		log.Fatalf("[ERROR] can't find uploaded episodes from %s, %v", podcastID, err)
	// 	}
	// }
	//
	return false, nil
}

func (p *Processor) UploadNewEpisodes(podcastID, podcastFolder string, maxSize int64) {
	episodes, err := p.Storage.FindEpisodesBySize(podcastID, podcast.New, maxSize)
	if err != nil {
		log.Fatalf("[ERROR] can't find episodes %s, %v", podcastID, err)
	}

	uploadCh := make(chan UploadEpisode)
	wg := sync.WaitGroup{}
	ctx := context.Background()
	for _, episode := range episodes {
		wg.Add(1)

		go func(uploadCh chan UploadEpisode, podcastID string, episodeItem podcast.Episode) {
			log.Printf("[INFO] Started upload episode %s - %s", podcastID, episodeItem.Filename)
			uploadInfo, err := p.S3Client.UploadEpisode(ctx,
				fmt.Sprintf("%s/%s", podcastFolder, episodeItem.Filename),
				fmt.Sprintf("storage/%s/%s", podcastFolder, episodeItem.Filename))
			if err != nil {
				log.Printf("[ERROR] can't upload episode %s, %v", episodeItem.Filename, err)
				wg.Done()
				return
			}

			log.Printf("[INFO] Episode uploaded %s - %s", episodeItem.Filename, uploadInfo.Location)
			uploadCh <- UploadEpisode{PodcastID: podcastID, Filename: episodeItem.Filename}
			wg.Done()
		}(uploadCh, podcastID, episode)
	}

	go func() {
		wg.Wait()
		close(uploadCh)
	}()

	for uploadedEpisode := range uploadCh {
		episode, err := p.Storage.GetEpisodeByFilename(uploadedEpisode.PodcastID, uploadedEpisode.Filename)
		if err != nil {
			log.Printf("[ERROR] can't get episode by filename %s - %s, %v", uploadedEpisode.PodcastID, uploadedEpisode.Filename, err)
		}

		if err = p.Storage.ChangeEpisodeStatus(podcastID, episode, podcast.Uploaded); err != nil {
			log.Printf("[ERROR] can't change status episode %s, %v", episode.Filename, err)
		}
	}

}
