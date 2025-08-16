package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileStore implements StateStore interface using the local filesystem
type FileStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileStore creates a new filesystem-based state store
func NewFileStore(basePath string) (*FileStore, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory %s: %w", basePath, err)
	}

	return &FileStore{
		basePath: basePath,
	}, nil
}

// Put stores data with the given key
func (fs *FileStore) Put(ctx context.Context, key string, data []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	filePath := fs.keyToPath(key)

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the file atomically by writing to a temp file first
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomically rename the temp file to the final file
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Get retrieves data by key
func (fs *FileStore) Get(ctx context.Context, key string) ([]byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	filePath := fs.keyToPath(key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Delete removes data by key
func (fs *FileStore) Delete(ctx context.Context, key string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	filePath := fs.keyToPath(key)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Try to remove empty parent directories
	fs.cleanupEmptyDirs(filepath.Dir(filePath))

	return nil
}

// List returns all keys with the given prefix
func (fs *FileStore) List(ctx context.Context, prefix string) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var keys []string
	prefixPath := fs.keyToPath(prefix)

	if info, err := os.Stat(prefixPath); err == nil && !info.IsDir() {
		keys = append(keys, prefix)
		return keys, nil
	}

	// Walk the directory tree looking for files that match the prefix
	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		key := fs.pathToKey(path)
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return keys, nil
}

// Exists checks if a key exists
func (fs *FileStore) Exists(ctx context.Context, key string) (bool, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	filePath := fs.keyToPath(key)
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// Close releases any resources held by the store
func (fs *FileStore) Close() error {
	return nil
}

// Stats returns file store statistics
func (fs *FileStore) Stats(ctx context.Context) (map[string]interface{}, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	totalSize := int64(0)
	count := 0

	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			count++
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate stats: %w", err)
	}

	return map[string]interface{}{
		"type":       "file",
		"base_path":  fs.basePath,
		"count":      count,
		"total_size": totalSize,
	}, nil
}

// keyToPath converts a key to a filesystem path
func (fs *FileStore) keyToPath(key string) string {
	// Replace colons with directory separators for better organization
	safePath := strings.ReplaceAll(key, ":", string(filepath.Separator))
	return filepath.Join(fs.basePath, safePath)
}

// pathToKey converts a filesystem path back to a key
func (fs *FileStore) pathToKey(path string) string {
	// Get relative path from base directory
	relPath, err := filepath.Rel(fs.basePath, path)
	if err != nil {
		return path // Fallback to full path
	}

	// Convert directory separators back to colons
	return strings.ReplaceAll(relPath, string(filepath.Separator), ":")
}

// cleanupEmptyDirs removes empty parent directories up to the base path
func (fs *FileStore) cleanupEmptyDirs(dir string) {
	// Don't remove the base directory itself
	if dir == fs.basePath || !strings.HasPrefix(dir, fs.basePath) {
		return
	}

	// Check if directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return // Directory not empty or can't read it
	}

	// Remove empty directory
	if err := os.Remove(dir); err == nil {
		// Recursively cleanup parent directories
		fs.cleanupEmptyDirs(filepath.Dir(dir))
	}
}
