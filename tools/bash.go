package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/schema"
)

// BashTool executes shell commands.
// Streams stdout+stderr via ReportToolProgress for real-time display.
// Final result applies tail truncation (2000 lines / 50KB).
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
	return schema.Object(
		schema.Property("command", schema.String("Shell command to execute")).Required(),
		schema.Property("timeout", schema.Int("Timeout in seconds (default: 120)")),
	)
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

	// Merge stdout and stderr into a single pipe for streaming
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	// Stream output line by line via tool progress
	var output []byte
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 256*1024), 256*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			output = append(output, line...)
			output = append(output, '\n')
			// Report each line as progress for real-time display
			agentcore.ReportToolProgress(ctx, line)
		}
	}()

	err := cmd.Wait()
	pw.Close()
	<-done

	outStr := string(output)
	if outStr == "" {
		outStr = "(no output)"
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
	truncated, totalLines, outputLines, wasTruncated := truncateTail(outStr, defaultMaxLines, defaultMaxBytes)
	if wasTruncated {
		startLine := totalLines - outputLines + 1
		truncated += fmt.Sprintf("\n\n[Showing lines %d-%d of %d.]", startLine, totalLines, totalLines)
	}
	if exitCode != 0 {
		truncated += fmt.Sprintf("\n\nCommand exited with code %d", exitCode)
	}

	return json.Marshal(truncated)
}
