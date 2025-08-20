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

// FileSandbox defines allowed paths for file operations
type FileSandbox struct {
	// AllowedPaths is a list of allowed directory paths
	AllowedPaths []string
	// AllowCurrentDir allows operations in current working directory
	AllowCurrentDir bool
}

// DefaultSandbox returns a sandbox that only allows current directory
func DefaultSandbox() *FileSandbox {
	return &FileSandbox{
		AllowCurrentDir: true,
	}
}

// NoSandbox returns nil (no restrictions)
func NoSandbox() *FileSandbox {
	return nil
}

// validatePath checks if a path is allowed by the sandbox
func validatePath(path string, sandbox *FileSandbox) error {
	if sandbox == nil {
		return nil // No restrictions
	}

	// Get absolute path for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if current directory is allowed
	if sandbox.AllowCurrentDir {
		cwd, err := os.Getwd()
		if err == nil {
			if strings.HasPrefix(absPath, cwd) {
				return nil
			}
		}
	}

	// Check allowed paths
	for _, allowedPath := range sandbox.AllowedPaths {
		allowedAbs, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, allowedAbs) {
			return nil
		}
	}

	return fmt.Errorf("path '%s' is not allowed by sandbox", path)
}

// FileReader creates a file reading tool
func FileReader() mas.Tool {
	return FileReaderWithSandbox(nil)
}

// FileReaderWithSandbox creates a file reading tool with sandbox restrictions
func FileReaderWithSandbox(sandbox *FileSandbox) mas.Tool {
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

			// Validate path against sandbox
			if err := validatePath(path, sandbox); err != nil {
				return nil, err
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
	return FileWriterWithSandbox(nil)
}

// FileWriterWithSandbox creates a file writing tool with sandbox restrictions
func FileWriterWithSandbox(sandbox *FileSandbox) mas.Tool {
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

			// Validate path against sandbox
			if err := validatePath(path, sandbox); err != nil {
				return nil, err
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
	return DirectoryListerWithSandbox(nil)
}

// DirectoryListerWithSandbox creates a directory listing tool with sandbox restrictions
func DirectoryListerWithSandbox(sandbox *FileSandbox) mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"path":        mas.StringProperty("Directory path to list"),
			"recursive":   mas.BooleanProperty("List directories recursively"),
			"show_hidden": mas.BooleanProperty("Include hidden files and directories"),
			"pattern":     mas.StringProperty("Filename pattern to match (e.g., '*.txt')"),
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

			// Validate path against sandbox
			if err := validatePath(path, sandbox); err != nil {
				return nil, err
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
	return FileInfoWithSandbox(nil)
}

// FileInfoWithSandbox creates a file information tool with sandbox restrictions
func FileInfoWithSandbox(sandbox *FileSandbox) mas.Tool {
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

			// Validate path against sandbox
			if err := validatePath(path, sandbox); err != nil {
				return nil, err
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

// AdvancedFileSystem creates an advanced file system tool with enhanced operations
func AdvancedFileSystem() mas.Tool {
	return AdvancedFileSystemWithSandbox(nil)
}

// AdvancedFileSystemWithSandbox creates an advanced file system tool with sandbox restrictions
func AdvancedFileSystemWithSandbox(sandbox *FileSandbox) mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"operation": mas.EnumProperty("File operation", []string{
				"read", "write", "append", "list", "search", "copy", "move", "delete", "mkdir", "info",
			}),
			"path":      mas.StringProperty("File or directory path"),
			"content":   mas.StringProperty("Content for write/append operations"),
			"target":    mas.StringProperty("Target path for copy/move operations"),
			"pattern":   mas.StringProperty("Search pattern (glob or regex)"),
			"recursive": mas.BooleanProperty("Recursive operation"),
			"encoding":  mas.StringProperty("File encoding (default: utf-8)"),
			"backup":    mas.BooleanProperty("Create backup before modification"),
		},
		Required: []string{"operation", "path"},
	}

	return mas.NewTool(
		"advanced_filesystem",
		"Advanced file system operations with pattern matching, search, and backup support",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter is required")
			}

			path, ok := params["path"].(string)
			if !ok {
				return nil, fmt.Errorf("path parameter is required")
			}

			// Validate path against sandbox
			if err := validatePath(path, sandbox); err != nil {
				return nil, err
			}

			switch operation {
			case "read":
				return handleRead(path, params)
			case "write":
				return handleWrite(path, params, sandbox)
			case "append":
				return handleAppend(path, params, sandbox)
			case "list":
				return handleList(path, params, sandbox)
			case "search":
				return handleSearch(path, params, sandbox)
			case "copy":
				return handleCopy(path, params, sandbox)
			case "move":
				return handleMove(path, params, sandbox)
			case "delete":
				return handleDelete(path, params, sandbox)
			case "mkdir":
				return handleMkdir(path, params, sandbox)
			case "info":
				return handleInfo(path, params, sandbox)
			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}
		},
	)
}

// Handler functions for advanced file operations

func handleRead(path string, params map[string]any) (any, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("file error: %v", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	return map[string]interface{}{
		"path":     path,
		"size":     info.Size(),
		"modified": info.ModTime().Format(time.RFC3339),
		"content":  string(data),
	}, nil
}

func handleWrite(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required for write operation")
	}

	if backup, _ := params["backup"].(bool); backup {
		if _, err := os.Stat(path); err == nil {
			backupPath := path + ".backup." + fmt.Sprintf("%d", time.Now().Unix())
			if err := copyFile(path, backupPath); err != nil {
				return nil, fmt.Errorf("failed to create backup: %v", err)
			}
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	return map[string]interface{}{
		"path":    path,
		"size":    len(content),
		"success": true,
	}, nil
}

func handleAppend(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required for append operation")
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for append: %v", err)
	}
	defer file.Close()

	bytesWritten, err := file.WriteString(content)
	if err != nil {
		return nil, fmt.Errorf("failed to append to file: %v", err)
	}

	return map[string]interface{}{
		"path":          path,
		"bytes_written": bytesWritten,
		"success":       true,
	}, nil
}

func handleList(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	recursive, _ := params["recursive"].(bool)
	pattern, _ := params["pattern"].(string)

	var items []map[string]interface{}

	if recursive {
		err := filepath.Walk(path, func(itemPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if pattern == "" || matchesPattern(info.Name(), pattern) {
				items = append(items, createFileInfo(itemPath, info))
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %v", err)
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %v", err)
		}

		for _, entry := range entries {
			if pattern == "" || matchesPattern(entry.Name(), pattern) {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				itemPath := filepath.Join(path, entry.Name())
				items = append(items, createFileInfo(itemPath, info))
			}
		}
	}

	return map[string]interface{}{
		"path":      path,
		"count":     len(items),
		"recursive": recursive,
		"pattern":   pattern,
		"items":     items,
	}, nil
}

func handleSearch(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern parameter is required for search operation")
	}

	recursive, _ := params["recursive"].(bool)
	var matches []string

	searchFunc := func(itemPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if matchesPattern(info.Name(), pattern) {
			matches = append(matches, itemPath)
		}
		return nil
	}

	if recursive {
		err := filepath.Walk(path, searchFunc)
		if err != nil {
			return nil, fmt.Errorf("failed to search directory: %v", err)
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %v", err)
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			itemPath := filepath.Join(path, entry.Name())
			if err := searchFunc(itemPath, info, nil); err != nil {
				continue
			}
		}
	}

	return map[string]interface{}{
		"path":      path,
		"pattern":   pattern,
		"recursive": recursive,
		"matches":   matches,
		"count":     len(matches),
	}, nil
}

func handleCopy(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	target, ok := params["target"].(string)
	if !ok {
		return nil, fmt.Errorf("target parameter is required for copy operation")
	}

	if err := validatePath(target, sandbox); err != nil {
		return nil, err
	}

	err := copyFile(path, target)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %v", err)
	}

	return map[string]interface{}{
		"source":  path,
		"target":  target,
		"success": true,
	}, nil
}

func handleMove(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	target, ok := params["target"].(string)
	if !ok {
		return nil, fmt.Errorf("target parameter is required for move operation")
	}

	if err := validatePath(target, sandbox); err != nil {
		return nil, err
	}

	err := os.Rename(path, target)
	if err != nil {
		return nil, fmt.Errorf("failed to move file: %v", err)
	}

	return map[string]interface{}{
		"source":  path,
		"target":  target,
		"success": true,
	}, nil
}

func handleDelete(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	recursive, _ := params["recursive"].(bool)

	var err error
	if recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to delete: %v", err)
	}

	return map[string]interface{}{
		"path":      path,
		"recursive": recursive,
		"success":   true,
	}, nil
}

func handleMkdir(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	return map[string]interface{}{
		"path":    path,
		"success": true,
	}, nil
}

func handleInfo(path string, params map[string]any, sandbox *FileSandbox) (any, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	return createFileInfo(path, info), nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func matchesPattern(name, pattern string) bool {
	if pattern == "" {
		return true
	}

	// Try glob pattern first
	if matched, err := filepath.Match(pattern, name); err == nil && matched {
		return true
	}

	// Try simple substring match
	return strings.Contains(strings.ToLower(name), strings.ToLower(pattern))
}
