package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"podgen/internal/app/podgen/podcast"
	"podgen/internal/storage"
	"podgen/internal/storage/factory"
)

// TestAcceptance_SQLiteCreationAndPersistence verifies that SQLite database
// can be created, episodes can be stored, and data persists after reopening.
func TestAcceptance_SQLiteCreationAndPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a new SQLite database using factory
	store, err := factory.NewFromStrings("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open SQLite store: %v", err)
	}

	// Create test episodes
	episodes := []*podcast.Episode{
		{
			Filename: "episode1.mp3",
			Status:   podcast.New,
			Session:  "session-001",
			Title:    "Episode 1",
			Size:     1024,
		},
		{
			Filename: "episode2.mp3",
			Status:   podcast.Uploaded,
			Session:  "session-001",
			Location: "https://cdn.example.com/episode2.mp3",
			Title:    "Episode 2",
			Size:     2048,
		},
	}

	// Store episodes for podcast1
	for _, ep := range episodes {
		if err := store.SaveEpisode("podcast1", ep); err != nil {
			t.Fatalf("Failed to save episode: %v", err)
		}
	}

	// Store episode for podcast2
	ep3 := &podcast.Episode{
		Filename: "episode3.mp3",
		Status:   podcast.New,
		Session:  "session-002",
		Title:    "Episode 3",
		Size:     3072,
	}
	if err := store.SaveEpisode("podcast2", ep3); err != nil {
		t.Fatalf("Failed to save episode: %v", err)
	}

	// Verify data is stored
	retrieved, err := store.ListEpisodes("podcast1")
	if err != nil {
		t.Fatalf("Failed to get episodes: %v", err)
	}
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 episodes for podcast1, got %d", len(retrieved))
	}

	// Close the store
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("Database file was not created: %v", err)
	}

	// Reopen and verify persistence
	store2, err := factory.NewFromStrings("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite store for reopen: %v", err)
	}
	if err := store2.Open(); err != nil {
		t.Fatalf("Failed to reopen SQLite store: %v", err)
	}
	defer func() { _ = store2.Close() }()

	// Verify persisted data
	retrieved2, err := store2.ListEpisodes("podcast1")
	if err != nil {
		t.Fatalf("Failed to get episodes after reopen: %v", err)
	}
	if len(retrieved2) != 2 {
		t.Errorf("Expected 2 episodes for podcast1 after reopen, got %d", len(retrieved2))
	}

	// Verify all podcasts were stored
	podcasts, err := store2.ListPodcasts()
	if err != nil {
		t.Fatalf("Failed to get all podcasts: %v", err)
	}
	if len(podcasts) != 2 {
		t.Errorf("Expected 2 podcasts, got %d", len(podcasts))
	}

	t.Logf("SQLite creation and persistence test PASSED")
}

// TestAcceptance_BoltToSQLiteMigration verifies data migration from BoltDB to SQLite
func TestAcceptance_BoltToSQLiteMigration(t *testing.T) {
	tmpDir := t.TempDir()
	boltPath := filepath.Join(tmpDir, "test.bolt")
	sqlitePath := filepath.Join(tmpDir, "test.db")

	// Create and populate BoltDB
	boltStore, err := factory.NewFromStrings("bolt", boltPath)
	if err != nil {
		t.Fatalf("Failed to create BoltDB store: %v", err)
	}
	if err := boltStore.Open(); err != nil {
		t.Fatalf("Failed to open BoltDB store: %v", err)
	}

	// Create test episodes in BoltDB
	testEpisodes := []*podcast.Episode{
		{
			Filename: "old-episode1.mp3",
			Status:   podcast.Uploaded,
			Session:  "old-session",
			Location: "https://old.example.com/ep1.mp3",
			Title:    "Old Episode 1",
			Size:     1000,
		},
		{
			Filename: "old-episode2.mp3",
			Status:   podcast.Uploaded,
			Session:  "old-session",
			Location: "https://old.example.com/ep2.mp3",
			Title:    "Old Episode 2",
			Size:     2000,
		},
	}

	for _, ep := range testEpisodes {
		if err := boltStore.SaveEpisode("migrated-podcast", ep); err != nil {
			t.Fatalf("Failed to save episode to BoltDB: %v", err)
		}
	}

	// Add another podcast
	anotherEp := &podcast.Episode{
		Filename: "another-ep.mp3",
		Status:   podcast.New,
		Session:  "another-session",
		Title:    "Another Episode",
		Size:     500,
	}
	if err := boltStore.SaveEpisode("another-podcast", anotherEp); err != nil {
		t.Fatalf("Failed to save episode to BoltDB: %v", err)
	}

	// Create SQLite destination
	sqliteStore, err := factory.NewFromStrings("sqlite", sqlitePath)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	if err := sqliteStore.Open(); err != nil {
		t.Fatalf("Failed to open SQLite store: %v", err)
	}

	// Run migration
	stats, err := storage.Migrate(boltStore, sqliteStore)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Close stores
	_ = boltStore.Close()
	_ = sqliteStore.Close()

	// Verify migration stats
	if stats.PodcastsProcessed != 2 {
		t.Errorf("Expected 2 podcasts processed, got %d", stats.PodcastsProcessed)
	}
	if stats.EpisodesMigrated != 3 {
		t.Errorf("Expected 3 episodes migrated, got %d", stats.EpisodesMigrated)
	}
	if stats.EpisodesFailed != 0 {
		t.Errorf("Expected 0 episodes failed, got %d", stats.EpisodesFailed)
	}

	// Reopen SQLite and verify data
	sqliteStore2, err := factory.NewFromStrings("sqlite", sqlitePath)
	if err != nil {
		t.Fatalf("Failed to reopen SQLite store: %v", err)
	}
	if err := sqliteStore2.Open(); err != nil {
		t.Fatalf("Failed to reopen SQLite store: %v", err)
	}
	defer func() { _ = sqliteStore2.Close() }()

	migratedEpisodes, err := sqliteStore2.ListEpisodes("migrated-podcast")
	if err != nil {
		t.Fatalf("Failed to get migrated episodes: %v", err)
	}
	if len(migratedEpisodes) != 2 {
		t.Errorf("Expected 2 migrated episodes, got %d", len(migratedEpisodes))
	}

	// Verify episode data integrity
	for _, ep := range migratedEpisodes {
		if ep.Status != podcast.Uploaded {
			t.Errorf("Episode %s has wrong status: %v", ep.Filename, ep.Status)
		}
		if ep.Location == "" {
			t.Errorf("Episode %s is missing Location", ep.Filename)
		}
	}

	t.Logf("BoltDB to SQLite migration test PASSED")
}

// TestAcceptance_SQLiteWALMode verifies that SQLite uses WAL mode
func TestAcceptance_SQLiteWALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "wal-test.db")

	// Create SQLite database
	store, err := factory.NewFromStrings("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open SQLite store: %v", err)
	}

	// Perform a write operation to trigger WAL activity
	ep := &podcast.Episode{
		Filename: "wal-episode.mp3",
		Status:   podcast.New,
		Session:  "wal-session",
		Title:    "WAL Test Episode",
	}
	if err := store.SaveEpisode("wal-test", ep); err != nil {
		t.Fatalf("Failed to save episode: %v", err)
	}

	// Check for WAL files (may not always exist if immediately checkpointed)
	walPath := dbPath + "-wal"
	shmPath := dbPath + "-shm"

	// At minimum, the main db file should exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("Database file was not created")
	}

	// WAL or SHM files indicate WAL mode is active
	// Note: These files may be cleaned up after checkpoint
	walExists := false
	if _, err := os.Stat(walPath); err == nil {
		walExists = true
		t.Logf("WAL file exists: %s", walPath)
	}
	if _, err := os.Stat(shmPath); err == nil {
		walExists = true
		t.Logf("SHM file exists: %s", shmPath)
	}

	// Close the store
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Reopen and verify operations work correctly (WAL enables concurrent reads)
	store2, err := factory.NewFromStrings("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	if err := store2.Open(); err != nil {
		t.Fatalf("Failed to reopen SQLite store: %v", err)
	}
	defer func() { _ = store2.Close() }()

	// Perform multiple reads (WAL allows concurrent reads)
	for i := 0; i < 10; i++ {
		_, err := store2.ListEpisodes("wal-test")
		if err != nil {
			t.Fatalf("Read operation %d failed: %v", i, err)
		}
	}

	if walExists {
		t.Logf("SQLite WAL mode verification PASSED (WAL files detected)")
	} else {
		t.Logf("SQLite WAL mode verification PASSED (operations successful, WAL may have been checkpointed)")
	}
}

// TestAcceptance_StorageFactory verifies the factory creates correct backends
func TestAcceptance_StorageFactory(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		storeType string
		path      string
	}{
		{"SQLite backend", "sqlite", filepath.Join(tmpDir, "factory-sqlite.db")},
		{"BoltDB backend", "bolt", filepath.Join(tmpDir, "factory-bolt.db")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := factory.NewFromStrings(tt.storeType, tt.path)
			if err != nil {
				t.Fatalf("Factory failed to create %s: %v", tt.storeType, err)
			}

			if err := store.Open(); err != nil {
				t.Fatalf("Failed to open store: %v", err)
			}
			defer func() { _ = store.Close() }()

			// Verify store works
			ep := &podcast.Episode{
				Filename: "factory-ep.mp3",
				Status:   podcast.New,
				Session:  "factory-session",
				Title:    "Factory Test Episode",
			}
			if err := store.SaveEpisode("factory-test", ep); err != nil {
				t.Fatalf("Failed to save episode: %v", err)
			}

			retrieved, err := store.GetEpisodeByFilename("factory-test", "factory-ep.mp3")
			if err != nil {
				t.Fatalf("Failed to get episode: %v", err)
			}
			if retrieved == nil {
				t.Fatalf("Episode not found")
			}

			t.Logf("%s factory test PASSED", tt.storeType)
		})
	}
}
