// Package storage provides migration utilities for transferring data between storage backends.
package storage

import (
	"fmt"

	log "github.com/go-pkgz/lgr"
)

// MigrateStats holds statistics about a migration operation.
type MigrateStats struct {
	PodcastsProcessed int
	EpisodesMigrated  int
	EpisodesFailed    int
}

// Migrate transfers all data from source store to destination store.
// Both stores must be opened before calling this function.
// Returns statistics about the migration and any error encountered.
func Migrate(from, to Store) (*MigrateStats, error) {
	if from == nil {
		return nil, fmt.Errorf("source store is nil")
	}
	if to == nil {
		return nil, fmt.Errorf("destination store is nil")
	}

	stats := &MigrateStats{}

	// Get all podcasts from source
	podcasts, err := from.ListPodcasts()
	if err != nil {
		return stats, fmt.Errorf("failed to list podcasts from source: %w", err)
	}

	log.Printf("[INFO] Migration started: found %d podcasts to migrate", len(podcasts))

	// Migrate each podcast
	for _, podcastID := range podcasts {
		episodeCount, failedCount, err := migratePodcast(from, to, podcastID)
		if err != nil {
			log.Printf("[ERROR] Failed to migrate podcast %s: %v", podcastID, err)
			stats.EpisodesFailed += failedCount
			continue
		}

		stats.PodcastsProcessed++
		stats.EpisodesMigrated += episodeCount
		stats.EpisodesFailed += failedCount

		log.Printf("[INFO] Migrated podcast %s: %d episodes", podcastID, episodeCount)
	}

	log.Printf("[INFO] Migration completed: %d podcasts, %d episodes migrated, %d failed",
		stats.PodcastsProcessed, stats.EpisodesMigrated, stats.EpisodesFailed)

	return stats, nil
}

// migratePodcast migrates all episodes for a single podcast.
// Returns the number of episodes migrated, failed count, and any error.
func migratePodcast(from, to Store, podcastID string) (migrated, failed int, err error) {
	episodes, listErr := from.ListEpisodes(podcastID)
	if listErr != nil {
		return 0, 0, fmt.Errorf("failed to list episodes: %w", listErr)
	}

	for _, episode := range episodes {
		if err := to.SaveEpisode(podcastID, episode); err != nil {
			log.Printf("[WARN] Failed to migrate episode %s/%s: %v", podcastID, episode.Filename, err)
			failed++
			continue
		}
		migrated++
	}

	return migrated, failed, nil
}

// MigrateWithProgress transfers data with progress callback.
// The callback is called after each podcast is processed.
type MigrateProgressCallback func(podcastID string, podcastNum, totalPodcasts int, episodesMigrated int)

// MigrateWithProgressCallback transfers all data with progress reporting.
func MigrateWithProgressCallback(from, to Store, callback MigrateProgressCallback) (*MigrateStats, error) {
	if from == nil {
		return nil, fmt.Errorf("source store is nil")
	}
	if to == nil {
		return nil, fmt.Errorf("destination store is nil")
	}

	stats := &MigrateStats{}

	// Get all podcasts from source
	podcasts, err := from.ListPodcasts()
	if err != nil {
		return stats, fmt.Errorf("failed to list podcasts from source: %w", err)
	}

	totalPodcasts := len(podcasts)
	log.Printf("[INFO] Migration started: found %d podcasts to migrate", totalPodcasts)

	// Migrate each podcast
	for i, podcastID := range podcasts {
		episodeCount, failedCount, err := migratePodcast(from, to, podcastID)
		if err != nil {
			log.Printf("[ERROR] Failed to migrate podcast %s: %v", podcastID, err)
			stats.EpisodesFailed += failedCount
			continue
		}

		stats.PodcastsProcessed++
		stats.EpisodesMigrated += episodeCount
		stats.EpisodesFailed += failedCount

		if callback != nil {
			callback(podcastID, i+1, totalPodcasts, episodeCount)
		}

		log.Printf("[INFO] Migrated podcast %s: %d episodes (%d/%d)", podcastID, episodeCount, i+1, totalPodcasts)
	}

	log.Printf("[INFO] Migration completed: %d podcasts, %d episodes migrated, %d failed",
		stats.PodcastsProcessed, stats.EpisodesMigrated, stats.EpisodesFailed)

	return stats, nil
}
