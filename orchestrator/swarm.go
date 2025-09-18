package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// Swarm represents a dynamic multi-agent collaboration group.
type Swarm interface {
	// AddAgent adds an agent to the group.
	AddAgent(agent agent.Agent) error

	// RemoveAgent removes an agent from the group.
	RemoveAgent(agentID string) error

	// Execute executes a task, and the group autonomously decides how to collaborate.
	Execute(ctx runtime.Context, task schema.Message) (schema.Message, error)

	// ExecuteStream executes a task in streaming mode.
	ExecuteStream(ctx runtime.Context, task schema.Message) (<-chan schema.StreamEvent, error)

	// GetAgents gets all agents.
	GetAgents() []agent.Agent

	// SetStrategy sets the collaboration strategy.
	SetStrategy(strategy SwarmStrategy)

	// GetMetrics gets the group's metrics.
	GetMetrics() SwarmMetrics
}

// SwarmStrategy is the group collaboration strategy.
type SwarmStrategy interface {
	// SelectNext selects the next agent to execute.
	SelectNext(ctx runtime.Context, agents []agent.Agent, task schema.Message, history []SwarmStep) (agent.Agent, error)

	// ShouldContinue determines whether the collaboration should continue.
	ShouldContinue(ctx runtime.Context, steps []SwarmStep, maxSteps int) bool

	// Name is the name of the strategy.
	Name() string
}

// SwarmStep is a step in the group's execution.
type SwarmStep struct {
	Agent     string                 `json:"agent"`
	Input     schema.Message         `json:"input"`
	Output    schema.Message         `json:"output"`
	Handoff   *schema.Handoff        `json:"handoff,omitempty"`
	StartTime time.Time              `json:"start_time"`
	Duration  time.Duration          `json:"duration"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SwarmMetrics are the metrics for the group.
type SwarmMetrics struct {
	TotalExecutions int            `json:"total_executions"`
	SuccessRate     float64        `json:"success_rate"`
	AverageSteps    float64        `json:"average_steps"`
	AverageDuration time.Duration  `json:"average_duration"`
	AgentUsage      map[string]int `json:"agent_usage"`
}

// BaseSwarm is the base implementation of a group.
type BaseSwarm struct {
	agents   []agent.Agent
	strategy SwarmStrategy
	metrics  SwarmMetrics
	mutex    sync.RWMutex
}

// NewSwarm creates a new group.
func NewSwarm(strategy SwarmStrategy) *BaseSwarm {
	return &BaseSwarm{
		agents:   make([]agent.Agent, 0),
		strategy: strategy,
		metrics: SwarmMetrics{
			AgentUsage: make(map[string]int),
		},
	}
}

func (s *BaseSwarm) AddAgent(ag agent.Agent) error {
	if ag == nil {
		return schema.NewValidationError("agent", ag, "agent cannot be nil")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if it already exists.
	for _, existing := range s.agents {
		if existing.ID() == ag.ID() {
			return schema.NewAgentError(ag.ID(), "add", schema.ErrAgentAlreadyExists)
		}
	}

	s.agents = append(s.agents, ag)
	return nil
}

func (s *BaseSwarm) RemoveAgent(agentID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i, ag := range s.agents {
		if ag.ID() == agentID {
			s.agents = append(s.agents[:i], s.agents[i+1:]...)
			return nil
		}
	}

	return schema.NewAgentError(agentID, "remove", schema.ErrAgentNotFound)
}

func (s *BaseSwarm) GetAgents() []agent.Agent {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return a copy to avoid concurrent modification.
	agents := make([]agent.Agent, len(s.agents))
	copy(agents, s.agents)
	return agents
}

func (s *BaseSwarm) SetStrategy(strategy SwarmStrategy) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.strategy = strategy
}

func (s *BaseSwarm) GetMetrics() SwarmMetrics {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.metrics
}

func (s *BaseSwarm) Execute(ctx runtime.Context, task schema.Message) (schema.Message, error) {
	startTime := time.Now()
	s.mutex.Lock()
	s.metrics.TotalExecutions++
	s.mutex.Unlock()

	agents := s.GetAgents()
	if len(agents) == 0 {
		return schema.Message{}, schema.NewValidationError("agents", agents, "no agents in swarm")
	}

	var steps []SwarmStep
	currentMessage := task
	maxSteps := 10 // Prevent infinite loops.

	for len(steps) < maxSteps {
		// Select the next agent.
		selectedAgent, err := s.strategy.SelectNext(ctx, agents, currentMessage, steps)
		if err != nil {
			return schema.Message{}, fmt.Errorf("failed to select next agent: %w", err)
		}

		// Execute the agent.
		stepStartTime := time.Now()
		response, handoff, err := selectedAgent.ExecuteWithHandoff(ctx, currentMessage)
		stepDuration := time.Since(stepStartTime)

		step := SwarmStep{
			Agent:     selectedAgent.ID(),
			Input:     currentMessage,
			Output:    response,
			Handoff:   handoff,
			StartTime: stepStartTime,
			Duration:  stepDuration,
			Metadata:  make(map[string]interface{}),
		}

		if err != nil {
			step.Error = err.Error()
			steps = append(steps, step)
			return schema.Message{}, fmt.Errorf("agent %s execution failed: %w", selectedAgent.ID(), err)
		}

		steps = append(steps, step)

		// Update agent usage statistics.
		s.mutex.Lock()
		s.metrics.AgentUsage[selectedAgent.ID()]++
		s.mutex.Unlock()

		// Check for handoff.
		if handoff != nil && handoff.Target != "" {
			// Find the target agent.
			var targetAgent agent.Agent
			for _, ag := range agents {
				if ag.ID() == handoff.Target || ag.Name() == handoff.Target {
					targetAgent = ag
					break
				}
			}

			if targetAgent != nil {
				// Prepare the handoff message.
				currentMessage = schema.Message{
					Role:      schema.RoleUser,
					Content:   response.Content,
					Timestamp: time.Now(),
				}
				continue
			}
		}

		// Check if we should continue.
		if !s.strategy.ShouldContinue(ctx, steps, maxSteps) {
			break
		}

		currentMessage = response
	}

	// Update metrics.
	duration := time.Since(startTime)
	s.updateMetrics(steps, duration, true)

	if len(steps) > 0 {
		return steps[len(steps)-1].Output, nil
	}

	return schema.Message{}, fmt.Errorf("no steps executed")
}

func (s *BaseSwarm) ExecuteStream(ctx runtime.Context, task schema.Message) (<-chan schema.StreamEvent, error) {
	eventChan := make(chan schema.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		eventChan <- schema.NewStreamEvent(schema.EventStart, nil)

		result, err := s.Execute(ctx, task)
		if err != nil {
			eventChan <- schema.NewErrorEvent(err, "swarm_execute")
			return
		}

		eventChan <- schema.NewStreamEvent(schema.EventEnd, result)
	}()

	return eventChan, nil
}

func (s *BaseSwarm) updateMetrics(steps []SwarmStep, duration time.Duration, success bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update success rate.
	if success {
		s.metrics.SuccessRate = (s.metrics.SuccessRate*float64(s.metrics.TotalExecutions-1) + 1.0) / float64(s.metrics.TotalExecutions)
	} else {
		s.metrics.SuccessRate = (s.metrics.SuccessRate * float64(s.metrics.TotalExecutions-1)) / float64(s.metrics.TotalExecutions)
	}

	// Update average steps.
	s.metrics.AverageSteps = (s.metrics.AverageSteps*float64(s.metrics.TotalExecutions-1) + float64(len(steps))) / float64(s.metrics.TotalExecutions)

	// Update average duration.
	s.metrics.AverageDuration = time.Duration((int64(s.metrics.AverageDuration)*int64(s.metrics.TotalExecutions-1) + int64(duration)) / int64(s.metrics.TotalExecutions))
}

// RoundRobinStrategy is a round-robin strategy.
type RoundRobinStrategy struct {
	lastIndex int
	mutex     sync.Mutex
}

func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{lastIndex: -1}
}

func (r *RoundRobinStrategy) SelectNext(ctx runtime.Context, agents []agent.Agent, task schema.Message, history []SwarmStep) (agent.Agent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.lastIndex = (r.lastIndex + 1) % len(agents)
	return agents[r.lastIndex], nil
}

func (r *RoundRobinStrategy) ShouldContinue(ctx runtime.Context, steps []SwarmStep, maxSteps int) bool {
	// Simple strategy: stop if there is no handoff.
	if len(steps) == 0 {
		return true
	}

	lastStep := steps[len(steps)-1]
	return lastStep.Handoff != nil && lastStep.Handoff.Target != ""
}

func (r *RoundRobinStrategy) Name() string {
	return "round_robin"
}

// ExpertRoutingStrategy is an expert routing strategy.
type ExpertRoutingStrategy struct {
	llmModel llm.ChatModel // LLM model for intelligent judgment.
}

// NewExpertRoutingStrategy creates a new expert routing strategy.
func NewExpertRoutingStrategy(llmModel llm.ChatModel) *ExpertRoutingStrategy {
	return &ExpertRoutingStrategy{
		llmModel: llmModel,
	}
}

func (e *ExpertRoutingStrategy) SelectNext(ctx runtime.Context, agents []agent.Agent, task schema.Message, history []SwarmStep) (agent.Agent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}

	// Intelligent matching based on agent capability declarations.
	bestAgent := e.findBestAgentByCapabilities(agents, task)
	if bestAgent != nil {
		return bestAgent, nil
	}

	// If no suitable agent is found, return the first available agent.
	return agents[0], nil
}

func (e *ExpertRoutingStrategy) ShouldContinue(ctx runtime.Context, steps []SwarmStep, maxSteps int) bool {
	if len(steps) >= maxSteps {
		return false
	}

	if len(steps) == 0 {
		return true
	}

	lastStep := steps[len(steps)-1]
	// If there is a handoff request, continue execution.
	return lastStep.Handoff != nil && lastStep.Handoff.Target != ""
}

func (e *ExpertRoutingStrategy) Name() string {
	return "expert_routing"
}

// findBestAgentByCapabilities finds the best agent based on agent capability declarations.
func (e *ExpertRoutingStrategy) findBestAgentByCapabilities(agents []agent.Agent, task schema.Message) agent.Agent {
	var bestAgent agent.Agent
	var bestScore float64

	for _, ag := range agents {
		score := e.calculateAgentScore(ag, task)
		if score > bestScore {
			bestScore = score
			bestAgent = ag
		}
	}

	return bestAgent
}

// calculateAgentScore calculates the matching score between an agent and a task.
func (e *ExpertRoutingStrategy) calculateAgentScore(ag agent.Agent, task schema.Message) float64 {
	capabilities := ag.GetCapabilities()
	if capabilities == nil {
		return 0.1 // Minimum score.
	}

	score := 0.0

	// Match based on area of expertise.
	for _, expertise := range capabilities.Expertise {
		if e.taskMatchesExpertise(task.Content, expertise) {
			score += 2.0
		}
	}

	// Match based on core capabilities.
	for _, capability := range capabilities.CoreCapabilities {
		if e.taskRequiresCapability(task.Content, capability) {
			score += 1.0
		}
	}

	// Match based on complexity.
	taskComplexity := e.estimateTaskComplexity(task.Content)
	if capabilities.ComplexityLevel >= taskComplexity {
		score += 0.5
	} else {
		score -= 0.5 // Deduct points for insufficient capability.
	}

	// Match based on custom tags.
	for _, tag := range capabilities.CustomTags {
		if strings.Contains(strings.ToLower(task.Content), strings.ToLower(tag)) {
			score += 0.3
		}
	}

	return score
}

// taskMatchesExpertise uses an LLM to intelligently determine if a task matches an area of expertise.
func (e *ExpertRoutingStrategy) taskMatchesExpertise(content, expertise string) bool {
	if e.llmModel == nil {
		return false
	}

	prompt := fmt.Sprintf(`Determine if the following task requires knowledge and skills in the \"%s\" area of expertise.\n\nTask content: %s\nArea of expertise: %s\n\nPlease analyze whether the task content is related to this area of expertise and whether it requires professional knowledge in this field to complete.\n\nPlease answer only: Yes or No`, expertise, content, expertise)

	messages := []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: prompt,
		},
	}

	ctx := runtime.NewContext(context.Background(), "swarm", "expertise_check")
	response, err := e.llmModel.Generate(ctx, messages)
	if err != nil {
		// If the LLM call fails, return false.
		return false
	}

	result := strings.TrimSpace(strings.ToLower(response.Content))
	return strings.Contains(result, "yes") || strings.Contains(result, "true")
}

// taskRequiresCapability uses an LLM to intelligently determine if a task requires a specific capability.
func (e *ExpertRoutingStrategy) taskRequiresCapability(content string, capability agent.Capability) bool {
	if e.llmModel == nil {
		return false
	}

	prompt := fmt.Sprintf(`Analyze if the following task requires the \"%s\" capability.\n\nTask content: %s\n\nCapability descriptions:\n- ToolUse: Requires using tools, calculations, searches, executing external operations.\n- Analysis: Requires analyzing, researching, investigating, statistical data.\n- Writing: Requires writing, composing, creating document content.\n- Engineering: Requires development, programming, code, system design.\n- Design: Requires design, UI/UX, interfaces, prototyping.\n- Planning: Requires planning, strategizing, scheduling.\n- Reasoning: Requires reasoning, logical thinking, analytical judgment.\n\nPlease answer only: Yes or No`, capability, content)

	// Use LLM for intelligent judgment.
	messages := []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: prompt,
		},
	}

	ctx := runtime.NewContext(context.Background(), "swarm", "capability_check")
	response, err := e.llmModel.Generate(ctx, messages)
	if err != nil {
		return false
	}

	result := strings.TrimSpace(strings.ToLower(response.Content))
	return strings.Contains(result, "yes") || strings.Contains(result, "true")
}

// estimateTaskComplexity uses an LLM to intelligently estimate the task complexity (1-10).
func (e *ExpertRoutingStrategy) estimateTaskComplexity(content string) int {
	if e.llmModel == nil {
		return 5
	}

	prompt := fmt.Sprintf(`Please evaluate the complexity of the following task on a scale of 1-10 (1=very simple, 10=extremely complex).\n\nTask content: %s\n\nEvaluation criteria:\n1-2: Simple Q&A, basic information retrieval.\n3-4: General analysis, simple creation.\n5-6: Moderately complex analysis, multi-step tasks.\n7-8: Complex system design, in-depth analysis.\n9-10: Extremely complex architectural design, comprehensive projects.\n\nPlease answer with only a number (1-10):`, content)

	messages := []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: prompt,
		},
	}

	ctx := runtime.NewContext(context.Background(), "swarm", "complexity_estimate")
	response, err := e.llmModel.Generate(ctx, messages)
	if err != nil {
		return 5
	}

	// Parse the complexity number returned by the LLM.
	result := strings.TrimSpace(response.Content)
	if complexity := e.parseComplexityFromResponse(result); complexity > 0 {
		return complexity
	}

	// If parsing fails, return the default complexity.
	return 5
}

// parseComplexityFromResponse parses the complexity number from the LLM response.
func (e *ExpertRoutingStrategy) parseComplexityFromResponse(response string) int {
	// Find a number from 1-10.
	for i := 1; i <= 10; i++ {
		if strings.Contains(response, fmt.Sprintf("%d", i)) {
			return i
		}
	}
	return 0 // Parsing failed.
}

// LoadBalancingStrategy is a load balancing strategy.
type LoadBalancingStrategy struct {
	agentLoad map[string]int // Agent ID -> current load
	mutex     sync.RWMutex
}

// NewLoadBalancingStrategy creates a new load balancing strategy.
func NewLoadBalancingStrategy() *LoadBalancingStrategy {
	return &LoadBalancingStrategy{
		agentLoad: make(map[string]int),
	}
}

func (l *LoadBalancingStrategy) SelectNext(ctx runtime.Context, agents []agent.Agent, task schema.Message, history []SwarmStep) (agent.Agent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Find the agent with the minimum load.
	var selectedAgent agent.Agent
	minLoad := int(^uint(0) >> 1) // Maximum int value.

	for _, ag := range agents {
		load := l.agentLoad[ag.ID()]
		if load < minLoad {
			minLoad = load
			selectedAgent = ag
		}
	}

	// Increase the load of the selected agent.
	if selectedAgent != nil {
		l.agentLoad[selectedAgent.ID()]++
	}

	return selectedAgent, nil
}

func (l *LoadBalancingStrategy) ShouldContinue(ctx runtime.Context, steps []SwarmStep, maxSteps int) bool {
	if len(steps) >= maxSteps {
		return false
	}

	if len(steps) == 0 {
		return true
	}

	lastStep := steps[len(steps)-1]

	// Decrease the load after completion.
	l.mutex.Lock()
	if l.agentLoad[lastStep.Agent] > 0 {
		l.agentLoad[lastStep.Agent]--
	}
	l.mutex.Unlock()

	// If there is a handoff request, continue execution.
	return lastStep.Handoff != nil && lastStep.Handoff.Target != ""
}

func (l *LoadBalancingStrategy) Name() string {
	return "load_balancing"
}

// GetAgentLoad gets the load of an agent.
func (l *LoadBalancingStrategy) GetAgentLoad(agentID string) int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.agentLoad[agentID]
}

// ResetLoad resets all loads.
func (l *LoadBalancingStrategy) ResetLoad() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for k := range l.agentLoad {
		l.agentLoad[k] = 0
	}
}

// CapabilityMatchingStrategy is a capability matching strategy.
type CapabilityMatchingStrategy struct {
	llmModel llm.ChatModel // LLM model for intelligent judgment.
}

// NewCapabilityMatchingStrategy creates a new capability matching strategy.
func NewCapabilityMatchingStrategy(llmModel llm.ChatModel) *CapabilityMatchingStrategy {
	return &CapabilityMatchingStrategy{
		llmModel: llmModel,
	}
}

func (c *CapabilityMatchingStrategy) SelectNext(ctx runtime.Context, agents []agent.Agent, task schema.Message, history []SwarmStep) (agent.Agent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}

	// Calculate the matching score for each agent.
	var bestAgent agent.Agent
	var bestScore float64

	for _, ag := range agents {
		score := c.calculateAgentCapabilityScore(ag, task)
		if score > bestScore {
			bestScore = score
			bestAgent = ag
		}
	}

	if bestAgent == nil {
		return agents[0], nil // Return the first one by default.
	}

	return bestAgent, nil
}

func (c *CapabilityMatchingStrategy) ShouldContinue(ctx runtime.Context, steps []SwarmStep, maxSteps int) bool {
	if len(steps) >= maxSteps {
		return false
	}

	if len(steps) == 0 {
		return true
	}

	lastStep := steps[len(steps)-1]
	return lastStep.Handoff != nil && lastStep.Handoff.Target != ""
}

func (c *CapabilityMatchingStrategy) Name() string {
	return "capability_matching"
}

// calculateAgentCapabilityScore calculates the agent's capability matching score.
func (c *CapabilityMatchingStrategy) calculateAgentCapabilityScore(ag agent.Agent, task schema.Message) float64 {
	capabilities := ag.GetCapabilities()
	if capabilities == nil {
		return 0.1 // Minimum score.
	}

	score := 0.0

	// Match based on core capabilities.
	for _, capability := range capabilities.CoreCapabilities {
		if c.taskRequiresCapability(task.Content, capability) {
			score += c.getCapabilityWeight(capability)
		}
	}

	// Match based on area of expertise.
	for _, expertise := range capabilities.Expertise {
		if c.taskMatchesExpertise(task.Content, expertise) {
			score += 2.0 // Higher weight for matching expertise.
		}
	}

	// Match based on complexity.
	taskComplexity := c.estimateTaskComplexity(task.Content)
	if capabilities.ComplexityLevel >= taskComplexity {
		score += 1.0
	} else {
		score -= 0.5 // Deduct points for insufficient capability.
	}

	// Match based on tool types.
	for _, toolType := range capabilities.ToolTypes {
		if c.taskRequiresToolType(task.Content, toolType) {
			score += 0.5
		}
	}

	return score
}

// getCapabilityWeight gets the capability weight.
func (c *CapabilityMatchingStrategy) getCapabilityWeight(capability agent.Capability) float64 {
	weights := map[agent.Capability]float64{
		agent.CapabilityToolUse:     1.0,
		agent.CapabilityMemory:      0.8,
		agent.CapabilityStreaming:   0.6,
		agent.CapabilityMultimodal:  1.2,
		agent.CapabilityReasoning:   1.5,
		agent.CapabilityPlanning:    1.3,
		agent.CapabilityHandoff:     0.9,
		agent.CapabilityAnalysis:    1.4,
		agent.CapabilityWriting:     1.1,
		agent.CapabilityResearch:    1.2,
		agent.CapabilityEngineering: 1.3,
		agent.CapabilityDesign:      1.1,
		agent.CapabilityMarketing:   1.0,
		agent.CapabilityFinance:     1.2,
		agent.CapabilityLegal:       1.3,
		agent.CapabilitySupport:     0.9,
		agent.CapabilityManagement:  1.1,
		agent.CapabilityEducation:   1.0,
	}

	if weight, exists := weights[capability]; exists {
		return weight
	}
	return 1.0 // Default weight.
}

// taskRequiresCapability uses an LLM to intelligently determine if a task requires a specific capability.
func (c *CapabilityMatchingStrategy) taskRequiresCapability(content string, capability agent.Capability) bool {
	if c.llmModel == nil {
		return false
	}

	prompt := fmt.Sprintf(`Analyze if the following task requires the \"%s\" capability.\n\nTask content: %s\n\nCapability descriptions:\n- ToolUse: Requires using tools, calculations, searches, executing external operations.\n- Analysis: Requires analyzing, researching, investigating, statistical data.\n- Writing: Requires writing, composing, creating document content.\n- Engineering: Requires development, programming, code, system design.\n- Design: Requires design, UI/UX, interfaces, prototyping.\n- Planning: Requires planning, strategizing, scheduling.\n- Reasoning: Requires reasoning, logical thinking, analytical judgment.\n\nPlease answer only: Yes or No`, capability, content)

	messages := []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: prompt,
		},
	}

	ctx := runtime.NewContext(context.Background(), "swarm", "capability_check")
	response, err := c.llmModel.Generate(ctx, messages)
	if err != nil {
		return false
	}

	result := strings.TrimSpace(strings.ToLower(response.Content))
	return strings.Contains(result, "yes") || strings.Contains(result, "true")
}

// taskMatchesExpertise uses an LLM to intelligently determine if a task matches an area of expertise.
func (c *CapabilityMatchingStrategy) taskMatchesExpertise(content, expertise string) bool {
	if c.llmModel == nil {
		return false
	}

	prompt := fmt.Sprintf(`Determine if the following task requires knowledge and skills in the \"%s\" area of expertise.\n\nTask content: %s\nArea of expertise: %s\n\nPlease analyze whether the task content is related to this area of expertise and whether it requires professional knowledge in this field to complete.\n\nPlease answer only: Yes or No`, expertise, content, expertise)

	messages := []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: prompt,
		},
	}

	ctx := runtime.NewContext(context.Background(), "swarm", "expertise_check")
	response, err := c.llmModel.Generate(ctx, messages)
	if err != nil {
		return false
	}

	result := strings.TrimSpace(strings.ToLower(response.Content))
	return strings.Contains(result, "yes") || strings.Contains(result, "true")
}

// estimateTaskComplexity uses an LLM to intelligently estimate the task complexity (1-10).
func (c *CapabilityMatchingStrategy) estimateTaskComplexity(content string) int {
	if c.llmModel == nil {
		return 5
	}

	prompt := fmt.Sprintf(`Please evaluate the complexity of the following task on a scale of 1-10 (1=very simple, 10=extremely complex).\n\nTask content: %s\n\nEvaluation criteria:\n1-2: Simple Q&A, basic information retrieval.\n3-4: General analysis, simple creation.\n5-6: Moderately complex analysis, multi-step tasks.\n7-8: Complex system design, in-depth analysis.\n9-10: Extremely complex architectural design, comprehensive projects.\n\nPlease answer with only a number (1-10):`, content)

	messages := []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: prompt,
		},
	}

	ctx := runtime.NewContext(context.Background(), "swarm", "complexity_estimate")
	response, err := c.llmModel.Generate(ctx, messages)
	if err != nil {
		return 5
	}

	// Parse the complexity number returned by the LLM.
	result := strings.TrimSpace(response.Content)
	if complexity := c.parseComplexityFromResponse(result); complexity > 0 {
		return complexity
	}

	// If parsing fails, return the default complexity.
	return 5
}

// parseComplexityFromResponse parses the complexity number from the LLM response.
func (c *CapabilityMatchingStrategy) parseComplexityFromResponse(response string) int {
	// Find a number from 1-10.
	for i := 1; i <= 10; i++ {
		if strings.Contains(response, fmt.Sprintf("%d", i)) {
			return i
		}
	}
	return 0 // Parsing failed.
}

// taskRequiresToolType checks if the task requires a specific tool type.
func (c *CapabilityMatchingStrategy) taskRequiresToolType(content, toolType string) bool {
	content = strings.ToLower(content)
	toolType = strings.ToLower(toolType)

	// Simple keyword matching.
	return strings.Contains(content, toolType)
}
