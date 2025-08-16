package checkpoint

import (
	"context"
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

// CheckpointInfo provides summary information about a checkpoint
type CheckpointInfo struct {
	ID            string                 `json:"id"`
	WorkflowID    string                 `json:"workflow_id"`
	Timestamp     time.Time              `json:"timestamp"`
	CurrentNode   string                 `json:"current_node"`
	Type          CheckpointType         `json:"type"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Size          int64                  `json:"size,omitempty"`
}

// CheckpointType indicates the reason for creating a checkpoint
type CheckpointType string

const (
	CheckpointTypeAuto       CheckpointType = "auto"        // Automatic checkpoint
	CheckpointTypeManual     CheckpointType = "manual"      // User-triggered checkpoint
	CheckpointTypeBeforeNode CheckpointType = "before_node" // Before node execution
	CheckpointTypeAfterNode  CheckpointType = "after_node"  // After node execution
)

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

// DefaultCheckpointConfig returns a default checkpoint configuration
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

// CheckpointOption allows configuring checkpoint behavior
type CheckpointOption func(*CheckpointConfig)

// WithAutoSave enables or disables automatic checkpointing
func WithAutoSave(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.AutoSave = enabled
	}
}

// WithSaveInterval sets the interval between automatic checkpoints
func WithSaveInterval(interval time.Duration) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.SaveInterval = interval
	}
}

// WithMaxCheckpoints sets the maximum number of checkpoints to keep
func WithMaxCheckpoints(max int) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.MaxCheckpoints = max
	}
}

// WithCompression enables or disables checkpoint compression
func WithCompression(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.Compression = enabled
	}
}

// WithRetentionDays sets how many days to retain checkpoints
func WithRetentionDays(days int) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.RetentionDays = days
	}
}

// WithSaveBeforeNode enables saving checkpoints before node execution
func WithSaveBeforeNode(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.SaveBeforeNode = enabled
	}
}

// WithSaveAfterNode enables saving checkpoints after node execution
func WithSaveAfterNode(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.SaveAfterNode = enabled
	}
}

// WithAutoCleanup enables or disables automatic cleanup
func WithAutoCleanup(enabled bool) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.AutoCleanup = enabled
	}
}

// WithCleanupInterval sets the interval for automatic cleanup
func WithCleanupInterval(interval time.Duration) CheckpointOption {
	return func(c *CheckpointConfig) {
		c.CleanupInterval = interval
	}
}

// WorkflowContext represents the execution context for a workflow
// This is imported from the main mas package to avoid circular imports
type WorkflowContext = interface{}