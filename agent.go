package mas

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/voocel/mas/llm"
)

type agent struct {
	name         string
	model        string
	systemPrompt string
	temperature  float64
	maxTokens    int
	tools        []Tool
	memory       Memory
	state        map[string]interface{}
	provider     llm.Provider
	eventBus     EventBus

	// Cognitive ability
	skillLibrary   SkillLibrary
	cognitiveState *CognitiveState
	cognitiveMode  CognitiveMode

	// Autonomous capability
	goalManager GoalManager

	// Learning capability
	learningEngine LearningEngine

	mu sync.RWMutex
}

func NewAgent(model, apiKey string) Agent {
	provider, err := llm.NewProvider(model, apiKey)
	if err != nil {
		provider = nil
	}

	return &agent{
		name:         "assistant",
		model:        model,
		temperature:  0.7,
		maxTokens:    2000,
		tools:        make([]Tool, 0),
		state:        make(map[string]interface{}),
		provider:     provider,
		skillLibrary: NewSkillLibrary(),
		cognitiveState: &CognitiveState{
			CurrentLayer:    CortexLayer,
			Mode:            AutomaticMode,
			LoadedSkills:    make([]string, 0),
			RecentDecisions: make([]*Decision, 0),
			LastUpdate:      time.Now(),
		},
		cognitiveMode: AutomaticMode,
	}
}

func NewAgentWithConfig(config AgentConfig) Agent {
	var provider llm.Provider
	if config.Provider != nil {
		provider = nil
	}
	if provider == nil && config.APIKey != "" {
		providerConfig := llm.ProviderConfig{
			Model:       config.Model,
			APIKey:      config.APIKey,
			BaseURL:     config.BaseURL,
			Temperature: config.Temperature,
			MaxTokens:   config.MaxTokens,
		}
		provider, _ = llm.NewProviderWithConfig(providerConfig)
	}

	if config.State == nil {
		config.State = make(map[string]interface{})
	}

	return &agent{
		name:         config.Name,
		model:        config.Model,
		systemPrompt: config.SystemPrompt,
		temperature:  config.Temperature,
		maxTokens:    config.MaxTokens,
		tools:        config.Tools,
		memory:       config.Memory,
		state:        config.State,
		provider:     provider,
	}
}

func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Name:        "assistant",
		Model:       "gpt-4.1-mini",
		Temperature: 0.7,
		MaxTokens:   2000,
		State:       make(map[string]interface{}),
	}
}

func (a *agent) Chat(ctx context.Context, message string) (string, error) {
	// Emit chat start event
	if err := a.PublishEvent(ctx, EventAgentChatStart, EventData(
		"message", message,
		"agent_name", a.Name(),
	)); err != nil {
		// Log but don't fail
		fmt.Printf("Failed to publish chat start event: %v\n", err)
	}

	if a.provider == nil {
		return "", fmt.Errorf("no LLM provider configured")
	}

	if a.memory != nil {
		if err := a.memory.Add(ctx, RoleUser, message); err != nil {
			return "", fmt.Errorf("failed to add message to memory: %w", err)
		}
	}

	messages, err := a.prepareMessages(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to prepare messages: %w", err)
	}

	var tools []llm.ToolDefinition
	if len(a.tools) > 0 {
		tools = a.convertToolsToLLMFormat()
	}

	req := llm.ChatRequest{
		Messages:    a.convertMessagesToLLM(messages),
		Model:       a.model,
		Temperature: a.temperature,
		MaxTokens:   a.maxTokens,
		Tools:       tools,
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	choice := resp.Choices[0]

	if len(choice.Message.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, a.convertMessageFromLLM(choice.Message), messages)
	}

	if a.memory != nil {
		if err := a.memory.Add(ctx, RoleAssistant, choice.Message.Content); err != nil {
			return "", fmt.Errorf("failed to add response to memory: %w", err)
		}
	}

	// Emit chat end event
	if err := a.PublishEvent(ctx, EventAgentChatEnd, EventData(
		"message", message,
		"response", choice.Message.Content,
		"agent_name", a.Name(),
		"success", true,
	)); err != nil {
		fmt.Printf("Failed to publish chat end event: %v\n", err)
	}

	return choice.Message.Content, nil
}

func (a *agent) WithTools(tools ...Tool) Agent {
	return &agent{
		name:         a.name,
		model:        a.model,
		systemPrompt: a.systemPrompt,
		temperature:  a.temperature,
		maxTokens:    a.maxTokens,
		tools:        append(a.tools, tools...),
		memory:       a.memory,
		state:        a.state,
		provider:     a.provider,
		eventBus:     a.eventBus,
	}
}

func (a *agent) WithMemory(memory Memory) Agent {
	return &agent{
		name:         a.name,
		model:        a.model,
		systemPrompt: a.systemPrompt,
		temperature:  a.temperature,
		maxTokens:    a.maxTokens,
		tools:        a.tools,
		memory:       memory,
		state:        a.state,
		provider:     a.provider,
		eventBus:     a.eventBus,
	}
}

func (a *agent) WithSystemPrompt(prompt string) Agent {
	return &agent{
		name:         a.name,
		model:        a.model,
		systemPrompt: prompt,
		temperature:  a.temperature,
		maxTokens:    a.maxTokens,
		tools:        a.tools,
		memory:       a.memory,
		state:        a.state,
		provider:     a.provider,
		eventBus:     a.eventBus,
	}
}

func (a *agent) WithTemperature(temp float64) Agent {
	return &agent{
		name:         a.name,
		model:        a.model,
		systemPrompt: a.systemPrompt,
		temperature:  temp,
		maxTokens:    a.maxTokens,
		tools:        a.tools,
		memory:       a.memory,
		state:        a.state,
		provider:     a.provider,
		eventBus:     a.eventBus,
	}
}

func (a *agent) WithMaxTokens(tokens int) Agent {
	return &agent{
		name:         a.name,
		model:        a.model,
		systemPrompt: a.systemPrompt,
		temperature:  a.temperature,
		maxTokens:    tokens,
		tools:        a.tools,
		memory:       a.memory,
		state:        a.state,
		provider:     a.provider,
		eventBus:     a.eventBus,
	}
}

func (a *agent) WithEventBus(eventBus EventBus) Agent {
	return &agent{
		name:         a.name,
		model:        a.model,
		systemPrompt: a.systemPrompt,
		temperature:  a.temperature,
		maxTokens:    a.maxTokens,
		tools:        a.tools,
		memory:       a.memory,
		state:        a.state,
		provider:     a.provider,
		eventBus:     eventBus,
	}
}

func (a *agent) SetState(key string, value interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state[key] = value
}

func (a *agent) GetState(key string) interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state[key]
}

func (a *agent) ClearState() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = make(map[string]interface{})
}

func (a *agent) Name() string {
	return a.name
}

func (a *agent) Model() string {
	return a.model
}

// Event-related methods
func (a *agent) GetEventBus() EventBus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.eventBus
}

func (a *agent) StreamEvents(ctx context.Context, eventTypes ...EventType) (<-chan Event, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.eventBus == nil {
		return nil, fmt.Errorf("no event bus configured")
	}

	if streamBus, ok := a.eventBus.(StreamEventBus); ok {
		return streamBus.Stream(ctx, eventTypes...)
	}

	return nil, fmt.Errorf("event bus does not support streaming")
}

func (a *agent) PublishEvent(ctx context.Context, eventType EventType, data map[string]interface{}) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.eventBus == nil {
		return nil // Silently ignore if no event bus
	}

	event := NewEvent(eventType, a.Name(), data)
	return a.eventBus.Publish(ctx, event)
}

func (a *agent) prepareMessages(ctx context.Context, currentMessage string) ([]ChatMessage, error) {
	var messages []ChatMessage
	if a.systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    RoleSystem,
			Content: a.systemPrompt,
		})
	}

	if a.memory != nil {
		history, err := a.memory.GetHistory(ctx, 10)
		if err == nil {
			for _, msg := range history {
				if msg.Role == RoleUser && msg.Content == currentMessage {
					continue
				}
				messages = append(messages, ChatMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	if !a.messageExistsInHistory(messages, currentMessage) {
		messages = append(messages, ChatMessage{
			Role:    RoleUser,
			Content: currentMessage,
		})
	}

	return messages, nil
}

func (a *agent) messageExistsInHistory(messages []ChatMessage, content string) bool {
	for _, msg := range messages {
		if msg.Role == RoleUser && msg.Content == content {
			return true
		}
	}
	return false
}

func (a *agent) handleToolCalls(ctx context.Context, assistantMessage ChatMessage, conversationHistory []ChatMessage) (string, error) {
	if a.memory != nil {
		toolCallsJSON, _ := json.Marshal(assistantMessage.ToolCalls)
		if err := a.memory.Add(ctx, RoleAssistant, fmt.Sprintf("Tool calls: %s", string(toolCallsJSON))); err != nil {
			return "", fmt.Errorf("failed to add tool calls to memory: %w", err)
		}
	}

	var updatedMessages []ChatMessage
	updatedMessages = append(updatedMessages, conversationHistory...)
	updatedMessages = append(updatedMessages, assistantMessage)

	for _, toolCall := range assistantMessage.ToolCalls {
		if toolCall.Type != "function" {
			continue
		}

		var selectedTool Tool
		for _, tool := range a.tools {
			if tool.Name() == toolCall.Function.Name {
				selectedTool = tool
				break
			}
		}

		if selectedTool == nil {
			result := fmt.Sprintf("Error: Tool '%s' not found", toolCall.Function.Name)
			updatedMessages = append(updatedMessages, ChatMessage{
				Role:       RoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			})
			continue
		}

		var params map[string]interface{}
		if toolCall.Function.Arguments != "" {
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params)
			if err != nil {
				result := fmt.Sprintf("Error parsing tool parameters: %v", err)
				updatedMessages = append(updatedMessages, ChatMessage{
					Role:       RoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
				})
				continue
			}
		}

		toolResult, err := selectedTool.Execute(ctx, params)
		var result string
		if err != nil {
			result = fmt.Sprintf("Error executing tool: %v", err)
		} else {
			resultJSON, _ := json.Marshal(toolResult)
			result = string(resultJSON)
		}

		updatedMessages = append(updatedMessages, ChatMessage{
			Role:       RoleTool,
			Content:    result,
			ToolCallID: toolCall.ID,
		})

		if a.memory != nil {
			if err := a.memory.Add(ctx, RoleTool, fmt.Sprintf("Tool '%s' result: %s", selectedTool.Name(), result)); err != nil {
				return "", fmt.Errorf("failed to add tool result to memory: %w", err)
			}
		}
	}

	req := llm.ChatRequest{
		Messages:    a.convertMessagesToLLM(updatedMessages),
		Model:       a.model,
		Temperature: a.temperature,
		MaxTokens:   a.maxTokens,
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call after tool execution failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned after tool execution")
	}

	finalResponse := resp.Choices[0].Message.Content

	if a.memory != nil {
		if err := a.memory.Add(ctx, RoleAssistant, finalResponse); err != nil {
			return "", fmt.Errorf("failed to add final response to memory: %w", err)
		}
	}

	return finalResponse, nil
}

func (a *agent) convertMessagesToLLM(messages []ChatMessage) []llm.Message {
	result := make([]llm.Message, len(messages))
	for i, msg := range messages {
		result[i] = llm.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCalls:  a.convertToolCallsToLLM(msg.ToolCalls),
			ToolCallID: msg.ToolCallID,
		}
	}
	return result
}

func (a *agent) convertMessageFromLLM(msg llm.Message) ChatMessage {
	return ChatMessage{
		Role:       msg.Role,
		Content:    msg.Content,
		Name:       msg.Name,
		ToolCalls:  a.convertToolCallsFromLLM(msg.ToolCalls),
		ToolCallID: msg.ToolCallID,
	}
}

func (a *agent) convertToolCallsToLLM(toolCalls []ToolCall) []llm.ToolCall {
	result := make([]llm.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = llm.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: llm.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

func (a *agent) convertToolCallsFromLLM(toolCalls []llm.ToolCall) []ToolCall {
	result := make([]ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

func (a *agent) convertToolsToLLMFormat() []llm.ToolDefinition {
	var tools []llm.ToolDefinition

	for _, tool := range a.tools {
		schema := tool.Schema()
		parameters := make(map[string]interface{})

		if schema != nil {
			parameters["type"] = schema.Type
			parameters["properties"] = schema.Properties
			parameters["required"] = schema.Required
		}

		tools = append(tools, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  parameters,
			},
		})
	}

	return tools
}

// Cognitive capabilities implementation

func (a *agent) Plan(ctx context.Context, goal string) (*Plan, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Emit planning event
	if err := a.PublishEvent(ctx, EventType("agent.plan.start"), EventData(
		"goal", goal,
		"agent_name", a.name,
	)); err != nil {
		fmt.Printf("Failed to publish plan start event: %v\n", err)
	}

	// Use LLM to generate plan
	planPrompt := fmt.Sprintf(`Create a step-by-step plan to achieve this goal: %s

Please respond with a structured plan including:
1. Break down into logical steps
2. Identify which cognitive layer each step should use:
   - reflex: immediate responses
   - cerebellum: learned skills
   - cortex: reasoning and analysis
   - meta: high-level planning

Goal: %s`, goal, goal)

	response, err := a.Chat(ctx, planPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	plan := NewPlan(goal)
	plan.Context["llm_response"] = response

	// Update cognitive state
	a.cognitiveState.ActivePlan = plan
	a.cognitiveState.CurrentLayer = MetaLayer
	a.cognitiveState.LastUpdate = time.Now()

	// Emit planning completion event
	if err := a.PublishEvent(ctx, EventType("agent.plan.complete"), EventData(
		"plan_id", plan.ID,
		"goal", goal,
		"agent_name", a.name,
	)); err != nil {
		fmt.Printf("Failed to publish plan complete event: %v\n", err)
	}

	return plan, nil
}

func (a *agent) Reason(ctx context.Context, situation *Situation) (*Decision, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Emit reasoning event
	if err := a.PublishEvent(ctx, EventType("agent.reason.start"), EventData(
		"situation", situation,
		"agent_name", a.name,
	)); err != nil {
		fmt.Printf("Failed to publish reason start event: %v\n", err)
	}

	// Create reasoning prompt
	reasoningPrompt := fmt.Sprintf(`Analyze this situation and make a decision:

Context: %v
Inputs: %v
Constraints: %v
Goals: %v

Please provide:
1. Your analysis of the situation
2. Recommended action
3. Confidence level (0-1)
4. Which cognitive layer to use for execution`,
		situation.Context, situation.Inputs, situation.Constraints, situation.Goals)

	response, err := a.Chat(ctx, reasoningPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to reason about situation: %w", err)
	}

	// Create decision
	decision := &Decision{
		Action:     extractAction(response),
		Layer:      CortexLayer,
		Confidence: extractConfidence(response),
		Reasoning:  response,
		Parameters: make(map[string]interface{}),
		Timestamp:  time.Now(),
	}

	// Update cognitive state
	a.cognitiveState.RecentDecisions = append(a.cognitiveState.RecentDecisions, decision)
	if len(a.cognitiveState.RecentDecisions) > 10 {
		a.cognitiveState.RecentDecisions = a.cognitiveState.RecentDecisions[1:]
	}
	a.cognitiveState.CurrentLayer = CortexLayer
	a.cognitiveState.LastUpdate = time.Now()

	// Emit reasoning completion event
	if err := a.PublishEvent(ctx, EventType("agent.reason.complete"), EventData(
		"decision", decision,
		"agent_name", a.name,
	)); err != nil {
		fmt.Printf("Failed to publish reason complete event: %v\n", err)
	}

	return decision, nil
}

func (a *agent) ExecuteSkill(ctx context.Context, skillName string, params map[string]interface{}) (interface{}, error) {
	a.mu.RLock()
	skill, err := a.skillLibrary.GetSkill(skillName)
	a.mu.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("skill not found: %w", err)
	}

	// Emit skill execution event
	if err := a.PublishEvent(ctx, EventType("agent.skill.start"), EventData(
		"skill_name", skillName,
		"parameters", params,
		"agent_name", a.name,
	)); err != nil {
		fmt.Printf("Failed to publish skill start event: %v\n", err)
	}

	// Execute skill
	result, err := skill.Execute(ctx, params)

	// Update cognitive state
	a.mu.Lock()
	a.cognitiveState.CurrentLayer = skill.Layer()
	a.cognitiveState.LastUpdate = time.Now()
	a.mu.Unlock()

	// Emit skill completion event
	eventType := EventType("agent.skill.complete")
	eventData := EventData(
		"skill_name", skillName,
		"parameters", params,
		"agent_name", a.name,
		"success", err == nil,
	)

	if err != nil {
		eventType = EventType("agent.skill.error")
		eventData["error"] = err.Error()
	} else {
		eventData["result"] = result
	}

	if publishErr := a.PublishEvent(ctx, eventType, eventData); publishErr != nil {
		fmt.Printf("Failed to publish skill event: %v\n", publishErr)
	}

	return result, err
}

func (a *agent) React(ctx context.Context, stimulus *Stimulus) (*Action, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Emit reaction event
	if err := a.PublishEvent(ctx, EventType("agent.react.start"), EventData(
		"stimulus", stimulus,
		"agent_name", a.name,
	)); err != nil {
		fmt.Printf("Failed to publish react start event: %v\n", err)
	}

	// Quick reactive response based on stimulus urgency
	var action *Action

	if stimulus.Urgency > 0.8 {
		// High urgency - immediate reflex response
		action = &Action{
			Type:       "immediate_response",
			Layer:      ReflexLayer,
			Parameters: stimulus.Data,
			Priority:   int(stimulus.Urgency * 100),
			Timestamp:  time.Now(),
		}
	} else {
		// Lower urgency - deliberate response
		action = &Action{
			Type:       "deliberate_response",
			Layer:      CortexLayer,
			Parameters: stimulus.Data,
			Priority:   int(stimulus.Urgency * 100),
			Timestamp:  time.Now(),
		}
	}

	// Update cognitive state
	a.cognitiveState.CurrentLayer = action.Layer
	a.cognitiveState.LastUpdate = time.Now()

	// Emit reaction completion event
	if err := a.PublishEvent(ctx, EventType("agent.react.complete"), EventData(
		"action", action,
		"agent_name", a.name,
	)); err != nil {
		fmt.Printf("Failed to publish react complete event: %v\n", err)
	}

	return action, nil
}

func (a *agent) GetSkillLibrary() SkillLibrary {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.skillLibrary
}

func (a *agent) WithSkills(skills ...Skill) Agent {
	newAgent := &agent{
		name:         a.name,
		model:        a.model,
		systemPrompt: a.systemPrompt,
		temperature:  a.temperature,
		maxTokens:    a.maxTokens,
		tools:        a.tools,
		memory:       a.memory,
		state:        a.state,
		provider:     a.provider,
		eventBus:     a.eventBus,
		skillLibrary: NewSkillLibrary(),
		cognitiveState: &CognitiveState{
			CurrentLayer:    a.cognitiveState.CurrentLayer,
			Mode:            a.cognitiveState.Mode,
			ActivePlan:      a.cognitiveState.ActivePlan,
			LoadedSkills:    make([]string, 0),
			RecentDecisions: make([]*Decision, 0),
			LastUpdate:      time.Now(),
		},
		cognitiveMode: a.cognitiveMode,
	}

	// Copy existing skills
	for _, skill := range a.skillLibrary.ListSkills() {
		newAgent.skillLibrary.RegisterSkill(skill)
	}

	// Add new skills
	for _, skill := range skills {
		newAgent.skillLibrary.RegisterSkill(skill)
		newAgent.cognitiveState.LoadedSkills = append(newAgent.cognitiveState.LoadedSkills, skill.Name())
	}

	return newAgent
}

func (a *agent) GetCognitiveState() *CognitiveState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy to prevent external modification
	state := *a.cognitiveState
	return &state
}

func (a *agent) SetCognitiveMode(mode CognitiveMode) Agent {
	newAgent := &agent{
		name:         a.name,
		model:        a.model,
		systemPrompt: a.systemPrompt,
		temperature:  a.temperature,
		maxTokens:    a.maxTokens,
		tools:        a.tools,
		memory:       a.memory,
		state:        a.state,
		provider:     a.provider,
		eventBus:     a.eventBus,
		skillLibrary: a.skillLibrary,
		cognitiveState: &CognitiveState{
			CurrentLayer:    a.cognitiveState.CurrentLayer,
			Mode:            mode,
			ActivePlan:      a.cognitiveState.ActivePlan,
			LoadedSkills:    a.cognitiveState.LoadedSkills,
			RecentDecisions: a.cognitiveState.RecentDecisions,
			LastUpdate:      time.Now(),
		},
		cognitiveMode: mode,
	}

	return newAgent
}

// Helper functions for cognitive processing

func extractAction(response string) string {
	// Simple extraction logic - in production you might use more sophisticated parsing
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "action") || strings.Contains(strings.ToLower(line), "recommend") {
			return strings.TrimSpace(line)
		}
	}
	return "continue"
}

func extractConfidence(response string) float64 {
	// Simple confidence extraction - in production you might use regex or LLM structured output
	if strings.Contains(strings.ToLower(response), "high confidence") || strings.Contains(strings.ToLower(response), "very confident") {
		return 0.9
	} else if strings.Contains(strings.ToLower(response), "medium confidence") || strings.Contains(strings.ToLower(response), "somewhat confident") {
		return 0.7
	} else if strings.Contains(strings.ToLower(response), "low confidence") || strings.Contains(strings.ToLower(response), "uncertain") {
		return 0.4
	}
	return 0.6 // default
}

// Autonomous capabilities implementation

func (a *agent) WithGoalManager(manager GoalManager) Agent {
	return &agent{
		name:           a.name,
		model:          a.model,
		systemPrompt:   a.systemPrompt,
		temperature:    a.temperature,
		maxTokens:      a.maxTokens,
		tools:          a.tools,
		memory:         a.memory,
		state:          a.state,
		provider:       a.provider,
		eventBus:       a.eventBus,
		skillLibrary:   a.skillLibrary,
		cognitiveState: a.cognitiveState,
		cognitiveMode:  a.cognitiveMode,
		goalManager:    manager,
	}
}

func (a *agent) GetGoalManager() GoalManager {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.goalManager
}

func (a *agent) AddGoal(ctx context.Context, goal *Goal) error {
	if a.goalManager == nil {
		return fmt.Errorf("goal manager not configured - use WithGoalManager() first")
	}
	return a.goalManager.AddGoal(ctx, goal)
}

func (a *agent) StartAutonomous(ctx context.Context, strategy AutonomousStrategy) error {
	if a.goalManager == nil {
		return fmt.Errorf("goal manager not configured - use WithGoalManager() first")
	}

	// Emit autonomous start event
	if err := a.PublishEvent(ctx, EventType("agent.autonomous.start"), EventData(
		"agent_name", a.name,
		"strategy", strategy,
	)); err != nil {
		fmt.Printf("Failed to publish autonomous start event: %v\n", err)
	}

	return a.goalManager.StartAutonomousMode(ctx, strategy)
}

func (a *agent) StopAutonomous(ctx context.Context) error {
	if a.goalManager == nil {
		return fmt.Errorf("goal manager not configured")
	}

	err := a.goalManager.StopAutonomousMode(ctx)

	// Emit autonomous stop event
	if publishErr := a.PublishEvent(ctx, EventType("agent.autonomous.stop"), EventData(
		"agent_name", a.name,
	)); publishErr != nil {
		fmt.Printf("Failed to publish autonomous stop event: %v\n", publishErr)
	}

	return err
}

func (a *agent) IsAutonomous() bool {
	if a.goalManager == nil {
		return false
	}
	return a.goalManager.IsAutonomous()
}

// Learning capabilities implementation

func (a *agent) WithLearningEngine(engine LearningEngine) Agent {
	return &agent{
		name:           a.name,
		model:          a.model,
		systemPrompt:   a.systemPrompt,
		temperature:    a.temperature,
		maxTokens:      a.maxTokens,
		tools:          a.tools,
		memory:         a.memory,
		state:          a.state,
		provider:       a.provider,
		eventBus:       a.eventBus,
		skillLibrary:   a.skillLibrary,
		cognitiveState: a.cognitiveState,
		cognitiveMode:  a.cognitiveMode,
		goalManager:    a.goalManager,
		learningEngine: engine,
	}
}

func (a *agent) GetLearningEngine() LearningEngine {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.learningEngine
}

func (a *agent) RecordExperience(ctx context.Context, experience *Experience) error {
	if a.learningEngine == nil {
		return fmt.Errorf("learning engine not configured - use WithLearningEngine() first")
	}
	return a.learningEngine.RecordExperience(ctx, experience)
}

func (a *agent) SelfReflect(ctx context.Context) (*SelfReflection, error) {
	if a.learningEngine == nil {
		return nil, fmt.Errorf("learning engine not configured - use WithLearningEngine() first")
	}

	// Emit self-reflection start event
	if err := a.PublishEvent(ctx, EventType("agent.self_reflection.start"), EventData(
		"agent_name", a.name,
	)); err != nil {
		fmt.Printf("Failed to publish self-reflection start event: %v\n", err)
	}

	reflection, err := a.learningEngine.SelfReflect(ctx)

	// Emit self-reflection completion event
	eventType := EventType("agent.self_reflection.complete")
	eventData := EventData(
		"agent_name", a.name,
		"success", err == nil,
	)

	if err != nil {
		eventType = EventType("agent.self_reflection.error")
		eventData["error"] = err.Error()
	} else {
		eventData["learning_progress"] = reflection.LearningProgress
		eventData["self_confidence"] = reflection.SelfConfidence
		eventData["adaptation_needed"] = reflection.AdaptationNeeded
	}

	if publishErr := a.PublishEvent(ctx, eventType, eventData); publishErr != nil {
		fmt.Printf("Failed to publish self-reflection event: %v\n", publishErr)
	}

	return reflection, err
}

func (a *agent) GetLearningMetrics() *LearningMetrics {
	if a.learningEngine == nil {
		return &LearningMetrics{} // Return empty metrics if no learning engine
	}
	return a.learningEngine.GetLearningMetrics()
}
