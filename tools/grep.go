package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/voocel/agentcore/schema"
)

// GrepTool searches file contents by pattern.
// Uses ripgrep (rg) if available, falls back to regexp + bufio.Scanner.
type GrepTool struct {
	WorkDir string
}

func NewGrep(workDir string) *GrepTool { return &GrepTool{WorkDir: workDir} }

func (t *GrepTool) Name() string  { return "grep" }
func (t *GrepTool) Label() string { return "Search Content" }
func (t *GrepTool) Description() string {
	return "Search file contents by regex pattern. Returns matching lines with file paths and line numbers (max 100 matches)."
}
func (t *GrepTool) Schema() map[string]any {
	return schema.Object(
		schema.Property("pattern", schema.String("Search pattern (regex by default, or literal with literal=true)")).Required(),
		schema.Property("path", schema.String("File or directory to search (default: working directory)")),
		schema.Property("glob", schema.String("File glob filter (e.g. '*.go', '*.ts')")),
		schema.Property("ignoreCase", schema.Bool("Case insensitive search")),
		schema.Property("literal", schema.Bool("Treat pattern as literal string, not regex")),
		schema.Property("contextLines", schema.Int("Number of context lines around each match (default: 0)")),
	)
}

type grepArgs struct {
	Pattern      string `json:"pattern"`
	Path         string `json:"path"`
	Glob         string `json:"glob"`
	IgnoreCase   bool   `json:"ignoreCase"`
	Literal      bool   `json:"literal"`
	ContextLines int    `json:"contextLines"`
}

const (
	grepMaxMatches = 100
	grepMaxLineLen = 500
	grepMaxBytes   = 50 * 1024
)

func (t *GrepTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a grepArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	searchPath := t.WorkDir
	if a.Path != "" {
		if filepath.IsAbs(a.Path) {
			searchPath = a.Path
		} else {
			searchPath = filepath.Join(t.WorkDir, a.Path)
		}
	}

	// Try ripgrep first
	if result, err := t.grepWithRg(ctx, a, searchPath); err == nil {
		return result, nil
	}

	// Fallback to Go implementation
	return t.grepWithGo(ctx, a, searchPath)
}

func (t *GrepTool) grepWithRg(ctx context.Context, a grepArgs, searchPath string) (json.RawMessage, error) {
	rgPath, err := exec.LookPath("rg")
	if err != nil {
		return nil, err
	}

	cmdArgs := []string{"--line-number", "--no-heading", "--color", "never"}

	if a.IgnoreCase {
		cmdArgs = append(cmdArgs, "--ignore-case")
	}
	if a.Literal {
		cmdArgs = append(cmdArgs, "--fixed-strings")
	}
	if a.ContextLines > 0 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--context=%d", a.ContextLines))
	}
	if a.Glob != "" {
		cmdArgs = append(cmdArgs, "--glob", a.Glob)
	}

	cmdArgs = append(cmdArgs, a.Pattern, searchPath)

	cmd := exec.CommandContext(ctx, rgPath, cmdArgs...)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		// rg returns exit code 1 for no matches
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return json.Marshal("No matches found.")
		}
		return nil, err
	}

	result := truncateGrepOutput(string(out), searchPath)
	return json.Marshal(result)
}

func (t *GrepTool) grepWithGo(ctx context.Context, a grepArgs, searchPath string) (json.RawMessage, error) {
	pattern := a.Pattern
	if a.Literal {
		pattern = regexp.QuoteMeta(pattern)
	}
	if a.IgnoreCase {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var results []string
	matchCount := 0

	err = filepath.WalkDir(searchPath, func(path string, d os.DirEntry, err error) error {
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
		if a.Glob != "" {
			if matched, _ := filepath.Match(a.Glob, d.Name()); !matched {
				return nil
			}
		}
		// Skip binary/large files
		info, _ := d.Info()
		if info != nil && info.Size() > 1024*1024 {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		rel, _ := filepath.Rel(searchPath, path)
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				if len(line) > grepMaxLineLen {
					line = line[:grepMaxLineLen] + "..."
				}
				results = append(results, fmt.Sprintf("%s:%d:%s", rel, lineNum, line))
				matchCount++
				if matchCount >= grepMaxMatches {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		return json.Marshal("No matches found.")
	}

	result := strings.Join(results, "\n")
	if matchCount >= grepMaxMatches {
		result += fmt.Sprintf("\n\n[Results truncated at %d matches. Use a more specific pattern or path.]", grepMaxMatches)
	}
	return json.Marshal(result)
}

func truncateGrepOutput(output, searchPath string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	prefix := searchPath + string(filepath.Separator)

	var results []string
	matchCount := 0
	for _, line := range lines {
		// Make paths relative
		if rel, ok := strings.CutPrefix(line, prefix); ok {
			line = rel
		}
		if len(line) > grepMaxLineLen+50 { // extra space for path:line:
			line = line[:grepMaxLineLen+50] + "..."
		}
		// Count non-context lines as matches (context lines start with -)
		if line != "--" && !strings.HasPrefix(line, " ") {
			matchCount++
		}
		results = append(results, line)
		if matchCount >= grepMaxMatches {
			break
		}
	}

	result := strings.Join(results, "\n")
	if len(result) > grepMaxBytes {
		result = result[:grepMaxBytes] + "\n\n[Output truncated.]"
	} else if matchCount >= grepMaxMatches {
		result += fmt.Sprintf("\n\n[Results truncated at %d matches. Use a more specific pattern or path.]", grepMaxMatches)
	}
	return result
}
