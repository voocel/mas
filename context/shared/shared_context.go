package shared

import (
	"context"
	"sync"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// SharedContext manages context that can be shared between multiple agents
type SharedContext interface {
	// Context sharing
	ShareContext(ctx context.Context, fromAgent, toAgent string, data map[string]interface{}) error
	GetSharedContext(ctx context.Context, agentID string) (map[string]interface{}, error)
	
	// Global state management
	SetGlobalState(ctx context.Context, key string, value interface{}) error
	GetGlobalState(ctx context.Context, key string) (interface{}, error)
	DeleteGlobalState(ctx context.Context, key string) error
	
	// Agent registration and discovery
	RegisterAgent(ctx context.Context, agentID string, metadata AgentMetadata) error
	UnregisterAgent(ctx context.Context, agentID string) error
	GetActiveAgents(ctx context.Context) ([]string, error)
	GetAgentMetadata(ctx context.Context, agentID string) (*AgentMetadata, error)
	
	// Context synchronization
	SyncContext(ctx context.Context, agentID string, state *contextpkg.ContextState) error
	GetSyncedContext(ctx context.Context, agentID string) (*contextpkg.ContextState, error)
	
	// Event broadcasting
	BroadcastEvent(ctx context.Context, event *SharedEvent) error
	SubscribeToEvents(ctx context.Context, agentID string, eventTypes []string) (<-chan *SharedEvent, error)
	UnsubscribeFromEvents(ctx context.Context, agentID string) error
}

// AgentMetadata contains metadata about an agent
type AgentMetadata struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Capabilities []string               `json:"capabilities"`
	Status       AgentStatus            `json:"status"`
	LastSeen     time.Time              `json:"last_seen"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AgentStatus represents the status of an agent
type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"
	AgentStatusIdle     AgentStatus = "idle"
	AgentStatusBusy     AgentStatus = "busy"
	AgentStatusOffline  AgentStatus = "offline"
)

// SharedEvent represents an event that can be shared between agents
type SharedEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target,omitempty"` // Empty means broadcast
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	TTL       time.Duration          `json:"ttl,omitempty"`
}

// SharedData represents data that can be shared between agents
type SharedData struct {
	ID        string                 `json:"id"`
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	Owner     string                 `json:"owner"`
	Readers   []string               `json:"readers"`
	Writers   []string               `json:"writers"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// InMemorySharedContext implements SharedContext using in-memory storage
type InMemorySharedContext struct {
	globalState   map[string]interface{}
	sharedData    map[string]map[string]interface{} // agentID -> shared data
	agents        map[string]*AgentMetadata
	syncedContext map[string]*contextpkg.ContextState
	subscribers   map[string]chan *SharedEvent // agentID -> event channel
	mutex         sync.RWMutex
}

// NewInMemorySharedContext creates a new in-memory shared context
func NewInMemorySharedContext() *InMemorySharedContext {
	return &InMemorySharedContext{
		globalState:   make(map[string]interface{}),
		sharedData:    make(map[string]map[string]interface{}),
		agents:        make(map[string]*AgentMetadata),
		syncedContext: make(map[string]*contextpkg.ContextState),
		subscribers:   make(map[string]chan *SharedEvent),
	}
}

// ShareContext shares context data from one agent to another
func (imsc *InMemorySharedContext) ShareContext(ctx context.Context, fromAgent, toAgent string, data map[string]interface{}) error {
	imsc.mutex.Lock()
	defer imsc.mutex.Unlock()

	// Initialize shared data for target agent if not exists
	if imsc.sharedData[toAgent] == nil {
		imsc.sharedData[toAgent] = make(map[string]interface{})
	}

	// Share the data
	for key, value := range data {
		sharedKey := fromAgent + ":" + key
		imsc.sharedData[toAgent][sharedKey] = value
	}

	// Broadcast share event
	event := &SharedEvent{
		ID:        generateEventID(),
		Type:      "context.shared",
		Source:    fromAgent,
		Target:    toAgent,
		Data: map[string]interface{}{
			"shared_keys": getKeys(data),
		},
		Timestamp: time.Now(),
	}

	go imsc.broadcastEventAsync(event)

	return nil
}

// GetSharedContext retrieves shared context for an agent
func (imsc *InMemorySharedContext) GetSharedContext(ctx context.Context, agentID string) (map[string]interface{}, error) {
	imsc.mutex.RLock()
	defer imsc.mutex.RUnlock()

	sharedData := imsc.sharedData[agentID]
	if sharedData == nil {
		return make(map[string]interface{}), nil
	}

	// Create a copy to avoid concurrent modification
	result := make(map[string]interface{})
	for key, value := range sharedData {
		result[key] = value
	}

	return result, nil
}

// SetGlobalState sets a global state value
func (imsc *InMemorySharedContext) SetGlobalState(ctx context.Context, key string, value interface{}) error {
	imsc.mutex.Lock()
	defer imsc.mutex.Unlock()

	imsc.globalState[key] = value

	// Broadcast global state change event
	event := &SharedEvent{
		ID:   generateEventID(),
		Type: "global_state.changed",
		Data: map[string]interface{}{
			"key":   key,
			"value": value,
		},
		Timestamp: time.Now(),
	}

	go imsc.broadcastEventAsync(event)

	return nil
}

// GetGlobalState retrieves a global state value
func (imsc *InMemorySharedContext) GetGlobalState(ctx context.Context, key string) (interface{}, error) {
	imsc.mutex.RLock()
	defer imsc.mutex.RUnlock()

	value, exists := imsc.globalState[key]
	if !exists {
		return nil, nil
	}

	return value, nil
}

// DeleteGlobalState deletes a global state value
func (imsc *InMemorySharedContext) DeleteGlobalState(ctx context.Context, key string) error {
	imsc.mutex.Lock()
	defer imsc.mutex.Unlock()

	delete(imsc.globalState, key)

	// Broadcast global state deletion event
	event := &SharedEvent{
		ID:   generateEventID(),
		Type: "global_state.deleted",
		Data: map[string]interface{}{
			"key": key,
		},
		Timestamp: time.Now(),
	}

	go imsc.broadcastEventAsync(event)

	return nil
}

// RegisterAgent registers an agent with the shared context
func (imsc *InMemorySharedContext) RegisterAgent(ctx context.Context, agentID string, metadata AgentMetadata) error {
	imsc.mutex.Lock()
	defer imsc.mutex.Unlock()

	metadata.LastSeen = time.Now()
	imsc.agents[agentID] = &metadata

	// Initialize shared data for the agent
	if imsc.sharedData[agentID] == nil {
		imsc.sharedData[agentID] = make(map[string]interface{})
	}

	// Broadcast agent registration event
	event := &SharedEvent{
		ID:   generateEventID(),
		Type: "agent.registered",
		Data: map[string]interface{}{
			"agent_id":   agentID,
			"agent_name": metadata.Name,
			"agent_type": metadata.Type,
		},
		Timestamp: time.Now(),
	}

	go imsc.broadcastEventAsync(event)

	return nil
}

// UnregisterAgent unregisters an agent from the shared context
func (imsc *InMemorySharedContext) UnregisterAgent(ctx context.Context, agentID string) error {
	imsc.mutex.Lock()
	defer imsc.mutex.Unlock()

	// Remove agent metadata
	delete(imsc.agents, agentID)

	// Clean up shared data
	delete(imsc.sharedData, agentID)

	// Clean up synced context
	delete(imsc.syncedContext, agentID)

	// Close event subscription if exists
	if ch, exists := imsc.subscribers[agentID]; exists {
		close(ch)
		delete(imsc.subscribers, agentID)
	}

	// Broadcast agent unregistration event
	event := &SharedEvent{
		ID:   generateEventID(),
		Type: "agent.unregistered",
		Data: map[string]interface{}{
			"agent_id": agentID,
		},
		Timestamp: time.Now(),
	}

	go imsc.broadcastEventAsync(event)

	return nil
}

// GetActiveAgents returns a list of active agent IDs
func (imsc *InMemorySharedContext) GetActiveAgents(ctx context.Context) ([]string, error) {
	imsc.mutex.RLock()
	defer imsc.mutex.RUnlock()

	var activeAgents []string
	for agentID, metadata := range imsc.agents {
		if metadata.Status == AgentStatusActive || metadata.Status == AgentStatusBusy {
			activeAgents = append(activeAgents, agentID)
		}
	}

	return activeAgents, nil
}

// GetAgentMetadata retrieves metadata for a specific agent
func (imsc *InMemorySharedContext) GetAgentMetadata(ctx context.Context, agentID string) (*AgentMetadata, error) {
	imsc.mutex.RLock()
	defer imsc.mutex.RUnlock()

	metadata, exists := imsc.agents[agentID]
	if !exists {
		return nil, nil
	}

	// Return a copy to avoid concurrent modification
	metadataCopy := *metadata
	return &metadataCopy, nil
}

// SyncContext synchronizes context state for an agent
func (imsc *InMemorySharedContext) SyncContext(ctx context.Context, agentID string, state *contextpkg.ContextState) error {
	imsc.mutex.Lock()
	defer imsc.mutex.Unlock()

	imsc.syncedContext[agentID] = state.Copy()

	// Update agent last seen time
	if agent, exists := imsc.agents[agentID]; exists {
		agent.LastSeen = time.Now()
	}

	return nil
}

// GetSyncedContext retrieves synchronized context for an agent
func (imsc *InMemorySharedContext) GetSyncedContext(ctx context.Context, agentID string) (*contextpkg.ContextState, error) {
	imsc.mutex.RLock()
	defer imsc.mutex.RUnlock()

	state, exists := imsc.syncedContext[agentID]
	if !exists {
		return nil, nil
	}

	return state.Copy(), nil
}

// BroadcastEvent broadcasts an event to all subscribed agents
func (imsc *InMemorySharedContext) BroadcastEvent(ctx context.Context, event *SharedEvent) error {
	imsc.mutex.RLock()
	defer imsc.mutex.RUnlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	if event.ID == "" {
		event.ID = generateEventID()
	}

	// Broadcast to all subscribers or specific target
	for agentID, ch := range imsc.subscribers {
		if event.Target == "" || event.Target == agentID {
			select {
			case ch <- event:
			default:
				// Channel is full, skip this subscriber
			}
		}
	}

	return nil
}

// broadcastEventAsync broadcasts an event asynchronously
func (imsc *InMemorySharedContext) broadcastEventAsync(event *SharedEvent) {
	imsc.BroadcastEvent(context.Background(), event)
}

// SubscribeToEvents subscribes an agent to events
func (imsc *InMemorySharedContext) SubscribeToEvents(ctx context.Context, agentID string, eventTypes []string) (<-chan *SharedEvent, error) {
	imsc.mutex.Lock()
	defer imsc.mutex.Unlock()

	// Create event channel for the agent
	eventChan := make(chan *SharedEvent, 100) // Buffered channel
	imsc.subscribers[agentID] = eventChan

	return eventChan, nil
}

// UnsubscribeFromEvents unsubscribes an agent from events
func (imsc *InMemorySharedContext) UnsubscribeFromEvents(ctx context.Context, agentID string) error {
	imsc.mutex.Lock()
	defer imsc.mutex.Unlock()

	if ch, exists := imsc.subscribers[agentID]; exists {
		close(ch)
		delete(imsc.subscribers, agentID)
	}

	return nil
}

// Helper functions
func generateEventID() string {
	return "event_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func getKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	return keys
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
