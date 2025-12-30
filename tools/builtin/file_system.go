package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// FileSystemTool provides filesystem operations.
type FileSystemTool struct {
	*tools.BaseTool
	allowedPaths []string // Allowed paths.
	maxFileSize  int64    // Max file size in bytes.
}

// FileReadInput defines read parameters.
type FileReadInput struct {
	Path string `json:"path" description:"Path to read"`
}

// FileWriteInput defines write parameters.
type FileWriteInput struct {
	Path    string `json:"path" description:"Path to write"`
	Content string `json:"content" description:"Content to write"`
	Append  bool   `json:"append,omitempty" description:"Whether to append; defaults to overwrite"`
}

// FileListInput defines list parameters.
type FileListInput struct {
	Path      string `json:"path" description:"Directory path to list"`
	Recursive bool   `json:"recursive,omitempty" description:"Whether to list recursively"`
}

// FileDeleteInput defines delete parameters.
type FileDeleteInput struct {
	Path string `json:"path" description:"Path of the file or directory to delete"`
}

// FileOutput defines the response payload.
type FileOutput struct {
	Success bool       `json:"success"`
	Message string     `json:"message,omitempty"`
	Content string     `json:"content,omitempty"`
	Files   []FileInfo `json:"files,omitempty"`
	Error   string     `json:"error,omitempty"`
}

// FileInfo defines file metadata.
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"is_dir"`
	ModTime time.Time `json:"mod_time"`
}

// NewFileSystemTool creates a filesystem tool.
func NewFileSystemTool(allowedPaths []string, maxFileSize int64) *FileSystemTool {
	if maxFileSize <= 0 {
		maxFileSize = 10 * 1024 * 1024 // Default: 10MB.
	}

	schema := tools.CreateToolSchema(
		"Filesystem tool for reading, writing, listing, and deleting files",
		map[string]interface{}{
			"action":    tools.StringProperty("Operation type: read, write, list, or delete"),
			"path":      tools.StringProperty("File or directory path"),
			"content":   tools.StringProperty("Content to write (write action only)"),
			"append":    tools.BooleanProperty("Append mode (write action only)"),
			"recursive": tools.BooleanProperty("Recursive listing (list action only)"),
		},
		[]string{"action", "path"},
	)

	baseTool := tools.NewBaseTool("file_system", "Filesystem tool for reading, writing, listing, and deleting files", schema).
		WithCapabilities(tools.CapabilityFile)

	return &FileSystemTool{
		BaseTool:     baseTool,
		allowedPaths: allowedPaths,
		maxFileSize:  maxFileSize,
	}
}

// Execute runs a filesystem operation.
func (t *FileSystemTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, schema.NewToolError(t.Name(), "parse_input", err)
	}

	action, ok := params["action"].(string)
	if !ok {
		return nil, schema.NewToolError(t.Name(), "invalid_action", fmt.Errorf("action must be a string"))
	}

	path, ok := params["path"].(string)
	if !ok {
		return nil, schema.NewToolError(t.Name(), "invalid_path", fmt.Errorf("path must be a string"))
	}

	if !t.isPathAllowed(path) {
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("access denied: path %s is not allowed", path),
		}
		return json.Marshal(output)
	}

	switch action {
	case "read":
		return t.readFile(path)
	case "write":
		content, _ := params["content"].(string)
		append, _ := params["append"].(bool)
		return t.writeFile(path, content, append)
	case "list":
		recursive, _ := params["recursive"].(bool)
		return t.listFiles(path, recursive)
	case "delete":
		return t.deleteFile(path)
	default:
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("unsupported action: %s", action),
		}
		return json.Marshal(output)
	}
}

// readFile reads a file.
func (t *FileSystemTool) readFile(path string) (json.RawMessage, error) {
	info, err := os.Stat(path)
	if err != nil {
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to stat file: %v", err),
		}
		return json.Marshal(output)
	}

	if info.Size() > t.maxFileSize {
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("file too large: %d bytes (max: %d bytes)", info.Size(), t.maxFileSize),
		}
		return json.Marshal(output)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}
		return json.Marshal(output)
	}

	output := FileOutput{
		Success: true,
		Content: string(content),
		Message: fmt.Sprintf("successfully read file: %s (%d bytes)", path, len(content)),
	}
	return json.Marshal(output)
}

// writeFile writes a file.
func (t *FileSystemTool) writeFile(path, content string, append bool) (json.RawMessage, error) {
	if int64(len(content)) > t.maxFileSize {
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("content too large: %d bytes (max: %d bytes)", len(content), t.maxFileSize),
		}
		return json.Marshal(output)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to create directory: %v", err),
		}
		return json.Marshal(output)
	}

	var err error
	if append {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			output := FileOutput{
				Success: false,
				Error:   fmt.Sprintf("failed to open file for append: %v", err),
			}
			return json.Marshal(output)
		}
		defer file.Close()
		_, err = file.WriteString(content)
	} else {
		err = os.WriteFile(path, []byte(content), 0644)
	}

	if err != nil {
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}
		return json.Marshal(output)
	}

	mode := "written"
	if append {
		mode = "appended"
	}

	output := FileOutput{
		Success: true,
		Message: fmt.Sprintf("successfully %s file: %s (%d bytes)", mode, path, len(content)),
	}
	return json.Marshal(output)
}

// listFiles lists directory entries.
func (t *FileSystemTool) listFiles(path string, recursive bool) (json.RawMessage, error) {
	var files []FileInfo

	if recursive {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			files = append(files, FileInfo{
				Name:    info.Name(),
				Path:    filePath,
				Size:    info.Size(),
				IsDir:   info.IsDir(),
				ModTime: info.ModTime(),
			})
			return nil
		})
		if err != nil {
			output := FileOutput{
				Success: false,
				Error:   fmt.Sprintf("failed to walk directory: %v", err),
			}
			return json.Marshal(output)
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			output := FileOutput{
				Success: false,
				Error:   fmt.Sprintf("failed to read directory: %v", err),
			}
			return json.Marshal(output)
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, FileInfo{
				Name:    info.Name(),
				Path:    filepath.Join(path, info.Name()),
				Size:    info.Size(),
				IsDir:   info.IsDir(),
				ModTime: info.ModTime(),
			})
		}
	}

	output := FileOutput{
		Success: true,
		Files:   files,
		Message: fmt.Sprintf("found %d items in %s", len(files), path),
	}
	return json.Marshal(output)
}

// deleteFile deletes a file or directory.
func (t *FileSystemTool) deleteFile(path string) (json.RawMessage, error) {
	err := os.RemoveAll(path)
	if err != nil {
		output := FileOutput{
			Success: false,
			Error:   fmt.Sprintf("failed to delete: %v", err),
		}
		return json.Marshal(output)
	}

	output := FileOutput{
		Success: true,
		Message: fmt.Sprintf("successfully deleted: %s", path),
	}
	return json.Marshal(output)
}

// isPathAllowed checks whether a path is allowed.
func (t *FileSystemTool) isPathAllowed(path string) bool {
	if len(t.allowedPaths) == 0 {
		return true
	}

	absPath, err := resolvePath(path)
	if err != nil {
		return false
	}

	for _, allowedPath := range t.allowedPaths {
		absAllowed, err := resolvePath(allowedPath)
		if err != nil {
			continue
		}
		if absPath == absAllowed || strings.HasPrefix(absPath, absAllowed+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func resolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absPath = filepath.Clean(absPath)

	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		return resolved, nil
	}

	dir := absPath
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		if _, statErr := os.Stat(dir); statErr == nil {
			resolved, err := filepath.EvalSymlinks(dir)
			if err == nil {
				rel, relErr := filepath.Rel(dir, absPath)
				if relErr == nil && rel != "." {
					return filepath.Join(resolved, rel), nil
				}
				return resolved, nil
			}
		}
		dir = parent
	}
	return absPath, nil
}

// ExecuteAsync executes asynchronously.
func (t *FileSystemTool) ExecuteAsync(ctx context.Context, input json.RawMessage) (<-chan tools.ToolResult, error) {
	resultChan := make(chan tools.ToolResult, 1)

	go func() {
		defer close(resultChan)

		result, err := t.Execute(ctx, input)
		if err != nil {
			resultChan <- tools.ToolResult{
				Success: false,
				Error:   err.Error(),
			}
			return
		}

		resultChan <- tools.ToolResult{
			Success: true,
			Data:    result,
		}
	}()

	return resultChan, nil
}
