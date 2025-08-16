package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MemoryStore implements StateStore interface using in-memory storage
type MemoryStore struct {
	data sync.Map // map[string][]byte
}

// NewMemoryStore creates a new in-memory state store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// Put stores data with the given key
func (ms *MemoryStore) Put(ctx context.Context, key string, data []byte) error {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	ms.data.Store(key, dataCopy)
	return nil
}

// Get retrieves data by key
func (ms *MemoryStore) Get(ctx context.Context, key string) ([]byte, error) {
	value, exists := ms.data.Load(key)
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	data := value.([]byte)

	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// Delete removes data by key
func (ms *MemoryStore) Delete(ctx context.Context, key string) error {
	ms.data.Delete(key)
	return nil
}

// List returns all keys with the given prefix
func (ms *MemoryStore) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	ms.data.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if strings.HasPrefix(keyStr, prefix) {
			keys = append(keys, keyStr)
		}
		return true
	})

	sort.Strings(keys)

	return keys, nil
}

// Exists checks if a key exists
func (ms *MemoryStore) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := ms.data.Load(key)
	return exists, nil
}

// Close releases any resources held by the store
func (ms *MemoryStore) Close() error {
	// Clear all data
	ms.data.Range(func(key, value interface{}) bool {
		ms.data.Delete(key)
		return true
	})
	return nil
}

// Size returns the number of items stored (useful for testing)
func (ms *MemoryStore) Size() int {
	count := 0
	ms.data.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Keys returns all stored keys (useful for debugging)
func (ms *MemoryStore) Keys() []string {
	var keys []string
	ms.data.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	sort.Strings(keys)
	return keys
}

// Clear removes all data from the store
func (ms *MemoryStore) Clear() {
	ms.data.Range(func(key, value interface{}) bool {
		ms.data.Delete(key)
		return true
	})
}

// Stats returns memory store statistics
func (ms *MemoryStore) Stats(ctx context.Context) (map[string]interface{}, error) {
	totalSize := int64(0)
	count := 0

	ms.data.Range(func(key, value interface{}) bool {
		count++
		if data, ok := value.([]byte); ok {
			totalSize += int64(len(data))
		}
		return true
	})

	return map[string]interface{}{
		"type":       "memory",
		"count":      count,
		"total_size": totalSize,
	}, nil
}
