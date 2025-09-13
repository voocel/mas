package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// InMemoryCheckpointer implements Checkpointer interface using in-memory storage
type InMemoryCheckpointer struct {
	checkpoints    map[string]map[string]*contextpkg.Checkpoint // threadID -> checkpointID -> checkpoint
	mutex          sync.RWMutex
	maxCheckpoints int
}

// NewInMemoryCheckpointer creates a new in-memory checkpointer
func NewInMemoryCheckpointer(maxCheckpoints int) *InMemoryCheckpointer {
	if maxCheckpoints <= 0 {
		maxCheckpoints = 100 // Default limit
	}

	return &InMemoryCheckpointer{
		checkpoints:    make(map[string]map[string]*contextpkg.Checkpoint),
		maxCheckpoints: maxCheckpoints,
	}
}

// Save saves a checkpoint
func (imc *InMemoryCheckpointer) Save(ctx context.Context, checkpoint *contextpkg.Checkpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("checkpoint cannot be nil")
	}

	if checkpoint.ThreadID == "" {
		return fmt.Errorf("thread ID cannot be empty")
	}

	if checkpoint.ID == "" {
		checkpoint.ID = generateCheckpointID()
	}

	if checkpoint.Timestamp.IsZero() {
		checkpoint.Timestamp = time.Now()
	}

	imc.mutex.Lock()
	defer imc.mutex.Unlock()

	// Initialize thread map if it doesn't exist
	if imc.checkpoints[checkpoint.ThreadID] == nil {
		imc.checkpoints[checkpoint.ThreadID] = make(map[string]*contextpkg.Checkpoint)
	}

	// Save the checkpoint
	imc.checkpoints[checkpoint.ThreadID][checkpoint.ID] = checkpoint

	// Cleanup old checkpoints if we exceed the limit
	imc.cleanupOldCheckpoints(checkpoint.ThreadID)

	return nil
}

// Load loads a checkpoint
func (imc *InMemoryCheckpointer) Load(ctx context.Context, threadID, checkpointID string) (*contextpkg.Checkpoint, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread ID cannot be empty")
	}

	if checkpointID == "" {
		return nil, fmt.Errorf("checkpoint ID cannot be empty")
	}

	imc.mutex.RLock()
	defer imc.mutex.RUnlock()

	threadCheckpoints, exists := imc.checkpoints[threadID]
	if !exists {
		return nil, fmt.Errorf("no checkpoints found for thread %s", threadID)
	}

	checkpoint, exists := threadCheckpoints[checkpointID]
	if !exists {
		return nil, fmt.Errorf("checkpoint %s not found for thread %s", checkpointID, threadID)
	}

	return checkpoint, nil
}

// List lists all checkpoints for a thread
func (imc *InMemoryCheckpointer) List(ctx context.Context, threadID string) ([]*contextpkg.Checkpoint, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread ID cannot be empty")
	}

	imc.mutex.RLock()
	defer imc.mutex.RUnlock()

	threadCheckpoints, exists := imc.checkpoints[threadID]
	if !exists {
		return []*contextpkg.Checkpoint{}, nil
	}

	checkpoints := make([]*contextpkg.Checkpoint, 0, len(threadCheckpoints))
	for _, checkpoint := range threadCheckpoints {
		checkpoints = append(checkpoints, checkpoint)
	}

	// Sort by timestamp (newest first)
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Timestamp.After(checkpoints[j].Timestamp)
	})

	return checkpoints, nil
}

// Delete deletes a specific checkpoint
func (imc *InMemoryCheckpointer) Delete(ctx context.Context, threadID, checkpointID string) error {
	if threadID == "" {
		return fmt.Errorf("thread ID cannot be empty")
	}

	if checkpointID == "" {
		return fmt.Errorf("checkpoint ID cannot be empty")
	}

	imc.mutex.Lock()
	defer imc.mutex.Unlock()

	threadCheckpoints, exists := imc.checkpoints[threadID]
	if !exists {
		return fmt.Errorf("no checkpoints found for thread %s", threadID)
	}

	if _, exists := threadCheckpoints[checkpointID]; !exists {
		return fmt.Errorf("checkpoint %s not found for thread %s", checkpointID, threadID)
	}

	delete(threadCheckpoints, checkpointID)

	// Clean up empty thread map
	if len(threadCheckpoints) == 0 {
		delete(imc.checkpoints, threadID)
	}

	return nil
}

// DeleteThread deletes all checkpoints for a thread
func (imc *InMemoryCheckpointer) DeleteThread(ctx context.Context, threadID string) error {
	if threadID == "" {
		return fmt.Errorf("thread ID cannot be empty")
	}

	imc.mutex.Lock()
	defer imc.mutex.Unlock()

	delete(imc.checkpoints, threadID)
	return nil
}

// GetLatest gets the latest checkpoint for a thread
func (imc *InMemoryCheckpointer) GetLatest(ctx context.Context, threadID string) (*contextpkg.Checkpoint, error) {
	checkpoints, err := imc.List(ctx, threadID)
	if err != nil {
		return nil, err
	}

	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found for thread %s", threadID)
	}

	return checkpoints[0], nil // Already sorted by timestamp (newest first)
}

// cleanupOldCheckpoints removes old checkpoints if we exceed the limit
func (imc *InMemoryCheckpointer) cleanupOldCheckpoints(threadID string) {
	threadCheckpoints := imc.checkpoints[threadID]
	if len(threadCheckpoints) <= imc.maxCheckpoints {
		return
	}

	// Convert to slice for sorting
	checkpoints := make([]*contextpkg.Checkpoint, 0, len(threadCheckpoints))
	for _, checkpoint := range threadCheckpoints {
		checkpoints = append(checkpoints, checkpoint)
	}

	// Sort by timestamp (newest first)
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Timestamp.After(checkpoints[j].Timestamp)
	})

	// Keep only the newest maxCheckpoints
	toKeep := checkpoints[:imc.maxCheckpoints]
	toDelete := checkpoints[imc.maxCheckpoints:]

	// Delete old checkpoints
	for _, checkpoint := range toDelete {
		delete(threadCheckpoints, checkpoint.ID)
	}

	// Rebuild the map with only the checkpoints to keep
	newThreadCheckpoints := make(map[string]*contextpkg.Checkpoint)
	for _, checkpoint := range toKeep {
		newThreadCheckpoints[checkpoint.ID] = checkpoint
	}
	imc.checkpoints[threadID] = newThreadCheckpoints
}

// GetStats returns statistics about the checkpointer
func (imc *InMemoryCheckpointer) GetStats() CheckpointerStats {
	imc.mutex.RLock()
	defer imc.mutex.RUnlock()

	totalCheckpoints := 0
	threadCount := len(imc.checkpoints)

	for _, threadCheckpoints := range imc.checkpoints {
		totalCheckpoints += len(threadCheckpoints)
	}

	return CheckpointerStats{
		TotalThreads:     threadCount,
		TotalCheckpoints: totalCheckpoints,
		MaxCheckpoints:   imc.maxCheckpoints,
	}
}

// CheckpointerStats represents statistics about the checkpointer
type CheckpointerStats struct {
	TotalThreads     int `json:"total_threads"`
	TotalCheckpoints int `json:"total_checkpoints"`
	MaxCheckpoints   int `json:"max_checkpoints"`
}

// FileCheckpointer implements Checkpointer interface using file storage
type FileCheckpointer struct {
	basePath string
	mutex    sync.RWMutex
}

// NewFileCheckpointer creates a new file-based checkpointer
func NewFileCheckpointer(basePath string) *FileCheckpointer {
	return &FileCheckpointer{
		basePath: basePath,
	}
}

// Save saves a checkpoint to file
func (fc *FileCheckpointer) Save(ctx context.Context, checkpoint *contextpkg.Checkpoint) error {
	// Implementation would save to file system
	// For now, return not implemented
	return fmt.Errorf("file checkpointer not implemented yet")
}

// Load loads a checkpoint from file
func (fc *FileCheckpointer) Load(ctx context.Context, threadID, checkpointID string) (*contextpkg.Checkpoint, error) {
	// Implementation would load from file system
	return nil, fmt.Errorf("file checkpointer not implemented yet")
}

// List lists all checkpoints for a thread from files
func (fc *FileCheckpointer) List(ctx context.Context, threadID string) ([]*contextpkg.Checkpoint, error) {
	// Implementation would list files
	return nil, fmt.Errorf("file checkpointer not implemented yet")
}

// Delete deletes a checkpoint file
func (fc *FileCheckpointer) Delete(ctx context.Context, threadID, checkpointID string) error {
	// Implementation would delete file
	return fmt.Errorf("file checkpointer not implemented yet")
}
