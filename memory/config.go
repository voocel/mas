package memory

import (
	"context"
	"fmt"
	"time"
)

// ConfigOption allows configuring memory behavior
type ConfigOption func(*MemoryConfig)

// WithMaxMessages sets the maximum number of messages to keep
func WithMaxMessages(max int) ConfigOption {
	return func(c *MemoryConfig) {
		c.MaxMessages = max
	}
}

// WithTTL sets the time-to-live for messages
func WithTTL(ttl time.Duration) ConfigOption {
	return func(c *MemoryConfig) {
		c.TTL = ttl
	}
}

// WithMetadata sets metadata for the memory instance
func WithMetadata(metadata map[string]interface{}) ConfigOption {
	return func(c *MemoryConfig) {
		c.Metadata = metadata
	}
}

// ApplyOptions applies a list of options to the config
func ApplyOptions(config *MemoryConfig, options ...ConfigOption) {
	for _, option := range options {
		option(config)
	}
}

// ValidateConfig validates the memory configuration
func ValidateConfig(config MemoryConfig) error {
	if config.MaxMessages < 1 {
		return fmt.Errorf("max messages must be greater than 0")
	}
	
	if config.TTL < 0 {
		return fmt.Errorf("TTL cannot be negative")
	}
	
	return nil
}

// Specialized memory configurations

// NewChatMemoryConfig creates a configuration optimized for chat
func NewChatMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxMessages: 50,   // Keep recent conversation context
		TTL:         0,    // No expiration for chat
		Metadata:    make(map[string]interface{}),
	}
}

// NewShortTermMemoryConfig creates a configuration for short-term tasks
func NewShortTermMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxMessages: 20,             // Limited context
		TTL:         30 * time.Minute, // Expire after 30 minutes
		Metadata:    make(map[string]interface{}),
	}
}

// NewLongTermMemoryConfig creates a configuration for long-term conversations
func NewLongTermMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxMessages: 200,  // Extended context
		TTL:         0,    // No expiration
		Metadata:    make(map[string]interface{}),
	}
}

// Helper functions for creating specialized memory instances

// NewChatMemory creates a conversation memory optimized for chat
func NewChatMemory() Memory {
	return NewConversationWithConfig(NewChatMemoryConfig())
}

// NewShortTermMemory creates a memory with short TTL
func NewShortTermMemory() Memory {
	return NewConversationWithConfig(NewShortTermMemoryConfig())
}

// NewLongTermMemory creates a memory with extended capacity
func NewLongTermMemory() Memory {
	return NewConversationWithConfig(NewLongTermMemoryConfig())
}

// Memory utility functions

// CopyMemory creates a deep copy of memory messages
func CopyMemory(source Memory, target Memory, ctx context.Context) error {
	if source == nil || target == nil {
		return fmt.Errorf("source and target memories cannot be nil")
	}
	
	// Get all messages from source
	messages, err := source.GetHistory(ctx, -1) // Get all messages
	if err != nil {
		return fmt.Errorf("failed to get source memory history: %w", err)
	}
	
	// Clear target memory
	if err := target.Clear(); err != nil {
		return fmt.Errorf("failed to clear target memory: %w", err)
	}
	
	// Add all messages to target
	for _, msg := range messages {
		if err := target.Add(ctx, msg.Role, msg.Content); err != nil {
			return fmt.Errorf("failed to add message to target memory: %w", err)
		}
	}
	
	return nil
}

// MergeMemories combines messages from multiple memories
func MergeMemories(ctx context.Context, target Memory, sources ...Memory) error {
	if target == nil {
		return fmt.Errorf("target memory cannot be nil")
	}
	
	for i, source := range sources {
		if source == nil {
			return fmt.Errorf("source memory %d cannot be nil", i)
		}
		
		messages, err := source.GetHistory(ctx, -1)
		if err != nil {
			return fmt.Errorf("failed to get history from source %d: %w", i, err)
		}
		
		for _, msg := range messages {
			if err := target.Add(ctx, msg.Role, msg.Content); err != nil {
				return fmt.Errorf("failed to add message from source %d: %w", i, err)
			}
		}
	}
	
	return nil
}