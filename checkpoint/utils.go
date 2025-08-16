package checkpoint

import (
	"context"
	"fmt"
	"time"

	"github.com/voocel/mas/checkpoint/store"
)

// NewFileCheckpointer creates a checkpointer that uses filesystem storage
func NewFileCheckpointer(basePath string, options ...CheckpointOption) (Checkpointer, error) {
	fileStore, err := store.NewFileStore(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file store: %w", err)
	}
	
	return NewManager(fileStore, options...), nil
}

// NewMemoryCheckpointer creates a checkpointer that uses in-memory storage
func NewMemoryCheckpointer(options ...CheckpointOption) Checkpointer {
	memStore := store.NewMemoryStore()
	return NewManager(memStore, options...)
}

// NewDefaultCheckpointer creates a checkpointer with sensible defaults for development
func NewDefaultCheckpointer() (Checkpointer, error) {
	// Use ./checkpoints directory by default
	return NewFileCheckpointer("./checkpoints",
		WithAutoSave(true),
		WithSaveInterval(30*time.Second),
		WithMaxCheckpoints(5),
		WithCompression(true),
		WithRetentionDays(7),
	)
}

// Create creates a checkpoint from workflow context
func Create(workflowID, currentNode string, completedNodes []string, context *WorkflowContext, checkpointType CheckpointType) *Checkpoint {
	return &Checkpoint{
		ID:             generateCheckpointID(),
		WorkflowID:     workflowID,
		Timestamp:      time.Now(),
		CurrentNode:    currentNode,
		CompletedNodes: completedNodes,
		Context:        context,
		Type:           checkpointType,
		Metadata:       make(map[string]interface{}),
		Version:        CurrentCheckpointVersion,
	}
}

// QuickSave creates and saves a checkpoint with minimal configuration
func QuickSave(ctx context.Context, checkpointer Checkpointer, workflowID, currentNode string, context *WorkflowContext) (*Checkpoint, error) {
	checkpoint := Create(workflowID, currentNode, []string{}, context, CheckpointTypeManual)
	
	if err := checkpointer.Save(ctx, checkpoint); err != nil {
		return nil, fmt.Errorf("failed to save checkpoint: %w", err)
	}
	
	return checkpoint, nil
}

// QuickLoad loads the latest checkpoint for a workflow
func QuickLoad(ctx context.Context, checkpointer Checkpointer, workflowID string) (*Checkpoint, error) {
	checkpoint, err := checkpointer.Load(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}
	
	return checkpoint, nil
}

// Validate performs basic validation on a checkpoint
func Validate(checkpoint *Checkpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("checkpoint cannot be nil")
	}
	
	if checkpoint.WorkflowID == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}
	
	if checkpoint.Context == nil {
		return fmt.Errorf("workflow context cannot be nil")
	}
	
	if checkpoint.Version <= 0 {
		return fmt.Errorf("invalid checkpoint version: %d", checkpoint.Version)
	}
	
	return nil
}

// IsRecoverable checks if a workflow can be recovered from checkpoints
func IsRecoverable(ctx context.Context, checkpointer Checkpointer, workflowID string) bool {
	_, err := checkpointer.Load(ctx, workflowID)
	return err == nil
}