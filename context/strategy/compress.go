package strategy

import (
	"context"
	"fmt"
	"sort"
	"strings"

	contextpkg "github.com/voocel/mas/context"
)

// CompressStrategy implements the Compress strategy for context engineering
// This strategy focuses on compressing and summarizing context to fit within token limits
type CompressStrategy struct {
	BaseStrategy
	summarizer Summarizer
	config     CompressConfig
}

// Summarizer defines the interface for text summarization
type Summarizer interface {
	Summarize(ctx context.Context, text string, maxLength int) (string, error)
	SummarizeMessages(ctx context.Context, messages []contextpkg.Message) (string, error)
	ExtractKeyPoints(ctx context.Context, text string, maxPoints int) ([]string, error)
}

// CompressConfig configures the compress strategy
type CompressConfig struct {
	MaxTokens        int     `json:"max_tokens"`
	CompressionRatio float64 `json:"compression_ratio"`
	PreserveRecent   int     `json:"preserve_recent"`
	EnableSummary    bool    `json:"enable_summary"`
	EnableKeyPoints  bool    `json:"enable_key_points"`
	MinImportance    float64 `json:"min_importance"`
}

// NewCompressStrategy creates a new compress strategy
func NewCompressStrategy(summarizer Summarizer, config ...CompressConfig) *CompressStrategy {
	cfg := DefaultCompressConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &CompressStrategy{
		BaseStrategy: BaseStrategy{
			name:        "compress",
			priority:    5, // Medium-high priority
			description: "Compresses context through summarization and pruning",
		},
		summarizer: summarizer,
		config:     cfg,
	}
}

// DefaultCompressConfig returns the default compress configuration
func DefaultCompressConfig() CompressConfig {
	return CompressConfig{
		MaxTokens:        4000,
		CompressionRatio: 0.3,
		PreserveRecent:   5,
		EnableSummary:    true,
		EnableKeyPoints:  true,
		MinImportance:    0.5,
	}
}

// Apply applies the compress strategy to the context state
func (cs *CompressStrategy) Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	newState := state.Copy()

	// Calculate current token count
	currentTokens := cs.estimateTokenCount(newState)

	// Check if compression is needed
	if currentTokens <= cs.config.MaxTokens {
		return newState, nil // No compression needed
	}

	// Apply compression
	if err := cs.compressMessages(ctx, newState); err != nil {
		return nil, fmt.Errorf("failed to compress messages: %w", err)
	}

	if err := cs.compressScratchpad(ctx, newState); err != nil {
		return nil, fmt.Errorf("failed to compress scratchpad: %w", err)
	}

	if err := cs.compressSelectedData(ctx, newState); err != nil {
		return nil, fmt.Errorf("failed to compress selected data: %w", err)
	}

	// Update token count
	newState.TokenCount = cs.estimateTokenCount(newState)

	return newState, nil
}

// compressMessages compresses the message history
func (cs *CompressStrategy) compressMessages(ctx context.Context, state *contextpkg.ContextState) error {
	if len(state.Messages) <= cs.config.PreserveRecent {
		return nil // Not enough messages to compress
	}

	// Preserve recent messages
	recentMessages := state.Messages[len(state.Messages)-cs.config.PreserveRecent:]
	oldMessages := state.Messages[:len(state.Messages)-cs.config.PreserveRecent]

	// Create compressed context if not exists
	if state.CompressedCtx == nil {
		state.CompressedCtx = &contextpkg.CompressedContext{
			ImportantData: make(map[string]interface{}),
		}
	}

	// Summarize old messages
	if cs.config.EnableSummary && cs.summarizer != nil {
		summary, err := cs.summarizer.SummarizeMessages(ctx, oldMessages)
		if err != nil {
			return fmt.Errorf("failed to summarize messages: %w", err)
		}
		state.CompressedCtx.Summary = summary
	}

	// Extract key points
	if cs.config.EnableKeyPoints && cs.summarizer != nil {
		allText := cs.messagesToText(oldMessages)
		keyPoints, err := cs.summarizer.ExtractKeyPoints(ctx, allText, 10)
		if err != nil {
			return fmt.Errorf("failed to extract key points: %w", err)
		}
		state.CompressedCtx.KeyPoints = keyPoints
	}

	// Calculate compression metrics
	originalTokens := cs.estimateTokensForMessages(state.Messages)
	compressedTokens := cs.estimateTokensForMessages(recentMessages) +
		cs.estimateTokens(state.CompressedCtx.Summary) +
		cs.estimateTokensForKeyPoints(state.CompressedCtx.KeyPoints)

	state.CompressedCtx.OriginalTokens = originalTokens
	state.CompressedCtx.CompressedTokens = compressedTokens
	if originalTokens > 0 {
		state.CompressedCtx.CompressionRatio = float64(compressedTokens) / float64(originalTokens)
	}

	// Replace messages with recent ones only
	state.Messages = recentMessages

	return nil
}

// compressScratchpad compresses the scratchpad data
func (cs *CompressStrategy) compressScratchpad(ctx context.Context, state *contextpkg.ContextState) error {
	if len(state.Scratchpad) == 0 {
		return nil
	}

	// Rank scratchpad entries by importance
	entries := cs.rankScratchpadEntries(state.Scratchpad)

	// Keep only the most important entries
	maxEntries := int(float64(len(entries)) * cs.config.CompressionRatio)
	if maxEntries < 5 {
		maxEntries = 5 // Keep at least 5 entries
	}

	compressedScratchpad := make(map[string]interface{})
	for i := 0; i < maxEntries && i < len(entries); i++ {
		entry := entries[i]
		compressedScratchpad[entry.Key] = entry.Value
	}

	state.Scratchpad = compressedScratchpad
	return nil
}

// compressSelectedData compresses the selected data
func (cs *CompressStrategy) compressSelectedData(ctx context.Context, state *contextpkg.ContextState) error {
	if len(state.SelectedData) == 0 {
		return nil
	}

	// Compress memories
	if memories, ok := state.SelectedData["memories"].([]*contextpkg.Memory); ok {
		compressed := cs.compressMemories(memories)
		state.SelectedData["memories"] = compressed
	}

	// Compress tools (keep only the most relevant)
	if tools, ok := state.SelectedData["tools"].([]*contextpkg.ToolDescription); ok {
		compressed := cs.compressTools(tools)
		state.SelectedData["tools"] = compressed
	}

	// Compress knowledge
	if knowledge, ok := state.SelectedData["knowledge"].([]*contextpkg.KnowledgeItem); ok {
		compressed := cs.compressKnowledge(knowledge)
		state.SelectedData["knowledge"] = compressed
	}

	return nil
}

// ScratchpadEntry represents a scratchpad entry with importance
type ScratchpadEntry struct {
	Key        string
	Value      interface{}
	Importance float64
}

// rankScratchpadEntries ranks scratchpad entries by importance
func (cs *CompressStrategy) rankScratchpadEntries(scratchpad map[string]interface{}) []ScratchpadEntry {
	entries := make([]ScratchpadEntry, 0, len(scratchpad))

	for key, value := range scratchpad {
		importance := cs.calculateScratchpadImportance(key, value)
		entries = append(entries, ScratchpadEntry{
			Key:        key,
			Value:      value,
			Importance: importance,
		})
	}

	// Sort by importance (descending)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Importance > entries[j].Importance
	})

	return entries
}

// calculateScratchpadImportance calculates importance of a scratchpad entry
func (cs *CompressStrategy) calculateScratchpadImportance(key string, value interface{}) float64 {
	importance := 0.5 // Base importance

	// Increase importance for certain key patterns
	keyLower := strings.ToLower(key)
	if strings.Contains(keyLower, "decision") {
		importance += 0.3
	}
	if strings.Contains(keyLower, "task") {
		importance += 0.2
	}
	if strings.Contains(keyLower, "current") {
		importance += 0.2
	}
	if strings.Contains(keyLower, "summary") {
		importance += 0.1
	}

	// Increase importance based on value type and content
	switch v := value.(type) {
	case string:
		if len(v) > 100 {
			importance += 0.1
		}
	case map[string]interface{}:
		importance += 0.2 // Structured data is usually important
	case []interface{}:
		importance += 0.15 // Lists are moderately important
	}

	// Cap at 1.0
	if importance > 1.0 {
		importance = 1.0
	}

	return importance
}

// compressMemories compresses memory list
func (cs *CompressStrategy) compressMemories(memories []*contextpkg.Memory) []*contextpkg.Memory {
	// Filter by minimum importance
	var filtered []*contextpkg.Memory
	for _, memory := range memories {
		if memory.Importance >= cs.config.MinImportance {
			filtered = append(filtered, memory)
		}
	}

	// Sort by importance and keep top entries
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Importance > filtered[j].Importance
	})

	maxMemories := int(float64(len(filtered)) * cs.config.CompressionRatio)
	if maxMemories < 3 {
		maxMemories = 3 // Keep at least 3 memories
	}

	if len(filtered) <= maxMemories {
		return filtered
	}

	return filtered[:maxMemories]
}

// compressTools compresses tool list
func (cs *CompressStrategy) compressTools(tools []*contextpkg.ToolDescription) []*contextpkg.ToolDescription {
	// Sort by importance
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Importance > tools[j].Importance
	})

	maxTools := int(float64(len(tools)) * cs.config.CompressionRatio)
	if maxTools < 2 {
		maxTools = 2 // Keep at least 2 tools
	}

	if len(tools) <= maxTools {
		return tools
	}

	return tools[:maxTools]
}

// compressKnowledge compresses knowledge list
func (cs *CompressStrategy) compressKnowledge(knowledge []*contextpkg.KnowledgeItem) []*contextpkg.KnowledgeItem {
	// Sort by relevance
	sort.Slice(knowledge, func(i, j int) bool {
		return knowledge[i].Relevance > knowledge[j].Relevance
	})

	maxKnowledge := int(float64(len(knowledge)) * cs.config.CompressionRatio)
	if maxKnowledge < 3 {
		maxKnowledge = 3 // Keep at least 3 knowledge items
	}

	if len(knowledge) <= maxKnowledge {
		return knowledge
	}

	return knowledge[:maxKnowledge]
}

// Utility methods for token estimation
func (cs *CompressStrategy) estimateTokenCount(state *contextpkg.ContextState) int {
	tokens := 0
	tokens += cs.estimateTokensForMessages(state.Messages)
	tokens += cs.estimateTokensForScratchpad(state.Scratchpad)
	tokens += cs.estimateTokensForSelectedData(state.SelectedData)

	if state.CompressedCtx != nil {
		tokens += cs.estimateTokens(state.CompressedCtx.Summary)
		tokens += cs.estimateTokensForKeyPoints(state.CompressedCtx.KeyPoints)
	}

	return tokens
}

func (cs *CompressStrategy) estimateTokensForMessages(messages []contextpkg.Message) int {
	tokens := 0
	for _, msg := range messages {
		tokens += cs.estimateTokens(msg.Content)
	}
	return tokens
}

func (cs *CompressStrategy) estimateTokensForScratchpad(scratchpad map[string]interface{}) int {
	tokens := 0
	for _, value := range scratchpad {
		if str, ok := value.(string); ok {
			tokens += cs.estimateTokens(str)
		}
	}
	return tokens
}

func (cs *CompressStrategy) estimateTokensForSelectedData(selectedData map[string]interface{}) int {
	// Simplified estimation
	return len(selectedData) * 50 // Rough estimate
}

func (cs *CompressStrategy) estimateTokensForKeyPoints(keyPoints []string) int {
	tokens := 0
	for _, point := range keyPoints {
		tokens += cs.estimateTokens(point)
	}
	return tokens
}

func (cs *CompressStrategy) estimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 4 characters for English text
	return len(text) / 4
}

func (cs *CompressStrategy) messagesToText(messages []contextpkg.Message) string {
	var parts []string
	for _, msg := range messages {
		parts = append(parts, msg.Content)
	}
	return strings.Join(parts, " ")
}
