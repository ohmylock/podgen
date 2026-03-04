package storage_test

import (
	"testing"

	"podgen/internal/app/podgen/podcast"
	"podgen/internal/storage"
)

// MockStore is a test implementation of the Store interface.
type MockStore struct {
	episodes  map[string]map[string]*podcast.Episode // podcastID -> filename -> episode
	podcasts  []string
	openCalls int
	closed    bool
}

func NewMockStore() *MockStore {
	return &MockStore{
		episodes: make(map[string]map[string]*podcast.Episode),
		podcasts: []string{},
	}
}

func (m *MockStore) Open() error {
	m.openCalls++
	m.closed = false
	return nil
}

func (m *MockStore) Close() error {
	m.closed = true
	return nil
}

func (m *MockStore) SaveEpisode(podcastID string, episode *podcast.Episode) error {
	if m.closed {
		return storage.ErrClosed
	}
	if m.episodes[podcastID] == nil {
		m.episodes[podcastID] = make(map[string]*podcast.Episode)
		m.podcasts = append(m.podcasts, podcastID)
	}
	m.episodes[podcastID][episode.Filename] = episode
	return nil
}

func (m *MockStore) FindEpisodesByStatus(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
	if m.closed {
		return nil, storage.ErrClosed
	}
	bucket, ok := m.episodes[podcastID]
	if !ok {
		return nil, storage.ErrNoBucket
	}
	var result []*podcast.Episode
	for _, ep := range bucket {
		if ep.Status == status {
			result = append(result, ep)
		}
	}
	return result, nil
}

func (m *MockStore) FindEpisodesBySession(podcastID, session string) ([]*podcast.Episode, error) {
	if m.closed {
		return nil, storage.ErrClosed
	}
	bucket, ok := m.episodes[podcastID]
	if !ok {
		return nil, storage.ErrNoBucket
	}
	var result []*podcast.Episode
	for _, ep := range bucket {
		if ep.Session == session {
			result = append(result, ep)
		}
	}
	return result, nil
}

func (m *MockStore) FindEpisodesBySizeLimit(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
	if m.closed {
		return nil, storage.ErrClosed
	}
	bucket, ok := m.episodes[podcastID]
	if !ok {
		return nil, storage.ErrNoBucket
	}
	var result []*podcast.Episode
	var totalSize int64
	for _, ep := range bucket {
		if ep.Status == status {
			if sizeLimit > 0 && totalSize+ep.Size > sizeLimit {
				break
			}
			result = append(result, ep)
			totalSize += ep.Size
		}
	}
	return result, nil
}

func (m *MockStore) GetEpisodeByFilename(podcastID, fileName string) (*podcast.Episode, error) {
	if m.closed {
		return nil, storage.ErrClosed
	}
	bucket, ok := m.episodes[podcastID]
	if !ok {
		return nil, storage.ErrNoBucket
	}
	ep, ok := bucket[fileName]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return ep, nil
}

func (m *MockStore) GetLastEpisodeByNotStatus(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	if m.closed {
		return nil, storage.ErrClosed
	}
	bucket, ok := m.episodes[podcastID]
	if !ok {
		return nil, storage.ErrNoBucket
	}
	var lastEp *podcast.Episode
	for _, ep := range bucket {
		if ep.Status != status {
			lastEp = ep
		}
	}
	return lastEp, nil
}

func (m *MockStore) ListPodcasts() ([]string, error) {
	if m.closed {
		return nil, storage.ErrClosed
	}
	return m.podcasts, nil
}

func (m *MockStore) ListEpisodes(podcastID string) ([]*podcast.Episode, error) {
	if m.closed {
		return nil, storage.ErrClosed
	}
	bucket, ok := m.episodes[podcastID]
	if !ok {
		return nil, storage.ErrNoBucket
	}
	var result []*podcast.Episode
	for _, ep := range bucket {
		result = append(result, ep)
	}
	return result, nil
}

// Compile-time check that MockStore implements Store interface.
var _ storage.Store = (*MockStore)(nil)

func TestStoreInterface(t *testing.T) {
	store := NewMockStore()

	// Test Open
	if err := store.Open(); err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	if store.openCalls != 1 {
		t.Errorf("Open() calls = %d, want 1", store.openCalls)
	}

	// Test SaveEpisode
	podcastID := "test-podcast"
	episode := &podcast.Episode{
		Filename: "episode1.mp3",
		Status:   podcast.New,
		Size:     1000,
		Session:  "session1",
	}
	if err := store.SaveEpisode(podcastID, episode); err != nil {
		t.Fatalf("SaveEpisode() failed: %v", err)
	}

	// Test GetEpisodeByFilename
	retrieved, err := store.GetEpisodeByFilename(podcastID, "episode1.mp3")
	if err != nil {
		t.Fatalf("GetEpisodeByFilename() failed: %v", err)
	}
	if retrieved.Filename != episode.Filename {
		t.Errorf("GetEpisodeByFilename() filename = %s, want %s", retrieved.Filename, episode.Filename)
	}

	// Test GetEpisodeByFilename with non-existent episode
	_, err = store.GetEpisodeByFilename(podcastID, "nonexistent.mp3")
	if err != storage.ErrNotFound {
		t.Errorf("GetEpisodeByFilename() error = %v, want ErrNotFound", err)
	}

	// Test FindEpisodesByStatus
	episodes, err := store.FindEpisodesByStatus(podcastID, podcast.New)
	if err != nil {
		t.Fatalf("FindEpisodesByStatus() failed: %v", err)
	}
	if len(episodes) != 1 {
		t.Errorf("FindEpisodesByStatus() count = %d, want 1", len(episodes))
	}

	// Test FindEpisodesBySession
	episodes, err = store.FindEpisodesBySession(podcastID, "session1")
	if err != nil {
		t.Fatalf("FindEpisodesBySession() failed: %v", err)
	}
	if len(episodes) != 1 {
		t.Errorf("FindEpisodesBySession() count = %d, want 1", len(episodes))
	}

	// Test ListPodcasts
	podcasts, err := store.ListPodcasts()
	if err != nil {
		t.Fatalf("ListPodcasts() failed: %v", err)
	}
	if len(podcasts) != 1 || podcasts[0] != podcastID {
		t.Errorf("ListPodcasts() = %v, want [%s]", podcasts, podcastID)
	}

	// Test ListEpisodes
	episodes, err = store.ListEpisodes(podcastID)
	if err != nil {
		t.Fatalf("ListEpisodes() failed: %v", err)
	}
	if len(episodes) != 1 {
		t.Errorf("ListEpisodes() count = %d, want 1", len(episodes))
	}

	// Test Close
	if err := store.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Test operations after close
	_, err = store.GetEpisodeByFilename(podcastID, "episode1.mp3")
	if err != storage.ErrClosed {
		t.Errorf("GetEpisodeByFilename() after Close() error = %v, want ErrClosed", err)
	}
}

func TestEpisodeStoreInterface(t *testing.T) {
	// Verify that any EpisodeStore implementation can be used
	var store storage.EpisodeStore = NewMockStore()

	// This is a compile-time check that EpisodeStore methods work
	_ = store.SaveEpisode("podcast", &podcast.Episode{Filename: "test.mp3"})
}

func TestDefaultConfig(t *testing.T) {
	config := storage.DefaultConfig("/path/to/db.sqlite")

	if config.Type != storage.TypeSQLite {
		t.Errorf("DefaultConfig().Type = %s, want %s", config.Type, storage.TypeSQLite)
	}
	if config.DSN != "/path/to/db.sqlite" {
		t.Errorf("DefaultConfig().DSN = %s, want /path/to/db.sqlite", config.DSN)
	}
	if config.MaxOpenConns != 10 {
		t.Errorf("DefaultConfig().MaxOpenConns = %d, want 10", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 5 {
		t.Errorf("DefaultConfig().MaxIdleConns = %d, want 5", config.MaxIdleConns)
	}
}

func TestStorageTypes(t *testing.T) {
	// Verify storage type constants
	if storage.TypeSQLite != "sqlite" {
		t.Errorf("TypeSQLite = %s, want sqlite", storage.TypeSQLite)
	}
	if storage.TypeBolt != "bolt" {
		t.Errorf("TypeBolt = %s, want bolt", storage.TypeBolt)
	}
	if storage.TypePostgres != "postgres" {
		t.Errorf("TypePostgres = %s, want postgres", storage.TypePostgres)
	}
}

func TestFindEpisodesBySizeLimit(t *testing.T) {
	store := NewMockStore()
	_ = store.Open()
	defer store.Close()

	podcastID := "test-podcast"

	// Add multiple episodes with different sizes
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Status: podcast.Uploaded, Size: 100},
		{Filename: "ep2.mp3", Status: podcast.Uploaded, Size: 200},
		{Filename: "ep3.mp3", Status: podcast.Uploaded, Size: 300},
	}

	for _, ep := range episodes {
		if err := store.SaveEpisode(podcastID, ep); err != nil {
			t.Fatalf("SaveEpisode() failed: %v", err)
		}
	}

	// Test with size limit
	result, err := store.FindEpisodesBySizeLimit(podcastID, podcast.Uploaded, 250)
	if err != nil {
		t.Fatalf("FindEpisodesBySizeLimit() failed: %v", err)
	}

	// Should return episodes until size limit is exceeded
	// Note: due to map iteration order, actual results may vary
	var totalSize int64
	for _, ep := range result {
		totalSize += ep.Size
	}
	if totalSize > 250 {
		t.Errorf("FindEpisodesBySizeLimit() total size = %d, want <= 250", totalSize)
	}

	// Test with no limit (0)
	result, err = store.FindEpisodesBySizeLimit(podcastID, podcast.Uploaded, 0)
	if err != nil {
		t.Fatalf("FindEpisodesBySizeLimit() with no limit failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("FindEpisodesBySizeLimit() with no limit count = %d, want 3", len(result))
	}
}

func TestGetLastEpisodeByNotStatus(t *testing.T) {
	store := NewMockStore()
	_ = store.Open()
	defer store.Close()

	podcastID := "test-podcast"

	// Add episodes with different statuses
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Status: podcast.New},
		{Filename: "ep2.mp3", Status: podcast.Uploaded},
		{Filename: "ep3.mp3", Status: podcast.Deleted},
	}

	for _, ep := range episodes {
		if err := store.SaveEpisode(podcastID, ep); err != nil {
			t.Fatalf("SaveEpisode() failed: %v", err)
		}
	}

	// Test getting episode that is not Deleted
	result, err := store.GetLastEpisodeByNotStatus(podcastID, podcast.Deleted)
	if err != nil {
		t.Fatalf("GetLastEpisodeByNotStatus() failed: %v", err)
	}
	if result == nil {
		t.Fatal("GetLastEpisodeByNotStatus() returned nil, want episode")
	}
	if result.Status == podcast.Deleted {
		t.Errorf("GetLastEpisodeByNotStatus() returned episode with status Deleted")
	}
}

func TestNoBucketErrors(t *testing.T) {
	store := NewMockStore()
	_ = store.Open()
	defer store.Close()

	// Test operations on non-existent podcast
	_, err := store.FindEpisodesByStatus("nonexistent", podcast.New)
	if err != storage.ErrNoBucket {
		t.Errorf("FindEpisodesByStatus() error = %v, want ErrNoBucket", err)
	}

	_, err = store.FindEpisodesBySession("nonexistent", "session")
	if err != storage.ErrNoBucket {
		t.Errorf("FindEpisodesBySession() error = %v, want ErrNoBucket", err)
	}

	_, err = store.ListEpisodes("nonexistent")
	if err != storage.ErrNoBucket {
		t.Errorf("ListEpisodes() error = %v, want ErrNoBucket", err)
	}

	_, err = store.GetEpisodeByFilename("nonexistent", "file.mp3")
	if err != storage.ErrNoBucket {
		t.Errorf("GetEpisodeByFilename() error = %v, want ErrNoBucket", err)
	}

	_, err = store.GetLastEpisodeByNotStatus("nonexistent", podcast.New)
	if err != storage.ErrNoBucket {
		t.Errorf("GetLastEpisodeByNotStatus() error = %v, want ErrNoBucket", err)
	}

	_, err = store.FindEpisodesBySizeLimit("nonexistent", podcast.New, 100)
	if err != storage.ErrNoBucket {
		t.Errorf("FindEpisodesBySizeLimit() error = %v, want ErrNoBucket", err)
	}
}
