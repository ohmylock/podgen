package proc

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"podgen/internal/app/podgen/podcast"
)

func newTestDB(t *testing.T) *BoltDB {
	t.Helper()
	f, err := os.CreateTemp("", "podgen-test-*.db")
	require.NoError(t, err)
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	db, err := bolt.Open(f.Name(), 0o600, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return &BoltDB{DB: db}
}

func TestBoltDB_SaveAndGetEpisode(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	ep := &podcast.Episode{
		Filename: "episode1.mp3",
		PubDate:  "Mon, 01 Jan 2024 00:00:00 +0000",
		Size:     12345,
		Status:   podcast.New,
	}

	err := store.SaveEpisode(podcastID, ep)
	require.NoError(t, err)

	got, err := store.GetEpisodeByFilename(podcastID, "episode1.mp3")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "episode1.mp3", got.Filename)
	assert.Equal(t, int64(12345), got.Size)
	assert.Equal(t, podcast.New, got.Status)
}

func TestBoltDB_GetEpisodeByFilename_NotFound(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	// Create bucket first
	ep := &podcast.Episode{Filename: "exists.mp3", Size: 100, Status: podcast.New}
	err := store.SaveEpisode(podcastID, ep)
	require.NoError(t, err)

	got, err := store.GetEpisodeByFilename(podcastID, "nonexistent.mp3")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBoltDB_GetEpisodeByFilename_NoBucket(t *testing.T) {
	store := newTestDB(t)

	got, err := store.GetEpisodeByFilename("nonexistent-podcast", "ep.mp3")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBoltDB_FindEpisodesByStatus(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
		{Filename: "ep2.mp3", Size: 2000, Status: podcast.Uploaded},
		{Filename: "ep3.mp3", Size: 3000, Status: podcast.New},
		{Filename: "ep4.mp3", Size: 4000, Status: podcast.Deleted},
	}

	for _, ep := range episodes {
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	newEps, err := store.FindEpisodesByStatus(podcastID, podcast.New)
	require.NoError(t, err)
	assert.Len(t, newEps, 2)

	uploaded, err := store.FindEpisodesByStatus(podcastID, podcast.Uploaded)
	require.NoError(t, err)
	assert.Len(t, uploaded, 1)
	assert.Equal(t, "ep2.mp3", uploaded[0].Filename)

	deleted, err := store.FindEpisodesByStatus(podcastID, podcast.Deleted)
	require.NoError(t, err)
	assert.Len(t, deleted, 1)
}

func TestBoltDB_FindEpisodesByStatus_NoBucket(t *testing.T) {
	store := newTestDB(t)

	eps, err := store.FindEpisodesByStatus("nonexistent", podcast.New)
	assert.Error(t, err)
	assert.Nil(t, eps)
}

func TestBoltDB_FindEpisodesBySession(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.Uploaded, Session: "session-1"},
		{Filename: "ep2.mp3", Size: 2000, Status: podcast.Uploaded, Session: "session-2"},
		{Filename: "ep3.mp3", Size: 3000, Status: podcast.Uploaded, Session: "session-1"},
	}

	for _, ep := range episodes {
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	sess1, err := store.FindEpisodesBySession(podcastID, "session-1")
	require.NoError(t, err)
	assert.Len(t, sess1, 2)

	sess2, err := store.FindEpisodesBySession(podcastID, "session-2")
	require.NoError(t, err)
	assert.Len(t, sess2, 1)

	sessNone, err := store.FindEpisodesBySession(podcastID, "nonexistent")
	require.NoError(t, err)
	assert.Len(t, sessNone, 0)
}

func TestBoltDB_ChangeStatusEpisodes(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
		{Filename: "ep2.mp3", Size: 2000, Status: podcast.New},
		{Filename: "ep3.mp3", Size: 3000, Status: podcast.Uploaded},
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

func TestBoltDB_FindEpisodesBySizeLimit(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
		{Filename: "ep2.mp3", Size: 2000, Status: podcast.New},
		{Filename: "ep3.mp3", Size: 3000, Status: podcast.New},
	}

	for _, ep := range episodes {
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	// Limit to 2500 bytes - should get only first episode
	result, err := store.FindEpisodesBySizeLimit(podcastID, podcast.New, 2500)
	require.NoError(t, err)
	assert.Len(t, result, 1)

	// Limit to 3001 - should get first two (1000+2000=3000 < 3001)
	result, err = store.FindEpisodesBySizeLimit(podcastID, podcast.New, 3001)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// No limit (0) - should get all
	result, err = store.FindEpisodesBySizeLimit(podcastID, podcast.New, 0)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestBoltDB_GetLastEpisodeByStatus(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
		{Filename: "ep2.mp3", Size: 2000, Status: podcast.Uploaded},
		{Filename: "ep3.mp3", Size: 3000, Status: podcast.Uploaded},
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

func TestBoltDB_GetLastEpisodeByNotStatus(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
		{Filename: "ep2.mp3", Size: 2000, Status: podcast.Uploaded},
		{Filename: "ep3.mp3", Size: 3000, Status: podcast.New},
	}

	for _, ep := range episodes {
		err := store.SaveEpisode(podcastID, ep)
		require.NoError(t, err)
	}

	// Last episode that is NOT New - should be ep2 (Uploaded)
	last, err := store.GetLastEpisodeByNotStatus(podcastID, podcast.New)
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "ep2.mp3", last.Filename)
	assert.Equal(t, podcast.Uploaded, last.Status)
}

func TestBoltDB_SaveEpisode_UpdateExisting(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	ep := &podcast.Episode{
		Filename: "ep1.mp3",
		Size:     1000,
		Status:   podcast.New,
	}

	err := store.SaveEpisode(podcastID, ep)
	require.NoError(t, err)

	// Update status
	ep.Status = podcast.Uploaded
	ep.Location = "https://s3/bucket/ep1.mp3"
	err = store.SaveEpisode(podcastID, ep)
	require.NoError(t, err)

	got, err := store.GetEpisodeByFilename(podcastID, "ep1.mp3")
	require.NoError(t, err)
	assert.Equal(t, podcast.Uploaded, got.Status)
	assert.Equal(t, "https://s3/bucket/ep1.mp3", got.Location)
}

func TestBoltDB_SaveEpisode_WithMetadata(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	ep := &podcast.Episode{
		Filename: "ep-meta.mp3",
		PubDate:  "Mon, 01 Jan 2024 00:00:00 +0000",
		Size:     5000,
		Status:   podcast.New,
		Title:    "My Episode Title",
		Artist:   "Some Artist",
		Album:    "Best Of Album",
		Year:     "2024",
		Comment:  "A great episode",
		Duration: "45:30",
	}

	err := store.SaveEpisode(podcastID, ep)
	require.NoError(t, err)

	got, err := store.GetEpisodeByFilename(podcastID, "ep-meta.mp3")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "My Episode Title", got.Title)
	assert.Equal(t, "Some Artist", got.Artist)
	assert.Equal(t, "Best Of Album", got.Album)
	assert.Equal(t, "2024", got.Year)
	assert.Equal(t, "A great episode", got.Comment)
	assert.Equal(t, "45:30", got.Duration)
}

func TestBoltDB_BackwardCompat_OldEpisodeWithoutMetadata(t *testing.T) {
	store := newTestDB(t)
	podcastID := "test-podcast"

	// Write raw JSON without new metadata fields (simulates old data format)
	oldJSON := `{"Filename":"old-ep.mp3","PubDate":"Mon, 01 Jan 2023 00:00:00 +0000","Size":9999,"Status":1,"Location":"https://example.com/old-ep.mp3","Session":"sess-old"}`

	err := store.WithWriteTx(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(podcastID))
		if err != nil {
			return err
		}
		return bucket.Put([]byte("old-ep.mp3"), []byte(oldJSON))
	})
	require.NoError(t, err)

	got, err := store.GetEpisodeByFilename(podcastID, "old-ep.mp3")
	require.NoError(t, err)
	require.NotNil(t, got)

	// Core fields intact
	assert.Equal(t, "old-ep.mp3", got.Filename)
	assert.Equal(t, int64(9999), got.Size)
	assert.Equal(t, podcast.Uploaded, got.Status)

	// New metadata fields default to empty strings
	assert.Equal(t, "", got.Title)
	assert.Equal(t, "", got.Artist)
	assert.Equal(t, "", got.Album)
	assert.Equal(t, "", got.Year)
	assert.Equal(t, "", got.Comment)
	assert.Equal(t, "", got.Duration)
}

func TestBoltDB_WithWriteTx(t *testing.T) {
	store := newTestDB(t)

	err := store.WithWriteTx(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("test-bucket"))
		return err
	})
	require.NoError(t, err)

	// Verify bucket was created
	err = store.WithReadTx(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("test-bucket"))
		assert.NotNil(t, b)
		return nil
	})
	require.NoError(t, err)
}
