package mas

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Checkpointer manages workflow state checkpoints for persistence and recovery
type Checkpointer interface {
	// Save creates a checkpoint for the specified workflow
	Save(ctx context.Context, checkpoint *Checkpoint) error

	// Load retrieves the latest checkpoint for a workflow
	Load(ctx context.Context, workflowID string) (*Checkpoint, error)

	// LoadByID retrieves a specific checkpoint by ID
	LoadByID(ctx context.Context, workflowID, checkpointID string) (*Checkpoint, error)

	// List returns all checkpoints for a workflow, sorted by timestamp (newest first)
	List(ctx context.Context, workflowID string) ([]*CheckpointInfo, error)

	// Delete removes a specific checkpoint
	Delete(ctx context.Context, workflowID, checkpointID string) error

	// DeleteAll removes all checkpoints for a workflow
	DeleteAll(ctx context.Context, workflowID string) error

	// Cleanup removes checkpoints older than the specified duration
	Cleanup(ctx context.Context, olderThan time.Duration) error

	// Close releases any resources held by the checkpointer
	Close() error
}

// Checkpoint represents a saved workflow state at a specific point in time
type Checkpoint struct {
	// Unique checkpoint identifier
	ID string `json:"id"`

	// Workflow this checkpoint belongs to
	WorkflowID string `json:"workflow_id"`

	// When this checkpoint was created
	Timestamp time.Time `json:"timestamp"`

	// Current node being executed (or about to be executed)
	CurrentNode string `json:"current_node"`

	// Nodes that have completed execution
	CompletedNodes []string `json:"completed_nodes"`

	// Full workflow context state
	Context *WorkflowContext `json:"context"`

	// Type of checkpoint (auto, manual, before_node, after_node)
	Type CheckpointType `json:"type"`

	// Additional metadata for this checkpoint
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Version for checkpoint format compatibility
	Version int `json:"version"`
}

// CheckpointConfig contains configuration options for checkpointing behavior
type CheckpointConfig struct {
	// Enable automatic checkpointing
	AutoSave bool `json:"auto_save"`

	// Interval between automatic checkpoints
	SaveInterval time.Duration `json:"save_interval"`

	// Maximum number of checkpoints to keep per workflow
	MaxCheckpoints int `json:"max_checkpoints"`

	// Enable compression for checkpoint data
	Compression bool `json:"compression"`

	// Number of days to retain checkpoints
	RetentionDays int `json:"retention_days"`

	// Save checkpoint before each node execution
	SaveBeforeNode bool `json:"save_before_node"`

	// Save checkpoint after each node execution
	SaveAfterNode bool `json:"save_after_node"`

	// Enable automatic cleanup of old checkpoints
	AutoCleanup bool `json:"auto_cleanup"`

	// Interval for automatic cleanup operations
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// CheckpointOption allows configuring checkpoint behavior
type CheckpointOption func(*CheckpointConfig)

// Factory functions for creating checkpointers
func NewFileCheckpointer(dir string, options ...CheckpointOption) (Checkpointer, error) {
	// For now, return a simple error since the implementation needs to be completed
	return nil, fmt.Errorf("file checkpointer not yet implemented")
}

func NewMemoryCheckpointer(options ...CheckpointOption) Checkpointer {
	// For now, return a simple memory checkpointer stub
	return &memoryCheckpointer{
		checkpoints: make(map[string]*Checkpoint),
		config:      DefaultCheckpointConfig(),
	}
}

func NewDefaultCheckpointer() (Checkpointer, error) {
	return NewMemoryCheckpointer(), nil
}

// Simple memory checkpointer implementation
type memoryCheckpointer struct {
	checkpoints map[string]*Checkpoint
	config      CheckpointConfig
	mu          sync.RWMutex
}

func (c *memoryCheckpointer) Save(ctx context.Context, checkpoint *Checkpoint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if checkpoint.ID == "" {
		checkpoint.ID = generateCheckpointID()
	}
	if checkpoint.Timestamp.IsZero() {
		checkpoint.Timestamp = time.Now()
	}

	key := fmt.Sprintf("%s:%s", checkpoint.WorkflowID, checkpoint.ID)
	c.checkpoints[key] = checkpoint
	return nil
}

func (c *memoryCheckpointer) Load(ctx context.Context, workflowID string) (*Checkpoint, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var latest *Checkpoint
	for _, checkpoint := range c.checkpoints {
		if checkpoint.WorkflowID == workflowID {
			if latest == nil || checkpoint.Timestamp.After(latest.Timestamp) {
				latest = checkpoint
			}
		}
	}

	if latest == nil {
		return nil, ErrCheckpointNotFound
	}
	return latest, nil
}

func (c *memoryCheckpointer) LoadByID(ctx context.Context, workflowID, checkpointID string) (*Checkpoint, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", workflowID, checkpointID)
	checkpoint, exists := c.checkpoints[key]
	if !exists {
		return nil, ErrCheckpointNotFound
	}
	return checkpoint, nil
}

func (c *memoryCheckpointer) List(ctx context.Context, workflowID string) ([]*CheckpointInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var infos []*CheckpointInfo
	for _, checkpoint := range c.checkpoints {
		if checkpoint.WorkflowID == workflowID {
			info := &CheckpointInfo{
				ID:          checkpoint.ID,
				WorkflowID:  checkpoint.WorkflowID,
				Timestamp:   checkpoint.Timestamp,
				CurrentNode: checkpoint.CurrentNode,
				Type:        checkpoint.Type,
				Metadata:    checkpoint.Metadata,
				Size:        1024, // Placeholder size
			}
			infos = append(infos, info)
		}
	}
	return infos, nil
}

func (c *memoryCheckpointer) Delete(ctx context.Context, workflowID, checkpointID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%s", workflowID, checkpointID)
	delete(c.checkpoints, key)
	return nil
}

func (c *memoryCheckpointer) DeleteAll(ctx context.Context, workflowID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, checkpoint := range c.checkpoints {
		if checkpoint.WorkflowID == workflowID {
			delete(c.checkpoints, key)
		}
	}
	return nil
}

func (c *memoryCheckpointer) Cleanup(ctx context.Context, olderThan time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for key, checkpoint := range c.checkpoints {
		if checkpoint.Timestamp.Before(cutoff) {
			delete(c.checkpoints, key)
		}
	}
	return nil
}

func (c *memoryCheckpointer) Close() error {
	return nil
}

// Configuration options
func WithAutoSave(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.AutoSave = enabled
	}
}

func WithSaveInterval(interval time.Duration) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.SaveInterval = interval
	}
}

func WithMaxCheckpoints(max int) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.MaxCheckpoints = max
	}
}

func WithCompression(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.Compression = enabled
	}
}

func WithRetentionDays(days int) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.RetentionDays = days
	}
}

func WithSaveBeforeNode(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.SaveBeforeNode = enabled
	}
}

func WithSaveAfterNode(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.SaveAfterNode = enabled
	}
}

func WithAutoCleanup(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.AutoCleanup = enabled
	}
}

func WithCleanupInterval(interval time.Duration) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.CleanupInterval = interval
	}
}

// Utility functions
func CreateCheckpoint(workflowID, currentNode string, completedNodes []string, context *WorkflowContext, checkpointType CheckpointType) *Checkpoint {
	return &Checkpoint{
		ID:             generateCheckpointID(),
		WorkflowID:     workflowID,
		Timestamp:      time.Now(),
		CurrentNode:    currentNode,
		CompletedNodes: completedNodes,
		Context:        context,
		Type:           checkpointType,
		Metadata:       make(map[string]interface{}),
		Version:        1,
	}
}

func ValidateCheckpoint(checkpoint *Checkpoint) error {
	if checkpoint == nil {
		return ErrCheckpointCorrupted
	}
	if checkpoint.ID == "" || checkpoint.WorkflowID == "" {
		return ErrCheckpointCorrupted
	}
	return nil
}

func DefaultCheckpointConfig() CheckpointConfig {
	return CheckpointConfig{
		AutoSave:        true,
		SaveInterval:    30 * time.Second,
		MaxCheckpoints:  10,
		Compression:     true,
		RetentionDays:   7,
		SaveBeforeNode:  false,
		SaveAfterNode:   true,
		AutoCleanup:     true,
		CleanupInterval: time.Hour,
	}
}

// Helper function to generate checkpoint IDs
func generateCheckpointID() string {
	return fmt.Sprintf("checkpoint_%d", time.Now().UnixNano())
}
