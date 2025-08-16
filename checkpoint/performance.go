package checkpoint

import (
	"context"
	"sync"
	"time"
)

// PerformanceManager tracks and optimizes checkpoint operations
type PerformanceManager struct {
	metrics PerformanceMetrics
	mu      sync.RWMutex
}

// PerformanceMetrics contains performance statistics
type PerformanceMetrics struct {
	SaveOperations  int64         `json:"save_operations"`
	LoadOperations  int64         `json:"load_operations"`
	AverageSaveTime time.Duration `json:"average_save_time"`
	AverageLoadTime time.Duration `json:"average_load_time"`
	TotalSaveTime   time.Duration `json:"total_save_time"`
	TotalLoadTime   time.Duration `json:"total_load_time"`
	ErrorCount      int64         `json:"error_count"`
	CacheHits       int64         `json:"cache_hits"`
	CacheMisses     int64         `json:"cache_misses"`
	LastReset       time.Time     `json:"last_reset"`
}

// BatchOperation represents a batch of checkpoint operations
type BatchOperation struct {
	Checkpoints []*Checkpoint
	Operations  []BatchOpType
}

// BatchOpType defines the type of batch operation
type BatchOpType string

const (
	BatchOpSave   BatchOpType = "save"
	BatchOpDelete BatchOpType = "delete"
)

// NewPerformanceManager creates a new performance manager
func NewPerformanceManager() *PerformanceManager {
	return &PerformanceManager{
		metrics: PerformanceMetrics{
			LastReset: time.Now(),
		},
	}
}

// TrackSave tracks a save operation
func (pm *PerformanceManager) TrackSave(duration time.Duration, success bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.metrics.SaveOperations++
	pm.metrics.TotalSaveTime += duration
	pm.metrics.AverageSaveTime = pm.metrics.TotalSaveTime / time.Duration(pm.metrics.SaveOperations)

	if !success {
		pm.metrics.ErrorCount++
	}
}

// TrackLoad tracks a load operation
func (pm *PerformanceManager) TrackLoad(duration time.Duration, success bool, cacheHit bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.metrics.LoadOperations++
	pm.metrics.TotalLoadTime += duration
	pm.metrics.AverageLoadTime = pm.metrics.TotalLoadTime / time.Duration(pm.metrics.LoadOperations)

	if !success {
		pm.metrics.ErrorCount++
	}

	if cacheHit {
		pm.metrics.CacheHits++
	} else {
		pm.metrics.CacheMisses++
	}
}

// GetMetrics returns current performance metrics
func (pm *PerformanceManager) GetMetrics() PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.metrics
}

// ResetMetrics resets performance metrics
func (pm *PerformanceManager) ResetMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.metrics = PerformanceMetrics{
		LastReset: time.Now(),
	}
}

// BatchManager handles batch operations for improved performance
type BatchManager struct {
	manager      *Manager
	batchSize    int
	batchTimeout time.Duration
	pendingOps   []batchOp
	mu           sync.Mutex
	ticker       *time.Ticker
	stopChan     chan struct{}
}

type batchOp struct {
	opType     BatchOpType
	checkpoint *Checkpoint
	workflowID string
	resultChan chan batchResult
}

type batchResult struct {
	err error
}

// NewBatchManager creates a new batch manager
func NewBatchManager(manager *Manager, batchSize int, batchTimeout time.Duration) *BatchManager {
	bm := &BatchManager{
		manager:      manager,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		pendingOps:   make([]batchOp, 0, batchSize),
		stopChan:     make(chan struct{}),
	}

	bm.ticker = time.NewTicker(batchTimeout)
	go bm.processBatches()

	return bm
}

// SaveBatch queues a checkpoint for batch saving
func (bm *BatchManager) SaveBatch(ctx context.Context, checkpoint *Checkpoint) error {
	resultChan := make(chan batchResult, 1)

	bm.mu.Lock()
	bm.pendingOps = append(bm.pendingOps, batchOp{
		opType:     BatchOpSave,
		checkpoint: checkpoint,
		resultChan: resultChan,
	})

	// Process batch if it's full
	if len(bm.pendingOps) >= bm.batchSize {
		go bm.processPendingBatch()
	}
	bm.mu.Unlock()

	// Wait for result
	select {
	case result := <-resultChan:
		return result.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DeleteBatch queues a checkpoint for batch deletion
func (bm *BatchManager) DeleteBatch(ctx context.Context, workflowID, checkpointID string) error {
	resultChan := make(chan batchResult, 1)

	bm.mu.Lock()
	bm.pendingOps = append(bm.pendingOps, batchOp{
		opType:     BatchOpDelete,
		workflowID: workflowID,
		checkpoint: &Checkpoint{ID: checkpointID, WorkflowID: workflowID},
		resultChan: resultChan,
	})

	// Process batch if it's full
	if len(bm.pendingOps) >= bm.batchSize {
		go bm.processPendingBatch()
	}
	bm.mu.Unlock()

	select {
	case result := <-resultChan:
		return result.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// processBatches processes batches on timeout
func (bm *BatchManager) processBatches() {
	for {
		select {
		case <-bm.ticker.C:
			bm.processPendingBatch()
		case <-bm.stopChan:
			bm.ticker.Stop()
			return
		}
	}
}

// processPendingBatch processes all pending operations
func (bm *BatchManager) processPendingBatch() {
	bm.mu.Lock()
	if len(bm.pendingOps) == 0 {
		bm.mu.Unlock()
		return
	}

	ops := make([]batchOp, len(bm.pendingOps))
	copy(ops, bm.pendingOps)
	bm.pendingOps = bm.pendingOps[:0] // Clear pending ops
	bm.mu.Unlock()

	// Group operations by type
	saveOps := make([]batchOp, 0)
	deleteOps := make([]batchOp, 0)

	for _, op := range ops {
		switch op.opType {
		case BatchOpSave:
			saveOps = append(saveOps, op)
		case BatchOpDelete:
			deleteOps = append(deleteOps, op)
		}
	}

	if len(saveOps) > 0 {
		bm.processSaveBatch(saveOps)
	}
	if len(deleteOps) > 0 {
		bm.processDeleteBatch(deleteOps)
	}
}

// processSaveBatch processes a batch of save operations
func (bm *BatchManager) processSaveBatch(ops []batchOp) {
	if batchStore, ok := bm.manager.store.(interface {
		BatchPut(ctx context.Context, items map[string][]byte) error
	}); ok {
		ctx := context.Background()
		items := make(map[string][]byte)

		for _, op := range ops {
			data, err := bm.manager.serializeCheckpoint(op.checkpoint)
			if err != nil {
				op.resultChan <- batchResult{err: err}
				continue
			}

			key := bm.manager.checkpointKey(op.checkpoint.WorkflowID, op.checkpoint.ID)
			items[key] = data
		}

		err := batchStore.BatchPut(ctx, items)

		for _, op := range ops {
			op.resultChan <- batchResult{err: err}
		}
	} else {
		for _, op := range ops {
			err := bm.manager.Save(context.Background(), op.checkpoint)
			op.resultChan <- batchResult{err: err}
		}
	}
}

// processDeleteBatch processes a batch of delete operations
func (bm *BatchManager) processDeleteBatch(ops []batchOp) {
	if batchStore, ok := bm.manager.store.(interface {
		BatchDelete(ctx context.Context, keys []string) error
	}); ok {
		ctx := context.Background()
		keys := make([]string, 0, len(ops))

		for _, op := range ops {
			key := bm.manager.checkpointKey(op.checkpoint.WorkflowID, op.checkpoint.ID)
			keys = append(keys, key)
		}

		err := batchStore.BatchDelete(ctx, keys)
		for _, op := range ops {
			op.resultChan <- batchResult{err: err}
		}
	} else {
		for _, op := range ops {
			err := bm.manager.Delete(context.Background(), op.checkpoint.WorkflowID, op.checkpoint.ID)
			op.resultChan <- batchResult{err: err}
		}
	}
}

// Stop stops the batch manager
func (bm *BatchManager) Stop() {
	close(bm.stopChan)
	bm.processPendingBatch()
}
