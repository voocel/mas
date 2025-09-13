package context

import (
	"time"
)

// Message represents a conversation message
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Name      string                 `json:"name,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ContextState represents the complete context state for agents
type ContextState struct {
	// Core message flow
	Messages []Message `json:"messages"`

	// Context engineering strategy data
	Scratchpad    map[string]interface{} `json:"scratchpad"`     // Write strategy
	SelectedData  map[string]interface{} `json:"selected_data"`  // Select strategy
	CompressedCtx *CompressedContext     `json:"compressed"`     // Compress strategy
	IsolatedCtx   map[string]interface{} `json:"isolated"`       // Isolate strategy

	// Metadata
	TokenCount int       `json:"token_count"`
	Timestamp  time.Time `json:"timestamp"`
	AgentID    string    `json:"agent_id"`
	ThreadID   string    `json:"thread_id"`
	SessionID  string    `json:"session_id,omitempty"`
}

// CompressedContext represents compressed context data
type CompressedContext struct {
	Summary       string                 `json:"summary"`
	KeyPoints     []string               `json:"key_points"`
	ImportantData map[string]interface{} `json:"important_data"`
	CompressionRatio float64             `json:"compression_ratio"`
	OriginalTokens   int                 `json:"original_tokens"`
	CompressedTokens int                 `json:"compressed_tokens"`
}

// StateUpdate represents updates to context state
type StateUpdate struct {
	Messages     []Message              `json:"messages,omitempty"`
	Scratchpad   map[string]interface{} `json:"scratchpad,omitempty"`
	SelectedData map[string]interface{} `json:"selected_data,omitempty"`
	SharedData   map[string]interface{} `json:"shared_data,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Checkpoint represents a saved state checkpoint
type Checkpoint struct {
	ID        string        `json:"id"`
	ThreadID  string        `json:"thread_id"`
	State     *ContextState `json:"state"`
	Timestamp time.Time     `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// StateAnalysis represents analysis of current context state
type StateAnalysis struct {
	TokenCount     int     `json:"token_count"`
	MessageCount   int     `json:"message_count"`
	AgentCount     int     `json:"agent_count"`
	Complexity     float64 `json:"complexity"`
	MemoryPressure float64 `json:"memory_pressure"`
	RecentActivity int     `json:"recent_activity"`
}

// MemoryType represents different types of memory
type MemoryType string

const (
	EpisodicMemoryType   MemoryType = "episodic"   // Experience memories
	SemanticMemoryType   MemoryType = "semantic"   // Fact memories
	ProceduralMemoryType MemoryType = "procedural" // Instruction memories
)

// Memory represents a stored memory item
type Memory struct {
	ID          string                 `json:"id"`
	Type        MemoryType             `json:"type"`
	Content     string                 `json:"content"`
	Context     map[string]interface{} `json:"context"`
	Importance  float64                `json:"importance"`
	Timestamp   time.Time              `json:"timestamp"`
	AgentID     string                 `json:"agent_id"`
	SessionID   string                 `json:"session_id"`
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
}

// MemoryCriteria represents criteria for memory selection
type MemoryCriteria struct {
	Type       MemoryType `json:"type,omitempty"`
	AgentID    string     `json:"agent_id,omitempty"`
	SessionID  string     `json:"session_id,omitempty"`
	MinImportance float64 `json:"min_importance,omitempty"`
	MaxAge     time.Duration `json:"max_age,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Query      string     `json:"query,omitempty"`
}

// VectorSearchResult represents a vector search result
type VectorSearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ToolDescription represents a tool available to agents
type ToolDescription struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Category    string                 `json:"category,omitempty"`
	Importance  float64                `json:"importance,omitempty"`
}

// KnowledgeItem represents a piece of knowledge
type KnowledgeItem struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Category    string                 `json:"category"`
	Relevance   float64                `json:"relevance"`
	Source      string                 `json:"source,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ContextBundle represents a bundle of context information
type ContextBundle struct {
	Messages     []Message         `json:"messages"`
	Scratchpad   map[string]interface{} `json:"scratchpad"`
	Memories     []*Memory         `json:"memories"`
	Tools        []*ToolDescription `json:"tools"`
	Knowledge    []*KnowledgeItem  `json:"knowledge"`
	Metadata     map[string]interface{} `json:"metadata"`
	TokenCount   int               `json:"token_count"`
	Timestamp    time.Time         `json:"timestamp"`
}

// Copy creates a deep copy of ContextState
func (cs *ContextState) Copy() *ContextState {
	newState := &ContextState{
		Messages:      make([]Message, len(cs.Messages)),
		Scratchpad:    make(map[string]interface{}),
		SelectedData:  make(map[string]interface{}),
		IsolatedCtx:   make(map[string]interface{}),
		TokenCount:    cs.TokenCount,
		Timestamp:     cs.Timestamp,
		AgentID:       cs.AgentID,
		ThreadID:      cs.ThreadID,
		SessionID:     cs.SessionID,
	}

	copy(newState.Messages, cs.Messages)

	for k, v := range cs.Scratchpad {
		newState.Scratchpad[k] = v
	}

	for k, v := range cs.SelectedData {
		newState.SelectedData[k] = v
	}

	for k, v := range cs.IsolatedCtx {
		newState.IsolatedCtx[k] = v
	}

	if cs.CompressedCtx != nil {
		newState.CompressedCtx = &CompressedContext{
			Summary:          cs.CompressedCtx.Summary,
			KeyPoints:        make([]string, len(cs.CompressedCtx.KeyPoints)),
			ImportantData:    make(map[string]interface{}),
			CompressionRatio: cs.CompressedCtx.CompressionRatio,
			OriginalTokens:   cs.CompressedCtx.OriginalTokens,
			CompressedTokens: cs.CompressedCtx.CompressedTokens,
		}
		copy(newState.CompressedCtx.KeyPoints, cs.CompressedCtx.KeyPoints)
		for k, v := range cs.CompressedCtx.ImportantData {
			newState.CompressedCtx.ImportantData[k] = v
		}
	}

	return newState
}

// NewMessage creates a new message with timestamp
func NewMessage(role, content string) Message {
	return Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewContextState creates a new context state
func NewContextState(threadID, agentID string) *ContextState {
	return &ContextState{
		Messages:     make([]Message, 0),
		Scratchpad:   make(map[string]interface{}),
		SelectedData: make(map[string]interface{}),
		IsolatedCtx:  make(map[string]interface{}),
		Timestamp:    time.Now(),
		AgentID:      agentID,
		ThreadID:     threadID,
	}
}
