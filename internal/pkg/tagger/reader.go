// Package tagger reads ID3 metadata from MP3 files.
package tagger

import (
	"fmt"
	"os"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/tcolgate/mp3"
)

// Metadata holds ID3 tag fields extracted from an MP3 file.
type Metadata struct {
	Title    string
	Artist   string
	Album    string
	Year     string
	Comment  string
	Duration string // iTunes duration format: HH:MM:SS or MM:SS
}

// ReadMetadata opens the given MP3 file, reads its ID3 tags and audio duration,
// and returns the extracted Metadata. Missing or empty tags are returned as empty strings.
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

	// Extract duration from MP3 audio data
	m.Duration = readDuration(filePath)

	return m, nil
}

// readDuration reads MP3 file and returns duration in iTunes format (HH:MM:SS or MM:SS).
// Returns empty string if duration cannot be determined.
func readDuration(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	d := mp3.NewDecoder(f)
	var totalDuration time.Duration
	var frame mp3.Frame
	skipped := 0

	for {
		if err := d.Decode(&frame, &skipped); err != nil {
			break
		}
		totalDuration += frame.Duration()
	}

	if totalDuration == 0 {
		return ""
	}

	return formatDuration(totalDuration)
}

// formatDuration formats a duration as HH:MM:SS or MM:SS for iTunes.
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
