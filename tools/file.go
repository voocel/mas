package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/voocel/mas"
)

// FileReader creates a file reading tool
func FileReader() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"path":   mas.StringProperty("File path to read"),
			"format": mas.EnumProperty("Output format", []string{"text", "base64"}),
			"limit":  mas.NumberProperty("Maximum number of bytes to read (default: 10MB)"),
		},
		Required: []string{"path"},
	}

	return mas.NewTool(
		"file_reader",
		"Reads files from the local filesystem",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			path, ok := params["path"].(string)
			if !ok {
				return nil, fmt.Errorf("path parameter is required")
			}

			format := "text"
			if formatParam, exists := params["format"]; exists {
				if f, ok := formatParam.(string); ok {
					format = f
				}
			}

			limit := int64(10 * 1024 * 1024) // 10MB default
			if limitParam, exists := params["limit"]; exists {
				if l, ok := limitParam.(float64); ok {
					limit = int64(l)
				}
			}

			// Get file info
			fileInfo, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("file error: %v", err)
			}

			if fileInfo.IsDir() {
				return nil, fmt.Errorf("path is a directory, not a file")
			}

			// Check file size
			if fileInfo.Size() > limit {
				return nil, fmt.Errorf("file size (%d bytes) exceeds limit (%d bytes)", fileInfo.Size(), limit)
			}

			// Read file
			file, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("failed to open file: %v", err)
			}
			defer file.Close()

			data, err := io.ReadAll(file)
			if err != nil {
				return nil, fmt.Errorf("failed to read file: %v", err)
			}

			var content string
			switch format {
			case "base64":
				content = base64.StdEncoding.EncodeToString(data)
			case "text":
				content = string(data)
			default:
				return nil, fmt.Errorf("unsupported format: %s", format)
			}

			return map[string]interface{}{
				"path":     path,
				"size":     fileInfo.Size(),
				"modified": fileInfo.ModTime().Format(time.RFC3339),
				"format":   format,
				"content":  content,
			}, nil
		},
	)
}

// FileWriter creates a file writing tool
func FileWriter() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"path":    mas.StringProperty("File path to write to"),
			"content": mas.StringProperty("Content to write"),
			"format":  mas.EnumProperty("Input format", []string{"text", "base64"}),
			"mode":    mas.EnumProperty("Write mode", []string{"create", "append", "overwrite"}),
		},
		Required: []string{"path", "content"},
	}

	return mas.NewTool(
		"file_writer",
		"Writes content to files on the local filesystem",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			path, ok := params["path"].(string)
			if !ok {
				return nil, fmt.Errorf("path parameter is required")
			}

			content, ok := params["content"].(string)
			if !ok {
				return nil, fmt.Errorf("content parameter is required")
			}

			format := "text"
			if formatParam, exists := params["format"]; exists {
				if f, ok := formatParam.(string); ok {
					format = f
				}
			}

			mode := "create"
			if modeParam, exists := params["mode"]; exists {
				if m, ok := modeParam.(string); ok {
					mode = m
				}
			}

			// Prepare data to write
			var data []byte
			var err error
			switch format {
			case "base64":
				data, err = base64.StdEncoding.DecodeString(content)
				if err != nil {
					return nil, fmt.Errorf("invalid base64 content: %v", err)
				}
			case "text":
				data = []byte(content)
			default:
				return nil, fmt.Errorf("unsupported format: %s", format)
			}

			// Create directory if it doesn't exist
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory: %v", err)
			}

			// Determine file flags
			var flags int
			switch mode {
			case "create":
				flags = os.O_CREATE | os.O_WRONLY | os.O_EXCL
			case "append":
				flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
			case "overwrite":
				flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
			default:
				return nil, fmt.Errorf("unsupported mode: %s", mode)
			}

			// Write file
			file, err := os.OpenFile(path, flags, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to open file for writing: %v", err)
			}
			defer file.Close()

			bytesWritten, err := file.Write(data)
			if err != nil {
				return nil, fmt.Errorf("failed to write file: %v", err)
			}

			return map[string]interface{}{
				"path":          path,
				"bytes_written": bytesWritten,
				"format":        format,
				"mode":          mode,
				"success":       true,
			}, nil
		},
	)
}

// DirectoryLister creates a directory listing tool
func DirectoryLister() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"path":      mas.StringProperty("Directory path to list"),
			"recursive": mas.BooleanProperty("List directories recursively"),
			"show_hidden": mas.BooleanProperty("Include hidden files and directories"),
			"pattern":   mas.StringProperty("Filename pattern to match (e.g., '*.txt')"),
		},
		Required: []string{"path"},
	}

	return mas.NewTool(
		"directory_lister",
		"Lists files and directories in a given path",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			path, ok := params["path"].(string)
			if !ok {
				return nil, fmt.Errorf("path parameter is required")
			}

			recursive := false
			if recursiveParam, exists := params["recursive"]; exists {
				if r, ok := recursiveParam.(bool); ok {
					recursive = r
				}
			}

			showHidden := false
			if hiddenParam, exists := params["show_hidden"]; exists {
				if h, ok := hiddenParam.(bool); ok {
					showHidden = h
				}
			}

			pattern := ""
			if patternParam, exists := params["pattern"]; exists {
				if p, ok := patternParam.(string); ok {
					pattern = p
				}
			}

			// Check if path exists and is a directory
			fileInfo, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("path error: %v", err)
			}

			if !fileInfo.IsDir() {
				return nil, fmt.Errorf("path is not a directory")
			}

			var items []map[string]interface{}

			if recursive {
				err = filepath.Walk(path, func(itemPath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if shouldIncludeItem(info.Name(), showHidden, pattern) {
						items = append(items, createFileInfo(itemPath, info))
					}

					return nil
				})
			} else {
				entries, err := os.ReadDir(path)
				if err != nil {
					return nil, fmt.Errorf("failed to read directory: %v", err)
				}

				for _, entry := range entries {
					if shouldIncludeItem(entry.Name(), showHidden, pattern) {
						info, err := entry.Info()
						if err != nil {
							continue
						}
						itemPath := filepath.Join(path, entry.Name())
						items = append(items, createFileInfo(itemPath, info))
					}
				}
			}

			if err != nil {
				return nil, fmt.Errorf("failed to list directory: %v", err)
			}

			return map[string]interface{}{
				"path":      path,
				"count":     len(items),
				"recursive": recursive,
				"pattern":   pattern,
				"items":     items,
			}, nil
		},
	)
}

// FileInfo creates a file information tool
func FileInfo() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"path": mas.StringProperty("File or directory path to get information about"),
		},
		Required: []string{"path"},
	}

	return mas.NewTool(
		"file_info",
		"Gets detailed information about a file or directory",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			path, ok := params["path"].(string)
			if !ok {
				return nil, fmt.Errorf("path parameter is required")
			}

			info, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("path error: %v", err)
			}

			result := createFileInfo(path, info)

			// Add additional information
			result["absolute_path"], _ = filepath.Abs(path)
			result["directory"] = filepath.Dir(path)
			result["extension"] = filepath.Ext(path)

			return result, nil
		},
	)
}

// Helper functions

// shouldIncludeItem determines if a file/directory item should be included based on filters
func shouldIncludeItem(name string, showHidden bool, pattern string) bool {
	// Check hidden files
	if !showHidden && strings.HasPrefix(name, ".") {
		return false
	}

	// Check pattern
	if pattern != "" {
		matched, err := filepath.Match(pattern, name)
		if err != nil || !matched {
			return false
		}
	}

	return true
}

// createFileInfo creates a file information map
func createFileInfo(path string, info os.FileInfo) map[string]interface{} {
	return map[string]interface{}{
		"name":        info.Name(),
		"path":        path,
		"size":        info.Size(),
		"mode":        info.Mode().String(),
		"permissions": fmt.Sprintf("%o", info.Mode().Perm()),
		"modified":    info.ModTime().Format(time.RFC3339),
		"is_dir":      info.IsDir(),
		"is_regular":  info.Mode().IsRegular(),
	}
}