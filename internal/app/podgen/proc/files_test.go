package proc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"podgen/internal/app/podgen/podcast"
)

// writeTaggedMP3 creates an MP3 file with ID3 tags at the given path.
func writeTaggedMP3(t *testing.T, path, title, artist, album, year, comment string) {
	t.Helper()

	f, err := os.Create(path) //nolint:gosec // test helper creating file in t.TempDir()
	require.NoError(t, err)
	_ = f.Close()

	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	require.NoError(t, err)

	tag.SetTitle(title)
	tag.SetArtist(artist)
	tag.SetAlbum(album)
	tag.SetYear(year)

	if comment != "" {
		tag.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "",
			Text:        comment,
		})
	}

	require.NoError(t, tag.Save())
	_ = tag.Close()
}

func TestFindEpisodes_MetadataPopulated(t *testing.T) {
	storage := t.TempDir()
	podcast1 := filepath.Join(storage, "mypodcast")
	require.NoError(t, os.MkdirAll(podcast1, 0o750))

	mp3Path := filepath.Join(podcast1, "episode-one.mp3")
	writeTaggedMP3(t, mp3Path, "Episode One", "Host Name", "Season 1", "2023", "Great episode")

	f := &Files{Storage: storage}
	episodes, err := f.FindEpisodes("mypodcast")
	require.NoError(t, err)

	var ep *podcast.Episode
	for _, e := range episodes {
		if e != nil && e.Filename == "episode-one.mp3" {
			ep = e
			break
		}
	}
	require.NotNil(t, ep, "expected to find episode-one.mp3")

	assert.Equal(t, "Episode One", ep.Title)
	assert.Equal(t, "Host Name", ep.Artist)
	assert.Equal(t, "Season 1", ep.Album)
	assert.Equal(t, "2023", ep.Year)
	assert.Equal(t, "Great episode", ep.Comment)
}

func TestFindEpisodes_ID3YearUsedForPubDate(t *testing.T) {
	storage := t.TempDir()
	podcast1 := filepath.Join(storage, "mypodcast")
	require.NoError(t, os.MkdirAll(podcast1, 0o750))

	// Filename has no date; ID3 Year should be used.
	mp3Path := filepath.Join(podcast1, "no-date-in-name.mp3")
	writeTaggedMP3(t, mp3Path, "", "", "", "2021", "")

	f := &Files{Storage: storage}
	episodes, err := f.FindEpisodes("mypodcast")
	require.NoError(t, err)

	var ep *podcast.Episode
	for _, e := range episodes {
		if e != nil && e.Filename == "no-date-in-name.mp3" {
			ep = e
			break
		}
	}
	require.NotNil(t, ep)
	assert.Contains(t, ep.PubDate, "2021")
}

func TestFindEpisodes_FilenameDateFallback(t *testing.T) {
	storage := t.TempDir()
	podcast1 := filepath.Join(storage, "mypodcast")
	require.NoError(t, os.MkdirAll(podcast1, 0o750))

	// No ID3 Year; filename has date.
	mp3Path := filepath.Join(podcast1, "2022-05-10-episode.mp3")
	writeTaggedMP3(t, mp3Path, "", "", "", "", "")

	f := &Files{Storage: storage}
	episodes, err := f.FindEpisodes("mypodcast")
	require.NoError(t, err)

	var ep *podcast.Episode
	for _, e := range episodes {
		if e != nil && e.Filename == "2022-05-10-episode.mp3" {
			ep = e
			break
		}
	}
	require.NotNil(t, ep)
	assert.Contains(t, ep.PubDate, "2022")
	assert.Contains(t, ep.PubDate, "May")
}

func TestFindEpisodes_SkipsNonMP3(t *testing.T) {
	storage := t.TempDir()
	podcast1 := filepath.Join(storage, "mypodcast")
	require.NoError(t, os.MkdirAll(podcast1, 0o750))

	require.NoError(t, os.WriteFile(filepath.Join(podcast1, "readme.txt"), []byte("hi"), 0o600))
	mp3Path := filepath.Join(podcast1, "valid.mp3")
	writeTaggedMP3(t, mp3Path, "Valid", "", "", "2020", "")

	f := &Files{Storage: storage}
	episodes, err := f.FindEpisodes("mypodcast")
	require.NoError(t, err)

	var count int
	for _, e := range episodes {
		if e != nil {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestFindEpisodes_EmptyFolder(t *testing.T) {
	storage := t.TempDir()
	podcast1 := filepath.Join(storage, "empty")
	require.NoError(t, os.MkdirAll(podcast1, 0o750))

	f := &Files{Storage: storage}
	episodes, err := f.FindEpisodes("empty")
	require.NoError(t, err)
	assert.Empty(t, episodes)
}
