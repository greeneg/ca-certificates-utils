package pluginUtils

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type PluginUtils struct {
	RequiredTools []string
}

type ProgressWriter struct {
	writer    io.Writer
	total     int64
	written   int64
	lastPrint time.Time
	startTime time.Time
	mu        sync.Mutex
}

func NewPluginUtils() PluginUtils {
	p := PluginUtils{
		RequiredTools: []string{"trust", "certutil"},
	}
	return p
}

func NewProgressWriter(writer io.Writer, total int64) *ProgressWriter {
	return &ProgressWriter{
		writer:    writer,
		total:     total,
		startTime: time.Now(),
		lastPrint: time.Now(),
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if n > 0 {
		pw.mu.Lock()
		pw.written += int64(n)
		// Update display every 100ms to avoid spamming the terminal
		if time.Since(pw.lastPrint) >= 100*time.Millisecond {
			pw.printProgress()
			pw.lastPrint = time.Now()
		}
		pw.mu.Unlock()
	}
	return n, err
}

func (pw *ProgressWriter) printProgress() {
	elapsed := time.Since(pw.startTime)
	speed := float64(pw.written) / elapsed.Seconds()

	if pw.total > 0 {
		percent := float64(pw.written) / float64(pw.total) * 100
		// don't use blanking since it does not work well on all terminals.
		// Instead, just print spaces at the end to overwrite any leftover characters
		fmt.Fprintf(os.Stderr, "\rDownloading: %.2f%% (%s / %s) - %s/s   ",
			percent,
			pw.formatBytes(pw.written),
			pw.formatBytes(pw.total),
			pw.formatBytes(int64(speed)))
	} else {
		fmt.Fprintf(os.Stderr, "\rDownloading: %s - %s/s   ",
			pw.formatBytes(pw.written),
			pw.formatBytes(int64(speed)))
	}
}

// Finish prints the final progress line with a newline.
func (pw *ProgressWriter) Finish() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.printProgress()
	fmt.Fprintln(os.Stderr)
}

// formatBytes formats a byte count into a human-readable string.
func (pw *ProgressWriter) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
