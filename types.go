package mas

import (
	"context"
	"time"
)

// Message represents a single message in the system
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Role constants for messages
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

// WorkflowContext represents the execution context for a workflow
type WorkflowContext struct {
	ID       string         `json:"id"`
	Data     map[string]any `json:"data"`
	Messages []Message      `json:"messages"`
}

// ToolSchema defines the parameter schema for a tool
type ToolSchema struct {
	Type        string                     `json:"type"`
	Properties  map[string]*PropertySchema `json:"properties"`
	Required    []string                   `json:"required"`
	Description string                     `json:"description,omitempty"`
}

// PropertySchema defines a property in the tool schema
type PropertySchema struct {
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Enum        []string        `json:"enum,omitempty"`
	Items       *PropertySchema `json:"items,omitempty"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// CheckpointType indicates the reason for creating a checkpoint
type CheckpointType string

const (
	CheckpointTypeAuto       CheckpointType = "auto"
	CheckpointTypeManual     CheckpointType = "manual"
	CheckpointTypeBeforeNode CheckpointType = "before_node"
	CheckpointTypeAfterNode  CheckpointType = "after_node"
)

// CheckpointInfo provides summary information about a checkpoint
type CheckpointInfo struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Timestamp   time.Time              `json:"timestamp"`
	CurrentNode string                 `json:"current_node"`
	Type        CheckpointType         `json:"type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Size        int64                  `json:"size,omitempty"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages    []ChatMessage    `json:"messages"`
	Model       string           `json:"model,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  string           `json:"tool_choice,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToolDefinition represents a tool definition for function calling
type ToolDefinition struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef represents a function definition
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Agent represents an intelligent agent
type Agent interface {
	Chat(ctx context.Context, message string) (string, error)
	ChatStream(ctx context.Context, message string) (<-chan string, error)
	WithTools(tools ...Tool) Agent
	WithMemory(memory Memory) Agent
	WithSystemPrompt(prompt string) Agent
	WithTemperature(temp float64) Agent
	WithMaxTokens(tokens int) Agent
	WithEventBus(eventBus EventBus) Agent
	SetState(key string, value interface{})
	GetState(key string) interface{}
	ClearState()
	Name() string
	Model() string

	// Event-related Methods
	GetEventBus() EventBus
	StreamEvents(ctx context.Context, eventTypes ...EventType) (<-chan Event, error)
	PublishEvent(ctx context.Context, eventType EventType, data map[string]interface{}) error

	// Cognitive Ability Approach (Optional)
	Plan(ctx context.Context, goal string) (*Plan, error)
	Reason(ctx context.Context, situation *Situation) (*Decision, error)
	ExecuteSkill(ctx context.Context, skillName string, params map[string]interface{}) (interface{}, error)
	React(ctx context.Context, stimulus *Stimulus) (*Action, error)
	GetSkillLibrary() SkillLibrary
	WithSkills(skills ...Skill) Agent
	GetCognitiveState() *CognitiveState
	SetCognitiveMode(mode CognitiveMode) Agent

	// Autonomous capability methods (optional)
	WithGoalManager(manager GoalManager) Agent
	GetGoalManager() GoalManager
	AddGoal(ctx context.Context, goal *Goal) error
	StartAutonomous(ctx context.Context, strategy AutonomousStrategy) error
	StopAutonomous(ctx context.Context) error
	IsAutonomous() bool

	// Learning capability methods (optional)
	WithLearningEngine(engine LearningEngine) Agent
	GetLearningEngine() LearningEngine
	RecordExperience(ctx context.Context, experience *Experience) error
	SelfReflect(ctx context.Context) (*SelfReflection, error)
	GetLearningMetrics() *LearningMetrics
}

// Memory represents the memory system for agents
type Memory interface {
	Add(ctx context.Context, role, content string) error
	GetHistory(ctx context.Context, limit int) ([]Message, error)
	Clear() error
	Count() int
}

// WorkflowNode represents a node in the workflow
type WorkflowNode interface {
	ID() string
	Execute(ctx context.Context, wfCtx *WorkflowContext) error
}

// WorkflowBuilder provides a fluent API for building workflows
type WorkflowBuilder interface {
	AddNode(node WorkflowNode) WorkflowBuilder
	AddEdge(from, to string) WorkflowBuilder
	SetStart(nodeID string) WorkflowBuilder
	AddConditionalRoute(fromNodeID string, condition func(*WorkflowContext) bool, trueTarget, falseTarget string) WorkflowBuilder
	WithCheckpointer(checkpointer Checkpointer) WorkflowBuilder
	WithEventBus(eventBus EventBus) WorkflowBuilder
	Execute(ctx context.Context, initialData map[string]any) (*WorkflowContext, error)
	ExecuteWithCheckpoint(ctx context.Context, initialData map[string]any) (*WorkflowContext, error)
	ResumeFromCheckpoint(ctx context.Context, workflowID string) (*WorkflowContext, error)

	GetEventBus() EventBus
	StreamEvents(ctx context.Context, eventTypes ...EventType) (<-chan Event, error)
	ExecuteWithEvents(ctx context.Context, initialData map[string]any) (*WorkflowContext, <-chan Event, error)
}

// Provider represents an LLM provider
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	Model() string
	Close() error
}

// AgentConfig contains configuration for creating an agent
type AgentConfig struct {
	Name         string
	Model        string
	APIKey       string
	BaseURL      string
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
	Tools        []Tool
	Memory       Memory
	State        map[string]interface{}
	Provider     Provider
}

// MemoryConfig contains configuration for memory systems
type MemoryConfig struct {
	MaxMessages int                    `json:"max_messages"`
	TTL         time.Duration          `json:"ttl,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Condition represents a single condition with its target
type Condition struct {
	Check  func(*WorkflowContext) bool
	Target string
}

// HumanInputProvider handles human input collection
type HumanInputProvider interface {
	RequestInput(ctx context.Context, prompt string, options ...HumanInputOption) (*HumanInput, error)
}

// HumanInput represents input from a human user
type HumanInput struct {
	Value string
	Data  map[string]any
}

// HumanInputOption configures human input behavior
type HumanInputOption func(*HumanInputConfig)

// HumanInputConfig contains configuration for human input
type HumanInputConfig struct {
	Timeout   time.Duration
	Validator func(string) error
	Required  bool
}
