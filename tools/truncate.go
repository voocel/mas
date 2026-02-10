package tools

import (
	"fmt"
	"strings"
)

// Truncation limits defaults.
const (
	defaultMaxLines = 2000
	defaultMaxBytes = 50 * 1024 // 50KB
)

// truncateHead keeps the first N lines/bytes (for file reads).
// Returns truncated content, total line count, output line count, and whether truncation occurred.
func truncateHead(content string, maxLines, maxBytes int) (output string, totalLines, outputLines int, truncated bool) {
	if maxLines <= 0 {
		maxLines = defaultMaxLines
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}

	lines := strings.Split(content, "\n")
	totalLines = len(lines)

	if totalLines <= maxLines && len(content) <= maxBytes {
		return content, totalLines, totalLines, false
	}

	var kept []string
	byteCount := 0

	for i, line := range lines {
		lineBytes := len(line)
		if i > 0 {
			lineBytes++ // newline
		}
		if byteCount+lineBytes > maxBytes {
			break
		}
		if len(kept) >= maxLines {
			break
		}
		kept = append(kept, line)
		byteCount += lineBytes
	}

	return strings.Join(kept, "\n"), totalLines, len(kept), true
}

// truncateTail keeps the last N lines/bytes (for bash output).
// Returns truncated content, total line count, output line count, and whether truncation occurred.
func truncateTail(content string, maxLines, maxBytes int) (output string, totalLines, outputLines int, truncated bool) {
	if maxLines <= 0 {
		maxLines = defaultMaxLines
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}

	lines := strings.Split(content, "\n")
	totalLines = len(lines)

	if totalLines <= maxLines && len(content) <= maxBytes {
		return content, totalLines, totalLines, false
	}

	// Work backwards
	var kept []string
	byteCount := 0

	for i := len(lines) - 1; i >= 0 && len(kept) < maxLines; i-- {
		line := lines[i]
		lineBytes := len(line)
		if len(kept) > 0 {
			lineBytes++ // newline
		}
		if byteCount+lineBytes > maxBytes {
			break
		}
		kept = append([]string{line}, kept...)
		byteCount += lineBytes
	}

	return strings.Join(kept, "\n"), totalLines, len(kept), true
}

func formatSize(bytes int) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
}
