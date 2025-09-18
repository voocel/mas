package schema

import "time"

// EventType enumerates streaming event types
type EventType string

const (
	EventStart       EventType = "start"
	EventEnd         EventType = "end"
	EventError       EventType = "error"
	EventToken       EventType = "token"
	EventToolCall    EventType = "tool_call"
	EventToolResult  EventType = "tool_result"
	EventStateChange EventType = "state_change"
	EventAgentSwitch EventType = "agent_switch"
	EventStepStart   EventType = "step_start"
	EventStepEnd     EventType = "step_end"
	EventStepSkipped EventType = "step_skipped"
)

// StreamEvent represents an event emitted during streaming execution
type StreamEvent struct {
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	AgentID   string      `json:"agent_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Error     error       `json:"error,omitempty"`
}

// TokenEvent captures token-level streaming data
type TokenEvent struct {
	Token string `json:"token"`
	Delta string `json:"delta,omitempty"`
}

// ToolCallEvent captures a tool invocation event
type ToolCallEvent struct {
	ToolCall ToolCall `json:"tool_call"`
}

// ToolResultEvent captures the result of a tool invocation
type ToolResultEvent struct {
	ToolResult ToolResult `json:"tool_result"`
}

// StateChangeEvent records a state transition
type StateChangeEvent struct {
	Key      string      `json:"key"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value"`
}

// AgentSwitchEvent records an agent switch event
type AgentSwitchEvent struct {
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Reason    string `json:"reason,omitempty"`
}

// NewStreamEvent constructs a streaming event
func NewStreamEvent(eventType EventType, data interface{}) StreamEvent {
	return StreamEvent{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorEvent constructs an error event
func NewErrorEvent(err error, agentID string) StreamEvent {
	return StreamEvent{
		Type:      EventError,
		AgentID:   agentID,
		Error:     err,
		Timestamp: time.Now(),
	}
}

// NewTokenEvent constructs a token event
func NewTokenEvent(token, delta string, agentID string) StreamEvent {
	return StreamEvent{
		Type:    EventToken,
		AgentID: agentID,
		Data: TokenEvent{
			Token: token,
			Delta: delta,
		},
		Timestamp: time.Now(),
	}
}

// NewToolCallEvent constructs a tool call event
func NewToolCallEvent(toolCall ToolCall, agentID string) StreamEvent {
	return StreamEvent{
		Type:    EventToolCall,
		AgentID: agentID,
		Data: ToolCallEvent{
			ToolCall: toolCall,
		},
		Timestamp: time.Now(),
	}
}

// NewToolResultEvent constructs a tool result event
func NewToolResultEvent(toolResult ToolResult, agentID string) StreamEvent {
	return StreamEvent{
		Type:    EventToolResult,
		AgentID: agentID,
		Data: ToolResultEvent{
			ToolResult: toolResult,
		},
		Timestamp: time.Now(),
	}
}
