package proc

import (
	log "github.com/go-pkgz/lgr"
)

// Processor
type Processor struct {
	Storage *BoltDB
	Files   *Files
}

// Update
func (p *Processor) Update(folderName, podcastName string) (int64, error) {
	var countNew int64
	episodes, err := p.Files.FindEpisodes(folderName)
	if err != nil {
		log.Fatalf("[ERROR] can't scan folder %s, %v", folderName, err)
		return 0, err
	}

	for _, episode := range episodes {
		created, err := p.Storage.SaveEpisode(podcastName, episode)
		if err != nil {
			log.Fatalf("[ERROR] can't add episode %s to %s, %v", episode.Filename, podcastName, err)
		}
		if created {
			countNew++
		}

	}

	return countNew, nil
}
