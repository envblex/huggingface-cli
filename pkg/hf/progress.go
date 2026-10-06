package hf

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// ProgressWriter wraps an io.Writer and tracks progress for a download.
type ProgressWriter struct {
	Writer     io.Writer
	Total      int64
	Current    int64
	Filename   string
	Quiet      bool
	mu         sync.Mutex
	lastUpdate time.Time
}

func NewProgressWriter(w io.Writer, total int64, filename string, quiet bool) *ProgressWriter {
	return &ProgressWriter{
		Writer:   w,
		Total:    total,
		Filename: filename,
		Quiet:    quiet || HFHubDisableProgressBars(),
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	if n > 0 {
		pw.mu.Lock()
		pw.Current += int64(n)
		now := time.Now()
		if !pw.Quiet && (now.Sub(pw.lastUpdate) > 150*time.Millisecond || (pw.Total > 0 && pw.Current >= pw.Total)) {
			pw.lastUpdate = now
			pw.render()
		}
		pw.mu.Unlock()
	}
	return n, err
}

func (pw *ProgressWriter) Finish() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if !pw.Quiet && pw.Current > 0 {
		pw.render()
		fmt.Fprintln(os.Stderr)
	}
}

func (pw *ProgressWriter) render() {
	var percent float64
	if pw.Total > 0 {
		percent = float64(pw.Current) / float64(pw.Total) * 100
	}
	curStr := FormatSize(pw.Current)
	if pw.Total > 0 {
		totalStr := FormatSize(pw.Total)
		width := 25
		filled := int(float64(width) * (float64(pw.Current) / float64(pw.Total)))
		if filled > width {
			filled = width
		}
		bar := ""
		for i := 0; i < filled; i++ {
			bar += "█"
		}
		for i := filled; i < width; i++ {
			bar += " "
		}
		fmt.Fprintf(os.Stderr, "\r%-25s %3.0f%%|%s| %s/%s", pw.Filename, percent, bar, curStr, totalStr)
	} else {
		fmt.Fprintf(os.Stderr, "\r%-25s %s downloaded", pw.Filename, curStr)
	}
}
