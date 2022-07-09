package proc

import (
	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/podcast"
)

// Processor is searcher of episode files and writer to store
type Processor struct {
	Storage *BoltDB
	Files   *Files
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
		created, err := p.Storage.SaveEpisode(podcastID, episode)
		if err != nil {
			log.Fatalf("[ERROR] can't add episode %s to %s, %v", episode.Filename, podcastID, err)
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
