package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// BashTool executes shell commands.
// Applies tail truncation (2000 lines / 50KB)
type BashTool struct {
	WorkDir string
	Timeout time.Duration // default: 2 minutes
}

func NewBash(workDir string) *BashTool {
	return &BashTool{WorkDir: workDir, Timeout: 2 * time.Minute}
}

func (t *BashTool) Name() string  { return "bash" }
func (t *BashTool) Label() string { return "Execute Command" }
func (t *BashTool) Description() string {
	return fmt.Sprintf(
		"Execute a bash command. Output is truncated to last %d lines or %s (whichever is hit first). Optionally provide a timeout in seconds.",
		defaultMaxLines, formatSize(defaultMaxBytes),
	)
}
func (t *BashTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to execute",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds (default: 120)",
			},
		},
		"required": []string{"command"},
	}
}

type bashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

func (t *BashTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a bashArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	timeout := t.Timeout
	if a.Timeout > 0 {
		timeout = time.Duration(a.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", a.Command)
	if t.WorkDir != "" {
		cmd.Dir = t.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout + stderr
	output := stdout.String()
	if errOut := stderr.String(); errOut != "" {
		if output != "" {
			output += "\n"
		}
		output += errOut
	}

	if output == "" {
		output = "(no output)"
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	// Apply tail truncation
	truncated, totalLines, outputLines, wasTruncated := truncateTail(output, defaultMaxLines, defaultMaxBytes)

	if wasTruncated {
		startLine := totalLines - outputLines + 1
		truncated += fmt.Sprintf("\n\n[Showing lines %d-%d of %d.]",
			startLine, totalLines, totalLines)
	}

	if exitCode != 0 {
		truncated += fmt.Sprintf("\n\nCommand exited with code %d", exitCode)
	}

	return json.Marshal(truncated)
}
