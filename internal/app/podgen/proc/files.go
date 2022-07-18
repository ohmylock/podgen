package proc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/podcast"
)

// Files for work with files of episodes
type Files struct {
	Storage string
}

// FindEpisodes in folder and come back like slice
func (f *Files) FindEpisodes(folderName string) ([]*podcast.Episode, error) {
	entities, err := f.scanFolder(folderName)
	if err != nil {
		log.Fatalf("[ERROR] can't scan folder %s, %v", folderName, err)
		return nil, err
	}
	var re = regexp.MustCompile(`(?m)([12]\d{3}-(0[1-9]|1[012])-(0[1-9]|[12]\d|3[01]))`)
	var result = make([]*podcast.Episode, len(entities))
	for i, entity := range entities {
		if entity.IsDir() {
			continue
		}
		extension := filepath.Ext(entity.Name())
		if extension != ".mp3" {
			continue
		}

		entityInfo, err := entity.Info()
		if err != nil {
			log.Fatalf("[ERROR] can't get file info %s in %s, %v", entity.Name(), folderName, err)
			return nil, err
		}

		pubDate := time.Now()

		matches := re.FindAllString(entity.Name(), -1)
		if matches != nil {
			match := matches[0]
			formatDate := "2006-01-02"
			pubDate, err = time.Parse(formatDate, match)
			if err != nil {
				formatDate2 := "2006-01-2"
				pubDate, err = time.Parse(formatDate2, match)
				if err != nil {
					log.Printf("[WARN] %s, %v", match, err)
				}
			}
		}

		result[i] = &podcast.Episode{
			Filename: entity.Name(),
			Size:     entityInfo.Size(),
			Status:   podcast.New,
			PubDate:  pubDate.Format(time.RFC1123Z),
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i] == nil || result[j] == nil {
			return false
		}
		return result[i].Filename < result[j].Filename
	})

	return result, nil
}

// CheckFileExists in file store
func CheckFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func (f *Files) scanFolder(folderName string) ([]os.DirEntry, error) {
	dir, err := os.ReadDir(fmt.Sprintf("%s/%s", f.Storage, folderName))
	if err != nil {
		return nil, err
	}

	return dir, nil
}
