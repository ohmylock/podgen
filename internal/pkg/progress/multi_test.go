package progress

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderBar(t *testing.T) {
	tests := []struct {
		pct      int
		width    int
		expected string
	}{
		{0, 10, "░░░░░░░░░░"},
		{100, 10, "██████████"},
		{50, 10, "█████░░░░░"},
		{50, 4, "██░░"},
		{110, 10, "██████████"}, // clamped to 100
		{-5, 10, "░░░░░░░░░░"},  // clamped to 0
	}

	for _, tt := range tests {
		got := renderBar(tt.pct, tt.width)
		assert.Equal(t, tt.expected, got, "pct=%d width=%d", tt.pct, tt.width)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.bytes)
		assert.Equal(t, tt.expected, got, "bytes=%d", tt.bytes)
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hel.."},
		{"ab", 2, "ab"},
		{"abc", 2, "ab"},
		{"ab", 1, "a"},
		{"тест строка", 5, "тес.."},
	}

	for _, tt := range tests {
		got := truncateString(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, got, "input=%q maxLen=%d", tt.input, tt.maxLen)
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"hi", 5, "hi   "},
		{"hello", 5, "hello"},
		{"toolong", 3, "toolong"},
		{"тест", 6, "тест  "},
	}

	for _, tt := range tests {
		got := padRight(tt.input, tt.width)
		assert.Equal(t, tt.expected, got, "input=%q width=%d", tt.input, tt.width)
	}
}

func TestNewMulti(t *testing.T) {
	var buf strings.Builder
	m := NewMulti(&buf, 2, 5)

	assert.NotNil(t, m)
	assert.Equal(t, 2, len(m.workers))
	assert.Equal(t, 5, m.total)
}

func TestMultiStartFile(t *testing.T) {
	var buf strings.Builder
	m := NewMulti(&buf, 2, 3)

	m.StartFile(0, "episode.mp3", 1024*1024)

	m.mu.Lock()
	w := m.workers[0]
	m.mu.Unlock()

	assert.True(t, w.Active)
	assert.Equal(t, "episode.mp3", w.Filename)
	assert.Equal(t, int64(1024*1024), w.Total)
}

func TestMultiStartFileInvalidWorker(t *testing.T) {
	var buf strings.Builder
	m := NewMulti(&buf, 2, 3)

	// Should not panic on invalid worker IDs
	m.StartFile(-1, "file.mp3", 100)
	m.StartFile(99, "file.mp3", 100)
}

func TestMultiCompleteFile(t *testing.T) {
	var buf strings.Builder
	m := NewMulti(&buf, 2, 3)

	m.StartFile(0, "episode.mp3", 1024)
	m.CompleteFile(0, 1024, nil)

	m.mu.Lock()
	completed := m.completed
	totalBytes := m.totalBytes
	active := m.workers[0].Active
	m.mu.Unlock()

	assert.Equal(t, 1, completed)
	assert.Equal(t, int64(1024), totalBytes)
	assert.False(t, active)
}

func TestMultiCompleteFileWithError(t *testing.T) {
	var buf strings.Builder
	m := NewMulti(&buf, 2, 3)

	m.StartFile(0, "episode.mp3", 1024)
	m.CompleteFile(0, 0, assert.AnError)

	m.mu.Lock()
	errors := m.errors
	totalBytes := m.totalBytes
	m.mu.Unlock()

	assert.Equal(t, 1, errors)
	assert.Equal(t, int64(0), totalBytes)
}

func TestMultiUpdateProgress(t *testing.T) {
	var buf strings.Builder
	m := NewMulti(&buf, 2, 3)

	m.StartFile(0, "episode.mp3", 1024)
	m.UpdateProgress(0, 512, 1024)

	m.mu.Lock()
	w := m.workers[0]
	m.mu.Unlock()

	assert.Equal(t, int64(512), w.Uploaded)
	assert.Equal(t, int64(1024), w.Total)
}

func TestMultiFinish(t *testing.T) {
	var buf strings.Builder
	m := NewMulti(&buf, 1, 1)

	m.StartFile(0, "file.mp3", 100)
	// Should not panic
	m.Finish()
}

func TestMultiRenderContainsProgress(t *testing.T) {
	var buf strings.Builder
	m := NewMulti(&buf, 1, 2)

	m.StartFile(0, "podcast.mp3", 2048)

	output := buf.String()
	// Should contain total progress line
	assert.Contains(t, output, "Total:")
	assert.Contains(t, output, "0/2")
}
