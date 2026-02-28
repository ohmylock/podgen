package proc

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/podcast"
	"podgen/internal/configs"
)

// Processor is searcher of episode files and writer to store
type Processor struct {
	Storage     EpisodeStore
	Files       FileScanner
	S3Client    ObjectStorage
	StoragePath string
	ChunkSize   int
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
			return 0, fmt.Errorf("can't add episode %s to %s, %w", episode.Filename, podcastID, e)
		}
		countNew++
	}

	return countNew, nil
}

// DeleteOldEpisodesByPodcast from s3 storage
func (p *Processor) DeleteOldEpisodesByPodcast(podcastID, podcastFolder string) error {
	episodes, err := p.Storage.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	if err != nil {
		return fmt.Errorf("can't find episodes %s, %w", podcastID, err)
	}

	ctx := context.Background()

	// parallel S3 deletes - track which episodes were successfully deleted
	type deleteResult struct {
		filename string
		ok       bool
	}
	results := make([]deleteResult, len(episodes))

	tasks := make([]func(ctx context.Context) error, len(episodes))
	for i, episode := range episodes {
		results[i].filename = episode.Filename
		tasks[i] = func(ctx context.Context) error {
			log.Printf("[INFO] Started delete episode %s - %s", podcastID, episode.Filename)
			if err := p.S3Client.DeleteEpisode(ctx, fmt.Sprintf("%s/%s", podcastFolder, episode.Filename)); err != nil {
				return err
			}
			log.Printf("[INFO] Episode deleted %s - %s", episode.Filename, podcastID)
			return nil
		}
	}

	errs := RunParallel(ctx, p.ChunkSize, tasks)

	// mark successful results
	for i, err := range errs {
		if err != nil {
			log.Printf("[ERROR] can't delete episode %s, %v", results[i].filename, err)
			continue
		}
		results[i].ok = true
	}

	// sequential DB updates for successfully deleted episodes
	for _, r := range results {
		if !r.ok {
			continue
		}
		episode, err := p.Storage.GetEpisodeByFilename(podcastID, r.filename)
		if err != nil {
			log.Printf("[ERROR] can't get episode by filename %s - %s, %v", podcastID, r.filename, err)
			continue
		}
		episode.Status = podcast.Deleted
		if err = p.Storage.SaveEpisode(podcastID, episode); err != nil {
			log.Printf("[ERROR] can't change status episode %s, %v", episode.Filename, err)
		}
	}
	return nil
}

// RollbackLastEpisodes last deleted episode
func (p *Processor) RollbackLastEpisodes(podcastID string) error {
	episode, err := p.Storage.GetLastEpisodeByNotStatus(podcastID, podcast.New)
	if err != nil {
		log.Printf("[ERROR] can't find episodes %s, %v", podcastID, err)
		return err
	}

	if episode == nil {
		log.Printf("[INFO] Episode for rollback not found %s", podcastID)
		return nil
	}

	if episode.Session != "" {
		return p.RollbackEpisodesOfSession(podcastID, episode.Session)
	}

	episode.Status = podcast.New
	if err = p.Storage.SaveEpisode(podcastID, episode); err != nil {
		log.Printf("[ERROR] can't change status episode %s, %v", episode.Filename, err)
		return err
	}

	return nil
}

// RollbackEpisodesOfSession last deleted episode of session
func (p *Processor) RollbackEpisodesOfSession(podcastID, session string) error {
	episodes, err := p.Storage.FindEpisodesBySession(podcastID, session)
	if err != nil {
		log.Printf("[ERROR] can't find episodes %s, %v", podcastID, err)
		return err
	}

	if len(episodes) == 0 {
		log.Printf("[INFO] Episodes for rollback not found %s", podcastID)
		return nil
	}

	log.Printf("[INFO] Started rollback episodes %s", podcastID)

	for _, episode := range episodes {
		episode.Status = podcast.New
		if err = p.Storage.SaveEpisode(podcastID, episode); err != nil {
			log.Printf("[ERROR] can't change status episode %s, %v", episode.Filename, err)
			return err
		}

		log.Printf("[INFO] Episode rollback %s - %s", episode.Filename, podcastID)
	}

	return nil
}

// UploadPodcastImage to s3 storage
func (p *Processor) UploadPodcastImage(podcastID, podcastFolder, podcastImageFilename string) (string, error) {
	log.Printf("[INFO] Started upload podcast image %s - %s", podcastID, podcastImageFilename)
	ctx := context.Background()

	podcastImagePath := fmt.Sprintf("%s/%s/%s", p.StoragePath, podcastFolder, podcastImageFilename)
	if !CheckFileExists(podcastImagePath) {
		podcastImagePath = fmt.Sprintf("%s/%s", p.StoragePath, podcastImageFilename)
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

	imageInfo, err := p.S3Client.GetObjectInfo(ctx, fmt.Sprintf("%s/%s", podcastFolder, podcastImageFilename))

	if err != nil {
		log.Printf("[ERROR] can't image info %s, %v", podcastImageFilename, err)
		return ""
	}
	return imageInfo.Location
}

// UploadNewEpisodes get new episodes by total limit of size and upload to s3 storage
func (p *Processor) UploadNewEpisodes(session, podcastID, podcastFolder string, sizeLimit int64) error {
	episodes, err := p.Storage.FindEpisodesBySizeLimit(podcastID, podcast.New, sizeLimit)
	if err != nil {
		return fmt.Errorf("can't find episodes %s, %w", podcastID, err)
	}

	ctx := context.Background()

	// process in chunks - parallel S3 uploads, then sequential DB updates
	for i := 0; i < len(episodes); i += p.ChunkSize {
		end := i + p.ChunkSize
		if end > len(episodes) {
			end = len(episodes)
		}
		chunk := episodes[i:end]

		// parallel S3 uploads
		uploadResults := make([]UploadedEpisode, len(chunk))
		tasks := make([]func(ctx context.Context) error, len(chunk))
		for j, episode := range chunk {
			tasks[j] = func(ctx context.Context) error {
				result, err := p.uploadSingleEpisode(ctx, podcastID, podcastFolder, episode)
				if err != nil {
					return err
				}
				uploadResults[j] = result
				return nil
			}
		}

		errs := RunParallel(ctx, p.ChunkSize, tasks)

		// sequential DB updates for successful uploads
		for j, err := range errs {
			if err != nil {
				log.Printf("[ERROR] can't upload episode %s, %v", chunk[j].Filename, err)
				continue
			}
			uploaded := uploadResults[j]
			episode, err := p.Storage.GetEpisodeByFilename(uploaded.PodcastID, uploaded.Filename)
			if err != nil {
				return fmt.Errorf("can't get episode by filename %s, %w", uploaded.Filename, err)
			}
			episode.Session = session
			episode.Status = podcast.Uploaded
			episode.Location = uploaded.Location
			if err = p.Storage.SaveEpisode(podcastID, episode); err != nil {
				return fmt.Errorf("can't save episode %s, %w", episode.Filename, err)
			}
		}
	}
	return nil
}

// GenerateFeed to podcast
func (p *Processor) GenerateFeed(podcastID string, podcastEntity configs.Podcast, podcastImageURL string) (string, error) {
	episodes, err := p.Storage.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	if err != nil {
		return "", fmt.Errorf("can't find episodes %s, %w", podcastID, err)
	}

	var header, body, footer string

	info := map[string]string{
		"author":   "PodGen",
		"email":    "podgen@localhost.com",
		"owner":    "PodGen",
		"category": "History",
		"language": "EN",
	}

	if podcastEntity.Info.Author != "" {
		info["author"] = podcastEntity.Info.Author
	}

	if podcastEntity.Info.Email != "" {
		info["email"] = podcastEntity.Info.Email
	}

	if podcastEntity.Info.Owner != "" {
		info["owner"] = podcastEntity.Info.Owner
	}

	if podcastEntity.Info.Category != "" {
		info["category"] = podcastEntity.Info.Category
	}

	if podcastEntity.Info.Language != "" {
		info["language"] = podcastEntity.Info.Language
	}

	header = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n" +
		"<rss xmlns:itunes=\"http://www.itunes.com/dtds/podcast-1.0.dtd\" " +
		"xmlns:dc=\"http://purl.org/dc/elements/1.1/\" xmlns:atom=\"http://www.w3.org/2005/Atom\" " +
		"xmlns:googleplay=\"http://www.google.com/schemas/play-podcasts/1.0\" version=\"2.0\">\n" +
		"<channel>\n" +
		fmt.Sprintf("<title>%s</title>\n<description><![CDATA[%s]]></description>\n", podcastEntity.Title, podcastEntity.Title) +
		"<generator>PodGen</generator>\n" +
		fmt.Sprintf("<language>%s</language>\n", info["language"]) +
		"<itunes:explicit>No</itunes:explicit>\n" +
		fmt.Sprintf("<itunes:subtitle>%s</itunes:subtitle>\n<itunes:summary><![CDATA[%s]]></itunes:summary>\n", podcastEntity.Title, podcastEntity.Title) +
		fmt.Sprintf("<itunes:author>%s</itunes:author>\n", info["author"]) +
		fmt.Sprintf("<author>%s</author>\n", info["author"]) +
		"<image>\n" +
		fmt.Sprintf("<url>%s</url>\n", podcastImageURL) +
		"</image>\n" +
		fmt.Sprintf("<itunes:image href=%q />\n", podcastImageURL) +
		"<itunes:owner>\n" +
		fmt.Sprintf("<itunes:name>%s</itunes:name>\n", info["owner"]) +
		fmt.Sprintf("<itunes:email>%s</itunes:email>\n", info["email"]) +
		"</itunes:owner>\n" +
		fmt.Sprintf("<itunes:category text=%q />\n", info["category"])

	footer = "</channel>\n</rss>"
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
		return "", fmt.Errorf("can't generate feed key for %s, %w", podcastID, err)
	}
	feedFilename := fmt.Sprintf("%s.rss", feedKey)
	feedPath := fmt.Sprintf("%s/%s/%s", p.StoragePath, podcastEntity.Folder, feedFilename)
	f, err := os.Create(feedPath) // nolint
	if err != nil {
		return "", fmt.Errorf("can't create file %s, %w", feedPath, err)
	}
	defer func(f *os.File) {
		if err = f.Close(); err != nil {
			log.Printf("[ERROR] can't close file %s, %v", feedPath, err)
		}
	}(f)

	if _, err = f.WriteString(fmt.Sprintf("%s\n%s\n%s", header, body, footer)); err != nil {
		return "", fmt.Errorf("[ERROR] can't write to file %s, %v", feedPath, err)
	}

	return feedFilename, nil
}

// UploadFeed of podcast to s3 storage
func (p *Processor) UploadFeed(podcastFolder, feedName string) *UploadResult {
	uploadInfo, err := p.S3Client.UploadFeed(context.Background(),
		fmt.Sprintf("%s/%s", podcastFolder, feedName),
		fmt.Sprintf("%s/%s/%s", p.StoragePath, podcastFolder, feedName))

	if err != nil {
		log.Printf("[ERROR] can't upload feed %s, %v", feedName, err)
		return nil
	}

	return uploadInfo
}

func (p *Processor) getFeedKey(podcastID string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(podcastID)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (p *Processor) uploadSingleEpisode(ctx context.Context, podcastID, podcastFolder string, episodeItem *podcast.Episode) (UploadedEpisode, error) {
	log.Printf("[INFO] Started upload episode %s - %s", podcastID, episodeItem.Filename)
	objectInfo, _ := p.S3Client.GetObjectInfo(ctx, fmt.Sprintf("%s/%s", podcastFolder, episodeItem.Filename))
	var location string
	if objectInfo != nil && episodeItem.Size == objectInfo.Size {
		location = objectInfo.Location
	}

	if location == "" {
		uploadInfo, err := p.S3Client.UploadEpisode(ctx,
			fmt.Sprintf("%s/%s", podcastFolder, episodeItem.Filename),
			fmt.Sprintf("%s/%s/%s", p.StoragePath, podcastFolder, episodeItem.Filename))
		if err != nil {
			return UploadedEpisode{}, err
		}
		location = uploadInfo.Location
	}

	log.Printf("[INFO] Episode uploaded %s - %s", episodeItem.Filename, location)
	return UploadedEpisode{
		PodcastID: podcastID,
		Filename:  episodeItem.Filename,
		Location:  location,
	}, nil
}
