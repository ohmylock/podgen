// Package tagger reads ID3 metadata from MP3 files.
package tagger

import (
	"github.com/bogem/id3v2/v2"
)

// Metadata holds ID3 tag fields extracted from an MP3 file.
type Metadata struct {
	Title   string
	Artist  string
	Album   string
	Year    string
	Comment string
}

// ReadMetadata opens the given MP3 file, reads its ID3 tags, and returns the
// extracted Metadata. Missing or empty tags are returned as empty strings.
// Non-fatal parse errors (e.g. no tag present) are silently ignored.
func ReadMetadata(filePath string) (Metadata, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return Metadata{}, err
	}
	defer tag.Close()

	m := Metadata{
		Title:  tag.Title(),
		Artist: tag.Artist(),
		Album:  tag.Album(),
		Year:   tag.Year(),
	}

	// Comments are stored as COMM frames; take text from the first one found.
	frames := tag.GetFrames(tag.CommonID("Comments"))
	for _, f := range frames {
		cf, ok := f.(id3v2.CommentFrame)
		if ok && cf.Text != "" {
			m.Comment = cf.Text
			break
		}
	}

	return m, nil
}
