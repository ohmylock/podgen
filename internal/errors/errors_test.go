package errors

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEpisodeError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *EpisodeError
		want string
	}{
		{
			name: "with filename",
			err: &EpisodeError{
				PodcastID: "podcast1",
				Filename:  "ep01.mp3",
				Op:        "SaveEpisode",
				Err:       ErrNoBucket,
			},
			want: "op=SaveEpisode podcast=podcast1 file=ep01.mp3: no bucket",
		},
		{
			name: "without filename",
			err: &EpisodeError{
				PodcastID: "podcast1",
				Op:        "FindEpisodesBySession",
				Err:       ErrNoBucket,
			},
			want: "op=FindEpisodesBySession podcast=podcast1: no bucket",
		},
		{
			name: "episode not found",
			err: &EpisodeError{
				PodcastID: "mypodcast",
				Filename:  "ep02.mp3",
				Op:        "GetEpisodeByFilename",
				Err:       ErrEpisodeNotFound,
			},
			want: "op=GetEpisodeByFilename podcast=mypodcast file=ep02.mp3: episode not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestEpisodeError_Unwrap(t *testing.T) {
	err := &EpisodeError{
		PodcastID: "podcast1",
		Op:        "ChangeStatusEpisodes",
		Err:       ErrNoBucket,
	}

	assert.True(t, stderrors.Is(err, ErrNoBucket))
	assert.False(t, stderrors.Is(err, ErrEpisodeNotFound))
}

func TestEpisodeError_As(t *testing.T) {
	wrapped := &EpisodeError{
		PodcastID: "p1",
		Op:        "GetLastEpisodeByStatus",
		Err:       ErrNoBucket,
	}
	outerErr := fmt.Errorf("operation failed: %w", wrapped)

	var episodeErr *EpisodeError
	require.True(t, stderrors.As(outerErr, &episodeErr))
	assert.Equal(t, "p1", episodeErr.PodcastID)
	assert.Equal(t, "GetLastEpisodeByStatus", episodeErr.Op)
}

func TestSentinelErrors(t *testing.T) {
	assert.Error(t, ErrNoBucket)
	assert.Error(t, ErrEpisodeNotFound)
	assert.NotEqual(t, ErrNoBucket, ErrEpisodeNotFound)
	assert.Equal(t, "no bucket", ErrNoBucket.Error())
	assert.Equal(t, "episode not found", ErrEpisodeNotFound.Error())
}
