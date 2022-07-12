package proc

import (
	"fmt"
	"os"
	"sort"

	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/podcast"
)

// Files for work with files of episodes
type Files struct {
}

// FindEpisodes in folder and come back like slice
func (f *Files) FindEpisodes(folderName string) ([]*podcast.Episode, error) {
	entities, err := f.scanFolder(folderName)
	if err != nil {
		log.Fatalf("[ERROR] can't scan folder %s, %v", folderName, err)
		return nil, err
	}
	var result = make([]*podcast.Episode, len(entities))
	for i, entity := range entities {
		if entity.IsDir() {
			continue
		}

		entityInfo, err := entity.Info()
		if err != nil {
			log.Fatalf("[ERROR] can't get file info %s in %s, %v", entity.Name(), folderName, err)
			return nil, err
		}

		result[i] = &podcast.Episode{Filename: entity.Name(), Size: entityInfo.Size(), Status: podcast.New}
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Filename < result[j].Filename
	})

	return result, nil
}

func (f *Files) scanFolder(folderName string) ([]os.DirEntry, error) {
	dir, err := os.ReadDir(fmt.Sprintf("storage/%s", folderName))
	if err != nil {
		return nil, err
	}

	return dir, nil
}
