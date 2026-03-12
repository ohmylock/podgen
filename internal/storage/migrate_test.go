package storage_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"podgen/internal/app/podgen/podcast"
	"podgen/internal/storage"
	boltstore "podgen/internal/storage/bolt"
	"podgen/internal/storage/sqlite"
)

func newBoltStore(t *testing.T) (storage.Store, func()) { //nolint:gocritic // named returns not needed for test helper
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "bolt.db")

	cfg := storage.Config{
		Type: storage.TypeBolt,
		DSN:  dbPath,
	}

	store := boltstore.New(cfg)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open bolt store: %v", err)
	}

	return store, func() { _ = store.Close() }
}

func newSQLiteStore(t *testing.T) (storage.Store, func()) { //nolint:gocritic // named returns not needed for test helper
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sqlite.db")

	cfg := storage.Config{
		Type:         storage.TypeSQLite,
		DSN:          dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}

	store := sqlite.New(cfg)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open sqlite store: %v", err)
	}

	return store, func() { _ = store.Close() }
}

func TestMigrateBoltToSQLite(t *testing.T) {
	// Setup source (BoltDB)
	src, srcCleanup := newBoltStore(t)
	defer srcCleanup()

	// Setup destination (SQLite)
	dst, dstCleanup := newSQLiteStore(t)
	defer dstCleanup()

	// Populate source with test data
	testData := map[string][]*podcast.Episode{
		"podcast1": {
			{Filename: "ep1.mp3", Status: podcast.New, Title: "Episode 1", Size: 100},
			{Filename: "ep2.mp3", Status: podcast.Uploaded, Title: "Episode 2", Size: 200},
		},
		"podcast2": {
			{Filename: "ep1.mp3", Status: podcast.New, Title: "Podcast 2 Episode 1", Size: 300},
			{Filename: "ep2.mp3", Status: podcast.Deleted, Title: "Podcast 2 Episode 2", Size: 400},
			{Filename: "ep3.mp3", Status: podcast.New, Title: "Podcast 2 Episode 3", Size: 500},
		},
	}

	for podcastID, episodes := range testData {
		for _, ep := range episodes {
			err := src.SaveEpisode(podcastID, ep)
			require.NoError(t, err)
		}
	}

	// Run migration
	stats, err := storage.Migrate(src, dst)
	require.NoError(t, err)

	// Verify stats
	assert.Equal(t, 2, stats.PodcastsProcessed)
	assert.Equal(t, 5, stats.EpisodesMigrated)
	assert.Equal(t, 0, stats.EpisodesFailed)

	// Verify data was migrated correctly
	podcasts, err := dst.ListPodcasts()
	require.NoError(t, err)
	assert.Len(t, podcasts, 2)

	for podcastID, expectedEpisodes := range testData {
		episodes, err := dst.ListEpisodes(podcastID)
		require.NoError(t, err)
		assert.Len(t, episodes, len(expectedEpisodes), "podcast %s should have %d episodes", podcastID, len(expectedEpisodes))

		// Verify each episode
		for _, expected := range expectedEpisodes {
			ep, err := dst.GetEpisodeByFilename(podcastID, expected.Filename)
			require.NoError(t, err)
			assert.Equal(t, expected.Title, ep.Title)
			assert.Equal(t, expected.Status, ep.Status)
			assert.Equal(t, expected.Size, ep.Size)
		}
	}
}

func TestMigrateSQLiteToBolt(t *testing.T) {
	// Setup source (SQLite)
	src, srcCleanup := newSQLiteStore(t)
	defer srcCleanup()

	// Setup destination (BoltDB)
	dst, dstCleanup := newBoltStore(t)
	defer dstCleanup()

	// Populate source with test data
	testData := map[string][]*podcast.Episode{
		"podcast-a": {
			{Filename: "track1.mp3", Status: podcast.New, Title: "Track 1", Artist: "Artist 1"},
			{Filename: "track2.mp3", Status: podcast.Uploaded, Title: "Track 2", Artist: "Artist 2"},
		},
		"podcast-b": {
			{Filename: "show1.mp3", Status: podcast.New, Title: "Show 1"},
		},
	}

	for podcastID, episodes := range testData {
		for _, ep := range episodes {
			err := src.SaveEpisode(podcastID, ep)
			require.NoError(t, err)
		}
	}

	// Run migration
	stats, err := storage.Migrate(src, dst)
	require.NoError(t, err)

	// Verify stats
	assert.Equal(t, 2, stats.PodcastsProcessed)
	assert.Equal(t, 3, stats.EpisodesMigrated)
	assert.Equal(t, 0, stats.EpisodesFailed)

	// Verify data was migrated correctly
	for podcastID, expectedEpisodes := range testData {
		episodes, err := dst.ListEpisodes(podcastID)
		require.NoError(t, err)
		assert.Len(t, episodes, len(expectedEpisodes))

		for _, expected := range expectedEpisodes {
			ep, err := dst.GetEpisodeByFilename(podcastID, expected.Filename)
			require.NoError(t, err)
			assert.Equal(t, expected.Title, ep.Title)
			assert.Equal(t, expected.Status, ep.Status)
			assert.Equal(t, expected.Artist, ep.Artist)
		}
	}
}

func TestMigrateEmptyStore(t *testing.T) {
	// Setup source (empty BoltDB)
	src, srcCleanup := newBoltStore(t)
	defer srcCleanup()

	// Setup destination (SQLite)
	dst, dstCleanup := newSQLiteStore(t)
	defer dstCleanup()

	// Run migration with empty source
	stats, err := storage.Migrate(src, dst)
	require.NoError(t, err)

	assert.Equal(t, 0, stats.PodcastsProcessed)
	assert.Equal(t, 0, stats.EpisodesMigrated)
	assert.Equal(t, 0, stats.EpisodesFailed)

	// Verify destination is still empty
	podcasts, err := dst.ListPodcasts()
	require.NoError(t, err)
	assert.Len(t, podcasts, 0)
}

func TestMigrateNilSource(t *testing.T) {
	dst, dstCleanup := newSQLiteStore(t)
	defer dstCleanup()

	_, err := storage.Migrate(nil, dst)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source store is nil")
}

func TestMigrateNilDestination(t *testing.T) {
	src, srcCleanup := newBoltStore(t)
	defer srcCleanup()

	_, err := storage.Migrate(src, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination store is nil")
}

func TestMigratePreservesAllFields(t *testing.T) {
	src, srcCleanup := newBoltStore(t)
	defer srcCleanup()

	dst, dstCleanup := newSQLiteStore(t)
	defer dstCleanup()

	// Create episode with all fields populated
	original := &podcast.Episode{
		Filename: "complete.mp3",
		PubDate:  "2024-03-01",
		Size:     12345678,
		Status:   podcast.Uploaded,
		Location: "https://example.com/complete.mp3",
		Session:  "session-123",
		Title:    "Complete Episode",
		Artist:   "Test Artist",
		Album:    "Test Album",
		Year:     "2024",
		Comment:  "Test comment with special chars: <>&\"'",
		Duration: "1:30:45",
	}

	err := src.SaveEpisode("test-podcast", original)
	require.NoError(t, err)

	// Run migration
	stats, err := storage.Migrate(src, dst)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.EpisodesMigrated)

	// Verify all fields
	migrated, err := dst.GetEpisodeByFilename("test-podcast", original.Filename)
	require.NoError(t, err)

	assert.Equal(t, original.Filename, migrated.Filename)
	assert.Equal(t, original.PubDate, migrated.PubDate)
	assert.Equal(t, original.Size, migrated.Size)
	assert.Equal(t, original.Status, migrated.Status)
	assert.Equal(t, original.Location, migrated.Location)
	assert.Equal(t, original.Session, migrated.Session)
	assert.Equal(t, original.Title, migrated.Title)
	assert.Equal(t, original.Artist, migrated.Artist)
	assert.Equal(t, original.Album, migrated.Album)
	assert.Equal(t, original.Year, migrated.Year)
	assert.Equal(t, original.Comment, migrated.Comment)
	assert.Equal(t, original.Duration, migrated.Duration)
}

func TestMigrateWithProgressCallback(t *testing.T) {
	src, srcCleanup := newBoltStore(t)
	defer srcCleanup()

	dst, dstCleanup := newSQLiteStore(t)
	defer dstCleanup()

	// Populate source
	podcasts := []string{"podcast-1", "podcast-2", "podcast-3"}
	for _, podcastID := range podcasts {
		err := src.SaveEpisode(podcastID, &podcast.Episode{Filename: "ep.mp3"})
		require.NoError(t, err)
	}

	// Track progress callbacks
	var progressCalls []struct {
		podcastID   string
		podcastNum  int
		total       int
		episodeCnt  int
	}

	callback := func(podcastID string, podcastNum, totalPodcasts, episodesMigrated int) {
		progressCalls = append(progressCalls, struct {
			podcastID  string
			podcastNum int
			total      int
			episodeCnt int
		}{podcastID, podcastNum, totalPodcasts, episodesMigrated})
	}

	// Run migration with progress
	stats, err := storage.MigrateWithProgressCallback(src, dst, callback)
	require.NoError(t, err)

	assert.Equal(t, 3, stats.PodcastsProcessed)
	assert.Len(t, progressCalls, 3)

	// Verify progress callback parameters
	for i, call := range progressCalls {
		assert.Equal(t, i+1, call.podcastNum)
		assert.Equal(t, 3, call.total)
		assert.Equal(t, 1, call.episodeCnt)
	}
}

func TestMigrateLargeDataset(t *testing.T) {
	src, srcCleanup := newBoltStore(t)
	defer srcCleanup()

	dst, dstCleanup := newSQLiteStore(t)
	defer dstCleanup()

	// Create larger dataset
	numPodcasts := 5
	episodesPerPodcast := 20

	for i := 0; i < numPodcasts; i++ {
		podcastID := "podcast-" + string(rune('a'+i))
		for j := 0; j < episodesPerPodcast; j++ {
			ep := &podcast.Episode{
				Filename: "episode-" + string(rune('0'+j/10)) + string(rune('0'+j%10)) + ".mp3",
				Status:   podcast.Status(j % 4),
				Title:    "Episode " + string(rune('0'+j/10)) + string(rune('0'+j%10)),
				Size:     int64(j * 1000),
			}
			err := src.SaveEpisode(podcastID, ep)
			require.NoError(t, err)
		}
	}

	// Run migration
	stats, err := storage.Migrate(src, dst)
	require.NoError(t, err)

	expectedTotal := numPodcasts * episodesPerPodcast
	assert.Equal(t, numPodcasts, stats.PodcastsProcessed)
	assert.Equal(t, expectedTotal, stats.EpisodesMigrated)
	assert.Equal(t, 0, stats.EpisodesFailed)

	// Verify counts
	for i := 0; i < numPodcasts; i++ {
		podcastID := "podcast-" + string(rune('a'+i))
		episodes, err := dst.ListEpisodes(podcastID)
		require.NoError(t, err)
		assert.Len(t, episodes, episodesPerPodcast)
	}
}
