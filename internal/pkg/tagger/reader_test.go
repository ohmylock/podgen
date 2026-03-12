package tagger

import (
	"os"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTaggedMP3 creates a temp file with the given ID3 tags and returns its path.
// The caller is responsible for removing the file.
func createTaggedMP3(t *testing.T, title, artist, album, year, comment string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "test-*.mp3")
	require.NoError(t, err)
	name := f.Name()
	_ = f.Close()

	tag, err := id3v2.Open(name, id3v2.Options{Parse: true})
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

	return name
}

func TestReadMetadata_AllFields(t *testing.T) {
	path := createTaggedMP3(t, "My Title", "My Artist", "My Album", "2024", "A comment")

	m, err := ReadMetadata(path)
	require.NoError(t, err)

	assert.Equal(t, "My Title", m.Title)
	assert.Equal(t, "My Artist", m.Artist)
	assert.Equal(t, "My Album", m.Album)
	assert.Equal(t, "2024", m.Year)
	assert.Equal(t, "A comment", m.Comment)
}

func TestReadMetadata_EmptyTags(t *testing.T) {
	path := createTaggedMP3(t, "", "", "", "", "")

	m, err := ReadMetadata(path)
	require.NoError(t, err)

	assert.Equal(t, Metadata{}, m)
}

func TestReadMetadata_MissingComment(t *testing.T) {
	path := createTaggedMP3(t, "Title Only", "Artist", "Album", "2023", "")

	m, err := ReadMetadata(path)
	require.NoError(t, err)

	assert.Equal(t, "Title Only", m.Title)
	assert.Equal(t, "", m.Comment)
}

func TestReadMetadata_FileNotFound(t *testing.T) {
	_, err := ReadMetadata("/nonexistent/path/file.mp3")
	assert.Error(t, err)
}
