package proc_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"podgen/internal/app/podgen/podcast"
	"podgen/internal/app/podgen/proc"
	"podgen/internal/app/podgen/proc/mocks"
	"podgen/internal/configs"
)

func TestProcessor_Update(t *testing.T) {
	tests := []struct {
		name           string
		folderName     string
		podcastID      string
		scannedEps     []*podcast.Episode
		scanErr        error
		existingEps    map[string]*podcast.Episode
		saveErr        error
		wantCount      int64
		wantErr        bool
		wantErrContain string
	}{
		{
			name:       "no episodes found",
			folderName: "podcast1",
			podcastID:  "pod1",
			scannedEps: []*podcast.Episode{},
			wantCount:  0,
		},
		{
			name:       "all new episodes saved",
			folderName: "podcast1",
			podcastID:  "pod1",
			scannedEps: []*podcast.Episode{
				{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
				{Filename: "ep2.mp3", Size: 2000, Status: podcast.New},
			},
			existingEps: map[string]*podcast.Episode{},
			wantCount:   2,
		},
		{
			name:       "skip existing episodes",
			folderName: "podcast1",
			podcastID:  "pod1",
			scannedEps: []*podcast.Episode{
				{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
				{Filename: "ep2.mp3", Size: 2000, Status: podcast.New},
			},
			existingEps: map[string]*podcast.Episode{
				"ep1.mp3": {Filename: "ep1.mp3", Size: 1000, Status: podcast.Uploaded},
			},
			wantCount: 1,
		},
		{
			name:       "scan error",
			folderName: "podcast1",
			podcastID:  "pod1",
			scanErr:    errors.New("scan failed"),
			wantErr:    true,
		},
		{
			name:       "save error",
			folderName: "podcast1",
			podcastID:  "pod1",
			scannedEps: []*podcast.Episode{
				{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
			},
			existingEps:    map[string]*podcast.Episode{},
			saveErr:        errors.New("db write failed"),
			wantErr:        true,
			wantErrContain: "can't add episode",
		},
		{
			name:       "nil episodes in scan result are skipped",
			folderName: "podcast1",
			podcastID:  "pod1",
			scannedEps: []*podcast.Episode{
				nil,
				{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
				nil,
			},
			existingEps: map[string]*podcast.Episode{},
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mocks.EpisodeStoreMock{
				GetEpisodeByFilenameFunc: func(podcastID string, fileName string) (*podcast.Episode, error) {
					if tt.existingEps == nil {
						return nil, nil
					}
					ep, ok := tt.existingEps[fileName]
					if !ok {
						return nil, errors.New("no episode found")
					}
					return ep, nil
				},
				SaveEpisodeFunc: func(podcastID string, episode *podcast.Episode) error {
					return tt.saveErr
				},
			}

			scanner := &mocks.FileScannerMock{
				FindEpisodesFunc: func(folderName string) ([]*podcast.Episode, error) {
					return tt.scannedEps, tt.scanErr
				},
			}

			p := &proc.Processor{
				Storage: store,
				Files:   scanner,
			}

			count, err := p.Update(context.Background(), tt.folderName, tt.podcastID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContain != "" {
					assert.Contains(t, err.Error(), tt.wantErrContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
		})
	}
}

func TestProcessor_DeleteOldEpisodesByPodcast(t *testing.T) {
	tests := []struct {
		name          string
		podcastID     string
		podcastFolder string
		episodes      []*podcast.Episode
		findErr       error
		deleteErr     error
		wantErr       bool
	}{
		{
			name:          "no episodes to delete",
			podcastID:     "pod1",
			podcastFolder: "folder1",
			episodes:      []*podcast.Episode{},
		},
		{
			name:          "successful delete",
			podcastID:     "pod1",
			podcastFolder: "folder1",
			episodes: []*podcast.Episode{
				{Filename: "ep1.mp3", Size: 1000, Status: podcast.Uploaded},
			},
		},
		{
			name:          "find error",
			podcastID:     "pod1",
			podcastFolder: "folder1",
			findErr:       errors.New("find failed"),
			wantErr:       true,
		},
		{
			name:          "delete error is logged but does not fail",
			podcastID:     "pod1",
			podcastFolder: "folder1",
			episodes: []*podcast.Episode{
				{Filename: "ep1.mp3", Size: 1000, Status: podcast.Uploaded},
			},
			deleteErr: errors.New("s3 delete failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mocks.EpisodeStoreMock{
				FindEpisodesByStatusFunc: func(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
					return tt.episodes, tt.findErr
				},
				GetEpisodeByFilenameFunc: func(podcastID string, fileName string) (*podcast.Episode, error) {
					for _, ep := range tt.episodes {
						if ep.Filename == fileName {
							return ep, nil
						}
					}
					return nil, errors.New("not found")
				},
				SaveEpisodeFunc: func(podcastID string, episode *podcast.Episode) error {
					return nil
				},
			}

			s3 := &mocks.ObjectStorageMock{
				DeleteEpisodeFunc: func(ctx context.Context, objectName string) error {
					return tt.deleteErr
				},
			}

			p := &proc.Processor{
				Storage:   store,
				S3Client:  s3,
				ChunkSize: 2,
			}

			err := p.DeleteOldEpisodesByPodcast(context.Background(), tt.podcastID, tt.podcastFolder)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.deleteErr == nil && len(tt.episodes) > 0 {
				saveCalls := store.SaveEpisodeCalls()
				assert.NotEmpty(t, saveCalls)
			}
		})
	}
}

func TestProcessor_RollbackLastEpisodes(t *testing.T) {
	tests := []struct {
		name      string
		podcastID string
		episode   *podcast.Episode
		findErr   error
		saveErr   error
		wantErr   bool
	}{
		{
			name:      "no episode found for rollback",
			podcastID: "pod1",
			episode:   nil,
		},
		{
			name:      "rollback single episode",
			podcastID: "pod1",
			episode:   &podcast.Episode{Filename: "ep1.mp3", Status: podcast.Uploaded},
		},
		{
			name:      "find error",
			podcastID: "pod1",
			findErr:   errors.New("find failed"),
			wantErr:   true,
		},
		{
			name:      "save error",
			podcastID: "pod1",
			episode:   &podcast.Episode{Filename: "ep1.mp3", Status: podcast.Uploaded},
			saveErr:   errors.New("save failed"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mocks.EpisodeStoreMock{
				GetLastEpisodeByNotStatusFunc: func(podcastID string, status podcast.Status) (*podcast.Episode, error) {
					return tt.episode, tt.findErr
				},
				SaveEpisodeFunc: func(podcastID string, episode *podcast.Episode) error {
					return tt.saveErr
				},
				FindEpisodesBySessionFunc: func(podcastID string, session string) ([]*podcast.Episode, error) {
					return nil, nil
				},
			}

			p := &proc.Processor{Storage: store}

			err := p.RollbackLastEpisodes(context.Background(), tt.podcastID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.episode != nil && tt.saveErr == nil {
				saveCalls := store.SaveEpisodeCalls()
				require.NotEmpty(t, saveCalls)
				assert.Equal(t, podcast.New, saveCalls[0].Episode.Status)
			}
		})
	}
}

func TestProcessor_RollbackEpisodesOfSession(t *testing.T) {
	tests := []struct {
		name      string
		podcastID string
		session   string
		episodes  []*podcast.Episode
		findErr   error
		saveErr   error
		wantErr   bool
	}{
		{
			name:      "no episodes in session",
			podcastID: "pod1",
			session:   "sess1",
			episodes:  []*podcast.Episode{},
		},
		{
			name:      "rollback session episodes",
			podcastID: "pod1",
			session:   "sess1",
			episodes: []*podcast.Episode{
				{Filename: "ep1.mp3", Status: podcast.Uploaded, Session: "sess1"},
				{Filename: "ep2.mp3", Status: podcast.Uploaded, Session: "sess1"},
			},
		},
		{
			name:      "find error",
			podcastID: "pod1",
			session:   "sess1",
			findErr:   errors.New("find failed"),
			wantErr:   true,
		},
		{
			name:      "save error stops rollback",
			podcastID: "pod1",
			session:   "sess1",
			episodes: []*podcast.Episode{
				{Filename: "ep1.mp3", Status: podcast.Uploaded, Session: "sess1"},
			},
			saveErr: errors.New("save failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mocks.EpisodeStoreMock{
				FindEpisodesBySessionFunc: func(podcastID string, session string) ([]*podcast.Episode, error) {
					return tt.episodes, tt.findErr
				},
				SaveEpisodeFunc: func(podcastID string, episode *podcast.Episode) error {
					return tt.saveErr
				},
			}

			p := &proc.Processor{Storage: store}

			err := p.RollbackEpisodesOfSession(context.Background(), tt.podcastID, tt.session)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestProcessor_UploadNewEpisodes(t *testing.T) {
	tests := []struct {
		name          string
		podcastID     string
		podcastFolder string
		session       string
		sizeLimit     int64
		episodes      []*podcast.Episode
		findErr       error
		uploadErr     error
		wantErr       bool
	}{
		{
			name:          "no episodes to upload",
			podcastID:     "pod1",
			podcastFolder: "folder1",
			session:       "sess1",
			sizeLimit:     100000,
			episodes:      []*podcast.Episode{},
		},
		{
			name:          "find error",
			podcastID:     "pod1",
			podcastFolder: "folder1",
			session:       "sess1",
			findErr:       errors.New("find failed"),
			wantErr:       true,
		},
		{
			name:          "successful upload and save",
			podcastID:     "pod1",
			podcastFolder: "folder1",
			session:       "sess1",
			sizeLimit:     100000,
			episodes: []*podcast.Episode{
				{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mocks.EpisodeStoreMock{
				FindEpisodesBySizeLimitFunc: func(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
					return tt.episodes, tt.findErr
				},
				GetEpisodeByFilenameFunc: func(podcastID string, fileName string) (*podcast.Episode, error) {
					for _, ep := range tt.episodes {
						if ep.Filename == fileName {
							epCopy := *ep
							return &epCopy, nil
						}
					}
					return nil, errors.New("not found")
				},
				SaveEpisodeFunc: func(podcastID string, episode *podcast.Episode) error {
					return nil
				},
			}

			s3 := &mocks.ObjectStorageMock{
				GetObjectInfoFunc: func(ctx context.Context, objectName string) (*proc.ObjectInfo, error) {
					return nil, errors.New("not found")
				},
				UploadEpisodeFunc: func(ctx context.Context, objectName string, filePath string) (*proc.UploadResult, error) {
					if tt.uploadErr != nil {
						return nil, tt.uploadErr
					}
					return &proc.UploadResult{Location: "https://s3/bucket/" + objectName}, nil
				},
			}

			p := &proc.Processor{
				Storage:     store,
				S3Client:    s3,
				StoragePath: "/tmp/storage",
				ChunkSize:   2,
			}

			err := p.UploadNewEpisodes(context.Background(), tt.session, tt.podcastID, tt.podcastFolder, tt.sizeLimit)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestProcessor_GetPodcastImage(t *testing.T) {
	tests := []struct {
		name         string
		folder       string
		imageFile    string
		objectInfo   *proc.ObjectInfo
		infoErr      error
		wantLocation string
	}{
		{
			name:         "successful get",
			folder:       "folder1",
			imageFile:    "cover.png",
			objectInfo:   &proc.ObjectInfo{Location: "https://s3/bucket/folder1/cover.png", Size: 500},
			wantLocation: "https://s3/bucket/folder1/cover.png",
		},
		{
			name:         "error returns empty",
			folder:       "folder1",
			imageFile:    "cover.png",
			infoErr:      errors.New("not found"),
			wantLocation: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3 := &mocks.ObjectStorageMock{
				GetObjectInfoFunc: func(ctx context.Context, objectName string) (*proc.ObjectInfo, error) {
					return tt.objectInfo, tt.infoErr
				},
			}

			p := &proc.Processor{S3Client: s3}
			result := p.GetPodcastImage(context.Background(), tt.folder, tt.imageFile)
			assert.Equal(t, tt.wantLocation, result)
		})
	}
}

func TestProcessor_UploadFeed(t *testing.T) {
	tests := []struct {
		name      string
		folder    string
		feedName  string
		uploadRes *proc.UploadResult
		uploadErr error
		wantNil   bool
	}{
		{
			name:      "successful upload",
			folder:    "folder1",
			feedName:  "feed.rss",
			uploadRes: &proc.UploadResult{Location: "https://s3/bucket/folder1/feed.rss"},
		},
		{
			name:      "upload error returns nil",
			folder:    "folder1",
			feedName:  "feed.rss",
			uploadErr: errors.New("upload failed"),
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3 := &mocks.ObjectStorageMock{
				UploadFeedFunc: func(ctx context.Context, objectName string, filePath string) (*proc.UploadResult, error) {
					return tt.uploadRes, tt.uploadErr
				},
			}

			p := &proc.Processor{
				S3Client:    s3,
				StoragePath: "/tmp/storage",
			}

			result := p.UploadFeed(context.Background(), tt.folder, tt.feedName)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.uploadRes.Location, result.Location)
			}
		})
	}
}

func TestProcessor_UploadNewEpisodes_WithProgress(t *testing.T) {
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
		{Filename: "ep2.mp3", Size: 2000, Status: podcast.New},
	}

	store := &mocks.EpisodeStoreMock{
		FindEpisodesBySizeLimitFunc: func(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
			return episodes, nil
		},
		GetEpisodeByFilenameFunc: func(podcastID string, fileName string) (*podcast.Episode, error) {
			for _, ep := range episodes {
				if ep.Filename == fileName {
					epCopy := *ep
					return &epCopy, nil
				}
			}
			return nil, errors.New("not found")
		},
		SaveEpisodeFunc: func(podcastID string, episode *podcast.Episode) error {
			return nil
		},
	}

	s3 := &mocks.ObjectStorageMock{
		GetObjectInfoFunc: func(ctx context.Context, objectName string) (*proc.ObjectInfo, error) {
			return nil, errors.New("not found")
		},
		UploadEpisodeFunc: func(ctx context.Context, objectName string, filePath string) (*proc.UploadResult, error) {
			return &proc.UploadResult{Location: "https://s3/bucket/" + objectName}, nil
		},
	}

	progress := &mocks.ProgressReporterMock{
		StartFileFunc:    func(workerID int, filename string, totalSize int64) {},
		CompleteFileFunc: func(workerID int, fileSize int64, err error) {},
		FinishFunc:       func() {},
	}

	p := &proc.Processor{
		Storage:     store,
		S3Client:    s3,
		Progress:    progress,
		StoragePath: "/tmp/storage",
		ChunkSize:   2,
	}

	err := p.UploadNewEpisodes(context.Background(), "sess1", "pod1", "folder1", 100000)
	require.NoError(t, err)

	startCalls := progress.StartFileCalls()
	completeCalls := progress.CompleteFileCalls()
	finishCalls := progress.FinishCalls()

	assert.Len(t, startCalls, 2)
	assert.Len(t, completeCalls, 2)
	assert.Len(t, finishCalls, 1)

	for _, c := range completeCalls {
		assert.NoError(t, c.Err)
	}
}

func TestProcessor_DeleteOldEpisodesByPodcast_WithProgress(t *testing.T) {
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.Uploaded},
	}

	store := &mocks.EpisodeStoreMock{
		FindEpisodesByStatusFunc: func(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
			return episodes, nil
		},
		GetEpisodeByFilenameFunc: func(podcastID string, fileName string) (*podcast.Episode, error) {
			for _, ep := range episodes {
				if ep.Filename == fileName {
					return ep, nil
				}
			}
			return nil, errors.New("not found")
		},
		SaveEpisodeFunc: func(podcastID string, episode *podcast.Episode) error {
			return nil
		},
	}

	s3 := &mocks.ObjectStorageMock{
		DeleteEpisodeFunc: func(ctx context.Context, objectName string) error {
			return nil
		},
	}

	progress := &mocks.ProgressReporterMock{
		StartFileFunc:    func(workerID int, filename string, totalSize int64) {},
		CompleteFileFunc: func(workerID int, fileSize int64, err error) {},
		FinishFunc:       func() {},
	}

	p := &proc.Processor{
		Storage:   store,
		S3Client:  s3,
		Progress:  progress,
		ChunkSize: 2,
	}

	err := p.DeleteOldEpisodesByPodcast(context.Background(), "pod1", "folder1")
	require.NoError(t, err)

	assert.Len(t, progress.StartFileCalls(), 1)
	assert.Len(t, progress.CompleteFileCalls(), 1)
	assert.Len(t, progress.FinishCalls(), 1)
	assert.Equal(t, "ep1.mp3", progress.StartFileCalls()[0].Filename)
	assert.NoError(t, progress.CompleteFileCalls()[0].Err)
}

func TestProcessor_UploadNewEpisodes_NoProgress(t *testing.T) {
	// Verify nil Progress field causes no panic
	episodes := []*podcast.Episode{
		{Filename: "ep1.mp3", Size: 1000, Status: podcast.New},
	}

	store := &mocks.EpisodeStoreMock{
		FindEpisodesBySizeLimitFunc: func(podcastID string, status podcast.Status, sizeLimit int64) ([]*podcast.Episode, error) {
			return episodes, nil
		},
		GetEpisodeByFilenameFunc: func(podcastID string, fileName string) (*podcast.Episode, error) {
			epCopy := *episodes[0]
			return &epCopy, nil
		},
		SaveEpisodeFunc: func(podcastID string, episode *podcast.Episode) error {
			return nil
		},
	}

	s3 := &mocks.ObjectStorageMock{
		GetObjectInfoFunc: func(ctx context.Context, objectName string) (*proc.ObjectInfo, error) {
			return nil, errors.New("not found")
		},
		UploadEpisodeFunc: func(ctx context.Context, objectName string, filePath string) (*proc.UploadResult, error) {
			return &proc.UploadResult{Location: "https://s3/bucket/" + objectName}, nil
		},
	}

	p := &proc.Processor{
		Storage:     store,
		S3Client:    s3,
		StoragePath: "/tmp/storage",
		ChunkSize:   2,
		// Progress is nil - should not panic
	}

	err := p.UploadNewEpisodes(context.Background(), "sess1", "pod1", "folder1", 100000)
	require.NoError(t, err)
}

func TestProcessor_GenerateFeed(t *testing.T) {
	t.Run("find episodes error", func(t *testing.T) {
		store := &mocks.EpisodeStoreMock{
			FindEpisodesByStatusFunc: func(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
				return nil, fmt.Errorf("db error")
			},
		}

		p := &proc.Processor{Storage: store}
		_, err := p.GenerateFeed(context.Background(), "pod1", configs.Podcast{Title: "Test"}, "https://img.png")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't find episodes")
	})

	t.Run("episode with full metadata uses title and metadata description", func(t *testing.T) {
		dir := t.TempDir()
		episodes := []*podcast.Episode{
			{
				Filename: "2024-01-01-ep1.mp3",
				PubDate:  "Mon, 01 Jan 2024 00:00:00 +0000",
				Size:     1000,
				Status:   podcast.Uploaded,
				Location: "https://s3/ep1.mp3",
				Title:    "My Episode Title",
				Artist:   "Test Artist",
				Album:    "Test Album",
				Year:     "2024",
				Comment:  "Episode comment",
				Duration: "01:23:45",
			},
		}

		store := &mocks.EpisodeStoreMock{
			FindEpisodesByStatusFunc: func(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
				return episodes, nil
			},
		}

		p := &proc.Processor{Storage: store, StoragePath: dir}
		podcastEntity := configs.Podcast{
			Title:  "My Podcast",
			Folder: "",
		}

		_, err := p.GenerateFeed(context.Background(), "pod1", podcastEntity, "https://img.png")
		require.NoError(t, err)

		// Read generated file to check content
		files, globErr := filepath.Glob(dir + "/*.rss")
		require.NoError(t, globErr)
		require.Len(t, files, 1)

		content, readErr := os.ReadFile(files[0])
		require.NoError(t, readErr)
		feedContent := string(content)

		assert.Contains(t, feedContent, "<title>My Episode Title</title>")
		assert.Contains(t, feedContent, "Test Artist - Test Album (2024)")
		assert.Contains(t, feedContent, "Episode comment")
		assert.Contains(t, feedContent, "<itunes:duration>01:23:45</itunes:duration>")
	})

	t.Run("episode without title falls back to filename", func(t *testing.T) {
		dir := t.TempDir()
		episodes := []*podcast.Episode{
			{
				Filename: "2024-01-01-ep1.mp3",
				PubDate:  "Mon, 01 Jan 2024 00:00:00 +0000",
				Size:     1000,
				Status:   podcast.Uploaded,
				Location: "https://s3/ep1.mp3",
			},
		}

		store := &mocks.EpisodeStoreMock{
			FindEpisodesByStatusFunc: func(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
				return episodes, nil
			},
		}

		p := &proc.Processor{Storage: store, StoragePath: dir}
		_, err := p.GenerateFeed(context.Background(), "pod1", configs.Podcast{Title: "My Podcast"}, "https://img.png")
		require.NoError(t, err)

		files, globErr := filepath.Glob(dir + "/*.rss")
		require.NoError(t, globErr)
		require.Len(t, files, 1)

		content, readErr := os.ReadFile(files[0])
		require.NoError(t, readErr)
		feedContent := string(content)

		assert.Contains(t, feedContent, "<title>2024-01-01-ep1.mp3</title>")
		assert.Contains(t, feedContent, "<![CDATA[2024-01-01-ep1.mp3]]>")
		assert.NotContains(t, feedContent, "<itunes:duration>")
	})

	t.Run("episode without duration omits itunes:duration", func(t *testing.T) {
		dir := t.TempDir()
		episodes := []*podcast.Episode{
			{
				Filename: "ep1.mp3",
				Size:     500,
				Status:   podcast.Uploaded,
				Location: "https://s3/ep1.mp3",
				Title:    "EP 1",
				Artist:   "Artist",
			},
		}

		store := &mocks.EpisodeStoreMock{
			FindEpisodesByStatusFunc: func(podcastID string, status podcast.Status) ([]*podcast.Episode, error) {
				return episodes, nil
			},
		}

		p := &proc.Processor{Storage: store, StoragePath: dir}
		_, err := p.GenerateFeed(context.Background(), "pod1", configs.Podcast{Title: "Pod"}, "https://img.png")
		require.NoError(t, err)

		files, _ := filepath.Glob(dir + "/*.rss")
		require.Len(t, files, 1)
		content, _ := os.ReadFile(files[0])
		assert.NotContains(t, string(content), "<itunes:duration>")
	})
}

func TestBuildItemDescription(t *testing.T) {
	tests := []struct {
		name     string
		episode  podcast.Episode
		wantDesc string
	}{
		{
			name:     "all metadata",
			episode:  podcast.Episode{Filename: "ep.mp3", Artist: "A", Album: "B", Year: "2024", Comment: "C"},
			wantDesc: "A - B (2024)\nC",
		},
		{
			name:     "artist and album, no year",
			episode:  podcast.Episode{Filename: "ep.mp3", Artist: "A", Album: "B"},
			wantDesc: "A - B",
		},
		{
			name:     "artist only",
			episode:  podcast.Episode{Filename: "ep.mp3", Artist: "A"},
			wantDesc: "A",
		},
		{
			name:     "album and year only",
			episode:  podcast.Episode{Filename: "ep.mp3", Album: "B", Year: "2024"},
			wantDesc: "B (2024)",
		},
		{
			name:     "album only",
			episode:  podcast.Episode{Filename: "ep.mp3", Album: "B"},
			wantDesc: "B",
		},
		{
			name:     "year only",
			episode:  podcast.Episode{Filename: "ep.mp3", Year: "2024"},
			wantDesc: "2024",
		},
		{
			name:     "comment only",
			episode:  podcast.Episode{Filename: "ep.mp3", Comment: "C"},
			wantDesc: "C",
		},
		{
			name:     "no metadata falls back to filename",
			episode:  podcast.Episode{Filename: "ep.mp3"},
			wantDesc: "ep.mp3",
		},
		{
			name:     "artist and comment, no album/year",
			episode:  podcast.Episode{Filename: "ep.mp3", Artist: "A", Comment: "C"},
			wantDesc: "A\nC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proc.BuildItemDescription(&tt.episode)
			assert.Equal(t, tt.wantDesc, result)
		})
	}
}
