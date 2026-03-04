package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	filenameWidth = 40
	barWidth      = 20
	sizeWidth     = 18 // "999.9MB/999.9MB"
)

// WorkerState represents the current state of an upload worker.
type WorkerState struct {
	Active    bool
	Filename  string
	Uploaded  int64
	Total     int64
	StartTime time.Time
}

// Multi manages multi-line progress display for concurrent uploads.
type Multi struct {
	mu           sync.Mutex
	out          io.Writer
	workers      []WorkerState
	completed    int
	total        int
	totalBytes   int64 // bytes uploaded so far
	errors       int
	startTime    time.Time
	lastRender   time.Time
	linesWritten int
}

// NewMulti creates a new multi-progress display.
func NewMulti(out io.Writer, workerCount, totalTasks int) *Multi {
	return &Multi{
		out:       out,
		workers:   make([]WorkerState, workerCount),
		total:     totalTasks,
		startTime: time.Now(),
	}
}

// StartFile marks a worker as starting a new file upload.
func (m *Multi) StartFile(workerID int, filename string, totalSize int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if workerID >= 0 && workerID < len(m.workers) {
		m.workers[workerID] = WorkerState{
			Active:    true,
			Filename:  filename,
			Total:     totalSize,
			StartTime: time.Now(),
		}
	}
	m.render()
}

// UpdateProgress updates the upload progress for a worker.
func (m *Multi) UpdateProgress(workerID int, uploaded, total int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if workerID >= 0 && workerID < len(m.workers) {
		m.workers[workerID].Uploaded = uploaded
		if total > 0 {
			m.workers[workerID].Total = total
		}
	}

	if time.Since(m.lastRender) > 100*time.Millisecond {
		m.render()
	}
}

// CompleteFile marks a file as completed.
func (m *Multi) CompleteFile(workerID int, fileSize int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if workerID >= 0 && workerID < len(m.workers) {
		m.workers[workerID].Active = false
	}

	m.completed++
	if err != nil {
		m.errors++
	} else {
		m.totalBytes += fileSize
	}
	m.render()
}

// Finish clears the progress display.
func (m *Multi) Finish() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clearLines()
}

// render updates the terminal display.
func (m *Multi) render() {
	m.clearLines()

	var lines []string

	pct := 0
	if m.total > 0 {
		pct = m.completed * 100 / m.total
	}
	elapsed := time.Since(m.startTime).Truncate(time.Second)

	bar := renderBar(pct, 30)
	statusParts := []string{
		fmt.Sprintf("%d/%d", m.completed, m.total),
		formatBytes(m.totalBytes),
		elapsed.String(),
	}
	if m.errors > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d errors", m.errors))
	}

	lines = append(lines, fmt.Sprintf("\033[1mTotal:\033[0m %s %3d%% │ %s",
		bar, pct, strings.Join(statusParts, " │ ")))

	lines = append(lines, strings.Repeat("─", 80))

	for i, w := range m.workers {
		line := m.formatWorker(i, w)
		lines = append(lines, line)
	}

	for _, line := range lines {
		fmt.Fprintln(m.out, line)
	}
	m.linesWritten = len(lines)
	m.lastRender = time.Now()
}

// clearLines moves cursor up and clears previous output.
func (m *Multi) clearLines() {
	for i := 0; i < m.linesWritten; i++ {
		fmt.Fprint(m.out, "\033[A\033[2K")
	}
}

// formatWorker formats a single worker's progress line.
func (m *Multi) formatWorker(id int, w WorkerState) string {
	if !w.Active {
		idleName := padRight("idle", filenameWidth)
		return fmt.Sprintf("\033[90m[%d] %s %s %3s │ %-*s\033[0m",
			id+1, idleName, renderBar(0, barWidth), "-",
			sizeWidth, "-")
	}

	name := truncateString(w.Filename, filenameWidth)
	name = padRight(name, filenameWidth)

	pct := 0
	if w.Total > 0 {
		pct = int(w.Uploaded * 100 / w.Total)
	}

	bar := renderBar(pct, barWidth)
	sizeInfo := fmt.Sprintf("%s/%s", formatBytes(w.Uploaded), formatBytes(w.Total))

	return fmt.Sprintf("[%d] %s %s %3d%% │ %-*s",
		id+1, name, bar, pct, sizeWidth, sizeInfo)
}

// renderBar generates a Unicode progress bar.
func renderBar(pct, width int) string {
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}

	filled := pct * width / 100
	empty := width - filled

	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// truncateString truncates a string to maxLen runes, adding ellipsis if needed.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 2 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-2]) + ".."
}

// padRight pads string to width based on visual width (rune count).
func padRight(s string, width int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		return s
	}
	return s + strings.Repeat(" ", width-runeCount)
}

// formatBytes formats bytes as human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	if exp >= len("KMGTPE") {
		exp = len("KMGTPE") - 1
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}
