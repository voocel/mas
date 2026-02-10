package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteTool writes content to a file, creating directories as needed.
type WriteTool struct{}

func NewWrite() *WriteTool { return &WriteTool{} }

func (t *WriteTool) Name() string        { return "write" }
func (t *WriteTool) Description() string { return "Write content to a file. Creates parent directories if needed. Overwrites existing files." }
func (t *WriteTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

type writeArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *WriteTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a writeArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(a.Path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	if err := os.WriteFile(a.Path, []byte(a.Content), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", a.Path, err)
	}

	return json.Marshal(fmt.Sprintf("wrote %d bytes to %s", len(a.Content), a.Path))
}
