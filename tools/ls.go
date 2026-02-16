package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/voocel/agentcore/schema"
)

// LsTool lists directory contents with optional depth control.
type LsTool struct {
	WorkDir string
}

func NewLs(workDir string) *LsTool { return &LsTool{WorkDir: workDir} }

func (t *LsTool) Name() string  { return "ls" }
func (t *LsTool) Label() string { return "List Directory" }
func (t *LsTool) Description() string {
	return "List directory contents. Returns file/directory names with sizes. Depth controls recursive listing (default 1, max 5)."
}
func (t *LsTool) Schema() map[string]any {
	return schema.Object(
		schema.Property("path", schema.String("Directory path (default: working directory)")),
		schema.Property("depth", schema.Int("Recursion depth (default: 1, max: 5)")),
	)
}

type lsArgs struct {
	Path  string `json:"path"`
	Depth int    `json:"depth"`
}

const lsMaxEntries = 200

func (t *LsTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a lsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	dir := t.WorkDir
	if a.Path != "" {
		if filepath.IsAbs(a.Path) {
			dir = a.Path
		} else {
			dir = filepath.Join(t.WorkDir, a.Path)
		}
	}

	depth := a.Depth
	if depth <= 0 {
		depth = 1
	}
	if depth > 5 {
		depth = 5
	}

	var entries []string
	count := 0

	err := walkDepth(ctx, dir, dir, 0, depth, func(rel string, info os.FileInfo, isDir bool) bool {
		if count >= lsMaxEntries {
			return false
		}
		count++

		if isDir {
			entries = append(entries, rel+"/")
		} else {
			entries = append(entries, fmt.Sprintf("%s  %s", rel, formatSize(int(info.Size()))))
		}
		return true
	})

	if err != nil {
		return nil, fmt.Errorf("ls %s: %w", dir, err)
	}

	if len(entries) == 0 {
		return json.Marshal("(empty directory)")
	}

	result := strings.Join(entries, "\n")
	if count >= lsMaxEntries {
		result += fmt.Sprintf("\n\n[Listing truncated at %d entries. Use a specific subdirectory or lower depth.]", lsMaxEntries)
	}
	return json.Marshal(result)
}

func walkDepth(ctx context.Context, root, dir string, current, maxDepth int, fn func(rel string, info os.FileInfo, isDir bool) bool) error {
	if current >= maxDepth {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range dirEntries {
		name := e.Name()
		if name == ".git" || name == "node_modules" || name == "__pycache__" || name == ".venv" {
			continue
		}

		path := filepath.Join(dir, name)
		rel, _ := filepath.Rel(root, path)
		info, err := e.Info()
		if err != nil {
			continue
		}

		isDir := e.IsDir()
		if !fn(rel, info, isDir) {
			return nil
		}

		if isDir {
			if err := walkDepth(ctx, root, path, current+1, maxDepth, fn); err != nil {
				return err
			}
		}
	}
	return nil
}
