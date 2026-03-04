package proc

import (
	"github.com/boltdb/bolt"

	"podgen/internal/app/podgen/podcast"
	boltstore "podgen/internal/storage/bolt"
)

// BoltDB is a backward-compatible wrapper around storage/bolt.Store.
//
// Deprecated: New code should use storage/bolt.Store directly via the factory.
type BoltDB struct {
	// DB is kept for backward compatibility with existing code that creates
	// BoltDB by setting DB directly. New code should use storage/bolt.New()
	// and call Open() instead.
	DB *bolt.DB

	// store is the underlying storage implementation.
	// It's lazily initialized from DB if not set via newFromStore.
	store *boltstore.Store
}

// NewBoltDBFromStore creates a BoltDB wrapper from an existing storage/bolt.Store.
// This is the preferred way to create a BoltDB in new code.
func NewBoltDBFromStore(store *boltstore.Store) *BoltDB {
	return &BoltDB{
		DB:    store.DB(),
		store: store,
	}
}

// SaveEpisode saves an episode to the store.
func (b *BoltDB) SaveEpisode(podcastID string, episode *podcast.Episode) error {
	if b.store != nil {
		return b.store.SaveEpisode(podcastID, episode)
	}
	// Legacy path: use the DB directly with the old implementation
	return b.saveLegacy(podcastID, episode)
}

// FindEpisodesByStatus retrieves all episodes with the given status.
func (b *BoltDB) FindEpisodesByStatus(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
	if b.store != nil {
		return b.store.FindEpisodesByStatus(podcastID, status)
	}
	return b.findByStatusLegacy(podcastID, status)
}

// FindEpisodesBySession retrieves all episodes for a given session.
func (b *BoltDB) FindEpisodesBySession(podcastID, session string) ([]*podcast.Episode, error) {
	if b.store != nil {
		return b.store.FindEpisodesBySession(podcastID, session)
	}
	return b.findBySessionLegacy(podcastID, session)
}

// ChangeStatusEpisodes changes the status of all episodes matching fromStatus to toStatus.
func (b *BoltDB) ChangeStatusEpisodes(podcastID string, fromStatus, toStatus podcast.Status) error {
	if b.store != nil {
		return b.store.ChangeStatusEpisodes(podcastID, fromStatus, toStatus)
	}
	return b.changeStatusLegacy(podcastID, fromStatus, toStatus)
}

// FindEpisodesBySizeLimit retrieves episodes up to a total size limit.
func (b *BoltDB) FindEpisodesBySizeLimit(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
	if b.store != nil {
		return b.store.FindEpisodesBySizeLimit(podcastID, status, sizeLimit)
	}
	return b.findBySizeLimitLegacy(podcastID, status, sizeLimit)
}

// GetEpisodeByFilename retrieves an episode by its filename.
func (b *BoltDB) GetEpisodeByFilename(podcastID, fileName string) (*podcast.Episode, error) {
	if b.store != nil {
		return b.store.GetEpisodeByFilename(podcastID, fileName)
	}
	return b.getByFilenameLegacy(podcastID, fileName)
}

// GetLastEpisodeByStatus retrieves the last episode with the given status.
func (b *BoltDB) GetLastEpisodeByStatus(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	if b.store != nil {
		return b.store.GetLastEpisodeByStatus(podcastID, status)
	}
	return b.getLastByStatusLegacy(podcastID, status)
}

// GetLastEpisodeByNotStatus retrieves the last episode that doesn't have the given status.
func (b *BoltDB) GetLastEpisodeByNotStatus(podcastID string, status podcast.Status) (*podcast.Episode, error) {
	if b.store != nil {
		return b.store.GetLastEpisodeByNotStatus(podcastID, status)
	}
	return b.getLastByNotStatusLegacy(podcastID, status)
}

// WithWriteTx executes fn within a serialized write transaction.
//
// Deprecated: This method exposes bolt internals. Use the storage interface methods instead.
func (b *BoltDB) WithWriteTx(fn func(*bolt.Tx) error) error {
	if b.store != nil {
		return b.store.WithWriteTx(fn)
	}
	return b.DB.Update(fn)
}

// WithReadTx executes fn within a read-only transaction.
//
// Deprecated: This method exposes bolt internals. Use the storage interface methods instead.
func (b *BoltDB) WithReadTx(fn func(*bolt.Tx) error) error {
	if b.store != nil {
		return b.store.WithReadTx(fn)
	}
	return b.DB.View(fn)
}
