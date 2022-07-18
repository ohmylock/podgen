package proc

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"sync"

	log "github.com/go-pkgz/lgr"
	"github.com/minio/minio-go/v7"
	"podgen/internal/app/podgen/podcast"
)

// Processor is searcher of episode files and writer to store
type Processor struct {
	Storage   *BoltDB
	Files     *Files
	S3Client  *S3Store
	ChunkSize int
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
		if episode == nil {
			continue
		}
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
			defer wg.Done()
			log.Printf("[INFO] Started upload episode %s - %s", podcastID, episodeItem.Filename)
			err := p.S3Client.DeleteEpisode(ctx, fmt.Sprintf("%s/%s", podcastFolder, episodeItem.Filename))
			if err != nil {
				log.Printf("[ERROR] can't delete episode %s, %v", episodeItem.Filename, err)
				return
			}

			log.Printf("[INFO] Episode deleted %s - %s", episodeItem.Filename, podcastID)
			deleteCh <- DeletedEpisode{PodcastID: podcastID, Filename: episodeItem.Filename}
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
			episode, err := p.Storage.GetEpisodeByFilename(deletedEpisode.PodcastID, deletedEpisode.Filename)
			if err != nil {
				log.Printf("[ERROR] can't get episode by filename %s - %s, %v", deletedEpisode.PodcastID, deletedEpisode.Filename, err)
			}
			episode.Status = podcast.Deleted
			if err = p.Storage.SaveEpisode(podcastID, episode); err != nil {
				log.Printf("[ERROR] can't change status episode %s, %v", episode.Filename, err)
			}
		case <-done:
			close(deleteCh)
			close(done)
			break Loop
		}
	}
	return nil
}

// UploadPodcastImage to s3 storage
func (p *Processor) UploadPodcastImage(podcastID, podcastFolder, podcastImageFilename string) (string, error) {
	log.Printf("[INFO] Started upload podcast image %s - %s", podcastID, podcastImageFilename)
	ctx := context.Background()

	podcastImagePath := fmt.Sprintf("%s/%s/%s", p.Files.Storage, podcastFolder, podcastImageFilename)
	if !CheckFileExists(podcastImagePath) {
		podcastImagePath = fmt.Sprintf("%s/%s", p.Files.Storage, podcastImageFilename)
	}

	if !CheckFileExists(podcastImagePath) {
		return "", errors.New("podcast image not found")
	}

	uploadInfo, err := p.S3Client.UploadImage(ctx,
		fmt.Sprintf("%s/%s", podcastFolder, podcastImageFilename),
		podcastImagePath)

	if err != nil {
		return "", fmt.Errorf("[ERROR] can't upload image %s, %v", podcastImageFilename, err)
	}

	log.Printf("[INFO] Image of podcast uploaded %s - %s", podcastImageFilename, uploadInfo.Location)
	return uploadInfo.Location, nil
}

// GetPodcastImage from s3 storage
func (p *Processor) GetPodcastImage(podcastFolder, podcastImageFilename string) string {
	ctx := context.Background()
	imageInfo, err := p.S3Client.GetObjectInfo(ctx,
		fmt.Sprintf("%s/%s", podcastFolder, podcastImageFilename))
	if err != nil {
		log.Printf("[ERROR] can't image info %s, %v", podcastImageFilename, err)
		return ""
	}
	return imageInfo.Key
}

// UploadNewEpisodes get new episodes by total limit of size and upload to s3 storage
func (p *Processor) UploadNewEpisodes(podcastID, podcastFolder string, sizeLimit int64) {
	episodes, err := p.Storage.FindEpisodesBySizeLimit(podcastID, podcast.New, sizeLimit)
	if err != nil {
		log.Fatalf("[ERROR] can't find episodes %s, %v", podcastID, err)
	}

	wg := sync.WaitGroup{}
	ctx := context.Background()

	for i := 0; i < len(episodes); i += p.ChunkSize {
		uploadCh := make(chan UploadedEpisode)
		done := make(chan bool)
		end := i + p.ChunkSize

		if end > len(episodes) {
			end = len(episodes)
		}

		for _, episode := range episodes[i:end] {
			wg.Add(1)
			go p.uploadProcess(ctx, &wg, uploadCh, podcastID, podcastFolder, episode)
		}

		go func() {
			wg.Wait()
			done <- true
		}()

	Loop:
		for {
			select {
			case uploadedEpisode := <-uploadCh:
				episode, err := p.Storage.GetEpisodeByFilename(uploadedEpisode.PodcastID, uploadedEpisode.Filename)
				if err != nil {
					log.Printf("[ERROR] can't get episode by filename %s - %s, %v", uploadedEpisode.PodcastID, uploadedEpisode.Filename, err)
				}
				episode.Status = podcast.Uploaded
				episode.Location = uploadedEpisode.Location
				if err = p.Storage.SaveEpisode(podcastID, episode); err != nil {
					log.Printf("[ERROR] can't change status episode %s, %v", episode.Filename, err)
				}
			case <-done:
				close(uploadCh)
				close(done)
				break Loop
			}
		}
	}

}

// GenerateFeed to podcast
func (p *Processor) GenerateFeed(podcastID, podcastTitle, podcastFolder, podcastImageURL string) (string, error) {
	episodes, err := p.Storage.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	if err != nil {
		log.Fatalf("[ERROR] can't find episodes %s, %v", podcastID, err)
	}

	var header, body, footer string

	header = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n" +
		"<rss xmlns:itunes=\"http://www.itunes.com/dtds/podcast-1.0.dtd\" " +
		"xmlns:dc=\"http://purl.org/dc/elements/1.1/\" xmlns:atom=\"http://www.w3.org/2005/Atom\" " +
		"xmlns:googleplay=\"http://www.google.com/schemas/play-podcasts/1.0\" version=\"2.0\">\n" +
		"<channel>\n" +
		fmt.Sprintf("<title>%s</title>\n<description><![CDATA[%s]]></description>\n", podcastTitle, podcastTitle) +
		"<generator>PodGen</generator>\n<language>EN</language>\n" +
		"<itunes:explicit>No</itunes:explicit>\n" +
		fmt.Sprintf("<itunes:subtitle>%s</itunes:subtitle>\n<itunes:summary><![CDATA[%s]]></itunes:summary>\n", podcastTitle, podcastTitle) +
		"<itunes:author>PodGen</itunes:author>\n" +
		"<author>PodGen</author>\n" +
		"<image>\n" +
		fmt.Sprintf("<url>%s</url>\n", podcastImageURL) +
		"</image>\n" +
		fmt.Sprintf("<itunes:image href=%q />\n", podcastImageURL) +
		"<itunes:owner>\n" +
		"<itunes:name>PodGen</itunes:name>\n" +
		"<itunes:email>podgen@localhost.com</itunes:email>\n" +
		"</itunes:owner>\n" +
		"<itunes:category text=\"Technology\"/>"

	footer = "\n</channel>\n</rss>"
	for _, episode := range episodes {
		body += "<item>\n" +
			fmt.Sprintf("<title>%s</title>\n", episode.Filename) +
			fmt.Sprintf("<description><![CDATA[%s]]></description>\n", episode.Filename) +
			fmt.Sprintf("<itunes:summary><![CDATA[%s]]></itunes:summary>\n", episode.Filename) +
			fmt.Sprintf("<pubDate>%s</pubDate>\n", episode.PubDate) +
			fmt.Sprintf("<itunes:image href=%q />\n", podcastImageURL) +
			fmt.Sprintf("<enclosure url=%q type=\"audio/mp3\" length=\"%d\" />\n", episode.Location, episode.Size) +
			fmt.Sprintf("<media:content url=%q fileSize=\"%d\" type=\"audio/mp3\" />\n", episode.Location, episode.Size) +
			"<itunes:explicit>No</itunes:explicit>\n" +
			"</item>\n"
	}

	feedKey, err := p.getFeedKey(podcastID)
	if err != nil {
		log.Fatalf("[ERROR] can't generate feed key for %s, %v", podcastID, err)
	}
	feedFilename := fmt.Sprintf("%s.rss", feedKey)
	feedPath := fmt.Sprintf("%s/%s/%s", p.Files.Storage, podcastFolder, feedFilename)
	f, err := os.Create(feedPath) // nolint
	if err != nil {
		log.Fatalf("[ERROR] can't create file %s, %v", feedPath, err)
	}
	defer func(f *os.File) {
		if err = f.Close(); err != nil {
			log.Fatalf("[ERROR] can't close file %s, %v", feedPath, err)
		}
	}(f)

	if _, err = f.WriteString(fmt.Sprintf("%s\n%s\n%s", header, body, footer)); err != nil {
		return "", fmt.Errorf("[ERROR] can't write to file %s, %v", feedPath, err)
	}

	return feedFilename, nil
}

// UploadFeed of podcast to s3 storage
func (p *Processor) UploadFeed(podcastFolder, feedName string) *minio.UploadInfo {
	uploadInfo, err := p.S3Client.UploadFeed(context.Background(),
		fmt.Sprintf("%s/%s", podcastFolder, feedName),
		fmt.Sprintf("%s/%s/%s", p.Files.Storage, podcastFolder, feedName))

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

func (p *Processor) uploadProcess(ctx context.Context, wg *sync.WaitGroup, uploadCh chan UploadedEpisode, podcastID, podcastFolder string, episodeItem *podcast.Episode) {
	defer wg.Done()
	log.Printf("[INFO] Started upload episode %s - %s", podcastID, episodeItem.Filename)
	objectInfo, _ := p.S3Client.GetObjectInfo(ctx, fmt.Sprintf("%s/%s", podcastFolder, episodeItem.Filename))
	var location string
	if objectInfo != nil && episodeItem.Size == objectInfo.Size {
		location = objectInfo.Location
	}

	if location == "" {
		uploadInfo, err := p.S3Client.UploadEpisode(ctx,
			fmt.Sprintf("%s/%s", podcastFolder, episodeItem.Filename),
			fmt.Sprintf("%s/%s/%s", p.Files.Storage, podcastFolder, episodeItem.Filename))
		if err != nil {
			log.Printf("[ERROR] can't upload episode %s, %v", episodeItem.Filename, err)
			return
		}
		location = uploadInfo.Location
	}

	log.Printf("[INFO] Episode uploaded %s - %s", episodeItem.Filename, location)
	uploadCh <- UploadedEpisode{
		PodcastID: podcastID,
		Filename:  episodeItem.Filename,
		Location:  location,
	}
}
