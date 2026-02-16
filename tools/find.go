package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/voocel/agentcore/schema"
)

// FindTool searches for files matching a glob pattern.
// Uses fd if available, falls back to filepath.WalkDir + filepath.Match.
type FindTool struct {
	WorkDir string
}

func NewFind(workDir string) *FindTool { return &FindTool{WorkDir: workDir} }

func (t *FindTool) Name() string  { return "find" }
func (t *FindTool) Label() string { return "Find Files" }
func (t *FindTool) Description() string {
	return "Search for files by glob pattern. Returns matching file paths (max 1000 results)."
}
func (t *FindTool) Schema() map[string]any {
	return schema.Object(
		schema.Property("pattern", schema.String("Glob pattern to match (e.g. '*.go', 'src/**/*.ts')")).Required(),
		schema.Property("path", schema.String("Directory to search in (default: working directory)")),
	)
}

type findArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

const findMaxResults = 1000

func (t *FindTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a findArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	searchDir := t.WorkDir
	if a.Path != "" {
		if filepath.IsAbs(a.Path) {
			searchDir = a.Path
		} else {
			searchDir = filepath.Join(t.WorkDir, a.Path)
		}
	}

	// Try fd first
	if result, err := t.findWithFd(ctx, a.Pattern, searchDir); err == nil {
		return result, nil
	}

	// Fallback to filepath.WalkDir
	return t.findWithWalk(ctx, a.Pattern, searchDir)
}

func (t *FindTool) findWithFd(ctx context.Context, pattern, dir string) (json.RawMessage, error) {
	fdPath, err := exec.LookPath("fd")
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, fdPath, "--type", "f", "--follow", "--glob", pattern, dir)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, err
	}

	return t.formatResults(string(out), dir)
}

func (t *FindTool) findWithWalk(ctx context.Context, pattern, dir string) (json.RawMessage, error) {
	var matches []string

	truncated := false
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "__pycache__" || name == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		if matched, _ := filepath.Match(pattern, d.Name()); matched {
			rel, _ := filepath.Rel(dir, path)
			matches = append(matches, rel)
		}
		if len(matches) >= findMaxResults {
			truncated = true
			return filepath.SkipAll
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, fmt.Errorf("walk: %w", err)
	}

	if len(matches) == 0 {
		return json.Marshal("No files found matching pattern.")
	}

	result := strings.Join(matches, "\n")
	if truncated {
		result += fmt.Sprintf("\n\n[Results truncated at %d. Use a more specific pattern.]", findMaxResults)
	}
	return json.Marshal(result)
}

func (t *FindTool) formatResults(output, dir string) (json.RawMessage, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var results []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if rel, err := filepath.Rel(dir, line); err == nil {
			results = append(results, rel)
		} else {
			results = append(results, line)
		}
		if len(results) >= findMaxResults {
			break
		}
	}

	if len(results) == 0 {
		return json.Marshal("No files found matching pattern.")
	}

	result := strings.Join(results, "\n")
	if len(results) >= findMaxResults {
		result += fmt.Sprintf("\n\n[Results truncated at %d. Use a more specific pattern.]", findMaxResults)
	}
	return json.Marshal(result)
}
