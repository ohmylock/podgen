package proc

import (
	"fmt"
	"os"

	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/podcast"
)

// Files for work with files of episodes
type Files struct {
}

func (f *Files) FindEpisodes(folderName string) ([]podcast.Episode, error) {
	var result []podcast.Episode

	entities, err := f.scanFolder(folderName)
	if err != nil {
		log.Fatalf("[ERROR] can't scan folder %s, %v", folderName, err)
		return nil, err
	}

	for _, entity := range entities {
		if entity.IsDir() {
			continue
		}

		entityInfo, err := entity.Info()
		if err != nil {
			log.Fatalf("[ERROR] can't get file info %s in %s, %v", entity.Name(), folderName, err)
			return nil, err
		}

		result = append(result, podcast.Episode{Filename: entity.Name(), Size: entityInfo.Size(), Status: podcast.New})
	}

	return result, nil
}

func (f *Files) scanFolder(folderName string) ([]os.DirEntry, error) {
	dir, err := os.ReadDir(fmt.Sprintf("storage/%s", folderName))
	if err != nil {
		return nil, err
	}

	return dir, nil
}
