package bolt_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"podgen/internal/app/podgen/podcast"
	"podgen/internal/storage"
	boltstore "podgen/internal/storage/bolt"
)

func newTestStore(t *testing.T) (*boltstore.Store, func()) { //nolint:gocritic // named returns not needed for test helper
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := storage.Config{
		Type: storage.TypeBolt,
		DSN:  dbPath,
	}

	store := boltstore.New(cfg)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	cleanup := func() {
		_ = store.Close()
	}

	return store, cleanup
}

func TestNew(t *testing.T) {
	cfg := storage.Config{
		Type: storage.TypeBolt,
		DSN:  "/tmp/test.db",
	}

	store := boltstore.New(cfg)
	if store == nil {
		t.Fatal("New() returned nil")
	}
}

func TestOpenClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := storage.Config{
		Type: storage.TypeBolt,
		DSN:  dbPath,
	}

	store := boltstore.New(cfg)

	// Test Open
	if err := store.Open(); err != nil {
		t.Fatalf("Open() failed: %v", err)
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

func TestOpenEmptyPath(t *testing.T) {
	cfg := storage.Config{
		Type: storage.TypeBolt,
		DSN:  "",
	}

	store := boltstore.New(cfg)
	err := store.Open()
	if err == nil {
		_ = store.Close()
		t.Fatal("Open() with empty path should fail")
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

	err := store.SaveEpisode(podcastID, episode)
	require.NoError(t, err)

	// Verify the episode was saved
	retrieved, err := store.GetEpisodeByFilename(podcastID, episode.Filename)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, episode.Filename, retrieved.Filename)
	assert.Equal(t, episode.PubDate, retrieved.PubDate)
	assert.Equal(t, episode.Size, retrieved.Size)
	assert.Equal(t, episode.Status, retrieved.Status)
	assert.Equal(t, episode.Location, retrieved.Location)
	assert.Equal(t, episode.Session, retrieved.Session)
	assert.Equal(t, episode.Title, retrieved.Title)
	assert.Equal(t, episode.Artist, retrieved.Artist)
	assert.Equal(t, episode.Album, retrieved.Album)
	assert.Equal(t, episode.Year, retrieved.Year)
	assert.Equal(t, episode.Comment, retrieved.Comment)
	assert.Equal(t, episode.Duration, retrieved.Duration)
}

func TestSaveEpisodeUpdate(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episode := &podcast.Episode{
		Filename: "episode1.mp3",
		Status:   podcast.New,
		Title:    "Original Title",
	}

	err := store.SaveEpisode(podcastID, episode)
	require.NoError(t, err)

	// Update the episode
	episode.Status = podcast.Uploaded
	episode.Title = "Updated Title"
	err = store.SaveEpisode(podcastID, episode)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.GetEpisodeByFilename(podcastID, episode.Filename)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, podcast.Uploaded, retrieved.Status)
	assert.Equal(t, "Updated Title", retrieved.Title)
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
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	// Find New episodes
	newEpisodes, err := store.FindEpisodesByStatus(podcastID, podcast.New)
	require.NoError(t, err)
	assert.Len(t, newEpisodes, 2)

	// Find Uploaded episodes
	uploadedEpisodes, err := store.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	require.NoError(t, err)
	assert.Len(t, uploadedEpisodes, 1)

	// Find Deleted episodes
	deletedEpisodes, err := store.FindEpisodesByStatus(podcastID, podcast.Deleted)
	require.NoError(t, err)
	assert.Len(t, deletedEpisodes, 1)
}

func TestFindEpisodesByStatusNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// For new podcasts (no bucket yet), should return empty slice, not error.
	// This matches SQLite behavior and allows scanning new podcasts.
	episodes, err := store.FindEpisodesByStatus("nonexistent-podcast", podcast.New)
	require.NoError(t, err)
	assert.Empty(t, episodes)
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
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	// Find session1 episodes
	session1Episodes, err := store.FindEpisodesBySession(podcastID, "session1")
	require.NoError(t, err)
	assert.Len(t, session1Episodes, 2)

	// Find session2 episodes
	session2Episodes, err := store.FindEpisodesBySession(podcastID, "session2")
	require.NoError(t, err)
	assert.Len(t, session2Episodes, 1)

	// Find nonexistent session
	noSessionEpisodes, err := store.FindEpisodesBySession(podcastID, "nonexistent")
	require.NoError(t, err)
	assert.Len(t, noSessionEpisodes, 0)
}

func TestFindEpisodesBySessionNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// For new podcasts (no bucket yet), should return empty slice, not error.
	// This matches SQLite behavior and allows querying new podcasts.
	episodes, err := store.FindEpisodesBySession("nonexistent-podcast", "session1")
	require.NoError(t, err)
	assert.Empty(t, episodes)
}

func TestChangeStatusEpisodes(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Status: podcast.New},
		{Filename: "ep2.mp3", Status: podcast.New},
		{Filename: "ep3.mp3", Status: podcast.Uploaded},
	}

	for _, ep := range episodes {
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	err := store.ChangeStatusEpisodes(podcastID, podcast.New, podcast.Uploaded)
	require.NoError(t, err)

	uploaded, err := store.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	require.NoError(t, err)
	assert.Len(t, uploaded, 3)

	newEps, err := store.FindEpisodesByStatus(podcastID, podcast.New)
	require.NoError(t, err)
	assert.Len(t, newEps, 0)
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
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	// Test with size limit 250 - should get at most 250 bytes total
	result, err := store.FindEpisodesBySizeLimit(podcastID, podcast.New, 250)
	require.NoError(t, err)

	var totalSize int64
	for _, ep := range result {
		totalSize += ep.Size
	}
	assert.LessOrEqual(t, totalSize, int64(250))

	// Test with no limit (0) - should return all matching
	allNew, err := store.FindEpisodesBySizeLimit(podcastID, podcast.New, 0)
	require.NoError(t, err)
	assert.Len(t, allNew, 3)

	// Test with large limit - should return all matching
	allNewLarge, err := store.FindEpisodesBySizeLimit(podcastID, podcast.New, 10000)
	require.NoError(t, err)
	assert.Len(t, allNewLarge, 3)
}

func TestFindEpisodesBySizeLimitNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// For new podcasts (no bucket yet), should return empty slice, not error.
	// This matches SQLite behavior and allows querying new podcasts.
	result, err := store.FindEpisodesBySizeLimit("nonexistent-podcast", podcast.New, 100)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetEpisodeByFilename(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episode := &podcast.Episode{
		Filename: "episode1.mp3",
		Title:    "Test Episode",
	}

	err := store.SaveEpisode(podcastID, episode)
	require.NoError(t, err)

	// Test successful retrieval
	retrieved, err := store.GetEpisodeByFilename(podcastID, "episode1.mp3")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, episode.Filename, retrieved.Filename)
	assert.Equal(t, episode.Title, retrieved.Title)

	// Test non-existent episode
	_, err = store.GetEpisodeByFilename(podcastID, "nonexistent.mp3")
	assert.Error(t, err)
}

func TestGetEpisodeByFilenameNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	_, err := store.GetEpisodeByFilename("nonexistent-podcast", "file.mp3")
	assert.Error(t, err)
}

func TestGetLastEpisodeByStatus(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	podcastID := "test-podcast"
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Status: podcast.New},
		{Filename: "ep2.mp3", Status: podcast.Uploaded},
		{Filename: "ep3.mp3", Status: podcast.Uploaded},
	}

	for _, ep := range episodes {
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	last, err := store.GetLastEpisodeByStatus(podcastID, podcast.Uploaded)
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "ep3.mp3", last.Filename)

	lastNew, err := store.GetLastEpisodeByStatus(podcastID, podcast.New)
	require.NoError(t, err)
	require.NotNil(t, lastNew)
	assert.Equal(t, "ep1.mp3", lastNew.Filename)
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
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	// Get last episode that is not Deleted
	result, err := store.GetLastEpisodeByNotStatus(podcastID, podcast.Deleted)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEqual(t, podcast.Deleted, result.Status)

	// Get last episode that is not New
	result, err = store.GetLastEpisodeByNotStatus(podcastID, podcast.New)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEqual(t, podcast.New, result.Status)
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
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	// All episodes are Deleted, so looking for not Deleted should return nil
	result, err := store.GetLastEpisodeByNotStatus(podcastID, podcast.Deleted)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetLastEpisodeByNotStatusNoBucket(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	_, err := store.GetLastEpisodeByNotStatus("nonexistent-podcast", podcast.New)
	assert.Error(t, err)
}

func TestListPodcasts(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Initially empty
	podcasts, err := store.ListPodcasts()
	require.NoError(t, err)
	assert.Len(t, podcasts, 0)

	// Add episodes to different podcasts
	err = store.SaveEpisode("podcast1", &podcast.Episode{Filename: "ep1.mp3"})
	require.NoError(t, err)
	err = store.SaveEpisode("podcast2", &podcast.Episode{Filename: "ep2.mp3"})
	require.NoError(t, err)
	err = store.SaveEpisode("podcast1", &podcast.Episode{Filename: "ep3.mp3"})
	require.NoError(t, err)

	podcasts, err = store.ListPodcasts()
	require.NoError(t, err)
	assert.Len(t, podcasts, 2)
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
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	result, err := store.ListEpisodes(podcastID)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestListEpisodesEmpty(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Save and then get from a different podcast
	err := store.SaveEpisode("podcast1", &podcast.Episode{Filename: "ep1.mp3"})
	require.NoError(t, err)

	result, err := store.ListEpisodes("podcast2")
	require.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestOperationsOnClosedStore(t *testing.T) {
	store, cleanup := newTestStore(t)
	cleanup() // Close immediately

	podcastID := "test-podcast"
	episode := &podcast.Episode{Filename: "ep1.mp3"}

	// All operations should return ErrClosed
	assert.Equal(t, storage.ErrClosed, store.SaveEpisode(podcastID, episode))

	_, err := store.FindEpisodesByStatus(podcastID, podcast.New)
	assert.Equal(t, storage.ErrClosed, err)

	_, err = store.FindEpisodesBySession(podcastID, "session")
	assert.Equal(t, storage.ErrClosed, err)

	_, err = store.FindEpisodesBySizeLimit(podcastID, podcast.New, 100)
	assert.Equal(t, storage.ErrClosed, err)

	_, err = store.GetEpisodeByFilename(podcastID, "file.mp3")
	assert.Equal(t, storage.ErrClosed, err)

	_, err = store.GetLastEpisodeByNotStatus(podcastID, podcast.New)
	assert.Equal(t, storage.ErrClosed, err)

	_, err = store.GetLastEpisodeByStatus(podcastID, podcast.New)
	assert.Equal(t, storage.ErrClosed, err)

	_, err = store.ListPodcasts()
	assert.Equal(t, storage.ErrClosed, err)

	_, err = store.ListEpisodes(podcastID)
	assert.Equal(t, storage.ErrClosed, err)

	assert.Equal(t, storage.ErrClosed, store.ChangeStatusEpisodes(podcastID, podcast.New, podcast.Uploaded))
}

func TestStoreImplementsInterface(t *testing.T) {
	// Compile-time check that Store implements storage.Store
	var _ storage.Store = (*boltstore.Store)(nil)
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
			err := store.SaveEpisode(podcastID, ep)
			require.NoError(t, err)
		}
	}

	// Verify each podcast has correct episodes
	for _, podcastID := range podcasts {
		episodes, err := store.ListEpisodes(podcastID)
		require.NoError(t, err)
		assert.Len(t, episodes, 3)
	}

	// Verify ListPodcasts
	allPodcasts, err := store.ListPodcasts()
	require.NoError(t, err)
	assert.Len(t, allPodcasts, 3)
}

func TestDBAccessor(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// Verify DB() returns non-nil
	db := store.DB()
	assert.NotNil(t, db)
}
