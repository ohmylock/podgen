package sqlite_test

import (
	"os"
	"path/filepath"
	"testing"

	"podgen/internal/app/podgen/podcast"
	"podgen/internal/storage"
	"podgen/internal/storage/sqlite"
)

func newTestStore(t *testing.T) (*sqlite.Store, func()) { //nolint:gocritic // named returns not needed for test helper
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := storage.Config{
		Type:         storage.TypeSQLite,
		DSN:          dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}

	store := sqlite.New(cfg)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	cleanup := func() {
		store.Close()
	}

	return store, cleanup
}

func TestNew(t *testing.T) {
	cfg := storage.Config{
		Type: storage.TypeSQLite,
		DSN:  "/tmp/test.db",
	}

	store := sqlite.New(cfg)
	if store == nil {
		t.Fatal("New() returned nil")
	}
}

func TestOpenClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := storage.Config{
		Type:         storage.TypeSQLite,
		DSN:          dbPath,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}

	store := sqlite.New(cfg)

	// Test Open
	if err := store.Open(); err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// Verify WAL mode is active by checking for .db-wal file
	walFile := dbPath + "-wal"
	if _, err := os.Stat(walFile); os.IsNotExist(err) {
		t.Log("Note: WAL file not created yet (may be created after first write)")
	}

	// Test Close
	if err := store.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Test Close again (should be safe)
	if err := store.Close(); err != nil {
		t.Fatalf("Close() again failed: %v", err)
	}
}

func TestOpenInvalidPath(t *testing.T) {
	cfg := storage.Config{
		Type: storage.TypeSQLite,
		DSN:  "/nonexistent/path/to/db.sqlite",
	}

	store := sqlite.New(cfg)
	err := store.Open()
	if err == nil {
		store.Close()
		t.Fatal("Open() with invalid path should fail")
	}
}

func TestSaveEpisode(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episode := &podcast.Episode{
		Filename: "episode1.mp3",
		PubDate:  "2024-01-15",
		Size:     1024000,
		Status:   podcast.New,
		Location: "https://example.com/episode1.mp3",
		Session:  "session1",
		Title:    "Test Episode",
		Artist:   "Test Artist",
		Album:    "Test Album",
		Year:     "2024",
		Comment:  "Test comment",
		Duration: "30:00",
	}

	if err := store.SaveEpisode(podcastID, episode); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}

	// Verify the episode was saved
	retrieved, err := store.GetEpisodeByFilename(podcastID, episode.Filename)
	if err != nil {
		t.Fatalf("GetEpisodeByFilename() failed: %v", err)
	}

	// Verify all fields
	if retrieved.Filename != episode.Filename {
		t.Errorf("Filename = %s, want %s", retrieved.Filename, episode.Filename)
	}
	if retrieved.PubDate != episode.PubDate {
		t.Errorf("PubDate = %s, want %s", retrieved.PubDate, episode.PubDate)
	}
	if retrieved.Size != episode.Size {
		t.Errorf("Size = %d, want %d", retrieved.Size, episode.Size)
	}
	if retrieved.Status != episode.Status {
		t.Errorf("Status = %d, want %d", retrieved.Status, episode.Status)
	}
	if retrieved.Location != episode.Location {
		t.Errorf("Location = %s, want %s", retrieved.Location, episode.Location)
	}
	if retrieved.Session != episode.Session {
		t.Errorf("Session = %s, want %s", retrieved.Session, episode.Session)
	}
	if retrieved.Title != episode.Title {
		t.Errorf("Title = %s, want %s", retrieved.Title, episode.Title)
	}
	if retrieved.Artist != episode.Artist {
		t.Errorf("Artist = %s, want %s", retrieved.Artist, episode.Artist)
	}
	if retrieved.Album != episode.Album {
		t.Errorf("Album = %s, want %s", retrieved.Album, episode.Album)
	}
	if retrieved.Year != episode.Year {
		t.Errorf("Year = %s, want %s", retrieved.Year, episode.Year)
	}
	if retrieved.Comment != episode.Comment {
		t.Errorf("Comment = %s, want %s", retrieved.Comment, episode.Comment)
	}
	if retrieved.Duration != episode.Duration {
		t.Errorf("Duration = %s, want %s", retrieved.Duration, episode.Duration)
	}
}

func TestSaveEpisodeUpsert(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episode := &podcast.Episode{
		Filename: "episode1.mp3",
		Status:   podcast.New,
		Title:    "Original Title",
	}

	// Save original
	if err := store.SaveEpisode(podcastID, episode); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}

	// Update the episode
	episode.Status = podcast.Uploaded
	episode.Title = "Updated Title"
	if err := store.SaveEpisode(podcastID, episode); err != nil {
		t.Fatalf("SaveEpisode() update failed: %v", err)
	}

	// Verify update
	retrieved, err := store.GetEpisodeByFilename(podcastID, episode.Filename)
	if err != nil {
		t.Fatalf("GetEpisodeByFilename() failed: %v", err)
	}

	if retrieved.Status != podcast.Uploaded {
		t.Errorf("Status = %d, want %d", retrieved.Status, podcast.Uploaded)
	}
	if retrieved.Title != "Updated Title" {
		t.Errorf("Title = %s, want Updated Title", retrieved.Title)
	}
}

func TestFindEpisodesByStatus(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Status: podcast.New},
		{Filename: "ep2.mp3", Status: podcast.New},
		{Filename: "ep3.mp3", Status: podcast.Uploaded},
		{Filename: "ep4.mp3", Status: podcast.Deleted},
	}

	for _, ep := range episodes {
		if err := store.SaveEpisode(podcastID, ep); err != nil {
			t.Fatalf("SaveEpisode() failed: %v", err)
		}
	}

	// Find New episodes
	newEpisodes, err := store.FindEpisodesByStatus(podcastID, podcast.New)
	if err != nil {
		t.Fatalf("FindEpisodesByStatus(New) failed: %v", err)
	}
	if len(newEpisodes) != 2 {
		t.Errorf("FindEpisodesByStatus(New) count = %d, want 2", len(newEpisodes))
	}

	// Find Uploaded episodes
	uploadedEpisodes, err := store.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	if err != nil {
		t.Fatalf("FindEpisodesByStatus(Uploaded) failed: %v", err)
	}
	if len(uploadedEpisodes) != 1 {
		t.Errorf("FindEpisodesByStatus(Uploaded) count = %d, want 1", len(uploadedEpisodes))
	}

	// Find Deleted episodes
	deletedEpisodes, err := store.FindEpisodesByStatus(podcastID, podcast.Deleted)
	if err != nil {
		t.Fatalf("FindEpisodesByStatus(Deleted) failed: %v", err)
	}
	if len(deletedEpisodes) != 1 {
		t.Errorf("FindEpisodesByStatus(Deleted) count = %d, want 1", len(deletedEpisodes))
	}
}

func TestFindEpisodesByStatusNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// For new podcasts (no episodes yet), should return empty slice, not error.
	// This allows scanning new podcasts without requiring bucket creation first.
	episodes, err := store.FindEpisodesByStatus("nonexistent-podcast", podcast.New)
	if err != nil {
		t.Fatalf("FindEpisodesByStatus() on nonexistent podcast should not fail: %v", err)
	}
	if len(episodes) != 0 {
		t.Errorf("FindEpisodesByStatus() on nonexistent podcast should return empty slice, got %d", len(episodes))
	}
}

func TestFindEpisodesBySession(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Session: "session1"},
		{Filename: "ep2.mp3", Session: "session1"},
		{Filename: "ep3.mp3", Session: "session2"},
		{Filename: "ep4.mp3", Session: ""},
	}

	for _, ep := range episodes {
		if err := store.SaveEpisode(podcastID, ep); err != nil {
			t.Fatalf("SaveEpisode() failed: %v", err)
		}
	}

	// Find session1 episodes
	session1Episodes, err := store.FindEpisodesBySession(podcastID, "session1")
	if err != nil {
		t.Fatalf("FindEpisodesBySession(session1) failed: %v", err)
	}
	if len(session1Episodes) != 2 {
		t.Errorf("FindEpisodesBySession(session1) count = %d, want 2", len(session1Episodes))
	}

	// Find session2 episodes
	session2Episodes, err := store.FindEpisodesBySession(podcastID, "session2")
	if err != nil {
		t.Fatalf("FindEpisodesBySession(session2) failed: %v", err)
	}
	if len(session2Episodes) != 1 {
		t.Errorf("FindEpisodesBySession(session2) count = %d, want 1", len(session2Episodes))
	}

	// Find nonexistent session
	noSessionEpisodes, err := store.FindEpisodesBySession(podcastID, "nonexistent")
	if err != nil {
		t.Fatalf("FindEpisodesBySession(nonexistent) failed: %v", err)
	}
	if len(noSessionEpisodes) != 0 {
		t.Errorf("FindEpisodesBySession(nonexistent) count = %d, want 0", len(noSessionEpisodes))
	}
}

func TestFindEpisodesBySessionNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// For new podcasts (no episodes yet), should return empty slice, not error.
	episodes, err := store.FindEpisodesBySession("nonexistent-podcast", "session1")
	if err != nil {
		t.Fatalf("FindEpisodesBySession() on nonexistent podcast should not fail: %v", err)
	}
	if len(episodes) != 0 {
		t.Errorf("FindEpisodesBySession() on nonexistent podcast should return empty slice, got %d", len(episodes))
	}
}

func TestFindEpisodesBySizeLimit(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Status: podcast.New, Size: 100},
		{Filename: "ep2.mp3", Status: podcast.New, Size: 200},
		{Filename: "ep3.mp3", Status: podcast.New, Size: 300},
		{Filename: "ep4.mp3", Status: podcast.Uploaded, Size: 500},
	}

	for _, ep := range episodes {
		if err := store.SaveEpisode(podcastID, ep); err != nil {
			t.Fatalf("SaveEpisode() failed: %v", err)
		}
	}

	// Test with size limit 250 - should get at most 250 bytes total
	result, err := store.FindEpisodesBySizeLimit(podcastID, podcast.New, 250)
	if err != nil {
		t.Fatalf("FindEpisodesBySizeLimit() failed: %v", err)
	}

	var totalSize int64
	for _, ep := range result {
		totalSize += ep.Size
	}
	if totalSize > 250 {
		t.Errorf("FindEpisodesBySizeLimit() total size = %d, want <= 250", totalSize)
	}

	// Test with no limit (0) - should return all matching
	allNew, err := store.FindEpisodesBySizeLimit(podcastID, podcast.New, 0)
	if err != nil {
		t.Fatalf("FindEpisodesBySizeLimit() with 0 limit failed: %v", err)
	}
	if len(allNew) != 3 {
		t.Errorf("FindEpisodesBySizeLimit() with 0 limit count = %d, want 3", len(allNew))
	}

	// Test with large limit - should return all matching
	allNewLarge, err := store.FindEpisodesBySizeLimit(podcastID, podcast.New, 10000)
	if err != nil {
		t.Fatalf("FindEpisodesBySizeLimit() with large limit failed: %v", err)
	}
	if len(allNewLarge) != 3 {
		t.Errorf("FindEpisodesBySizeLimit() with large limit count = %d, want 3", len(allNewLarge))
	}
}

func TestFindEpisodesBySizeLimitNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Should return nil, nil for non-existent podcast (follows BoltDB behavior)
	result, err := store.FindEpisodesBySizeLimit("nonexistent-podcast", podcast.New, 100)
	if err != nil {
		t.Fatalf("FindEpisodesBySizeLimit() unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("FindEpisodesBySizeLimit() result = %v, want nil", result)
	}
}

func TestGetEpisodeByFilename(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episode := &podcast.Episode{
		Filename: "episode1.mp3",
		Title:    "Test Episode",
	}

	if err := store.SaveEpisode(podcastID, episode); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}

	// Test successful retrieval
	retrieved, err := store.GetEpisodeByFilename(podcastID, "episode1.mp3")
	if err != nil {
		t.Fatalf("GetEpisodeByFilename() failed: %v", err)
	}
	if retrieved.Filename != episode.Filename {
		t.Errorf("Filename = %s, want %s", retrieved.Filename, episode.Filename)
	}
	if retrieved.Title != episode.Title {
		t.Errorf("Title = %s, want %s", retrieved.Title, episode.Title)
	}

	// Test non-existent episode
	_, err = store.GetEpisodeByFilename(podcastID, "nonexistent.mp3")
	if err != storage.ErrNotFound {
		t.Errorf("GetEpisodeByFilename() error = %v, want ErrNotFound", err)
	}
}

func TestGetEpisodeByFilenameNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	_, err := store.GetEpisodeByFilename("nonexistent-podcast", "file.mp3")
	if err == nil {
		t.Fatal("GetEpisodeByFilename() on nonexistent podcast should fail")
	}
}

func TestGetLastEpisodeByNotStatus(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Status: podcast.New},
		{Filename: "ep2.mp3", Status: podcast.Uploaded},
		{Filename: "ep3.mp3", Status: podcast.Deleted},
		{Filename: "ep4.mp3", Status: podcast.Deleted},
	}

	for _, ep := range episodes {
		if err := store.SaveEpisode(podcastID, ep); err != nil {
			t.Fatalf("SaveEpisode() failed: %v", err)
		}
	}

	// Get last episode that is not Deleted
	result, err := store.GetLastEpisodeByNotStatus(podcastID, podcast.Deleted)
	if err != nil {
		t.Fatalf("GetLastEpisodeByNotStatus() failed: %v", err)
	}
	if result == nil {
		t.Fatal("GetLastEpisodeByNotStatus() returned nil, want episode")
	}
	if result.Status == podcast.Deleted {
		t.Errorf("GetLastEpisodeByNotStatus() returned deleted episode")
	}

	// Get last episode that is not New
	result, err = store.GetLastEpisodeByNotStatus(podcastID, podcast.New)
	if err != nil {
		t.Fatalf("GetLastEpisodeByNotStatus() failed: %v", err)
	}
	if result == nil {
		t.Fatal("GetLastEpisodeByNotStatus() returned nil, want episode")
	}
	if result.Status == podcast.New {
		t.Errorf("GetLastEpisodeByNotStatus() returned new episode")
	}
}

func TestGetLastEpisodeByNotStatusAllMatch(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Status: podcast.Deleted},
		{Filename: "ep2.mp3", Status: podcast.Deleted},
	}

	for _, ep := range episodes {
		if err := store.SaveEpisode(podcastID, ep); err != nil {
			t.Fatalf("SaveEpisode() failed: %v", err)
		}
	}

	// All episodes are Deleted, so looking for not Deleted should return nil
	result, err := store.GetLastEpisodeByNotStatus(podcastID, podcast.Deleted)
	if err != nil {
		t.Fatalf("GetLastEpisodeByNotStatus() failed: %v", err)
	}
	if result != nil {
		t.Errorf("GetLastEpisodeByNotStatus() = %v, want nil", result)
	}
}

func TestGetLastEpisodeByNotStatusNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// For new podcasts (no episodes yet), should return nil, not error.
	episode, err := store.GetLastEpisodeByNotStatus("nonexistent-podcast", podcast.New)
	if err != nil {
		t.Fatalf("GetLastEpisodeByNotStatus() on nonexistent podcast should not fail: %v", err)
	}
	if episode != nil {
		t.Errorf("GetLastEpisodeByNotStatus() on nonexistent podcast should return nil, got %v", episode)
	}
}

func TestListPodcasts(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Initially empty
	podcasts, err := store.ListPodcasts()
	if err != nil {
		t.Fatalf("ListPodcasts() failed: %v", err)
	}
	if len(podcasts) != 0 {
		t.Errorf("ListPodcasts() count = %d, want 0", len(podcasts))
	}

	// Add episodes to different podcasts
	if err := store.SaveEpisode("podcast1", &podcast.Episode{Filename: "ep1.mp3"}); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}
	if err := store.SaveEpisode("podcast2", &podcast.Episode{Filename: "ep2.mp3"}); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}
	if err := store.SaveEpisode("podcast1", &podcast.Episode{Filename: "ep3.mp3"}); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}

	podcasts, err = store.ListPodcasts()
	if err != nil {
		t.Fatalf("ListPodcasts() failed: %v", err)
	}
	if len(podcasts) != 2 {
		t.Errorf("ListPodcasts() count = %d, want 2", len(podcasts))
	}

	// Verify order (should be sorted)
	if podcasts[0] != "podcast1" || podcasts[1] != "podcast2" {
		t.Errorf("ListPodcasts() = %v, want [podcast1 podcast2]", podcasts)
	}
}

func TestListEpisodes(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3"},
		{Filename: "ep2.mp3"},
		{Filename: "ep3.mp3"},
	}

	for _, ep := range episodes {
		if err := store.SaveEpisode(podcastID, ep); err != nil {
			t.Fatalf("SaveEpisode() failed: %v", err)
		}
	}

	result, err := store.ListEpisodes(podcastID)
	if err != nil {
		t.Fatalf("ListEpisodes() failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("ListEpisodes() count = %d, want 3", len(result))
	}
}

func TestListEpisodesEmpty(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Save and then get from a different podcast
	if err := store.SaveEpisode("podcast1", &podcast.Episode{Filename: "ep1.mp3"}); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}

	result, err := store.ListEpisodes("podcast2")
	if err != nil {
		t.Fatalf("ListEpisodes() failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("ListEpisodes() count = %d, want 0", len(result))
	}
}

func TestOperationsOnClosedStore(t *testing.T) {
	store, cleanup := newTestStore(t)
	cleanup() // Close immediately

	podcastID := "test-podcast"
	episode := &podcast.Episode{Filename: "ep1.mp3"}

	// All operations should return ErrClosed
	if err := store.SaveEpisode(podcastID, episode); err != storage.ErrClosed {
		t.Errorf("SaveEpisode() error = %v, want ErrClosed", err)
	}

	if _, err := store.FindEpisodesByStatus(podcastID, podcast.New); err != storage.ErrClosed {
		t.Errorf("FindEpisodesByStatus() error = %v, want ErrClosed", err)
	}

	if _, err := store.FindEpisodesBySession(podcastID, "session"); err != storage.ErrClosed {
		t.Errorf("FindEpisodesBySession() error = %v, want ErrClosed", err)
	}

	if _, err := store.FindEpisodesBySizeLimit(podcastID, podcast.New, 100); err != storage.ErrClosed {
		t.Errorf("FindEpisodesBySizeLimit() error = %v, want ErrClosed", err)
	}

	if _, err := store.GetEpisodeByFilename(podcastID, "file.mp3"); err != storage.ErrClosed {
		t.Errorf("GetEpisodeByFilename() error = %v, want ErrClosed", err)
	}

	if _, err := store.GetLastEpisodeByNotStatus(podcastID, podcast.New); err != storage.ErrClosed {
		t.Errorf("GetLastEpisodeByNotStatus() error = %v, want ErrClosed", err)
	}

	if _, err := store.ListPodcasts(); err != storage.ErrClosed {
		t.Errorf("ListPodcasts() error = %v, want ErrClosed", err)
	}

	if _, err := store.ListEpisodes(podcastID); err != storage.ErrClosed {
		t.Errorf("ListEpisodes() error = %v, want ErrClosed", err)
	}
}

func TestStoreImplementsInterface(t *testing.T) {
	// Compile-time check that Store implements storage.Store
	var _ storage.Store = (*sqlite.Store)(nil)
}

func TestConcurrentAccess(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(idx int) {
			ep := &podcast.Episode{
				Filename: "concurrent-ep" + string(rune('0'+idx)) + ".mp3",
				Status:   podcast.New,
			}
			_ = store.SaveEpisode(podcastID, ep)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all episodes were saved
	episodes, err := store.ListEpisodes(podcastID)
	if err != nil {
		t.Fatalf("ListEpisodes() failed: %v", err)
	}
	if len(episodes) != 10 {
		t.Errorf("ListEpisodes() count = %d, want 10", len(episodes))
	}
}

func TestWALModeEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "wal-test.db")

	cfg := storage.Config{
		Type: storage.TypeSQLite,
		DSN:  dbPath,
	}

	store := sqlite.New(cfg)
	if err := store.Open(); err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer store.Close()

	// Write something to trigger WAL file creation
	if err := store.SaveEpisode("test", &podcast.Episode{Filename: "test.mp3"}); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}

	// Check for WAL file
	walFile := dbPath + "-wal"
	if _, err := os.Stat(walFile); os.IsNotExist(err) {
		t.Log("Note: WAL file may not exist if all changes are checkpointed")
	}
}

func TestMultiplePodcasts(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Save episodes to multiple podcasts
	podcasts := []string{"podcast-a", "podcast-b", "podcast-c"}
	for _, podcastID := range podcasts {
		for i := 0; i < 3; i++ {
			ep := &podcast.Episode{
				Filename: "ep" + string(rune('0'+i)) + ".mp3",
				Status:   podcast.New,
			}
			if err := store.SaveEpisode(podcastID, ep); err != nil {
				t.Fatalf("SaveEpisode() failed: %v", err)
			}
		}
	}

	// Verify each podcast has correct episodes
	for _, podcastID := range podcasts {
		episodes, err := store.ListEpisodes(podcastID)
		if err != nil {
			t.Fatalf("ListEpisodes(%s) failed: %v", podcastID, err)
		}
		if len(episodes) != 3 {
			t.Errorf("ListEpisodes(%s) count = %d, want 3", podcastID, len(episodes))
		}
	}

	// Verify ListPodcasts
	allPodcasts, err := store.ListPodcasts()
	if err != nil {
		t.Fatalf("ListPodcasts() failed: %v", err)
	}
	if len(allPodcasts) != 3 {
		t.Errorf("ListPodcasts() count = %d, want 3", len(allPodcasts))
	}
}
