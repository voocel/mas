package agent

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// Agent defines the interface for an agent.
type Agent interface {
	// ID unique identifier of the agent.
	ID() string

	// Name of the agent.
	Name() string

	// Execute performs a single turn of conversation.
	Execute(ctx runtime.Context, input schema.Message) (schema.Message, error)

	// ExecuteStream performs a streaming conversation.
	ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error)

	// ExecuteWithHandoff executes a conversation with handoff support.
	ExecuteWithHandoff(ctx runtime.Context, input schema.Message) (schema.Message, *schema.Handoff, error)

	// Tools list of tools available to the agent.
	Tools() []tools.Tool

	// Capabilities list of capabilities of the agent.
	Capabilities() []Capability

	// GetCapabilities the full capability declaration of the agent.
	GetCapabilities() *AgentCapabilities

	// GetModel the model used by the agent.
	GetModel() llm.ChatModel

	// GetSystemPrompt the system prompt of the agent.
	GetSystemPrompt() string

	// CanHandoff checks if the agent supports handoff.
	CanHandoff() bool
}

// Capability defines an agent's capability.
type Capability string

const (
	CapabilityToolUse     Capability = "tool_use"
	CapabilityMemory      Capability = "memory"
	CapabilityStreaming   Capability = "streaming"
	CapabilityMultimodal  Capability = "multimodal"
	CapabilityReasoning   Capability = "reasoning"
	CapabilityPlanning    Capability = "planning"
	CapabilityHandoff     Capability = "handoff"
	CapabilityAnalysis    Capability = "analysis"
	CapabilityWriting     Capability = "writing"
	CapabilityResearch    Capability = "research"
	CapabilityEngineering Capability = "engineering"
	CapabilityDesign      Capability = "design"
	CapabilityMarketing   Capability = "marketing"
	CapabilityFinance     Capability = "finance"
	CapabilityLegal       Capability = "legal"
	CapabilitySupport     Capability = "support"
	CapabilityManagement  Capability = "management"
	CapabilityEducation   Capability = "education"
)

// AgentCapabilities declares the capabilities of an agent.
type AgentCapabilities struct {
	// Core capabilities
	CoreCapabilities []Capability `json:"core_capabilities"`

	// Areas of expertise
	Expertise []string `json:"expertise"`

	// Supported tool types
	ToolTypes []string `json:"tool_types"`

	// Supported languages
	Languages []string `json:"languages"`

	// Processing complexity (1-10, 10 is the highest)
	ComplexityLevel int `json:"complexity_level"`

	// Concurrent processing capability
	ConcurrencyLevel int `json:"concurrency_level"`

	// Custom capability tags
	CustomTags []string `json:"custom_tags"`

	// Capability description
	Description string `json:"description"`
}

type StateInputFunc func(interface{}) string
type StateOutputFunc func(interface{}, string)

// AgentConfig is the configuration for an agent.
type AgentConfig struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	SystemPrompt string             `json:"system_prompt"`
	Model        llm.ChatModel      `json:"-"`
	Tools        []tools.Tool       `json:"-"`
	Capabilities *AgentCapabilities `json:"capabilities"`
	MaxHistory   int                `json:"max_history"`
	Temperature  float64            `json:"temperature"`
	MaxTokens    int                `json:"max_tokens"`

	StateKey       string          `json:"state_key"`
	InputFromState StateInputFunc  `json:"-"`
	OutputToState  StateOutputFunc `json:"-"`
}

// DefaultAgentConfig is the default agent configuration.
var DefaultAgentConfig = &AgentConfig{
	MaxHistory:  10,
	Temperature: 0.7,
	MaxTokens:   1000,
}

// BaseAgent is the base implementation of an agent.
type BaseAgent struct {
	config       *AgentConfig
	model        llm.ChatModel
	toolRegistry *tools.Registry
	executor     *tools.Executor
	history      []schema.Message
	capabilities *AgentCapabilities
	historyMu    sync.RWMutex
}

// NewAgent creates a new agent.
func NewAgent(id, name string, model llm.ChatModel, opts ...Option) *BaseAgent {
	config := &AgentConfig{
		ID:           id,
		Name:         name,
		Model:        model,
		SystemPrompt: "",
		Tools:        []tools.Tool{},
		Capabilities: &AgentCapabilities{
			CoreCapabilities: []Capability{CapabilityToolUse},
			ComplexityLevel:  5,
			ConcurrencyLevel: 1,
			Languages:        []string{"zh", "en"},
		},
		MaxHistory:  DefaultAgentConfig.MaxHistory,
		Temperature: DefaultAgentConfig.Temperature,
		MaxTokens:   DefaultAgentConfig.MaxTokens,
	}

	for _, opt := range opts {
		opt(config)
	}

	// Create tool registry and executor
	toolRegistry := tools.NewRegistry()
	for _, tool := range config.Tools {
		toolRegistry.Register(tool)
	}

	executor := tools.NewExecutor(toolRegistry, nil)

	agent := &BaseAgent{
		config:       config,
		model:        model,
		toolRegistry: toolRegistry,
		executor:     executor,
		history:      make([]schema.Message, 0),
		capabilities: config.Capabilities,
	}

	return agent
}

func (a *BaseAgent) ID() string {
	return a.config.ID
}

func (a *BaseAgent) Name() string {
	return a.config.Name
}

func (a *BaseAgent) GetModel() llm.ChatModel {
	return a.model
}

func (a *BaseAgent) GetSystemPrompt() string {
	return a.config.SystemPrompt
}

// ExecuteWithHandoff executes a conversation with handoff support.
func (a *BaseAgent) ExecuteWithHandoff(ctx runtime.Context, input schema.Message) (schema.Message, *schema.Handoff, error) {
	// Execute directly, handoff is handled by the function calling mechanism.
	response, err := a.Execute(ctx, input)
	if err != nil {
		return schema.Message{}, nil, err
	}

	// Check for handoff tool calls.
	handoff := a.extractHandoffFromResponse(response)

	return response, handoff, nil
}

// CanHandoff checks if the agent supports handoff.
func (a *BaseAgent) CanHandoff() bool {
	// Check if the agent has the handoff capability.
	for _, cap := range a.Capabilities() {
		if cap == CapabilityHandoff {
			return true
		}
	}
	return false
}

// extractHandoffFromResponse extracts handoff information from the response.
func (a *BaseAgent) extractHandoffFromResponse(response schema.Message) *schema.Handoff {
	// Check for transfer function in tool calls.
	for _, toolCall := range response.ToolCalls {
		if strings.HasPrefix(toolCall.Name, "transfer_to_") {
			// Parse the target agent.
			target := strings.TrimPrefix(toolCall.Name, "transfer_to_")

			var args map[string]interface{}
			if err := json.Unmarshal(toolCall.Args, &args); err == nil {
				handoff := schema.NewHandoff(target)

				// Extract reason and priority from arguments.
				if reason, ok := args["reason"].(string); ok {
					handoff.WithContext("reason", reason)
				}
				if priority, ok := args["priority"].(float64); ok {
					handoff.WithPriority(int(priority))
				}
				if contextData, ok := args["context"].(map[string]interface{}); ok {
					for k, v := range contextData {
						handoff.WithContext(k, v)
					}
				}

				return handoff
			}

			// If parsing fails, return a basic handoff.
			return schema.NewHandoff(target).WithContext("reason", "function_call")
		}
	}

	return nil
}

func (a *BaseAgent) Tools() []tools.Tool {
	return a.toolRegistry.List()
}

func (a *BaseAgent) Capabilities() []Capability {
	if a.capabilities != nil {
		return a.capabilities.CoreCapabilities
	}

	// Compatibility: if no capability declaration is set, use the old logic.
	capabilities := []Capability{CapabilityReasoning}

	// Determine capabilities based on model and tools.
	if a.model.SupportsStreaming() {
		capabilities = append(capabilities, CapabilityStreaming)
	}

	if len(a.Tools()) > 0 {
		capabilities = append(capabilities, CapabilityToolUse)
	}

	if a.model.SupportsTools() {
		capabilities = append(capabilities, CapabilityPlanning)
	}

	return capabilities
}

// GetCapabilities returns the full capability declaration of the agent.
func (a *BaseAgent) GetCapabilities() *AgentCapabilities {
	if a.capabilities != nil {
		return a.capabilities
	}

	// If not set, return a default declaration based on current capabilities.
	return &AgentCapabilities{
		CoreCapabilities: a.Capabilities(),
		ComplexityLevel:  5,
		ConcurrencyLevel: 1,
		Languages:        []string{"zh", "en"},
		Description:      "General purpose agent",
	}
}

// Execute performs a single turn of conversation.
func (a *BaseAgent) Execute(ctx runtime.Context, input schema.Message) (schema.Message, error) {
	// If state handling is configured, generate inputs from the state
	actualInput := input
	if a.config.StateKey != "" && a.config.InputFromState != nil {
		if state := ctx.GetStateValue(a.config.StateKey); state != nil {
			stateInput := a.config.InputFromState(state)
			actualInput = schema.Message{
				Role:    "user",
				Content: stateInput,
			}
		}
	}

	a.addToHistory(actualInput)
	messages := a.buildMessages()

	req := a.buildLLMRequest(messages)
	resp, err := a.model.Generate(ctx, req)
	if err != nil {
		return schema.Message{}, schema.NewAgentError(a.ID(), "execute", err)
	}

	// Handle tool calls.
	response := resp.Message
	if response.HasToolCalls() {
		response, err = a.handleToolCalls(ctx, response)
		if err != nil {
			return schema.Message{}, schema.NewAgentError(a.ID(), "handle_tool_calls", err)
		}
	}

	// If state handling is configured, write the output to the state
	if a.config.StateKey != "" && a.config.OutputToState != nil {
		if state := ctx.GetStateValue(a.config.StateKey); state != nil {
			a.config.OutputToState(state, response.Content)
		}
	}

	a.addToHistory(response)

	return response, nil
}

// ExecuteStream performs a streaming conversation.
func (a *BaseAgent) ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error) {
	if !a.model.SupportsStreaming() {
		return nil, schema.NewAgentError(a.ID(), "execute_stream", schema.ErrModelNotSupported)
	}

	a.addToHistory(input)
	messages := a.buildMessages()

	req := a.buildLLMRequest(messages)
	return a.model.GenerateStream(ctx, req)
}

// handleToolCalls handles tool calls.
func (a *BaseAgent) handleToolCalls(ctx runtime.Context, message schema.Message) (schema.Message, error) {
	if !message.HasToolCalls() {
		return message, nil
	}

	// Execute tool calls.
	results, err := a.executor.ExecuteParallel(ctx, message.ToolCalls)
	if err != nil {
		return schema.Message{}, err
	}

	toolMessages := make([]schema.Message, len(results))
	for i, result := range results {
		toolMessages[i] = schema.Message{
			ID:      result.ID,
			Role:    schema.RoleTool,
			Content: string(result.Result),
		}
		if result.Error != "" {
			toolMessages[i].Content = result.Error
			toolMessages[i].SetMetadata("error", result.Error)
		}
	}

	// Add tool calls and results to history.
	a.addToHistory(message)
	for _, toolMsg := range toolMessages {
		a.addToHistory(toolMsg)
	}

	// Recall the model to generate the final response.
	messages := a.buildMessages()
	req := a.buildLLMRequest(messages)
	r, err := a.model.Generate(ctx, req)
	if err != nil {
		return schema.Message{}, err
	}
	return r.Message, nil
}

// buildMessages builds the list of messages.
func (a *BaseAgent) buildMessages() []schema.Message {
	messages := make([]schema.Message, 0)

	if a.config.SystemPrompt != "" {
		systemMsg := schema.Message{
			Role:    schema.RoleSystem,
			Content: a.config.SystemPrompt,
		}
		messages = append(messages, systemMsg)
	}

	a.historyMu.RLock()
	historyCopy := make([]schema.Message, len(a.history))
	copy(historyCopy, a.history)
	a.historyMu.RUnlock()

	messages = append(messages, historyCopy...)

	return messages
}

func (a *BaseAgent) addToHistory(message schema.Message) {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()

	a.history = append(a.history, message)
	if len(a.history) > a.config.MaxHistory {
		// Keep recent messages.
		a.history = a.history[len(a.history)-a.config.MaxHistory:]
	}
}

func (a *BaseAgent) ClearHistory() {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	a.history = make([]schema.Message, 0)
}

func (a *BaseAgent) GetHistory() []schema.Message {
	// Return a copy to prevent external modification.
	a.historyMu.RLock()
	defer a.historyMu.RUnlock()
	history := make([]schema.Message, len(a.history))
	copy(history, a.history)
	return history
}

func (a *BaseAgent) AddTool(tool tools.Tool) error {
	return a.toolRegistry.Register(tool)
}

func (a *BaseAgent) RemoveTool(name string) error {
	return a.toolRegistry.Unregister(name)
}

func (a *BaseAgent) HasTool(name string) bool {
	return a.toolRegistry.Has(name)
}

func (a *BaseAgent) UpdateConfig(config *AgentConfig) {
	if config.MaxHistory > 0 {
		a.config.MaxHistory = config.MaxHistory
	}
	if config.Temperature >= 0 {
		a.config.Temperature = config.Temperature
	}
	if config.MaxTokens > 0 {
		a.config.MaxTokens = config.MaxTokens
	}
	if config.SystemPrompt != "" {
		a.config.SystemPrompt = config.SystemPrompt
	}
}

func (a *BaseAgent) buildLLMRequest(messages []schema.Message) *llm.Request {
	req := &llm.Request{Messages: messages}

	cfg := *llm.DefaultGenerationConfig
	if a.config.Temperature >= 0 {
		cfg.Temperature = a.config.Temperature
	}
	if a.config.MaxTokens > 0 {
		cfg.MaxTokens = a.config.MaxTokens
	}
	req.Config = &cfg

	if a.model.SupportsTools() && len(a.toolRegistry.List()) > 0 {
		req.Tools = a.collectToolSpecs()
		req.ToolChoice = &llm.ToolChoiceOption{Type: "auto"}
	}
	return req
}

func (a *BaseAgent) collectToolSpecs() []llm.ToolSpec {
	toolsList := a.toolRegistry.List()
	specs := make([]llm.ToolSpec, 0, len(toolsList))
	for _, t := range toolsList {
		if t.Schema() == nil {
			continue
		}
		params := map[string]interface{}{"type": "object"}
		if t.Schema().Type != "" {
			params["type"] = t.Schema().Type
		}
		if len(t.Schema().Properties) > 0 {
			params["properties"] = t.Schema().Properties
		}
		if len(t.Schema().Required) > 0 {
			params["required"] = t.Schema().Required
		}
		specs = append(specs, llm.ToolSpec{Name: t.Name(), Description: t.Description(), Parameters: params})
	}
	return specs
}

func (a *BaseAgent) GetConfig() *AgentConfig {
	// Return a copy to prevent external modification.
	configCopy := *a.config
	return &configCopy
}
