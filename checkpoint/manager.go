package checkpoint

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/voocel/mas/checkpoint/store"
)

const (
	// Current checkpoint format version
	CurrentCheckpointVersion = 1

	// Key prefixes for different data types
	CheckpointKeyPrefix = "checkpoint:"
	WorkflowKeyPrefix   = "workflow:"
)

// Manager implements the Checkpointer interface with auto-cleanup support
type Manager struct {
	store       store.StateStore
	config      CheckpointConfig
	mu          sync.RWMutex
	autoCleaner *AutoCleaner
	compressor  *AdvancedCompressor
}

// NewManager creates a new checkpoint manager with the given store and configuration
func NewManager(store store.StateStore, options ...CheckpointOption) *Manager {
	config := DefaultCheckpointConfig()
	for _, option := range options {
		option(&config)
	}

	manager := &Manager{
		store:  store,
		config: config,
	}

	if config.Compression {
		compConfig := DefaultCompressionConfig()
		manager.compressor = NewAdvancedCompressor(compConfig)
	}

	if config.AutoCleanup {
		cleanupConfig := DefaultCleanupConfig()
		cleanupConfig.MaxAge = time.Duration(config.RetentionDays) * 24 * time.Hour
		cleanupConfig.Interval = config.CleanupInterval

		manager.autoCleaner = NewAutoCleaner(manager, cleanupConfig)
		manager.autoCleaner.Start()
	}

	return manager
}

// Save creates a checkpoint for the specified workflow
func (cm *Manager) Save(ctx context.Context, checkpoint *Checkpoint) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if checkpoint.ID == "" {
		checkpoint.ID = generateCheckpointID()
	}

	checkpoint.Version = CurrentCheckpointVersion
	if checkpoint.Timestamp.IsZero() {
		checkpoint.Timestamp = time.Now()
	}

	data, err := cm.serializeCheckpoint(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to serialize checkpoint: %w", err)
	}

	// Store the checkpoint
	key := cm.checkpointKey(checkpoint.WorkflowID, checkpoint.ID)
	if err := cm.store.Put(ctx, key, data); err != nil {
		return fmt.Errorf("failed to store checkpoint: %w", err)
	}

	// Update the latest checkpoint pointer
	latestKey := cm.latestCheckpointKey(checkpoint.WorkflowID)
	latestData, _ := json.Marshal(map[string]string{
		"checkpoint_id": checkpoint.ID,
		"timestamp":     checkpoint.Timestamp.Format(time.RFC3339),
	})
	if err := cm.store.Put(ctx, latestKey, latestData); err != nil {
		return fmt.Errorf("failed to update latest checkpoint pointer: %w", err)
	}

	// Cleanup old checkpoints if needed
	if cm.config.MaxCheckpoints > 0 {
		go cm.cleanupOldCheckpoints(ctx, checkpoint.WorkflowID)
	}

	return nil
}

// Load retrieves the latest checkpoint for a workflow
func (cm *Manager) Load(ctx context.Context, workflowID string) (*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Get the latest checkpoint ID
	latestKey := cm.latestCheckpointKey(workflowID)
	latestData, err := cm.store.Get(ctx, latestKey)
	if err != nil {
		return nil, fmt.Errorf("no checkpoints found for workflow %s", workflowID)
	}

	var latest map[string]string
	if err := json.Unmarshal(latestData, &latest); err != nil {
		return nil, fmt.Errorf("failed to parse latest checkpoint data: %w", err)
	}

	checkpointID := latest["checkpoint_id"]
	return cm.LoadByID(ctx, workflowID, checkpointID)
}

// LoadByID retrieves a specific checkpoint by ID
func (cm *Manager) LoadByID(ctx context.Context, workflowID, checkpointID string) (*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key := cm.checkpointKey(workflowID, checkpointID)
	data, err := cm.store.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("checkpoint not found: %s/%s", workflowID, checkpointID)
	}

	checkpoint, err := cm.deserializeCheckpoint(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint: %w", err)
	}

	return checkpoint, nil
}

// List returns all checkpoints for a workflow, sorted by timestamp (newest first)
func (cm *Manager) List(ctx context.Context, workflowID string) ([]*CheckpointInfo, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	prefix := cm.workflowCheckpointPrefix(workflowID)
	keys, err := cm.store.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	var infos []*CheckpointInfo
	for _, key := range keys {
		data, err := cm.store.Get(ctx, key)
		if err != nil {
			continue
		}

		checkpoint, err := cm.deserializeCheckpoint(data)
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
		infos = append(infos, info)
	}

	// Sort by timestamp (newest first)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Timestamp.After(infos[j].Timestamp)
	})

	return infos, nil
}

// Delete removes a specific checkpoint
func (cm *Manager) Delete(ctx context.Context, workflowID, checkpointID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	key := cm.checkpointKey(workflowID, checkpointID)
	return cm.store.Delete(ctx, key)
}

// DeleteAll removes all checkpoints for a workflow
func (cm *Manager) DeleteAll(ctx context.Context, workflowID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	prefix := cm.workflowCheckpointPrefix(workflowID)
	keys, err := cm.store.List(ctx, prefix)
	if err != nil {
		return fmt.Errorf("failed to list checkpoints for deletion: %w", err)
	}

	for _, key := range keys {
		if err := cm.store.Delete(ctx, key); err != nil {
			return fmt.Errorf("failed to delete checkpoint %s: %w", key, err)
		}
	}

	// Also delete the latest pointer
	latestKey := cm.latestCheckpointKey(workflowID)
	cm.store.Delete(ctx, latestKey) // Ignore error as it might not exist

	return nil
}

// Cleanup removes checkpoints older than the specified duration
func (cm *Manager) Cleanup(ctx context.Context, olderThan time.Duration) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	keys, err := cm.store.List(ctx, CheckpointKeyPrefix)
	if err != nil {
		return fmt.Errorf("failed to list checkpoints for cleanup: %w", err)
	}

	for _, key := range keys {
		data, err := cm.store.Get(ctx, key)
		if err != nil {
			continue
		}

		checkpoint, err := cm.deserializeCheckpoint(data)
		if err != nil {
			continue
		}

		if checkpoint.Timestamp.Before(cutoff) {
			if err := cm.store.Delete(ctx, key); err != nil {
				fmt.Printf("Warning: failed to delete old checkpoint %s: %v\n", key, err)
			}
		}
	}

	return nil
}

// Close releases any resources held by the checkpointer
func (cm *Manager) Close() error {
	if cm.autoCleaner != nil {
		cm.autoCleaner.Stop()
	}

	return cm.store.Close()
}

// serializeCheckpoint converts a checkpoint to bytes
func (cm *Manager) serializeCheckpoint(checkpoint *Checkpoint) ([]byte, error) {
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return nil, err
	}

	if cm.config.Compression && cm.compressor != nil {
		compressed, err := cm.compressor.Compress(context.Background(), data)
		if err != nil {
			return data, nil
		}

		// Store compressed data with metadata
		wrapper := struct {
			IsCompressed bool            `json:"is_compressed"`
			Compression  *CompressedData `json:"compression,omitempty"`
			Data         []byte          `json:"data,omitempty"`
		}{
			IsCompressed: true,
			Compression:  compressed,
		}

		return json.Marshal(wrapper)
	}

	return data, nil
}

// deserializeCheckpoint converts bytes to a checkpoint
func (cm *Manager) deserializeCheckpoint(data []byte) (*Checkpoint, error) {
	var wrapper struct {
		IsCompressed bool            `json:"is_compressed"`
		Compression  *CompressedData `json:"compression,omitempty"`
		Data         []byte          `json:"data,omitempty"`
	}

	// First try to unmarshal as wrapped data
	if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.IsCompressed && wrapper.Compression != nil {
		if cm.compressor != nil {
			decompressed, err := cm.compressor.Decompress(context.Background(), wrapper.Compression)
			if err == nil {
				data = decompressed
			}
		}
	} else {
		// Try legacy compression (gzip)
		if cm.config.Compression {
			if decompressed, err := cm.decompress(data); err == nil {
				data = decompressed
			}
		}
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, err
	}

	if checkpoint.Version == 0 {
		checkpoint.Version = 1 // Assume version 1 for old checkpoints
	}

	return &checkpoint, nil
}

// compress compresses data using gzip
func (cm *Manager) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		gz.Close()
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompress decompresses gzip data
func (cm *Manager) decompress(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gz); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// cleanupOldCheckpoints removes excess checkpoints beyond MaxCheckpoints
func (cm *Manager) cleanupOldCheckpoints(ctx context.Context, workflowID string) error {
	infos, err := cm.List(ctx, workflowID)
	if err != nil {
		return err
	}

	if len(infos) <= cm.config.MaxCheckpoints {
		return nil
	}

	// Delete oldest checkpoints
	toDelete := infos[cm.config.MaxCheckpoints:]
	for _, info := range toDelete {
		if err := cm.Delete(ctx, workflowID, info.ID); err != nil {
			return err
		}
	}

	return nil
}

// Key generation functions
func (cm *Manager) checkpointKey(workflowID, checkpointID string) string {
	return fmt.Sprintf("%s%s:%s", CheckpointKeyPrefix, workflowID, checkpointID)
}

func (cm *Manager) latestCheckpointKey(workflowID string) string {
	return fmt.Sprintf("%s%s:latest", WorkflowKeyPrefix, workflowID)
}

func (cm *Manager) workflowCheckpointPrefix(workflowID string) string {
	return fmt.Sprintf("%s%s:", CheckpointKeyPrefix, workflowID)
}

// generateCheckpointID creates a unique checkpoint identifier
func generateCheckpointID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("cp_%d_%s", time.Now().UnixNano(), hex.EncodeToString(bytes))
}
