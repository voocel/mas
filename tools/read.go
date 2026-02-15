package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/voocel/agentcore/schema"
)

// ReadTool reads file contents with optional offset and limit.
// Applies head truncation (2000 lines / 50KB).
type ReadTool struct{}

func NewRead() *ReadTool { return &ReadTool{} }

func (t *ReadTool) Name() string  { return "read" }
func (t *ReadTool) Label() string { return "Read File" }
func (t *ReadTool) Description() string {
	return fmt.Sprintf(
		"Read file contents. Output is truncated to %d lines or %s (whichever is hit first). Use offset/limit for large files.",
		defaultMaxLines, formatSize(defaultMaxBytes),
	)
}
func (t *ReadTool) Schema() map[string]any {
	return schema.Object(
		schema.Property("path", schema.String("Path to the file to read (relative or absolute)")).Required(),
		schema.Property("offset", schema.Int("Line number to start reading from (1-based, default: 1)")),
		schema.Property("limit", schema.Int("Maximum number of lines to read")),
	)
}

type readArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

func (t *ReadTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a readArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	data, err := os.ReadFile(a.Path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", a.Path, err)
	}

	allLines := strings.Split(string(data), "\n")
	totalFileLines := len(allLines)

	// Apply offset (1-based)
	startLine := 0
	if a.Offset > 0 {
		startLine = a.Offset - 1
	}
	if startLine >= len(allLines) {
		return nil, fmt.Errorf("offset %d is beyond end of file (%d lines)", a.Offset, totalFileLines)
	}

	lines := allLines[startLine:]

	// Apply user limit if specified
	userLimited := false
	if a.Limit > 0 && a.Limit < len(lines) {
		lines = lines[:a.Limit]
		userLimited = true
	}

	content := strings.Join(lines, "\n")

	// Apply head truncation
	truncated, _, outputLines, wasTruncated := truncateHead(content, defaultMaxLines, defaultMaxBytes)

	// Format with line numbers
	truncatedLines := strings.Split(truncated, "\n")
	var sb strings.Builder
	for i, line := range truncatedLines {
		fmt.Fprintf(&sb, "%d\t%s\n", startLine+i+1, line)
	}

	result := sb.String()

	// Add truncation notice
	if wasTruncated {
		endLine := startLine + outputLines
		nextOffset := endLine + 1
		result += fmt.Sprintf("\n[Showing lines %d-%d of %d. Use offset=%d to continue.]",
			startLine+1, endLine, totalFileLines, nextOffset)
	} else if userLimited && startLine+a.Limit < totalFileLines {
		remaining := totalFileLines - (startLine + a.Limit)
		nextOffset := startLine + a.Limit + 1
		result += fmt.Sprintf("\n[%d more lines in file. Use offset=%d to continue.]",
			remaining, nextOffset)
	}

	return json.Marshal(result)
}
