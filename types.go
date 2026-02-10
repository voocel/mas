package mas

import (
	"context"
	"encoding/json"
	"time"
)

// toolProgressKey is the context key for tool progress callbacks.
type toolProgressKey struct{}

// ToolProgressFunc is a callback for reporting tool execution progress.
// Tools call ReportToolProgress to emit partial results during long operations.
type ToolProgressFunc func(partialResult json.RawMessage)

// WithToolProgress injects a progress callback into the context.
func WithToolProgress(ctx context.Context, fn ToolProgressFunc) context.Context {
	return context.WithValue(ctx, toolProgressKey{}, fn)
}

// ReportToolProgress reports partial progress during tool execution.
// Silently ignored if no callback is registered in the context.
func ReportToolProgress(ctx context.Context, partial json.RawMessage) {
	if fn, ok := ctx.Value(toolProgressKey{}).(ToolProgressFunc); ok {
		fn(partial)
	}
}

// Role defines message roles.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// AgentMessage is the app-layer message abstraction.
// Message implements this interface. Users can define custom types
// (e.g. status notifications, UI hints) that flow through the context
// pipeline but get filtered out by ConvertToLLM.
type AgentMessage interface {
	GetRole() Role
	GetTimestamp() time.Time
}

// Message is an LLM-level message.
type Message struct {
	Role       Role           `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

func (m Message) GetRole() Role           { return m.Role }
func (m Message) GetTimestamp() time.Time { return m.Timestamp }

// HasToolCalls reports whether tool calls are present.
func (m Message) HasToolCalls() bool { return len(m.ToolCalls) > 0 }

// ToolCall represents a tool invocation request from the LLM.
type ToolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// ToolResult represents a tool execution outcome.
type ToolResult struct {
	ToolCallID string          `json:"tool_call_id"`
	Content    json.RawMessage `json:"content,omitempty"`
	IsError    bool            `json:"is_error,omitempty"`
}

// Tool defines the minimal tool interface.
// Timeout control goes through context.Context.
// Tools can report execution progress via ReportToolProgress(ctx, partial).
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

// ToolLabeler is an optional interface for tools to provide a human-readable label.
// Used by UI consumers for display purposes (e.g. "Read File" instead of "read").
type ToolLabeler interface {
	Label() string
}

// FuncTool wraps a function as a Tool (convenience helper).
type FuncTool struct {
	name        string
	description string
	schema      map[string]any
	fn          func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

func NewFuncTool(name, description string, schema map[string]any, fn func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)) *FuncTool {
	return &FuncTool{name: name, description: description, schema: schema, fn: fn}
}

func (t *FuncTool) Name() string           { return t.name }
func (t *FuncTool) Description() string    { return t.description }
func (t *FuncTool) Schema() map[string]any { return t.schema }
func (t *FuncTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	return t.fn(ctx, args)
}

// AgentContext holds the immutable context for a single agent loop invocation.
type AgentContext struct {
	SystemPrompt string
	Messages     []AgentMessage
	Tools        []Tool
}

// StreamFn is an injectable LLM call function.
// When nil, the loop uses model.Generate / model.GenerateStream directly.
// Use this to swap in a proxy, mock, or custom implementation.
type StreamFn func(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

// LLMRequest is the request passed to StreamFn.
type LLMRequest struct {
	Messages []Message
	Tools    []ToolSpec
}

// LLMResponse is the response from StreamFn.
type LLMResponse struct {
	Message Message
}

// ToolSpec describes a tool for the LLM (name + description + JSON schema).
type ToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// LoopConfig configures the agent loop.
// Replaces Runner.Config + Middleware + Observer + Tracer with function hooks.
type LoopConfig struct {
	Model    ChatModel
	StreamFn StreamFn // nil = use Model directly
	MaxTurns int      // safety limit, default 10

	// Two-stage pipeline: TransformContext → ConvertToLLM
	// TransformContext operates on AgentMessage[] (prune, inject external context).
	// ConvertToLLM filters to LLM-compatible Message[] at the call boundary.
	TransformContext func(ctx context.Context, msgs []AgentMessage) ([]AgentMessage, error)
	ConvertToLLM     func(msgs []AgentMessage) []Message

	// Steering: called after each tool execution to check for user interruptions.
	// If messages are returned, remaining tool calls are skipped.
	GetSteeringMessages func() []AgentMessage

	// FollowUp: called when the agent would otherwise stop.
	// If messages are returned, the agent continues with another turn.
	GetFollowUpMessages func() []AgentMessage
}

// ChatModel is the LLM provider interface.
type ChatModel interface {
	Generate(ctx context.Context, messages []Message, tools []ToolSpec) (*LLMResponse, error)
	GenerateStream(ctx context.Context, messages []Message, tools []ToolSpec) (<-chan StreamEvent, error)
	SupportsTools() bool
}

// StreamEvent is a streaming event from the LLM.
type StreamEvent struct {
	Type       StreamEventType
	Delta      string  // text delta for token events
	Message    Message // final message for done events
	StopReason string  // finish reason from LLM provider (for done events)
	Err        error   // error for error events
}

// StreamEventType identifies LLM streaming event types.
type StreamEventType string

const (
	StreamEventToken StreamEventType = "token"
	StreamEventDone  StreamEventType = "done"
	StreamEventError StreamEventType = "error"
)

// QueueMode controls how steering/follow-up queues are drained.
type QueueMode string

const (
	// QueueModeAll drains all queued messages at once (default).
	QueueModeAll QueueMode = "all"
	// QueueModeOneAtATime drains one message per turn, letting the agent respond to each individually.
	QueueModeOneAtATime QueueMode = "one-at-a-time"
)

// EventType identifies agent lifecycle event types.
type EventType string

const (
	EventAgentStart    EventType = "agent_start"
	EventAgentEnd      EventType = "agent_end"
	EventTurnStart     EventType = "turn_start"
	EventTurnEnd       EventType = "turn_end"
	EventMessageStart  EventType = "message_start"
	EventMessageUpdate EventType = "message_update"
	EventMessageEnd    EventType = "message_end"
	EventToolExecStart  EventType = "tool_exec_start"
	EventToolExecUpdate EventType = "tool_exec_update"
	EventToolExecEnd    EventType = "tool_exec_end"
	EventError          EventType = "error"
)

// Event is a lifecycle event emitted by the agent loop.
// This is the single output channel for all lifecycle information.
// Consumers (TUI, Slack bot, Web UI, logging) subscribe and filter by Type.
type Event struct {
	Type        EventType
	Message     AgentMessage // for message_start/update/end, turn_end, agent_end
	Delta       string       // text delta for message_update
	ToolID      string       // for tool_exec_*
	Tool        string       // tool name for tool_exec_*
	ToolLabel   string       // human-readable tool label for tool_exec_* (from ToolLabeler)
	Args        any          // tool args for tool_exec_start
	Result      any          // tool result for tool_exec_end/update
	IsError     bool         // tool error flag for tool_exec_end
	ToolResults []ToolResult // for turn_end: all tool results from this turn
	Err         error        // for error events
	Data        any          // generic payload (e.g. []AgentMessage for agent_end)
}
