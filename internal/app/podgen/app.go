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

// Update find and add to db new episodes of podcast
func (a *App) Update() {
	podcasts := a.findPodcasts()

	wg := sync.WaitGroup{}
	for i, p := range podcasts {
		wg.Add(1)
		go func(i string, p configs.Podcast) {
			countNew, err := a.updateFolder(p.Folder, i)
			if err != nil {
				wg.Done()
				return
			}
			if countNew > 0 {
				log.Printf("[INFO] found new %d episodes for %s", countNew, p.Title)
			}

			wg.Done()
		}(i, p)
	}
	wg.Wait()
}

func (a *App) Upload() {
	podcasts := a.findPodcasts()

	wg := sync.WaitGroup{}
	for _, p := range podcasts {
		wg.Add(1)
		go func(p configs.Podcast) {
			a.deleteOldEpisodes(p)
			// if err != nil {
			// 	wg.Done()
			// 	return
			// }
			// if countNew > 0 {
			// 	log.Printf("[INFO] found new %d episodes for %s", countNew, p.Title)
			// }
			//
			wg.Done()
		}(p)
	}
	wg.Wait()
}

func (a *App) findPodcasts() map[string]configs.Podcast {
	return a.config.Podcasts
}

func (a *App) updateFolder(folderName string, podcastID string) (int64, error) {
	countNew, err := a.processor.Update(folderName, podcastID)
	if err != nil {
		return 0, err
	}

	return countNew, nil
}

func (a *App) deleteOldEpisodes(p configs.Podcast) {
	a.processor.DeleteOldEpisodes(p.Folder)
}
