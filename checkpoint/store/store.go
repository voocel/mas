package store

import (
	"context"
)

// StateStore provides an abstraction layer for persistent storage backends
type StateStore interface {
	// Put stores data with the given key
	Put(ctx context.Context, key string, data []byte) error

	// Get retrieves data by key
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete removes data by key
	Delete(ctx context.Context, key string) error

	// List returns all keys with the given prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// Close releases any resources held by the store
	Close() error
}

// BatchStore extends StateStore with batch operations
type BatchStore interface {
	StateStore
	
	// BatchPut stores multiple key-value pairs in a single operation
	BatchPut(ctx context.Context, items map[string][]byte) error
	
	// BatchDelete removes multiple keys in a single operation
	BatchDelete(ctx context.Context, keys []string) error
}

// MetricsStore extends StateStore with metrics capabilities
type MetricsStore interface {
	StateStore
	
	// Stats returns storage statistics
	Stats(ctx context.Context) (map[string]interface{}, error)
}