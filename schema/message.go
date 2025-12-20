package schema

import (
	"encoding/json"
	"time"
)

// Role defines message roles.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Message represents a chat message.
type Message struct {
	ID        string                 `json:"id"`
	Role      Role                   `json:"role"`
	Content   string                 `json:"content"`
	ToolCalls []ToolCall            `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ToolCall represents a tool invocation request.
type ToolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// ToolResult represents a tool execution result.
type ToolResult struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Reset clears the message fields.
func (m *Message) Reset() {
	m.ID = ""
	m.Role = ""
	m.Content = ""
	m.ToolCalls = m.ToolCalls[:0]
	m.Metadata = nil
	m.Timestamp = time.Time{}
}

// Clone deep-copies the message.
func (m *Message) Clone() *Message {
	clone := &Message{
		ID:        m.ID,
		Role:      m.Role,
		Content:   m.Content,
		Timestamp: m.Timestamp,
	}
	
	// Deep-copy tool calls.
	if len(m.ToolCalls) > 0 {
		clone.ToolCalls = make([]ToolCall, len(m.ToolCalls))
		copy(clone.ToolCalls, m.ToolCalls)
	}
	
	// Deep-copy metadata.
	if m.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range m.Metadata {
			clone.Metadata[k] = v
		}
	}
	
	return clone
}

// HasToolCalls reports whether tool calls are present.
func (m *Message) HasToolCalls() bool {
	return len(m.ToolCalls) > 0
}

// AddToolCall appends a tool call.
func (m *Message) AddToolCall(toolCall ToolCall) {
	m.ToolCalls = append(m.ToolCalls, toolCall)
}

// SetMetadata sets metadata.
func (m *Message) SetMetadata(key string, value interface{}) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata[key] = value
}

// GetMetadata retrieves metadata.
func (m *Message) GetMetadata(key string) (interface{}, bool) {
	if m.Metadata == nil {
		return nil, false
	}
	value, exists := m.Metadata[key]
	return value, exists
}
