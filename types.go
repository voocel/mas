package agentcore

import (
	"context"
	"encoding/json"
	"strings"
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

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

// Role defines message roles.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// ---------------------------------------------------------------------------
// Content Blocks
// ---------------------------------------------------------------------------

// ContentType identifies the kind of content in a ContentBlock.
type ContentType string

const (
	ContentText     ContentType = "text"
	ContentThinking ContentType = "thinking"
	ContentToolCall ContentType = "toolCall"
	ContentImage    ContentType = "image"
)

// ContentBlock is a tagged union for message content.
// Exactly one payload field is populated, matching the Type value.
type ContentBlock struct {
	Type     ContentType `json:"type"`
	Text     string      `json:"text,omitempty"`
	Thinking string      `json:"thinking,omitempty"`
	ToolCall *ToolCall   `json:"tool_call,omitempty"`
	Image    *ImageData  `json:"image,omitempty"`
}

// ImageData holds base64-encoded image content.
type ImageData struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
}

// Block constructors

func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: ContentText, Text: text}
}

func ThinkingBlock(thinking string) ContentBlock {
	return ContentBlock{Type: ContentThinking, Thinking: thinking}
}

func ToolCallBlock(tc ToolCall) ContentBlock {
	return ContentBlock{Type: ContentToolCall, ToolCall: &tc}
}

func ImageBlock(data, mimeType string) ContentBlock {
	return ContentBlock{Type: ContentImage, Image: &ImageData{Data: data, MimeType: mimeType}}
}

// ---------------------------------------------------------------------------
// Stop Reason
// ---------------------------------------------------------------------------

// StopReason indicates why the LLM stopped generating.
type StopReason string

const (
	StopReasonStop    StopReason = "stop"
	StopReasonLength  StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonError   StopReason = "error"
	StopReasonAborted StopReason = "aborted"
)

// ---------------------------------------------------------------------------
// Usage
// ---------------------------------------------------------------------------

// Usage tracks token consumption for a single LLM call.
//
// Field semantics:
//   - Input: prompt tokens sent to the model (includes cached tokens for some providers)
//   - Output: completion tokens generated (includes reasoning tokens if applicable)
//   - CacheRead: tokens served from prompt cache (Anthropic: cache_read_input_tokens)
//   - CacheWrite: tokens written to prompt cache (Anthropic: cache_creation_input_tokens)
//   - TotalTokens: provider-reported total, typically Input + Output
type Usage struct {
	Input       int `json:"input"`
	Output      int `json:"output"`
	CacheRead   int `json:"cache_read"`
	CacheWrite  int `json:"cache_write"`
	TotalTokens int `json:"total_tokens"`
}

// Add accumulates another Usage into this one (nil-safe).
func (u *Usage) Add(other *Usage) {
	if other == nil {
		return
	}
	u.Input += other.Input
	u.Output += other.Output
	u.CacheRead += other.CacheRead
	u.CacheWrite += other.CacheWrite
	u.TotalTokens += other.TotalTokens
}

// ---------------------------------------------------------------------------
// Thinking Level
// ---------------------------------------------------------------------------

// ThinkingLevel configures the reasoning depth for models that support it.
type ThinkingLevel string

const (
	ThinkingOff     ThinkingLevel = "off"
	ThinkingMinimal ThinkingLevel = "minimal"
	ThinkingLow     ThinkingLevel = "low"
	ThinkingMedium  ThinkingLevel = "medium"
	ThinkingHigh    ThinkingLevel = "high"
	ThinkingXHigh   ThinkingLevel = "xhigh"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// AgentMessage is the app-layer message abstraction.
// Message implements this interface. Users can define custom types
// (e.g. status notifications, UI hints) that flow through the context
// pipeline but get filtered out by ConvertToLLM.
type AgentMessage interface {
	GetRole() Role
	GetTimestamp() time.Time
	TextContent() string
	ThinkingContent() string
	HasToolCalls() bool
}

// Message is an LLM-level message with structured content blocks.
type Message struct {
	Role       Role           `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason StopReason     `json:"stop_reason,omitempty"`
	Usage      *Usage         `json:"usage,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

func (m Message) GetRole() Role           { return m.Role }
func (m Message) GetTimestamp() time.Time  { return m.Timestamp }

// TextContent returns the concatenated text from all text blocks.
func (m Message) TextContent() string {
	var sb strings.Builder
	for _, b := range m.Content {
		if b.Type == ContentText {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// ThinkingContent returns the concatenated thinking text.
func (m Message) ThinkingContent() string {
	var sb strings.Builder
	for _, b := range m.Content {
		if b.Type == ContentThinking {
			sb.WriteString(b.Thinking)
		}
	}
	return sb.String()
}

// ToolCalls returns all tool call blocks.
func (m Message) ToolCalls() []ToolCall {
	var calls []ToolCall
	for _, b := range m.Content {
		if b.Type == ContentToolCall && b.ToolCall != nil {
			calls = append(calls, *b.ToolCall)
		}
	}
	return calls
}

// HasToolCalls reports whether any tool call blocks exist.
func (m Message) HasToolCalls() bool {
	for _, b := range m.Content {
		if b.Type == ContentToolCall {
			return true
		}
	}
	return false
}

// IsEmpty reports whether the message has no meaningful content.
func (m Message) IsEmpty() bool {
	return len(m.Content) == 0
}

// ---------------------------------------------------------------------------
// Message Serialization Helpers
// ---------------------------------------------------------------------------

// CollectMessages extracts concrete Messages from an AgentMessage slice,
// dropping custom types. Use this to serialize conversation history.
func CollectMessages(msgs []AgentMessage) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if msg, ok := m.(Message); ok {
			out = append(out, msg)
		}
	}
	return out
}

// ToAgentMessages converts a Message slice to AgentMessage slice.
// Use this to restore conversation history from deserialized Messages.
func ToAgentMessages(msgs []Message) []AgentMessage {
	out := make([]AgentMessage, len(msgs))
	for i, m := range msgs {
		out[i] = m
	}
	return out
}

// ---------------------------------------------------------------------------
// Message Constructors
// ---------------------------------------------------------------------------

// UserMsg creates a user message from plain text.
func UserMsg(text string) Message {
	return Message{
		Role:      RoleUser,
		Content:   []ContentBlock{TextBlock(text)},
		Timestamp: time.Now(),
	}
}

// SystemMsg creates a system message.
func SystemMsg(text string) Message {
	return Message{
		Role:      RoleSystem,
		Content:   []ContentBlock{TextBlock(text)},
		Timestamp: time.Now(),
	}
}

// ToolResultMsg creates a tool result message.
func ToolResultMsg(toolCallID string, content json.RawMessage, isError bool) Message {
	return Message{
		Role:    RoleTool,
		Content: []ContentBlock{TextBlock(string(content))},
		Metadata: map[string]any{
			"tool_call_id": toolCallID,
			"is_error":     isError,
		},
		Timestamp: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Tool Calls & Results
// ---------------------------------------------------------------------------

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
	Details    any             `json:"details,omitempty"` // optional metadata for UI display/logging
}

// ---------------------------------------------------------------------------
// Tool Interface
// ---------------------------------------------------------------------------

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
type ToolLabeler interface {
	Label() string
}

// PermissionFunc is called before each tool execution.
// Return nil to allow execution, or a non-nil error to deny.
// The error message is sent back to the LLM as a tool error result.
// Receives context.Context to support I/O (e.g. TUI confirmation, remote policy).
type PermissionFunc func(ctx context.Context, call ToolCall) error

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

// ---------------------------------------------------------------------------
// Agent Context & Loop Config
// ---------------------------------------------------------------------------

// AgentContext holds the immutable context for a single agent loop invocation.
type AgentContext struct {
	SystemPrompt string
	Messages     []AgentMessage
	Tools        []Tool
}

// StreamFn is an injectable LLM call function.
// When nil, the loop uses model.Generate / model.GenerateStream directly.
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
type LoopConfig struct {
	Model         ChatModel
	StreamFn      StreamFn      // nil = use Model directly
	MaxTurns      int           // safety limit, default 10
	MaxRetries    int           // LLM call retry limit for retryable errors, default 3
	MaxToolErrors int           // consecutive tool failure threshold per tool, 0 = unlimited
	ThinkingLevel ThinkingLevel // reasoning depth

	// Two-stage pipeline: TransformContext → ConvertToLLM
	TransformContext func(ctx context.Context, msgs []AgentMessage) ([]AgentMessage, error)
	ConvertToLLM     func(msgs []AgentMessage) []Message

	// CheckPermission is called before each tool execution.
	// Return nil to allow, or error to deny (error becomes tool error result).
	// When nil, all tools are allowed.
	CheckPermission PermissionFunc

	// GetApiKey resolves the API key before each LLM call.
	// The provider parameter identifies which provider is being called (e.g. "openai", "anthropic").
	// Enables per-provider key resolution, key rotation, OAuth tokens, and multi-tenant scenarios.
	// When nil or returns empty string, the model's default key is used.
	GetApiKey func(provider string) (string, error)

	// ThinkingBudgets maps each ThinkingLevel to a max thinking token count.
	// When set, the resolved budget is passed to the model alongside the level.
	ThinkingBudgets map[ThinkingLevel]int

	// SessionID enables provider-level session caching (e.g. Anthropic prompt cache).
	SessionID string

	// Steering: called after each tool execution to check for user interruptions.
	GetSteeringMessages func() []AgentMessage

	// FollowUp: called when the agent would otherwise stop.
	GetFollowUpMessages func() []AgentMessage
}

// ---------------------------------------------------------------------------
// Context Usage Estimation
// ---------------------------------------------------------------------------

// ContextEstimateFn estimates the current context token consumption from messages.
// Returns total tokens, tokens from LLM Usage, and estimated trailing tokens.
type ContextEstimateFn func(msgs []AgentMessage) (tokens, usageTokens, trailingTokens int)

// ContextUsage represents the current context window occupancy estimate.
type ContextUsage struct {
	Tokens         int     `json:"tokens"`          // estimated total tokens in context
	ContextWindow  int     `json:"context_window"`  // model's context window size
	Percent        float64 `json:"percent"`         // tokens / contextWindow * 100
	UsageTokens    int     `json:"usage_tokens"`    // from last LLM-reported Usage
	TrailingTokens int     `json:"trailing_tokens"` // chars/4 estimate for trailing messages
}

// ---------------------------------------------------------------------------
// Call Options
// ---------------------------------------------------------------------------

// CallOption configures per-call LLM parameters.
type CallOption func(*CallConfig)

// CallConfig holds per-call configuration resolved from CallOptions.
type CallConfig struct {
	ThinkingLevel  ThinkingLevel
	ThinkingBudget int    // max thinking tokens, 0 = use provider default
	APIKey         string // per-call API key override, empty = use model default
	SessionID      string // provider session caching identifier
}

// ResolveCallConfig applies options and returns the resolved config.
func ResolveCallConfig(opts []CallOption) CallConfig {
	var cfg CallConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithThinking sets the thinking level for a single LLM call.
func WithThinking(level ThinkingLevel) CallOption {
	return func(c *CallConfig) { c.ThinkingLevel = level }
}

// WithThinkingBudget sets the max thinking tokens for a single LLM call.
func WithThinkingBudget(tokens int) CallOption {
	return func(c *CallConfig) { c.ThinkingBudget = tokens }
}

// WithAPIKey overrides the API key for a single LLM call.
// Enables key rotation, OAuth short-lived tokens, and multi-tenant scenarios.
func WithAPIKey(key string) CallOption {
	return func(c *CallConfig) { c.APIKey = key }
}

// WithCallSessionID sets a session identifier for a single LLM call.
func WithCallSessionID(id string) CallOption {
	return func(c *CallConfig) { c.SessionID = id }
}

// ---------------------------------------------------------------------------
// ChatModel Interface
// ---------------------------------------------------------------------------

// ChatModel is the LLM provider interface.
type ChatModel interface {
	Generate(ctx context.Context, messages []Message, tools []ToolSpec, opts ...CallOption) (*LLMResponse, error)
	GenerateStream(ctx context.Context, messages []Message, tools []ToolSpec, opts ...CallOption) (<-chan StreamEvent, error)
	SupportsTools() bool
}

// ProviderNamer is an optional interface for ChatModel implementations
// to expose their provider name (e.g. "openai", "anthropic", "gemini").
// Used by the agent loop to pass provider context to GetApiKey callbacks.
type ProviderNamer interface {
	ProviderName() string
}

// ---------------------------------------------------------------------------
// Stream Events (fine-grained)
// ---------------------------------------------------------------------------

// StreamEventType identifies LLM streaming event types.
type StreamEventType string

const (
	// Text content streaming
	StreamEventTextStart StreamEventType = "text_start"
	StreamEventTextDelta StreamEventType = "text_delta"
	StreamEventTextEnd   StreamEventType = "text_end"

	// Thinking/reasoning streaming
	StreamEventThinkingStart StreamEventType = "thinking_start"
	StreamEventThinkingDelta StreamEventType = "thinking_delta"
	StreamEventThinkingEnd   StreamEventType = "thinking_end"

	// Tool call streaming
	StreamEventToolCallStart StreamEventType = "toolcall_start"
	StreamEventToolCallDelta StreamEventType = "toolcall_delta"
	StreamEventToolCallEnd   StreamEventType = "toolcall_end"

	// Terminal events
	StreamEventDone  StreamEventType = "done"
	StreamEventError StreamEventType = "error"
)

// StreamEvent is a streaming event from the LLM.
type StreamEvent struct {
	Type         StreamEventType
	ContentIndex int        // which content block is being updated
	Delta        string     // text/thinking/toolcall argument delta
	Message      Message    // partial (during streaming) or final (done)
	StopReason   StopReason // finish reason (for done events)
	Err          error      // for error events
}

// ---------------------------------------------------------------------------
// Queue Mode
// ---------------------------------------------------------------------------

// QueueMode controls how steering/follow-up queues are drained.
type QueueMode string

const (
	QueueModeAll        QueueMode = "all"
	QueueModeOneAtATime QueueMode = "one-at-a-time"
)

// ---------------------------------------------------------------------------
// Agent Events
// ---------------------------------------------------------------------------

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
	EventRetry          EventType = "retry"
	EventError          EventType = "error"
)

// Event is a lifecycle event emitted by the agent loop.
// This is the single output channel for all lifecycle information.
type Event struct {
	Type        EventType
	Message     AgentMessage    // for message_start/update/end, turn_end
	Delta       string          // text delta for message_update
	ToolID      string          // for tool_exec_*
	Tool        string          // tool name for tool_exec_*
	ToolLabel   string          // human-readable tool label (from ToolLabeler)
	Args        json.RawMessage // tool args for tool_exec_start
	Result      json.RawMessage // tool result for tool_exec_end/update
	IsError     bool            // tool error flag for tool_exec_end
	ToolResults []ToolResult    // for turn_end: all tool results from this turn
	Err         error           // for error events
	NewMessages []AgentMessage  // for agent_end: messages added during this loop
	RetryInfo   *RetryInfo      // for retry events
}

// RetryInfo carries retry context for EventRetry events.
type RetryInfo struct {
	Attempt    int
	MaxRetries int
	Delay      time.Duration
	Err        error
}
