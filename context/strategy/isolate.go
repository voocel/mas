package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// IsolateStrategy implements the Isolate strategy for context engineering
// This strategy focuses on isolating context for different agents and tasks
type IsolateStrategy struct {
	BaseStrategy
	isolationManager *IsolationManager
	config           IsolateConfig
}

// IsolateConfig configures the isolate strategy
type IsolateConfig struct {
	EnableAgentIsolation bool          `json:"enable_agent_isolation"`
	EnableTaskIsolation  bool          `json:"enable_task_isolation"`
	MaxIsolatedContexts  int           `json:"max_isolated_contexts"`
	IsolationTimeout     time.Duration `json:"isolation_timeout"`
	EnableSandbox        bool          `json:"enable_sandbox"`
}

// IsolationManager manages isolated contexts
type IsolationManager struct {
	contexts map[string]*IsolatedContext
	mutex    sync.RWMutex
	config   IsolateConfig
}

// IsolatedContext represents an isolated context for an agent or task
type IsolatedContext struct {
	ID         string                   `json:"id"`
	AgentID    string                   `json:"agent_id"`
	TaskID     string                   `json:"task_id,omitempty"`
	Context    *contextpkg.ContextState `json:"context"`
	Sandbox    Sandbox                  `json:"-"`
	SharedRefs map[string]interface{}   `json:"shared_refs"`
	CreatedAt  time.Time                `json:"created_at"`
	LastAccess time.Time                `json:"last_access"`
	mutex      sync.RWMutex
}

// Sandbox defines the interface for sandboxed execution environments
type Sandbox interface {
	Execute(ctx context.Context, code string) (string, error)
	StoreData(key string, data interface{}) error
	RetrieveData(key string) (interface{}, error)
	ListData() []string
	Clear() error
	GetStats() SandboxStats
}

// SandboxStats represents sandbox statistics
type SandboxStats struct {
	DataItems     int           `json:"data_items"`
	MemoryUsage   int64         `json:"memory_usage"`
	ExecutionTime time.Duration `json:"execution_time"`
}

// NewIsolateStrategy creates a new isolate strategy
func NewIsolateStrategy(config ...IsolateConfig) *IsolateStrategy {
	cfg := DefaultIsolateConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &IsolateStrategy{
		BaseStrategy: BaseStrategy{
			name:        "isolate",
			priority:    4, // Medium priority
			description: "Isolates context for different agents and tasks",
		},
		isolationManager: NewIsolationManager(cfg),
		config:           cfg,
	}
}

// DefaultIsolateConfig returns the default isolate configuration
func DefaultIsolateConfig() IsolateConfig {
	return IsolateConfig{
		EnableAgentIsolation: true,
		EnableTaskIsolation:  true,
		MaxIsolatedContexts:  50,
		IsolationTimeout:     1 * time.Hour,
		EnableSandbox:        true,
	}
}

// NewIsolationManager creates a new isolation manager
func NewIsolationManager(config IsolateConfig) *IsolationManager {
	return &IsolationManager{
		contexts: make(map[string]*IsolatedContext),
		config:   config,
	}
}

// Apply applies the isolate strategy to the context state
func (is *IsolateStrategy) Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	newState := state.Copy()

	// Create or update isolated context for the agent
	if is.config.EnableAgentIsolation && state.AgentID != "" {
		isolatedCtx, err := is.isolationManager.GetOrCreateIsolatedContext(state.AgentID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get isolated context: %w", err)
		}

		// Update the isolated context with current state
		if err := is.updateIsolatedContext(isolatedCtx, newState); err != nil {
			return nil, fmt.Errorf("failed to update isolated context: %w", err)
		}

		// Store reference to isolated context
		newState.IsolatedCtx["agent_context_id"] = isolatedCtx.ID
	}

	// Handle task isolation if task ID is available
	if is.config.EnableTaskIsolation {
		if taskID, ok := newState.Scratchpad["task_id"].(string); ok && taskID != "" {
			isolatedCtx, err := is.isolationManager.GetOrCreateIsolatedContext(state.AgentID, taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to get task isolated context: %w", err)
			}

			// Update the isolated context with current state
			if err := is.updateIsolatedContext(isolatedCtx, newState); err != nil {
				return nil, fmt.Errorf("failed to update task isolated context: %w", err)
			}

			// Store reference to task isolated context
			newState.IsolatedCtx["task_context_id"] = isolatedCtx.ID
		}
	}

	// Clean up expired contexts
	is.isolationManager.CleanupExpiredContexts()

	return newState, nil
}

// GetOrCreateIsolatedContext gets or creates an isolated context
func (im *IsolationManager) GetOrCreateIsolatedContext(agentID, taskID string) (*IsolatedContext, error) {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Generate context ID
	contextID := agentID
	if taskID != "" {
		contextID = fmt.Sprintf("%s_%s", agentID, taskID)
	}

	// Check if context already exists
	if ctx, exists := im.contexts[contextID]; exists {
		ctx.LastAccess = time.Now()
		return ctx, nil
	}

	// Check if we've reached the maximum number of contexts
	if len(im.contexts) >= im.config.MaxIsolatedContexts {
		// Remove the oldest context
		im.removeOldestContext()
	}

	// Create new isolated context
	isolatedCtx := &IsolatedContext{
		ID:         contextID,
		AgentID:    agentID,
		TaskID:     taskID,
		Context:    contextpkg.NewContextState("", agentID),
		SharedRefs: make(map[string]interface{}),
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
	}

	// Create sandbox if enabled
	if im.config.EnableSandbox {
		sandbox, err := NewInMemorySandbox(contextID)
		if err != nil {
			return nil, fmt.Errorf("failed to create sandbox: %w", err)
		}
		isolatedCtx.Sandbox = sandbox
	}

	im.contexts[contextID] = isolatedCtx
	return isolatedCtx, nil
}

// updateIsolatedContext updates an isolated context with new state
func (is *IsolateStrategy) updateIsolatedContext(isolatedCtx *IsolatedContext, state *contextpkg.ContextState) error {
	isolatedCtx.mutex.Lock()
	defer isolatedCtx.mutex.Unlock()

	// Update the isolated context with relevant information
	isolatedCtx.Context.Messages = append(isolatedCtx.Context.Messages, state.Messages...)

	// Merge scratchpad data
	for k, v := range state.Scratchpad {
		isolatedCtx.Context.Scratchpad[k] = v
	}

	// Merge selected data
	for k, v := range state.SelectedData {
		isolatedCtx.Context.SelectedData[k] = v
	}

	isolatedCtx.LastAccess = time.Now()
	return nil
}

// CleanupExpiredContexts removes expired isolated contexts
func (im *IsolationManager) CleanupExpiredContexts() {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	now := time.Now()
	for id, ctx := range im.contexts {
		if now.Sub(ctx.LastAccess) > im.config.IsolationTimeout {
			// Clean up sandbox if it exists
			if ctx.Sandbox != nil {
				ctx.Sandbox.Clear()
			}
			delete(im.contexts, id)
		}
	}
}

// removeOldestContext removes the oldest context to make room for new ones
func (im *IsolationManager) removeOldestContext() {
	var oldestID string
	var oldestTime time.Time

	for id, ctx := range im.contexts {
		if oldestID == "" || ctx.LastAccess.Before(oldestTime) {
			oldestID = id
			oldestTime = ctx.LastAccess
		}
	}

	if oldestID != "" {
		if ctx := im.contexts[oldestID]; ctx.Sandbox != nil {
			ctx.Sandbox.Clear()
		}
		delete(im.contexts, oldestID)
	}
}

// GetIsolatedContext retrieves an isolated context by ID
func (im *IsolationManager) GetIsolatedContext(contextID string) (*IsolatedContext, error) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	ctx, exists := im.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("isolated context %s not found", contextID)
	}

	ctx.LastAccess = time.Now()
	return ctx, nil
}

// ListIsolatedContexts returns all isolated context IDs
func (im *IsolationManager) ListIsolatedContexts() []string {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	ids := make([]string, 0, len(im.contexts))
	for id := range im.contexts {
		ids = append(ids, id)
	}
	return ids
}

// GetStats returns isolation manager statistics
func (im *IsolationManager) GetStats() IsolationStats {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	return IsolationStats{
		TotalContexts: len(im.contexts),
		MaxContexts:   im.config.MaxIsolatedContexts,
		ActiveAgents:  im.countActiveAgents(),
		ActiveTasks:   im.countActiveTasks(),
	}
}

// IsolationStats represents isolation manager statistics
type IsolationStats struct {
	TotalContexts int `json:"total_contexts"`
	MaxContexts   int `json:"max_contexts"`
	ActiveAgents  int `json:"active_agents"`
	ActiveTasks   int `json:"active_tasks"`
}

// countActiveAgents counts unique active agents
func (im *IsolationManager) countActiveAgents() int {
	agents := make(map[string]bool)
	for _, ctx := range im.contexts {
		agents[ctx.AgentID] = true
	}
	return len(agents)
}

// countActiveTasks counts unique active tasks
func (im *IsolationManager) countActiveTasks() int {
	tasks := make(map[string]bool)
	for _, ctx := range im.contexts {
		if ctx.TaskID != "" {
			tasks[ctx.TaskID] = true
		}
	}
	return len(tasks)
}

// InMemorySandbox implements Sandbox interface using in-memory storage
type InMemorySandbox struct {
	id    string
	data  map[string]interface{}
	stats SandboxStats
	mutex sync.RWMutex
}

// NewInMemorySandbox creates a new in-memory sandbox
func NewInMemorySandbox(id string) (*InMemorySandbox, error) {
	return &InMemorySandbox{
		id:    id,
		data:  make(map[string]interface{}),
		stats: SandboxStats{},
	}, nil
}

// Execute executes code in the sandbox (simplified implementation)
func (ims *InMemorySandbox) Execute(ctx context.Context, code string) (string, error) {
	start := time.Now()
	defer func() {
		ims.stats.ExecutionTime += time.Since(start)
	}()

	// This is a simplified implementation
	// In a real implementation, you would use a proper code execution sandbox
	return fmt.Sprintf("Executed code in sandbox %s: %s", ims.id, code), nil
}

// StoreData stores data in the sandbox
func (ims *InMemorySandbox) StoreData(key string, data interface{}) error {
	ims.mutex.Lock()
	defer ims.mutex.Unlock()

	ims.data[key] = data
	ims.stats.DataItems = len(ims.data)
	return nil
}

// RetrieveData retrieves data from the sandbox
func (ims *InMemorySandbox) RetrieveData(key string) (interface{}, error) {
	ims.mutex.RLock()
	defer ims.mutex.RUnlock()

	data, exists := ims.data[key]
	if !exists {
		return nil, fmt.Errorf("data with key %s not found", key)
	}
	return data, nil
}

// ListData lists all data keys in the sandbox
func (ims *InMemorySandbox) ListData() []string {
	ims.mutex.RLock()
	defer ims.mutex.RUnlock()

	keys := make([]string, 0, len(ims.data))
	for key := range ims.data {
		keys = append(keys, key)
	}
	return keys
}

// Clear clears all data in the sandbox
func (ims *InMemorySandbox) Clear() error {
	ims.mutex.Lock()
	defer ims.mutex.Unlock()

	ims.data = make(map[string]interface{})
	ims.stats.DataItems = 0
	return nil
}

// GetStats returns sandbox statistics
func (ims *InMemorySandbox) GetStats() SandboxStats {
	ims.mutex.RLock()
	defer ims.mutex.RUnlock()

	return ims.stats
}
