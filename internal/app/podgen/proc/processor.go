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

// UploadedEpisode struct for result of upload
type UploadedEpisode struct {
	PodcastID string
	Filename  string
	Location  string
}

// DeletedEpisode struct for result of delete
type DeletedEpisode struct {
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
		item, err := p.Storage.GetEpisodeByFilename(podcastID, episode.Filename)
		if err != nil {
			log.Printf("get episode by filename error, %v", err)
		}

		if item != nil {
			continue
		}

		e := p.Storage.SaveEpisode(podcastID, episode)
		if e != nil {
			log.Fatalf("[ERROR] can't add episode %s to %s, %v", episode.Filename, podcastID, e)
		}
		countNew++
	}

	return countNew, nil
}

// DeleteOldEpisodesByPodcast from s3 storage
func (p *Processor) DeleteOldEpisodesByPodcast(podcastID, podcastFolder string) error {
	episodes, err := p.Storage.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	if err != nil {
		log.Fatalf("[ERROR] can't find episodes %s, %v", podcastID, err)
	}
	deleteCh := make(chan DeletedEpisode)
	done := make(chan bool)
	wg := sync.WaitGroup{}
	ctx := context.Background()
	for _, episode := range episodes {
		wg.Add(1)

		go func(deleteCh chan DeletedEpisode, podcastID string, episodeItem *podcast.Episode) {
			log.Printf("[INFO] Started upload episode %s - %s", podcastID, episodeItem.Filename)
			err := p.S3Client.DeleteEpisode(ctx, fmt.Sprintf("%s/%s", podcastFolder, episodeItem.Filename))
			if err != nil {
				log.Printf("[ERROR] can't delete episode %s, %v", episodeItem.Filename, err)
				wg.Done()
				return
			}

			log.Printf("[INFO] Episode deleted %s - %s", episodeItem.Filename, podcastID)
			deleteCh <- DeletedEpisode{PodcastID: podcastID, Filename: episodeItem.Filename}
			wg.Done()
		}(deleteCh, podcastID, episode)
	}

	go func() {
		wg.Wait()
		done <- true
	}()

Loop:
	for {
		select {
		case deletedEpisode := <-deleteCh:
			log.Printf("%+v", deletedEpisode)
			episode, err := p.Storage.GetEpisodeByFilename(deletedEpisode.PodcastID, deletedEpisode.Filename)
			if err != nil {
				log.Printf("[ERROR] can't get episode by filename %s - %s, %v", deletedEpisode.PodcastID, deletedEpisode.Filename, err)
			}
			episode.Status = podcast.Deleted
			if err = p.Storage.SaveEpisode(podcastID, episode); err != nil {
				log.Printf("[ERROR] can't change status episode %s, %v", episode.Filename, err)
			}
			log.Printf("[INFO] episode saved %+v", episode)
		case <-done:
			close(deleteCh)
			break Loop
		}
	}
	return nil
}

// UploadNewEpisodes get new episodes by total limit of size and upload to s3 storage
func (p *Processor) UploadNewEpisodes(podcastID, podcastFolder string, sizeLimit int64) {
	episodes, err := p.Storage.FindEpisodesBySizeLimit(podcastID, podcast.New, sizeLimit)
	if err != nil {
		log.Fatalf("[ERROR] can't find episodes %s, %v", podcastID, err)
	}

	uploadCh := make(chan UploadedEpisode)
	done := make(chan bool)
	wg := sync.WaitGroup{}
	ctx := context.Background()
	for _, episode := range episodes {
		wg.Add(1)

		go func(uploadCh chan UploadedEpisode, podcastID string, episodeItem *podcast.Episode) {
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
			uploadCh <- UploadedEpisode{PodcastID: podcastID, Filename: episodeItem.Filename, Location: uploadInfo.Location}
			wg.Done()
		}(uploadCh, podcastID, episode)
	}

	go func() {
		wg.Wait()
		done <- true
	}()
Loop:
	for {
		select {
		case uploadedEpisode := <-uploadCh:
			log.Printf("%+v", uploadedEpisode)
			episode, err := p.Storage.GetEpisodeByFilename(uploadedEpisode.PodcastID, uploadedEpisode.Filename)
			if err != nil {
				log.Printf("[ERROR] can't get episode by filename %s - %s, %v", uploadedEpisode.PodcastID, uploadedEpisode.Filename, err)
			}
			episode.Status = podcast.Uploaded
			episode.Location = uploadedEpisode.Location
			if err = p.Storage.SaveEpisode(podcastID, episode); err != nil {
				log.Printf("[ERROR] can't change status episode %s, %v", episode.Filename, err)
			}
			log.Printf("[INFO] episode saved %+v", episode)
		case <-done:
			close(uploadCh)
			break Loop
		}
	}

}
