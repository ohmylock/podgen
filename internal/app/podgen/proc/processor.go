package proc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sync"

	log "github.com/go-pkgz/lgr"
	"github.com/minio/minio-go/v7"
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

// GenerateFeed to podcast
func (p *Processor) GenerateFeed(podcastID, podcastTitle, podcastFolder string) (string, error) {
	episodes, err := p.Storage.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	if err != nil {
		log.Fatalf("[ERROR] can't find episodes %s, %v", podcastID, err)
	}

	var header, body, footer string

	header = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n" +
		"<rss xmlns:itunes=\"http://www.itunes.com/dtds/podcast-1.0.dtd\" version=\"2.0\">\n" +
		"<channel>\n" +
		fmt.Sprintf("<title>%s</title>\n<description></description>\n", podcastTitle)

	footer = "\n</channel>\n</rss>"
	for _, episode := range episodes {
		body += "<item>\n" +
			fmt.Sprintf("<title>%s</title>\n", episode.Filename) +
			fmt.Sprintf("<description><![CDATA[%s]]></description>\n", episode.Filename) +
			fmt.Sprintf("<enclosure url=%q type=\"audio/mp3\" length=\"%d\" />\n", episode.Location, episode.Size) +
			"</item>\n"
	}

	feedKey, err := p.getFeedKey(podcastID)
	if err != nil {
		log.Fatalf("[ERROR] can't generate feed key for %s, %v", podcastID, err)
	}
	feedFilename := fmt.Sprintf("%s.xml", feedKey)
	feedPath := fmt.Sprintf("storage/%s/%s", podcastFolder, feedFilename)
	f, err := os.Create(feedPath) // nolint
	if err != nil {
		log.Fatalf("[ERROR] can't create file %s, %v", feedPath, err)
	}
	defer func(f *os.File) {
		if err = f.Close(); err != nil {
			log.Fatalf("[ERROR] can't close file %s, %v", feedPath, err)
		}
	}(f)

	if _, err = f.WriteString(header + body + footer); err != nil {
		return "", fmt.Errorf("[ERROR] can't write to file %s, %v", feedPath, err)
	}

	return feedFilename, nil
}

// UploadFeed of podcast to s3 storage
func (p *Processor) UploadFeed(podcastFolder, feedName string) *minio.UploadInfo {
	uploadInfo, err := p.S3Client.UploadFeed(context.Background(),
		fmt.Sprintf("%s/%s", podcastFolder, feedName),
		fmt.Sprintf("storage/%s/%s", podcastFolder, feedName))

	if err != nil {
		log.Printf("[ERROR] can't upload feed %s, %v", feedName, err)
		return nil
	}

	return uploadInfo
}

func (p *Processor) getFeedKey(podcastID string) (string, error) {
	key, err := func() (string, error) {
		h := sha256.New()
		if _, err := h.Write([]byte(podcastID)); err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", h.Sum(nil)), nil
	}()

	return key, err
}
