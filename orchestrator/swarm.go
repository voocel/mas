package orchestrator

import (
	"fmt"
	"sync"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/coordination"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

const (
	swarmContextKeyNextTarget = "swarm.next_target"
)

// SwarmOption customizes swarm behaviour.
type SwarmOption func(*BaseSwarm)

// WithSwarmPlanner injects a custom Planner.
func WithSwarmPlanner(planner coordination.Planner) SwarmOption {
	return func(s *BaseSwarm) {
		s.planner = planner
	}
}

// WithSwarmRouter injects a custom Router.
func WithSwarmRouter(router coordination.Router) SwarmOption {
	return func(s *BaseSwarm) {
		s.router = router
	}
}

// WithSwarmMaxSteps limits the maximum collaboration steps.
func WithSwarmMaxSteps(max int) SwarmOption {
	return func(s *BaseSwarm) {
		if max > 0 {
			s.maxSteps = max
		}
	}
}

// Swarm defines the multi-agent collaboration interface.
type Swarm interface {
	AddAgent(agent agent.Agent) error
	RemoveAgent(agentID string) error
	GetAgents() []agent.Agent
	SetPlanner(planner coordination.Planner)
	SetRouter(router coordination.Router)
	Execute(ctx runtime.Context, task schema.Message) (schema.Message, error)
	ExecuteStream(ctx runtime.Context, task schema.Message) (<-chan schema.StreamEvent, error)
	GetMetrics() SwarmMetrics
}

// SwarmMetrics holds aggregated collaboration statistics.
type SwarmMetrics struct {
	TotalExecutions int            `json:"total_executions"`
	SuccessRate     float64        `json:"success_rate"`
	AverageSteps    float64        `json:"average_steps"`
	AverageDuration time.Duration  `json:"average_duration"`
	AgentUsage      map[string]int `json:"agent_usage"`
}

// BaseSwarm implements Swarm using Planner/Router abstractions.
type BaseSwarm struct {
	agents []agent.Agent

	planner coordination.Planner
	router  coordination.Router

	maxSteps int

	metrics SwarmMetrics

	mu        sync.RWMutex
	lastIndex int
}

// NewSwarm builds a swarm with round-robin defaults.
func NewSwarm(options ...SwarmOption) *BaseSwarm {
	swarm := &BaseSwarm{
		agents:   make([]agent.Agent, 0),
		maxSteps: 10,
		metrics:  SwarmMetrics{AgentUsage: make(map[string]int)},
	}
	for _, opt := range options {
		if opt != nil {
			opt(swarm)
		}
	}
	if swarm.planner == nil {
		swarm.planner = swarm.defaultPlanner()
	}
	if swarm.router == nil {
		swarm.router = swarm.defaultRouter()
	}
	return swarm
}

// AddAgent registers an agent for collaboration.
func (s *BaseSwarm) AddAgent(ag agent.Agent) error {
	if ag == nil {
		return schema.NewValidationError("agent", ag, "agent cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.agents {
		if existing.ID() == ag.ID() {
			return schema.NewAgentError(ag.ID(), "add", schema.ErrAgentAlreadyExists)
		}
	}

	s.agents = append(s.agents, ag)
	return nil
}

// RemoveAgent removes an agent from the swarm.
func (s *BaseSwarm) RemoveAgent(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, ag := range s.agents {
		if ag.ID() == agentID {
			s.agents = append(s.agents[:i], s.agents[i+1:]...)
			delete(s.metrics.AgentUsage, agentID)
			return nil
		}
	}
	return schema.NewAgentError(agentID, "remove", schema.ErrAgentNotFound)
}

// GetAgents returns a copy of registered agents.
func (s *BaseSwarm) GetAgents() []agent.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]agent.Agent, len(s.agents))
	copy(agents, s.agents)
	return agents
}

// SetPlanner replaces the planner implementation.
func (s *BaseSwarm) SetPlanner(planner coordination.Planner) {
	if planner == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.planner = planner
}

// SetRouter replaces the router implementation.
func (s *BaseSwarm) SetRouter(router coordination.Router) {
	if router == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.router = router
}

// Execute coordinates multi-agent collaboration for one task.
func (s *BaseSwarm) Execute(ctx runtime.Context, task schema.Message) (schema.Message, error) {
	start := time.Now()

	s.mu.RLock()
	planner := s.planner
	router := s.router
	maxSteps := s.maxSteps
	s.mu.RUnlock()

	if planner == nil || router == nil {
		return schema.Message{}, fmt.Errorf("swarm: planner or router not configured")
	}

	req := coordination.Request{
		Input: task,
		Type:  coordination.RequestTypeAuto,
	}

	plan, err := planner.Plan(ctx, req)
	if err != nil {
		return schema.Message{}, err
	}
	if err := plan.Validate(); err != nil {
		return schema.Message{}, err
	}

	steps, err := plan.OrderedSteps()
	if err != nil {
		return schema.Message{}, err
	}

	if len(steps) > maxSteps {
		steps = steps[:maxSteps]
	}

	current := task
	executedSteps := 0
	s.clearNextTarget(ctx)

	for _, step := range steps {
		target, routeErr := router.Route(ctx, req, plan, step)
		if routeErr != nil {
			return schema.Message{}, routeErr
		}
		if err := coordination.EnsureTarget(target); err != nil {
			return schema.Message{}, err
		}
		if target.Type != coordination.TargetAgent {
			return schema.Message{}, fmt.Errorf("swarm: only agent targets supported, got %s", target.Type)
		}

		ag := s.findAgent(target.Name)
		if ag == nil {
			return schema.Message{}, schema.NewAgentError(target.Name, "execute", schema.ErrAgentNotFound)
		}

		response, handoff, execErr := ag.ExecuteWithHandoff(ctx, current)

		s.recordAgentUsage(ag.ID())

		if execErr != nil {
			s.updateMetrics(false, executedSteps+1, time.Since(start))
			return schema.Message{}, execErr
		}

		executedSteps++
		current = response

		if handoff != nil && handoff.Target != "" {
			s.storeNextTarget(ctx, handoff.Target)
		} else {
			s.clearNextTarget(ctx)
			break
		}
	}

	s.updateMetrics(true, executedSteps, time.Since(start))
	return current, nil
}

// ExecuteStream wraps Execute with a streaming event channel.
func (s *BaseSwarm) ExecuteStream(ctx runtime.Context, task schema.Message) (<-chan schema.StreamEvent, error) {
	events := make(chan schema.StreamEvent, 100)

	go func() {
		defer close(events)
		events <- schema.NewStreamEvent(schema.EventStart, nil)
		result, err := s.Execute(ctx, task)
		if err != nil {
			events <- schema.NewErrorEvent(err, "swarm_execute")
			return
		}
		events <- schema.NewStreamEvent(schema.EventEnd, result)
	}()

	return events, nil
}

// GetMetrics returns a snapshot of current metrics.
func (s *BaseSwarm) GetMetrics() SwarmMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	metrics := s.metrics
	metrics.AgentUsage = make(map[string]int, len(s.metrics.AgentUsage))
	for k, v := range s.metrics.AgentUsage {
		metrics.AgentUsage[k] = v
	}
	return metrics
}

func (s *BaseSwarm) findAgent(id string) agent.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ag := range s.agents {
		if ag.ID() == id || ag.Name() == id {
			return ag
		}
	}
	return nil
}

func (s *BaseSwarm) recordAgentUsage(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.AgentUsage[agentID]++
}

func (s *BaseSwarm) updateMetrics(success bool, steps int, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.TotalExecutions++
	total := float64(s.metrics.TotalExecutions)

	if success {
		s.metrics.SuccessRate = ((s.metrics.SuccessRate * (total - 1)) + 1) / total
	} else {
		s.metrics.SuccessRate = (s.metrics.SuccessRate * (total - 1)) / total
	}

	s.metrics.AverageSteps = ((s.metrics.AverageSteps * (total - 1)) + float64(steps)) / total
	avgDuration := (time.Duration(float64(s.metrics.AverageDuration)*(total-1)) + duration) / time.Duration(s.metrics.TotalExecutions)
	s.metrics.AverageDuration = avgDuration
}

func (s *BaseSwarm) storeNextTarget(ctx runtime.Context, target string) {
	ctx.SetStateValue(swarmContextKeyNextTarget, target)
	if target != "" {
		ctx.SetStateValue(handoffNextTargetKey, coordination.Target{Name: target, Type: coordination.TargetAgent})
	} else {
		ctx.SetStateValue(handoffNextTargetKey, nil)
	}
}

func (s *BaseSwarm) clearNextTarget(ctx runtime.Context) {
	ctx.SetStateValue(swarmContextKeyNextTarget, "")
	ctx.SetStateValue(handoffNextTargetKey, nil)
}

func (s *BaseSwarm) nextTargetFromContext(ctx runtime.Context) string {
	if value := ctx.GetStateValue(swarmContextKeyNextTarget); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (s *BaseSwarm) defaultPlanner() coordination.Planner {
	return coordination.PlannerFunc(func(ctx runtime.Context, req coordination.Request) (*coordination.Plan, error) {
		plan := coordination.NewPlan()
		var prev *coordination.Step
		for i := 0; i < s.maxSteps; i++ {
			step := &coordination.Step{
				ID:   fmt.Sprintf("swarm-step-%d", i),
				Name: fmt.Sprintf("Swarm iteration %d", i+1),
			}
			plan.AddStep(step)
			if prev != nil {
				prev.Next = append(prev.Next, step.ID)
			}
			prev = step
		}
		return plan, nil
	})
}

func (s *BaseSwarm) defaultRouter() coordination.Router {
	return coordination.RouterFunc(func(ctx runtime.Context, req coordination.Request, plan *coordination.Plan, step *coordination.Step) (coordination.Target, error) {
		if step == nil {
			return coordination.Target{}, fmt.Errorf("swarm: step is nil")
		}

		// Prefer handoff targets already stored in context.
		if target := s.nextTargetFromContext(ctx); target != "" {
			if ag := s.findAgent(target); ag != nil {
				return coordination.Target{Name: ag.ID(), Type: coordination.TargetAgent}, nil
			}
		}

		// Use explicit request target when provided.
		if req.Target != "" {
			if ag := s.findAgent(req.Target); ag != nil {
				return coordination.Target{Name: ag.ID(), Type: coordination.TargetAgent}, nil
			}
		}

		// Fallback to round-robin selection.
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.agents) == 0 {
			return coordination.Target{}, schema.NewValidationError("agents", nil, "no agents in swarm")
		}
		s.lastIndex = (s.lastIndex + 1) % len(s.agents)
		ag := s.agents[s.lastIndex]
		return coordination.Target{Name: ag.ID(), Type: coordination.TargetAgent}, nil
	})
}
