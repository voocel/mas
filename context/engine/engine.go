package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// ContextEngine represents the core context engineering engine
type ContextEngine struct {
	strategies   map[string]ContextStrategy
	memory       MemoryStore
	checkpointer Checkpointer
	vectorStore  VectorStore
	config       *Config
	mutex        sync.RWMutex
}

// ContextStrategy defines the interface for context strategies
type ContextStrategy interface {
	Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error)
	Name() string
	Priority() int
}

// MemoryStore defines the interface for memory storage
type MemoryStore interface {
	Store(ctx context.Context, memory *contextpkg.Memory) error
	Retrieve(ctx context.Context, criteria contextpkg.MemoryCriteria) ([]*contextpkg.Memory, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string, limit int) ([]*contextpkg.Memory, error)
}

// Checkpointer defines the interface for checkpoint management
type Checkpointer interface {
	Save(ctx context.Context, checkpoint *contextpkg.Checkpoint) error
	Load(ctx context.Context, threadID, checkpointID string) (*contextpkg.Checkpoint, error)
	List(ctx context.Context, threadID string) ([]*contextpkg.Checkpoint, error)
	Delete(ctx context.Context, threadID, checkpointID string) error
}

// VectorStore defines the interface for vector storage
type VectorStore interface {
	Store(ctx context.Context, id string, vector []float64, metadata map[string]interface{}) error
	Search(ctx context.Context, query string, limit int, category string) ([]contextpkg.VectorSearchResult, error)
	Delete(ctx context.Context, id string) error
}

// Config represents engine configuration
type Config struct {
	MaxTokens          int           `json:"max_tokens"`
	CompressionRatio   float64       `json:"compression_ratio"`
	MemoryRetention    time.Duration `json:"memory_retention"`
	CheckpointInterval time.Duration `json:"checkpoint_interval"`
	DefaultStrategy    string        `json:"default_strategy"`
}

// Option represents a configuration option
type Option func(*ContextEngine)

// NewContextEngine creates a new context engine
func NewContextEngine(opts ...Option) *ContextEngine {
	engine := &ContextEngine{
		strategies: make(map[string]ContextStrategy),
		config:     defaultConfig(),
	}

	for _, opt := range opts {
		opt(engine)
	}

	return engine
}

// WithMemory sets the memory store
func WithMemory(memory MemoryStore) Option {
	return func(ce *ContextEngine) {
		ce.memory = memory
	}
}

// WithCheckpointer sets the checkpointer
func WithCheckpointer(checkpointer Checkpointer) Option {
	return func(ce *ContextEngine) {
		ce.checkpointer = checkpointer
	}
}

// WithVectorStore sets the vector store
func WithVectorStore(vectorStore VectorStore) Option {
	return func(ce *ContextEngine) {
		ce.vectorStore = vectorStore
	}
}

// WithConfig sets the configuration
func WithConfig(config *Config) Option {
	return func(ce *ContextEngine) {
		ce.config = config
	}
}

// RegisterStrategy registers a context strategy
func (ce *ContextEngine) RegisterStrategy(strategy ContextStrategy) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()
	ce.strategies[strategy.Name()] = strategy
}

// ApplyStrategy applies a specific strategy to the context state
func (ce *ContextEngine) ApplyStrategy(
	ctx context.Context,
	strategyName string,
	state *contextpkg.ContextState,
) (*contextpkg.ContextState, error) {
	ce.mutex.RLock()
	strategy, exists := ce.strategies[strategyName]
	ce.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("strategy %s not found", strategyName)
	}

	return strategy.Apply(ctx, state)
}

// GetState retrieves the current context state for a thread
func (ce *ContextEngine) GetState(ctx context.Context, threadID string) (*contextpkg.ContextState, error) {
	if ce.checkpointer == nil {
		return nil, fmt.Errorf("checkpointer not configured")
	}

	checkpoints, err := ce.checkpointer.List(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	if len(checkpoints) == 0 {
		// Return new state if no checkpoints exist
		return contextpkg.NewContextState(threadID, ""), nil
	}

	// Return the latest checkpoint
	latest := checkpoints[0]
	for _, cp := range checkpoints {
		if cp.Timestamp.After(latest.Timestamp) {
			latest = cp
		}
	}

	return latest.State, nil
}

// UpdateState updates the context state for a thread
func (ce *ContextEngine) UpdateState(
	ctx context.Context,
	threadID string,
	update contextpkg.StateUpdate,
) error {
	state, err := ce.GetState(ctx, threadID)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	// Apply updates
	if len(update.Messages) > 0 {
		state.Messages = append(state.Messages, update.Messages...)
	}

	if update.Scratchpad != nil {
		for k, v := range update.Scratchpad {
			state.Scratchpad[k] = v
		}
	}

	if update.SelectedData != nil {
		for k, v := range update.SelectedData {
			state.SelectedData[k] = v
		}
	}

	state.Timestamp = time.Now()

	// Save checkpoint
	return ce.CreateCheckpoint(ctx, state)
}

// CreateCheckpoint creates a new checkpoint
func (ce *ContextEngine) CreateCheckpoint(
	ctx context.Context,
	state *contextpkg.ContextState,
) error {
	if ce.checkpointer == nil {
		return fmt.Errorf("checkpointer not configured")
	}

	checkpoint := &contextpkg.Checkpoint{
		ID:        generateCheckpointID(),
		ThreadID:  state.ThreadID,
		State:     state,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	return ce.checkpointer.Save(ctx, checkpoint)
}

// RestoreFromCheckpoint restores state from a checkpoint
func (ce *ContextEngine) RestoreFromCheckpoint(
	ctx context.Context,
	threadID, checkpointID string,
) (*contextpkg.ContextState, error) {
	if ce.checkpointer == nil {
		return nil, fmt.Errorf("checkpointer not configured")
	}

	checkpoint, err := ce.checkpointer.Load(ctx, threadID, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	return checkpoint.State, nil
}

// GetMemory retrieves the memory store
func (ce *ContextEngine) GetMemory() MemoryStore {
	return ce.memory
}

// GetVectorStore retrieves the vector store
func (ce *ContextEngine) GetVectorStore() VectorStore {
	return ce.vectorStore
}

// GetConfig retrieves the configuration
func (ce *ContextEngine) GetConfig() *Config {
	return ce.config
}

// ListStrategies returns all registered strategies
func (ce *ContextEngine) ListStrategies() []string {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	strategies := make([]string, 0, len(ce.strategies))
	for name := range ce.strategies {
		strategies = append(strategies, name)
	}
	return strategies
}

// defaultConfig returns the default configuration
func defaultConfig() *Config {
	return &Config{
		MaxTokens:          4000,
		CompressionRatio:   0.3,
		MemoryRetention:    24 * time.Hour,
		CheckpointInterval: 5 * time.Minute,
		DefaultStrategy:    "adaptive",
	}
}

// generateCheckpointID generates a unique checkpoint ID
func generateCheckpointID() string {
	return fmt.Sprintf("checkpoint_%d", time.Now().UnixNano())
}
