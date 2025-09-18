package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// conversationMemory is the in-memory implementation of conversation memory.
type conversationMemory struct {
	messages []schema.Message
	config   *MemoryConfig
	mutex    sync.RWMutex
}

// NewConversationMemory creates a new conversation memory.
func NewConversationMemory(config *MemoryConfig) ConversationMemory {
	if config == nil {
		config = DefaultMemoryConfig
	}

	return &conversationMemory{
		messages: make([]schema.Message, 0),
		config:   config,
	}
}

// Add adds a message to the conversation history.
func (c *conversationMemory) Add(ctx context.Context, message schema.Message) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	c.messages = append(c.messages, message)

	// Limit the history length.
	if len(c.messages) > c.config.MaxHistory {
		// Keep recent messages, but preserve system messages.
		systemMessages := make([]schema.Message, 0)
		recentMessages := make([]schema.Message, 0)

		// Separate system messages from other messages.
		for _, msg := range c.messages {
			if msg.Role == schema.RoleSystem {
				systemMessages = append(systemMessages, msg)
			} else {
				recentMessages = append(recentMessages, msg)
			}
		}

		// Keep recent non-system messages.
		keepCount := c.config.MaxHistory - len(systemMessages)
		if keepCount > 0 && len(recentMessages) > keepCount {
			recentMessages = recentMessages[len(recentMessages)-keepCount:]
		}

		// Recombine the messages.
		c.messages = make([]schema.Message, 0, len(systemMessages)+len(recentMessages))
		c.messages = append(c.messages, systemMessages...)
		c.messages = append(c.messages, recentMessages...)
	}

	return nil
}

// AddConversationTurn adds a turn of conversation.
func (c *conversationMemory) AddConversationTurn(ctx context.Context, userMsg, assistantMsg schema.Message) error {
	if err := c.Add(ctx, userMsg); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	if err := c.Add(ctx, assistantMsg); err != nil {
		return fmt.Errorf("failed to add assistant message: %w", err)
	}

	return nil
}

// Query queries for relevant messages (simple implementation based on content matching). todo RAG
func (c *conversationMemory) Query(ctx context.Context, query string, limit int) ([]MemoryItem, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.config.EnableSearch {
		return c.getRecentItems(limit), nil
	}

	// Simple text matching search.
	items := make([]MemoryItem, 0)
	for i := len(c.messages) - 1; i >= 0 && len(items) < limit; i-- {
		msg := c.messages[i]
		if containsIgnoreCase(msg.Content, query) {
			items = append(items, MemoryItem{
				ID:        msg.ID,
				Content:   msg.Content,
				Metadata:  map[string]interface{}{"role": msg.Role, "timestamp": msg.Timestamp},
				Timestamp: msg.Timestamp,
				Score:     1.0, // Simple match score.
			})
		}
	}

	return items, nil
}

// GetHistory gets the full conversation history.
func (c *conversationMemory) GetHistory(ctx context.Context) ([]schema.Message, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Return a copy.
	history := make([]schema.Message, len(c.messages))
	copy(history, c.messages)
	return history, nil
}

// GetRecentHistory gets the N most recent history records.
func (c *conversationMemory) GetRecentHistory(ctx context.Context, limit int) ([]schema.Message, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if limit <= 0 || limit >= len(c.messages) {
		return c.GetHistory(ctx)
	}

	// Return recent messages.
	start := len(c.messages) - limit
	recent := make([]schema.Message, limit)
	copy(recent, c.messages[start:])
	return recent, nil
}

// GetConversationContext gets the conversation context for LLM calls.
func (c *conversationMemory) GetConversationContext(ctx context.Context) ([]schema.Message, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	context := make([]schema.Message, 0, len(c.messages))

	// First, add system messages.
	for _, msg := range c.messages {
		if msg.Role == schema.RoleSystem {
			context = append(context, msg)
		}
	}

	// Then, add the conversation history (in chronological order).
	for _, msg := range c.messages {
		if msg.Role != schema.RoleSystem {
			context = append(context, msg)
		}
	}

	return context, nil
}

// Summarize summarizes the conversation history using AI.
func (c *conversationMemory) Summarize(ctx context.Context, model ...llm.ChatModel) (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.messages) == 0 {
		return "", fmt.Errorf("no conversation history to summarize")
	}

	// Determine the model to use.
	var summaryModel llm.ChatModel
	if len(model) > 0 && model[0] != nil {
		// Use the provided model.
		summaryModel = model[0]
	} else {
		// Use the configured model.
		if c.config.SummaryModel == nil {
			return "", fmt.Errorf("no summary model provided, please pass a model or configure SummaryModel in MemoryConfig")
		}
		summaryModel = c.config.SummaryModel
	}

	return c.summarizeWithAI(ctx, summaryModel)
}

// summarizeWithAI is the internal AI summarization implementation.
func (c *conversationMemory) summarizeWithAI(ctx context.Context, model llm.ChatModel) (string, error) {
	// Build the conversation history text.
	conversationText := c.buildConversationText()

	// Build the summarization prompt.
	prompt := fmt.Sprintf(`Please summarize the main content and key information of the following conversation. Requirements:
1. Be concise and highlight the key points.
2. Retain important technical details and decisions.
3. Organize the content by topic.
4. Keep the length within 200 words.

Conversation content:
%s

Summary:`, conversationText)

	// Call the LLM for summarization.
	messages := []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: prompt,
		},
	}

	// Create a runtime context.
	runtimeCtx := runtime.NewContext(ctx, "memory-summary", "summary-"+fmt.Sprintf("%d", time.Now().UnixNano()))

	response, err := model.Generate(runtimeCtx, messages)
	if err != nil {
		return "", fmt.Errorf("AI summary failed: %w", err)
	}

	return response.Content, nil
}

// buildConversationText builds the conversation text.
func (c *conversationMemory) buildConversationText() string {
	var builder strings.Builder

	for _, msg := range c.messages {
		if msg.Role == schema.RoleSystem {
			continue // Skip system messages.
		}

		roleText := "User"
		if msg.Role == schema.RoleAssistant {
			roleText = "Assistant"
		}

		builder.WriteString(fmt.Sprintf("%s: %s\n", roleText, msg.Content))
	}

	return builder.String()
}

// Clear clears the conversation history.
func (c *conversationMemory) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.messages = make([]schema.Message, 0)
	return nil
}

// getRecentItems gets the most recent memory items.
func (c *conversationMemory) getRecentItems(limit int) []MemoryItem {
	items := make([]MemoryItem, 0)
	start := len(c.messages) - limit
	if start < 0 {
		start = 0
	}

	for i := start; i < len(c.messages); i++ {
		msg := c.messages[i]
		items = append(items, MemoryItem{
			ID:        msg.ID,
			Content:   msg.Content,
			Metadata:  map[string]interface{}{"role": msg.Role, "timestamp": msg.Timestamp},
			Timestamp: msg.Timestamp,
			Score:     1.0,
		})
	}

	return items
}

func containsIgnoreCase(text, substr string) bool {
	// Simple case-insensitive matching.
	// More complex search algorithms can be used in a real implementation.
	return len(text) >= len(substr) &&
		len(substr) > 0 &&
		findSubstring(strings.ToLower(text), strings.ToLower(substr))
}

func findSubstring(text, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(text) < len(substr) {
		return false
	}

	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
