package podgen

import (
	"sync"

	log "github.com/go-pkgz/lgr"
	"podgen/internal/app/podgen/proc"
	"podgen/internal/configs"
)

type App struct {
	config    *configs.Conf
	processor *proc.Processor
}

// NewApplication Создание нового приложения
func NewApplication(conf *configs.Conf, p *proc.Processor) (*App, error) {
	app := App{config: conf, processor: p}
	return &app, nil
}

func (a *App) FindPodcasts() map[string]configs.Podcast {
	return a.config.Podcasts
}

func (a *App) updateFolder(folderName string, podcastName string) (int64, error) {
	countNew, err := a.processor.Update(folderName, podcastName)
	if err != nil {
		return 0, err
	}

	return countNew, nil
}

// Update find and add to db new episodes of podcast
func (a *App) Update() {
	podcasts := a.FindPodcasts()

	wg := sync.WaitGroup{}
	for _, p := range podcasts {
		wg.Add(1)
		go func(p configs.Podcast) {
			countNew, err := a.updateFolder(p.Folder, p.Title)
			if err != nil {
				wg.Done()
				return
			}
			if countNew > 0 {
				log.Printf("[INFO] found new %d episodes for %s", countNew, p.Title)
			}

			wg.Done()
		}(p)
	}
	wg.Wait()
}
