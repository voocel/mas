package checkpoint

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AutoCleaner manages automatic cleanup of expired checkpoints
type AutoCleaner struct {
	manager  *Manager
	config   CleanupConfig
	ticker   *time.Ticker
	stopChan chan struct{}
	mu       sync.Mutex
	running  bool
}

// CleanupConfig contains configuration for automatic cleanup
type CleanupConfig struct {
	// Interval between cleanup runs
	Interval time.Duration `json:"interval"`

	// Maximum age of checkpoints to keep
	MaxAge time.Duration `json:"max_age"`

	// Minimum number of checkpoints to keep per workflow
	MinKeepCount int `json:"min_keep_count"`

	// Maximum storage size before triggering cleanup
	MaxStorageSize int64 `json:"max_storage_size"`

	// Enable automatic cleanup
	Enabled bool `json:"enabled"`
}

// DefaultCleanupConfig returns a default cleanup configuration
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		Interval:       time.Hour,
		MaxAge:         7 * 24 * time.Hour, // 7 days
		MinKeepCount:   3,
		MaxStorageSize: 1024 * 1024 * 1024, // 1GB
		Enabled:        true,
	}
}

// NewAutoCleaner creates a new auto cleaner
func NewAutoCleaner(manager *Manager, config CleanupConfig) *AutoCleaner {
	return &AutoCleaner{
		manager:  manager,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start begins the automatic cleanup process
func (ac *AutoCleaner) Start() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.running {
		return fmt.Errorf("auto cleaner is already running")
	}

	if !ac.config.Enabled {
		return fmt.Errorf("auto cleanup is disabled")
	}

	ac.ticker = time.NewTicker(ac.config.Interval)
	ac.running = true

	go ac.cleanupLoop()

	return nil
}

// Stop stops the automatic cleanup process
func (ac *AutoCleaner) Stop() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.running {
		return nil
	}

	close(ac.stopChan)
	ac.ticker.Stop()
	ac.running = false

	return nil
}

// IsRunning returns true if the auto cleaner is running
func (ac *AutoCleaner) IsRunning() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return ac.running
}

// RunOnce performs a single cleanup operation
func (ac *AutoCleaner) RunOnce(ctx context.Context) (*CleanupStats, error) {
	stats := &CleanupStats{
		StartTime: time.Now(),
	}

	// Get storage statistics if supported
	if metricsStore, ok := ac.manager.store.(interface {
		Stats(ctx context.Context) (map[string]interface{}, error)
	}); ok {
		storeStats, err := metricsStore.Stats(ctx)
		if err == nil {
			if totalSize, ok := storeStats["total_size"].(int64); ok {
				stats.TotalSizeBefore = totalSize

				// If storage size exceeds limit, trigger aggressive cleanup
				if totalSize > ac.config.MaxStorageSize {
					stats.TriggeredBySize = true
				}
			}
		}
	}

	// Clean up by age
	ageDeleted, err := ac.cleanupByAge(ctx)
	if err != nil {
		return stats, fmt.Errorf("age-based cleanup failed: %w", err)
	}
	stats.DeletedByAge = ageDeleted

	// Clean up by count if needed
	countDeleted, err := ac.cleanupByCount(ctx)
	if err != nil {
		return stats, fmt.Errorf("count-based cleanup failed: %w", err)
	}
	stats.DeletedByCount = countDeleted

	// Clean up by size if needed
	if stats.TriggeredBySize {
		sizeDeleted, err := ac.cleanupBySize(ctx)
		if err != nil {
			return stats, fmt.Errorf("size-based cleanup failed: %w", err)
		}
		stats.DeletedBySize = sizeDeleted
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	stats.TotalDeleted = stats.DeletedByAge + stats.DeletedByCount + stats.DeletedBySize

	return stats, nil
}

// CleanupStats contains statistics about a cleanup operation
type CleanupStats struct {
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	Duration         time.Duration `json:"duration"`
	TotalDeleted     int           `json:"total_deleted"`
	DeletedByAge     int           `json:"deleted_by_age"`
	DeletedByCount   int           `json:"deleted_by_count"`
	DeletedBySize    int           `json:"deleted_by_size"`
	TotalSizeBefore  int64         `json:"total_size_before"`
	TotalSizeAfter   int64         `json:"total_size_after"`
	TriggeredBySize  bool          `json:"triggered_by_size"`
	WorkflowsCleaned int           `json:"workflows_cleaned"`
}

// cleanupLoop is the main cleanup loop
func (ac *AutoCleaner) cleanupLoop() {
	for {
		select {
		case <-ac.ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			stats, err := ac.RunOnce(ctx)
			cancel()

			if err != nil {
				fmt.Printf("Auto cleanup error: %v\n", err)
			} else if stats.TotalDeleted > 0 {
				fmt.Printf("Auto cleanup completed: deleted %d checkpoints in %v\n",
					stats.TotalDeleted, stats.Duration)
			}

		case <-ac.stopChan:
			return
		}
	}
}

// cleanupByAge removes checkpoints older than MaxAge
func (ac *AutoCleaner) cleanupByAge(ctx context.Context) (int, error) {
	deleted := 0
	cutoff := time.Now().Add(-ac.config.MaxAge)

	// Get all checkpoint keys
	keys, err := ac.manager.store.List(ctx, CheckpointKeyPrefix)
	if err != nil {
		return 0, err
	}

	for _, key := range keys {
		data, err := ac.manager.store.Get(ctx, key)
		if err != nil {
			continue
		}

		checkpoint, err := ac.manager.deserializeCheckpoint(data)
		if err != nil {
			continue
		}

		if checkpoint.Timestamp.Before(cutoff) {
			if err := ac.manager.store.Delete(ctx, key); err != nil {
				fmt.Printf("Warning: failed to delete expired checkpoint %s: %v\n", key, err)
			} else {
				deleted++
			}
		}
	}

	return deleted, nil
}

// cleanupByCount ensures we don't keep too many checkpoints per workflow
func (ac *AutoCleaner) cleanupByCount(ctx context.Context) (int, error) {
	deleted := 0

	// Group checkpoints by workflow
	workflowCheckpoints := make(map[string][]*CheckpointInfo)

	keys, err := ac.manager.store.List(ctx, CheckpointKeyPrefix)
	if err != nil {
		return 0, err
	}

	for _, key := range keys {
		data, err := ac.manager.store.Get(ctx, key)
		if err != nil {
			continue
		}

		checkpoint, err := ac.manager.deserializeCheckpoint(data)
		if err != nil {
			continue
		}

		info := &CheckpointInfo{
			ID:          checkpoint.ID,
			WorkflowID:  checkpoint.WorkflowID,
			Timestamp:   checkpoint.Timestamp,
			CurrentNode: checkpoint.CurrentNode,
			Type:        checkpoint.Type,
			Metadata:    checkpoint.Metadata,
			Size:        int64(len(data)),
		}

		workflowCheckpoints[checkpoint.WorkflowID] = append(workflowCheckpoints[checkpoint.WorkflowID], info)
	}

	// Clean up excess checkpoints for each workflow
	for workflowID, checkpoints := range workflowCheckpoints {
		if len(checkpoints) <= ac.config.MinKeepCount {
			continue
		}

		// Sort by timestamp (newest first)
		for i := 0; i < len(checkpoints)-1; i++ {
			for j := i + 1; j < len(checkpoints); j++ {
				if checkpoints[i].Timestamp.Before(checkpoints[j].Timestamp) {
					checkpoints[i], checkpoints[j] = checkpoints[j], checkpoints[i]
				}
			}
		}

		// Delete oldest checkpoints beyond MinKeepCount
		toDelete := checkpoints[ac.config.MinKeepCount:]
		for _, info := range toDelete {
			key := ac.manager.checkpointKey(workflowID, info.ID)
			if err := ac.manager.store.Delete(ctx, key); err != nil {
				fmt.Printf("Warning: failed to delete excess checkpoint %s: %v\n", key, err)
			} else {
				deleted++
			}
		}
	}

	return deleted, nil
}

// cleanupBySize removes oldest checkpoints when storage is too large
func (ac *AutoCleaner) cleanupBySize(ctx context.Context) (int, error) {
	deleted := 0

	// Get all checkpoints sorted by timestamp (oldest first)
	var allCheckpoints []*CheckpointInfo

	keys, err := ac.manager.store.List(ctx, CheckpointKeyPrefix)
	if err != nil {
		return 0, err
	}

	for _, key := range keys {
		data, err := ac.manager.store.Get(ctx, key)
		if err != nil {
			continue
		}

		checkpoint, err := ac.manager.deserializeCheckpoint(data)
		if err != nil {
			continue
		}

		info := &CheckpointInfo{
			ID:          checkpoint.ID,
			WorkflowID:  checkpoint.WorkflowID,
			Timestamp:   checkpoint.Timestamp,
			CurrentNode: checkpoint.CurrentNode,
			Type:        checkpoint.Type,
			Metadata:    checkpoint.Metadata,
			Size:        int64(len(data)),
		}

		allCheckpoints = append(allCheckpoints, info)
	}

	// Sort by timestamp (oldest first)
	for i := 0; i < len(allCheckpoints)-1; i++ {
		for j := i + 1; j < len(allCheckpoints); j++ {
			if allCheckpoints[i].Timestamp.After(allCheckpoints[j].Timestamp) {
				allCheckpoints[i], allCheckpoints[j] = allCheckpoints[j], allCheckpoints[i]
			}
		}
	}

	// Delete oldest checkpoints until we're under the size limit
	// But respect MinKeepCount per workflow
	workflowCounts := make(map[string]int)
	for _, checkpoint := range allCheckpoints {
		workflowCounts[checkpoint.WorkflowID]++
	}

	for _, checkpoint := range allCheckpoints {
		// Check if we can delete this checkpoint (respect MinKeepCount)
		if workflowCounts[checkpoint.WorkflowID] <= ac.config.MinKeepCount {
			continue
		}

		key := ac.manager.checkpointKey(checkpoint.WorkflowID, checkpoint.ID)
		if err := ac.manager.store.Delete(ctx, key); err != nil {
			fmt.Printf("Warning: failed to delete checkpoint for size limit %s: %v\n", key, err)
		} else {
			deleted++
			workflowCounts[checkpoint.WorkflowID]--

			// Check if we've freed enough space (simplified check)
			if deleted >= 10 {
				break
			}
		}
	}

	return deleted, nil
}
