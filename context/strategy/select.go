package strategy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// SelectStrategy implements the Select strategy for context engineering
// This strategy focuses on selecting relevant information from memory and knowledge
type SelectStrategy struct {
	BaseStrategy
	memoryStore MemoryStore
	vectorStore VectorStore
	config      SelectConfig
}

// VectorStore defines the interface for vector storage
type VectorStore interface {
	Store(ctx context.Context, id string, vector []float64, metadata map[string]interface{}) error
	Search(ctx context.Context, query string, limit int, category string) ([]contextpkg.VectorSearchResult, error)
	Delete(ctx context.Context, id string) error
}

// SelectConfig configures the select strategy
type SelectConfig struct {
	MaxMemories          int     `json:"max_memories"`
	MaxTools             int     `json:"max_tools"`
	MaxKnowledge         int     `json:"max_knowledge"`
	RelevanceThreshold   float64 `json:"relevance_threshold"`
	EnableSemanticSearch bool    `json:"enable_semantic_search"`
	MemoryDecayFactor    float64 `json:"memory_decay_factor"`
}

// NewSelectStrategy creates a new select strategy
func NewSelectStrategy(memoryStore MemoryStore, vectorStore VectorStore, config ...SelectConfig) *SelectStrategy {
	cfg := DefaultSelectConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &SelectStrategy{
		BaseStrategy: BaseStrategy{
			name:        "select",
			priority:    6, // High priority for relevance
			description: "Selects relevant memories, tools, and knowledge for context",
		},
		memoryStore: memoryStore,
		vectorStore: vectorStore,
		config:      cfg,
	}
}

// DefaultSelectConfig returns the default select configuration
func DefaultSelectConfig() SelectConfig {
	return SelectConfig{
		MaxMemories:          10,
		MaxTools:             5,
		MaxKnowledge:         8,
		RelevanceThreshold:   0.6,
		EnableSemanticSearch: true,
		MemoryDecayFactor:    0.1,
	}
}

// Apply applies the select strategy to the context state
func (ss *SelectStrategy) Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	newState := state.Copy()

	// Extract query from recent messages
	query := ss.extractQuery(state.Messages)

	// Select relevant memories
	if ss.memoryStore != nil {
		memories, err := ss.selectRelevantMemories(ctx, query, state.AgentID, state.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to select memories: %w", err)
		}
		newState.SelectedData["memories"] = memories
	}

	// Select relevant tools
	if ss.vectorStore != nil {
		tools, err := ss.selectRelevantTools(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to select tools: %w", err)
		}
		newState.SelectedData["tools"] = tools
	}

	// Select relevant knowledge
	if ss.vectorStore != nil {
		knowledge, err := ss.selectRelevantKnowledge(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to select knowledge: %w", err)
		}
		newState.SelectedData["knowledge"] = knowledge
	}

	return newState, nil
}

// extractQuery extracts a search query from recent messages
func (ss *SelectStrategy) extractQuery(messages []contextpkg.Message) string {
	if len(messages) == 0 {
		return ""
	}

	// Use the last few messages to build a query
	var queryParts []string
	start := len(messages) - 3
	if start < 0 {
		start = 0
	}

	for i := start; i < len(messages); i++ {
		msg := messages[i]
		if msg.Role == "user" || msg.Role == "assistant" {
			// Extract key terms from the message
			terms := ss.extractKeyTerms(msg.Content)
			queryParts = append(queryParts, terms...)
		}
	}

	return strings.Join(queryParts, " ")
}

// extractKeyTerms extracts key terms from text
func (ss *SelectStrategy) extractKeyTerms(text string) []string {
	// Simple keyword extraction - could be enhanced with NLP
	words := strings.Fields(strings.ToLower(text))
	var terms []string

	// Filter out common stop words and short words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
	}

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 2 && !stopWords[word] {
			terms = append(terms, word)
		}
	}

	return terms
}

// selectRelevantMemories selects relevant memories based on query and context
func (ss *SelectStrategy) selectRelevantMemories(ctx context.Context, query, agentID, sessionID string) ([]*contextpkg.Memory, error) {
	var allMemories []*contextpkg.Memory

	// Search by query if semantic search is enabled
	if ss.config.EnableSemanticSearch && query != "" {
		searchResults, err := ss.memoryStore.Search(ctx, query, ss.config.MaxMemories*2)
		if err == nil {
			allMemories = append(allMemories, searchResults...)
		}
	}

	// Get recent memories for the current agent/session
	criteria := contextpkg.MemoryCriteria{
		AgentID:   agentID,
		SessionID: sessionID,
		MaxAge:    24 * time.Hour, // Last 24 hours
		Limit:     ss.config.MaxMemories,
	}

	recentMemories, err := ss.memoryStore.Retrieve(ctx, criteria)
	if err != nil {
		return nil, err
	}

	allMemories = append(allMemories, recentMemories...)

	// Remove duplicates and rank by relevance
	uniqueMemories := ss.deduplicateMemories(allMemories)
	rankedMemories := ss.rankMemoriesByRelevance(uniqueMemories, query)

	// Apply memory decay based on age
	ss.applyMemoryDecay(rankedMemories)

	// Filter by relevance threshold and limit
	var selectedMemories []*contextpkg.Memory
	for _, memory := range rankedMemories {
		if len(selectedMemories) >= ss.config.MaxMemories {
			break
		}
		if memory.Importance >= ss.config.RelevanceThreshold {
			selectedMemories = append(selectedMemories, memory)
		}
	}

	return selectedMemories, nil
}

// selectRelevantTools selects relevant tools based on query
func (ss *SelectStrategy) selectRelevantTools(ctx context.Context, query string) ([]*contextpkg.ToolDescription, error) {
	if query == "" {
		return []*contextpkg.ToolDescription{}, nil
	}

	// Search for relevant tools using vector search
	results, err := ss.vectorStore.Search(ctx, query, ss.config.MaxTools, "tools")
	if err != nil {
		return nil, err
	}

	var tools []*contextpkg.ToolDescription
	for _, result := range results {
		if result.Score >= ss.config.RelevanceThreshold {
			tool := &contextpkg.ToolDescription{
				Name:        result.Metadata["name"].(string),
				Description: result.Content,
				Parameters:  result.Metadata["parameters"].(map[string]interface{}),
				Category:    result.Metadata["category"].(string),
				Importance:  result.Score,
			}
			tools = append(tools, tool)
		}
	}

	return tools, nil
}

// selectRelevantKnowledge selects relevant knowledge based on query
func (ss *SelectStrategy) selectRelevantKnowledge(ctx context.Context, query string) ([]*contextpkg.KnowledgeItem, error) {
	if query == "" {
		return []*contextpkg.KnowledgeItem{}, nil
	}

	// Search for relevant knowledge using vector search
	results, err := ss.vectorStore.Search(ctx, query, ss.config.MaxKnowledge, "knowledge")
	if err != nil {
		return nil, err
	}

	var knowledge []*contextpkg.KnowledgeItem
	for _, result := range results {
		if result.Score >= ss.config.RelevanceThreshold {
			item := &contextpkg.KnowledgeItem{
				ID:        result.ID,
				Content:   result.Content,
				Category:  result.Metadata["category"].(string),
				Relevance: result.Score,
				Source:    result.Metadata["source"].(string),
				Metadata:  result.Metadata,
				Timestamp: time.Now(),
			}
			knowledge = append(knowledge, item)
		}
	}

	return knowledge, nil
}

// deduplicateMemories removes duplicate memories
func (ss *SelectStrategy) deduplicateMemories(memories []*contextpkg.Memory) []*contextpkg.Memory {
	seen := make(map[string]bool)
	var unique []*contextpkg.Memory

	for _, memory := range memories {
		if !seen[memory.ID] {
			seen[memory.ID] = true
			unique = append(unique, memory)
		}
	}

	return unique
}

// rankMemoriesByRelevance ranks memories by relevance to query
func (ss *SelectStrategy) rankMemoriesByRelevance(memories []*contextpkg.Memory, query string) []*contextpkg.Memory {
	if query == "" {
		// Sort by importance and recency if no query
		sort.Slice(memories, func(i, j int) bool {
			if memories[i].Importance != memories[j].Importance {
				return memories[i].Importance > memories[j].Importance
			}
			return memories[i].Timestamp.After(memories[j].Timestamp)
		})
		return memories
	}

	// Calculate relevance scores
	queryTerms := ss.extractKeyTerms(query)
	for _, memory := range memories {
		relevanceScore := ss.calculateRelevanceScore(memory.Content, queryTerms)
		// Combine with existing importance
		memory.Importance = (memory.Importance + relevanceScore) / 2
	}

	// Sort by combined score
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].Importance > memories[j].Importance
	})

	return memories
}

// calculateRelevanceScore calculates relevance score based on term matching
func (ss *SelectStrategy) calculateRelevanceScore(content string, queryTerms []string) float64 {
	if len(queryTerms) == 0 {
		return 0.5
	}

	contentLower := strings.ToLower(content)
	matches := 0

	for _, term := range queryTerms {
		if strings.Contains(contentLower, term) {
			matches++
		}
	}

	return float64(matches) / float64(len(queryTerms))
}

// applyMemoryDecay applies decay factor based on memory age
func (ss *SelectStrategy) applyMemoryDecay(memories []*contextpkg.Memory) {
	now := time.Now()
	for _, memory := range memories {
		age := now.Sub(memory.Timestamp)
		decayFactor := 1.0 - (age.Hours()*ss.config.MemoryDecayFactor)/24.0
		if decayFactor < 0.1 {
			decayFactor = 0.1 // Minimum decay
		}
		memory.Importance *= decayFactor
	}
}
