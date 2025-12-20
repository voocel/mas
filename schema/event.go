package schema

import "time"

// EventType defines stream event types.
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

// StreamEvent represents a stream event.
type StreamEvent struct {
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	AgentID   string      `json:"agent_id,omitempty"`
	RunID     RunID       `json:"run_id,omitempty"`
	StepID    StepID      `json:"step_id,omitempty"`
	SpanID    SpanID      `json:"span_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Error     error       `json:"error,omitempty"`
}

// TokenEvent represents a token-level event.
type TokenEvent struct {
	Token string `json:"token"`
	Delta string `json:"delta,omitempty"`
}

// ToolCallEvent represents a tool call event.
type ToolCallEvent struct {
	ToolCall ToolCall `json:"tool_call"`
}

// ToolResultEvent represents a tool result event.
type ToolResultEvent struct {
	ToolResult ToolResult `json:"tool_result"`
}

// StateChangeEvent represents a state change event.
type StateChangeEvent struct {
	Key      string      `json:"key"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value"`
}

// AgentSwitchEvent represents an agent switch event.
type AgentSwitchEvent struct {
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Reason    string `json:"reason,omitempty"`
}

// NewStreamEvent creates a stream event.
func NewStreamEvent(eventType EventType, data interface{}) StreamEvent {
	return StreamEvent{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorEvent creates an error event.
func NewErrorEvent(err error, agentID string) StreamEvent {
	return StreamEvent{
		Type:      EventError,
		AgentID:   agentID,
		Error:     err,
		Timestamp: time.Now(),
	}
}

// NewTokenEvent creates a token event.
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

// NewToolCallEvent creates a tool call event.
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

// NewToolResultEvent creates a tool result event.
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

func (e StreamEvent) WithIDs(runID RunID, stepID StepID, spanID SpanID) StreamEvent {
	e.RunID = runID
	e.StepID = stepID
	e.SpanID = spanID
	return e
}
