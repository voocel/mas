package strategy

import (
	"context"
	"fmt"
	"strings"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// WriteStrategy implements the Write strategy for context engineering
// This strategy focuses on persisting information outside the context window
type WriteStrategy struct {
	BaseStrategy
	memoryStore MemoryStore
	config      WriteConfig
}

// MemoryStore defines the interface for memory storage
type MemoryStore interface {
	Store(ctx context.Context, memory *contextpkg.Memory) error
	Retrieve(ctx context.Context, criteria contextpkg.MemoryCriteria) ([]*contextpkg.Memory, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string, limit int) ([]*contextpkg.Memory, error)
}

// WriteConfig configures the write strategy
type WriteConfig struct {
	EnableScratchpad          bool    `json:"enable_scratchpad"`
	EnableMemoryStorage       bool    `json:"enable_memory_storage"`
	MemoryImportanceThreshold float64 `json:"memory_importance_threshold"`
	MaxScratchpadSize         int     `json:"max_scratchpad_size"`
	AutoSummarize             bool    `json:"auto_summarize"`
}

// NewWriteStrategy creates a new write strategy
func NewWriteStrategy(memoryStore MemoryStore, config ...WriteConfig) *WriteStrategy {
	cfg := DefaultWriteConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &WriteStrategy{
		BaseStrategy: BaseStrategy{
			name:        "write",
			priority:    7, // High priority for persistence
			description: "Persists important information to scratchpad and memory",
		},
		memoryStore: memoryStore,
		config:      cfg,
	}
}

// DefaultWriteConfig returns the default write configuration
func DefaultWriteConfig() WriteConfig {
	return WriteConfig{
		EnableScratchpad:          true,
		EnableMemoryStorage:       true,
		MemoryImportanceThreshold: 0.7,
		MaxScratchpadSize:         1000,
		AutoSummarize:             true,
	}
}

// Apply applies the write strategy to the context state
func (ws *WriteStrategy) Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	newState := state.Copy()

	// Apply scratchpad operations
	if ws.config.EnableScratchpad {
		if err := ws.applyScratchpad(ctx, newState); err != nil {
			return nil, fmt.Errorf("scratchpad operation failed: %w", err)
		}
	}

	// Apply memory storage operations
	if ws.config.EnableMemoryStorage && ws.memoryStore != nil {
		if err := ws.applyMemoryStorage(ctx, newState); err != nil {
			return nil, fmt.Errorf("memory storage operation failed: %w", err)
		}
	}

	return newState, nil
}

// applyScratchpad manages the scratchpad operations
func (ws *WriteStrategy) applyScratchpad(ctx context.Context, state *contextpkg.ContextState) error {
	// Extract important information from recent messages
	importantInfo := ws.extractImportantInfo(state.Messages)

	// Update scratchpad with important information
	for key, value := range importantInfo {
		state.Scratchpad[key] = value
	}

	// Add current task progress if available
	if state.AgentID != "" {
		state.Scratchpad["current_agent"] = state.AgentID
		state.Scratchpad["last_update"] = time.Now()
	}

	// Add conversation summary if auto-summarize is enabled
	if ws.config.AutoSummarize && len(state.Messages) > 5 {
		summary := ws.generateConversationSummary(state.Messages)
		state.Scratchpad["conversation_summary"] = summary
	}

	// Manage scratchpad size
	if err := ws.manageScratchpadSize(state); err != nil {
		return fmt.Errorf("failed to manage scratchpad size: %w", err)
	}

	return nil
}

// applyMemoryStorage manages memory storage operations
func (ws *WriteStrategy) applyMemoryStorage(ctx context.Context, state *contextpkg.ContextState) error {
	// Extract memories from recent messages
	memories := ws.extractMemories(state.Messages, state.AgentID, state.SessionID)

	// Store important memories
	for _, memory := range memories {
		if memory.Importance >= ws.config.MemoryImportanceThreshold {
			if err := ws.memoryStore.Store(ctx, memory); err != nil {
				return fmt.Errorf("failed to store memory %s: %w", memory.ID, err)
			}
		}
	}

	return nil
}

// extractImportantInfo extracts important information from messages
func (ws *WriteStrategy) extractImportantInfo(messages []contextpkg.Message) map[string]interface{} {
	info := make(map[string]interface{})

	// Extract key-value pairs, decisions, and important facts
	for i, msg := range messages {
		// Look for decision patterns
		if ws.containsDecision(msg.Content) {
			info[fmt.Sprintf("decision_%d", i)] = map[string]interface{}{
				"content":   msg.Content,
				"timestamp": msg.Timestamp,
				"role":      msg.Role,
			}
		}

		// Look for task assignments
		if ws.containsTaskAssignment(msg.Content) {
			info[fmt.Sprintf("task_%d", i)] = map[string]interface{}{
				"content":   msg.Content,
				"timestamp": msg.Timestamp,
				"role":      msg.Role,
			}
		}

		// Look for important facts or data
		if ws.containsImportantData(msg.Content) {
			info[fmt.Sprintf("data_%d", i)] = map[string]interface{}{
				"content":   msg.Content,
				"timestamp": msg.Timestamp,
				"role":      msg.Role,
			}
		}
	}

	return info
}

// extractMemories extracts memories from messages
func (ws *WriteStrategy) extractMemories(messages []contextpkg.Message, agentID, sessionID string) []*contextpkg.Memory {
	memories := make([]*contextpkg.Memory, 0)

	for i, msg := range messages {
		// Create episodic memory for significant interactions
		if ws.isSignificantInteraction(msg) {
			memory := &contextpkg.Memory{
				ID:      fmt.Sprintf("episodic_%s_%d", sessionID, i),
				Type:    contextpkg.EpisodicMemoryType,
				Content: msg.Content,
				Context: map[string]interface{}{
					"role":          msg.Role,
					"agent_id":      agentID,
					"session_id":    sessionID,
					"message_index": i,
				},
				Importance:  ws.calculateImportance(msg),
				Timestamp:   msg.Timestamp,
				AgentID:     agentID,
				SessionID:   sessionID,
				AccessCount: 0,
				LastAccess:  time.Now(),
			}
			memories = append(memories, memory)
		}

		// Create semantic memory for facts and knowledge
		if ws.containsFactualInformation(msg) {
			memory := &contextpkg.Memory{
				ID:      fmt.Sprintf("semantic_%s_%d", sessionID, i),
				Type:    contextpkg.SemanticMemoryType,
				Content: ws.extractFactualContent(msg.Content),
				Context: map[string]interface{}{
					"source":     "conversation",
					"agent_id":   agentID,
					"session_id": sessionID,
				},
				Importance:  ws.calculateFactualImportance(msg),
				Timestamp:   msg.Timestamp,
				AgentID:     agentID,
				SessionID:   sessionID,
				AccessCount: 0,
				LastAccess:  time.Now(),
			}
			memories = append(memories, memory)
		}

		// Create procedural memory for instructions and procedures
		if ws.containsProcedure(msg) {
			memory := &contextpkg.Memory{
				ID:      fmt.Sprintf("procedural_%s_%d", sessionID, i),
				Type:    contextpkg.ProceduralMemoryType,
				Content: ws.extractProcedureContent(msg.Content),
				Context: map[string]interface{}{
					"procedure_type": "instruction",
					"agent_id":       agentID,
					"session_id":     sessionID,
				},
				Importance:  ws.calculateProceduralImportance(msg),
				Timestamp:   msg.Timestamp,
				AgentID:     agentID,
				SessionID:   sessionID,
				AccessCount: 0,
				LastAccess:  time.Now(),
			}
			memories = append(memories, memory)
		}
	}

	return memories
}

// Helper methods for content analysis
func (ws *WriteStrategy) containsDecision(content string) bool {
	decisionKeywords := []string{"decide", "decision", "choose", "select", "determine"}
	content = strings.ToLower(content)
	for _, keyword := range decisionKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

func (ws *WriteStrategy) containsTaskAssignment(content string) bool {
	taskKeywords := []string{"task", "assign", "do", "complete", "work on", "handle"}
	content = strings.ToLower(content)
	for _, keyword := range taskKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

func (ws *WriteStrategy) containsImportantData(content string) bool {
	dataKeywords := []string{"data", "result", "output", "finding", "conclusion", "analysis"}
	content = strings.ToLower(content)
	for _, keyword := range dataKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

func (ws *WriteStrategy) isSignificantInteraction(msg contextpkg.Message) bool {
	// Consider interactions significant if they are long enough or contain important keywords
	return len(msg.Content) > 50 || ws.containsDecision(msg.Content) || ws.containsTaskAssignment(msg.Content)
}

func (ws *WriteStrategy) containsFactualInformation(msg contextpkg.Message) bool {
	factKeywords := []string{"fact", "information", "data", "statistic", "number", "percentage"}
	content := strings.ToLower(msg.Content)
	for _, keyword := range factKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

func (ws *WriteStrategy) containsProcedure(msg contextpkg.Message) bool {
	procedureKeywords := []string{"step", "procedure", "process", "method", "how to", "instruction"}
	for _, keyword := range procedureKeywords {
		if contains(msg.Content, keyword) {
			return true
		}
	}
	return false
}

// Content extraction methods
func (ws *WriteStrategy) extractFactualContent(content string) string {
	// Simple extraction - could be enhanced with NLP
	return content
}

func (ws *WriteStrategy) extractProcedureContent(content string) string {
	// Simple extraction - could be enhanced with NLP
	return content
}

// Importance calculation methods
func (ws *WriteStrategy) calculateImportance(msg contextpkg.Message) float64 {
	importance := 0.5 // Base importance

	// Increase importance based on content length
	if len(msg.Content) > 100 {
		importance += 0.1
	}
	if len(msg.Content) > 200 {
		importance += 0.1
	}

	// Increase importance for decisions and tasks
	if ws.containsDecision(msg.Content) {
		importance += 0.2
	}
	if ws.containsTaskAssignment(msg.Content) {
		importance += 0.2
	}

	// Cap at 1.0
	if importance > 1.0 {
		importance = 1.0
	}

	return importance
}

func (ws *WriteStrategy) calculateFactualImportance(msg contextpkg.Message) float64 {
	// Facts are generally important
	return 0.8
}

func (ws *WriteStrategy) calculateProceduralImportance(msg contextpkg.Message) float64 {
	// Procedures are very important for future reference
	return 0.9
}

// Utility methods
func (ws *WriteStrategy) generateConversationSummary(messages []contextpkg.Message) string {
	if len(messages) == 0 {
		return "No conversation to summarize"
	}

	// Simple summary generation - could be enhanced with LLM
	return fmt.Sprintf("Conversation with %d messages, latest from %s",
		len(messages), messages[len(messages)-1].Timestamp.Format("15:04:05"))
}

func (ws *WriteStrategy) manageScratchpadSize(state *contextpkg.ContextState) error {
	// Remove old entries if scratchpad is too large
	if len(state.Scratchpad) > ws.config.MaxScratchpadSize {
		// Simple cleanup - remove oldest entries
		// In a real implementation, you might want to be smarter about what to remove
		count := 0
		for key := range state.Scratchpad {
			if count >= ws.config.MaxScratchpadSize/2 {
				break
			}
			delete(state.Scratchpad, key)
			count++
		}
	}
	return nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	// Simple case-insensitive contains
	// Could be enhanced with better string matching
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
