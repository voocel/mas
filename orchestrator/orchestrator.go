package orchestrator

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/coordination"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/workflows"
)

// Option customizes orchestrator construction.
type Option func(*BaseOrchestrator)

// WithPlanner injects a custom Planner implementation.
func WithPlanner(planner coordination.Planner) Option {
	return func(o *BaseOrchestrator) {
		o.planner = planner
	}
}

// WithRouter injects a custom Router implementation.
func WithRouter(router coordination.Router) Option {
	return func(o *BaseOrchestrator) {
		o.router = router
	}
}

// Orchestrator exposes the orchestration capabilities.
type Orchestrator interface {
	AddAgent(name string, ag agent.Agent) error
	AddWorkflow(name string, workflow workflows.Workflow) error
	Execute(ctx runtime.Context, request ExecuteRequest) (ExecuteResponse, error)
	ExecuteStream(ctx runtime.Context, request ExecuteRequest) (<-chan schema.StreamEvent, error)
	GetAgent(name string) (agent.Agent, bool)
	ListAgents() []string
	RemoveAgent(name string) error
}

// ExecuteRequest captures the input for a single orchestration execution.
type ExecuteRequest struct {
	Input      schema.Message         `json:"input"`
	Target     string                 `json:"target"`
	Type       ExecuteType            `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ExecuteResponse returns the final output along with execution trace information.
type ExecuteResponse struct {
	Output   schema.Message         `json:"output"`
	Source   string                 `json:"source"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Trace    []ExecutionStep        `json:"trace"`
}

// ExecuteType declares how the orchestrator should interpret the request.
type ExecuteType string

const (
	ExecuteTypeAgent    ExecuteType = "agent"
	ExecuteTypeWorkflow ExecuteType = "workflow"
	ExecuteTypeAuto     ExecuteType = "auto"
)

const handoffNextTargetKey = schema.HandoffNextTargetStateKey

// ExecutionStep records key data for each executed step.
type ExecutionStep struct {
	StepID   string                 `json:"step_id"`
	StepName string                 `json:"step_name"`
	Agent    string                 `json:"agent,omitempty"`
	Target   coordination.Target    `json:"target"`
	Input    schema.Message         `json:"input"`
	Output   schema.Message         `json:"output"`
	Duration int64                  `json:"duration"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BaseOrchestrator implements Orchestrator using Planner/Router abstractions.
type BaseOrchestrator struct {
	agents    map[string]agent.Agent
	workflows map[string]workflows.Workflow

	planner coordination.Planner
	router  coordination.Router

	mu sync.RWMutex
}

// NewOrchestrator constructs an orchestrator with optional overrides.
func NewOrchestrator(options ...Option) *BaseOrchestrator {
	orch := &BaseOrchestrator{
		agents:    make(map[string]agent.Agent),
		workflows: make(map[string]workflows.Workflow),
	}
	for _, opt := range options {
		if opt != nil {
			opt(orch)
		}
	}
	if orch.planner == nil {
		orch.planner = orch.defaultPlanner()
	}
	if orch.router == nil {
		orch.router = orch.defaultRouter()
	}
	return orch
}

// AddAgent registers an agent instance.
func (o *BaseOrchestrator) AddAgent(name string, ag agent.Agent) error {
	if name == "" {
		return schema.NewValidationError("name", name, "agent name cannot be empty")
	}
	if ag == nil {
		return schema.NewValidationError("agent", ag, "agent cannot be nil")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.agents[name]; exists {
		return schema.NewAgentError(name, "add", schema.ErrAgentAlreadyExists)
	}
	o.agents[name] = ag
	return nil
}

// AddWorkflow registers a workflow instance.
func (o *BaseOrchestrator) AddWorkflow(name string, workflow workflows.Workflow) error {
	if name == "" {
		return schema.NewValidationError("name", name, "workflow name cannot be empty")
	}
	if workflow == nil {
		return schema.NewValidationError("workflow", workflow, "workflow cannot be nil")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.workflows[name]; exists {
		return schema.NewWorkflowError(name, "add", schema.ErrWorkflowNotFound)
	}
	o.workflows[name] = workflow
	return nil
}

// GetAgent retrieves a registered agent by name.
func (o *BaseOrchestrator) GetAgent(name string) (agent.Agent, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	ag, ok := o.agents[name]
	return ag, ok
}

// ListAgents returns all registered agent names.
func (o *BaseOrchestrator) ListAgents() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	names := make([]string, 0, len(o.agents))
	for name := range o.agents {
		names = append(names, name)
	}
	return names
}

// RemoveAgent unregisters an agent.
func (o *BaseOrchestrator) RemoveAgent(name string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.agents[name]; !exists {
		return schema.NewAgentError(name, "remove", schema.ErrAgentNotFound)
	}
	delete(o.agents, name)
	return nil
}

// Execute runs the orchestration pipeline using the configured planner and router.
func (o *BaseOrchestrator) Execute(ctx runtime.Context, request ExecuteRequest) (ExecuteResponse, error) {
	coordReq := coordination.Request{
		Input:      request.Input,
		Target:     request.Target,
		Type:       coordination.RequestType(request.Type),
		Parameters: request.Parameters,
		Metadata:   request.Metadata,
	}

	plan, err := o.planner.Plan(ctx, coordReq)
	if err != nil {
		return ExecuteResponse{}, err
	}
	if err := plan.Validate(); err != nil {
		return ExecuteResponse{}, err
	}

	runner := executionRunner{
		orchestrator: o,
		router:       o.router,
		request:      request,
		coordReq:     coordReq,
	}
	return runner.run(ctx, plan)
}

// ExecuteStream handles streaming execution; currently limited to single-step plans.
func (o *BaseOrchestrator) ExecuteStream(ctx runtime.Context, request ExecuteRequest) (<-chan schema.StreamEvent, error) {
	switch request.Type {
	case ExecuteTypeAgent:
		ag, ok := o.GetAgent(request.Target)
		if !ok {
			return nil, schema.NewAgentError(request.Target, "execute_stream", schema.ErrAgentNotFound)
		}
		return ag.ExecuteStream(ctx, request.Input)
	case ExecuteTypeWorkflow:
		o.mu.RLock()
		workflow, exists := o.workflows[request.Target]
		o.mu.RUnlock()
		if !exists {
			return nil, schema.NewWorkflowError(request.Target, "execute_stream", schema.ErrWorkflowNotFound)
		}
		return workflow.ExecuteStream(ctx, request.Input)
	case ExecuteTypeAuto:
		coordReq := coordination.Request{
			Input:      request.Input,
			Target:     request.Target,
			Type:       coordination.RequestType(request.Type),
			Parameters: request.Parameters,
			Metadata:   request.Metadata,
		}
		plan, err := o.planner.Plan(ctx, coordReq)
		if err != nil {
			return nil, err
		}
		steps, err := plan.OrderedSteps()
		if err != nil {
			return nil, err
		}
		if len(steps) != 1 {
			return nil, fmt.Errorf("orchestrator: streaming execution only supports single-step plans")
		}
		target, err := o.router.Route(ctx, coordReq, plan, steps[0])
		if err != nil {
			return nil, err
		}
		if err := coordination.EnsureTarget(target); err != nil {
			return nil, err
		}
		switch target.Type {
		case coordination.TargetAgent:
			ag, ok := o.GetAgent(target.Name)
			if !ok {
				return nil, schema.NewAgentError(target.Name, "execute_stream", schema.ErrAgentNotFound)
			}
			return ag.ExecuteStream(ctx, request.Input)
		case coordination.TargetWorkflow:
			o.mu.RLock()
			workflow, exists := o.workflows[target.Name]
			o.mu.RUnlock()
			if !exists {
				return nil, schema.NewWorkflowError(target.Name, "execute_stream", schema.ErrWorkflowNotFound)
			}
			return workflow.ExecuteStream(ctx, request.Input)
		default:
			return nil, fmt.Errorf("orchestrator: unsupported target type %s", target.Type)
		}
	default:
		return nil, schema.NewValidationError("type", request.Type, "unsupported execute type")
	}
}

// executionRunner ties together plan traversal, routing, and execution.
type executionRunner struct {
	orchestrator *BaseOrchestrator
	router       coordination.Router
	request      ExecuteRequest
	coordReq     coordination.Request
}

func (r executionRunner) run(ctx runtime.Context, plan *coordination.Plan) (ExecuteResponse, error) {
	steps, err := plan.OrderedSteps()
	if err != nil {
		return ExecuteResponse{}, err
	}

	current := r.request.Input
	trace := make([]ExecutionStep, 0, len(steps))
	var lastTarget coordination.Target
	r.clearNextHandoff(ctx)

	for _, step := range steps {
		target, routeErr := r.router.Route(ctx, r.coordReq, plan, step)
		if routeErr != nil {
			return ExecuteResponse{}, routeErr
		}
		if err := coordination.EnsureTarget(target); err != nil {
			return ExecuteResponse{}, err
		}

		start := time.Now()
		output, stepHandoff, execErr := r.executeStep(ctx, target, current)
		duration := time.Since(start).Milliseconds()

		stepTrace := ExecutionStep{
			StepID:   step.ID,
			StepName: step.Name,
			Target:   target,
			Input:    current,
			Duration: duration,
			Metadata: step.Metadata,
		}

		if execErr != nil {
			stepTrace.Error = execErr.Error()
			trace = append(trace, stepTrace)
			r.clearNextHandoff(ctx)
			return ExecuteResponse{}, execErr
		}

		stepTrace.Output = output
		if target.Type == coordination.TargetAgent {
			stepTrace.Agent = target.Name
		}
		if stepHandoff != nil && stepHandoff.Target != "" {
			if stepTrace.Metadata == nil {
				stepTrace.Metadata = make(map[string]interface{})
			}
			stepTrace.Metadata["handoff_target"] = stepHandoff.Target
			if reason, ok := stepHandoff.GetContext("reason"); ok {
				stepTrace.Metadata["handoff_reason"] = reason
			}
			if stepHandoff.Priority != 0 {
				stepTrace.Metadata["handoff_priority"] = stepHandoff.Priority
			}
		}
		trace = append(trace, stepTrace)

		current = output
		lastTarget = target
		r.applyHandoff(ctx, stepHandoff)
	}

	response := ExecuteResponse{
		Output: current,
		Trace:  trace,
	}
	if lastTarget.Name != "" {
		response.Source = lastTarget.Name
	}
	return response, nil
}

func (r executionRunner) executeStep(ctx runtime.Context, target coordination.Target, input schema.Message) (schema.Message, *schema.Handoff, error) {
	switch target.Type {
	case coordination.TargetAgent:
		ag, ok := r.orchestrator.GetAgent(target.Name)
		if !ok {
			return schema.Message{}, nil, schema.NewAgentError(target.Name, "execute", schema.ErrAgentNotFound)
		}
		return ag.ExecuteWithHandoff(ctx, input)
	case coordination.TargetWorkflow:
		r.orchestrator.mu.RLock()
		workflow, exists := r.orchestrator.workflows[target.Name]
		r.orchestrator.mu.RUnlock()
		if !exists {
			return schema.Message{}, nil, schema.NewWorkflowError(target.Name, "execute", schema.ErrWorkflowNotFound)
		}
		msg, err := workflow.Execute(ctx, input)
		return msg, nil, err
	default:
		return schema.Message{}, nil, fmt.Errorf("orchestrator: unsupported target type %s", target.Type)
	}
}

// defaultPlanner builds the fallback planner compatible with legacy behaviour.
func (o *BaseOrchestrator) defaultPlanner() coordination.Planner {
	return coordination.PlannerFunc(func(ctx runtime.Context, req coordination.Request) (*coordination.Plan, error) {
		switch req.Type {
		case coordination.RequestTypeAgent:
			if req.Target == "" {
				return nil, schema.NewValidationError("target", req.Target, "agent target cannot be empty")
			}
			return coordination.SingleStep("agent:"+req.Target, coordination.Target{
				Name: req.Target,
				Type: coordination.TargetAgent,
			}), nil
		case coordination.RequestTypeWorkflow:
			if req.Target == "" {
				return nil, schema.NewValidationError("target", req.Target, "workflow target cannot be empty")
			}
			return coordination.SingleStep("workflow:"+req.Target, coordination.Target{
				Name: req.Target,
				Type: coordination.TargetWorkflow,
			}), nil
		case coordination.RequestTypeAuto:
			return o.buildAutoPlan(req)
		default:
			return nil, schema.NewValidationError("type", req.Type, "unsupported execute type")
		}
	})
}

func (o *BaseOrchestrator) buildAutoPlan(req coordination.Request) (*coordination.Plan, error) {
	o.mu.RLock()
	agentRegistry := make(map[string]agent.Agent, len(o.agents))
	agentNames := make([]string, 0, len(o.agents))
	for name, ag := range o.agents {
		agentRegistry[name] = ag
		agentNames = append(agentNames, name)
	}
	workflowNames := make([]string, 0, len(o.workflows))
	for name := range o.workflows {
		workflowNames = append(workflowNames, name)
	}
	o.mu.RUnlock()

	sort.Strings(agentNames)
	sort.Strings(workflowNames)

	agentTargets := makeTargets(agentNames, coordination.TargetAgent)
	workflowTargets := makeTargets(workflowNames, coordination.TargetWorkflow)

	if req.Target != "" {
		if _, ok := agentRegistry[req.Target]; ok {
			return coordination.SingleStep("agent:"+req.Target, coordination.Target{Name: req.Target, Type: coordination.TargetAgent}), nil
		}
		for _, name := range workflowNames {
			if name == req.Target {
				return coordination.SingleStep("workflow:"+req.Target, coordination.Target{Name: req.Target, Type: coordination.TargetWorkflow}), nil
			}
		}
		return nil, schema.NewValidationError("target", req.Target, "target not found")
	}

	if len(agentTargets)+len(workflowTargets) == 0 {
		return nil, schema.NewValidationError("target", req.Target, "no agents or workflows available")
	}

	if len(agentTargets)+len(workflowTargets) == 1 {
		target := agentTargets
		if len(target) == 0 {
			target = workflowTargets
		}
		return coordination.SingleStep("auto-single", target[0]), nil
	}

	plan := coordination.NewPlan()

	analysisStep := &coordination.Step{
		ID:          "auto-analyze",
		Name:        "分析需求",
		Description: "理解用户意图与上下文",
		Metadata: map[string]interface{}{
			"phase":                "analyze",
			"preferred_capability": string(agent.CapabilityAnalysis),
		},
		Candidates: filterTargetsByCapability(agentNames, agentRegistry, []agent.Capability{
			agent.CapabilityAnalysis,
			agent.CapabilityResearch,
			agent.CapabilityReasoning,
		}),
	}
	if len(analysisStep.Candidates) == 0 {
		analysisStep.Candidates = append([]coordination.Target(nil), agentTargets...)
	}
	ensureDefaultTarget(analysisStep, agentTargets)
	plan.AddStep(analysisStep)

	executionStep := &coordination.Step{
		ID:          "auto-execute",
		Name:        "执行方案",
		Description: "调用最合适的智能体或工作流完成任务",
		Metadata: map[string]interface{}{
			"phase":                "execute",
			"preferred_capability": string(agent.CapabilityEngineering),
		},
	}
	actAgents := filterTargetsByCapability(agentNames, agentRegistry, []agent.Capability{
		agent.CapabilityEngineering,
		agent.CapabilityWriting,
		agent.CapabilityDesign,
		agent.CapabilityReasoning,
	})
	executionStep.Candidates = appendTargetsUnique([]coordination.Target(nil), actAgents)
	executionStep.Candidates = appendTargetsUnique(executionStep.Candidates, workflowTargets)
	if len(executionStep.Candidates) == 0 {
		executionStep.Candidates = appendTargetsUnique(append([]coordination.Target(nil), agentTargets...), workflowTargets)
	}
	fallbackTargets := appendTargetsUnique(append([]coordination.Target(nil), agentTargets...), workflowTargets)
	ensureDefaultTarget(executionStep, fallbackTargets)
	analysisStep.Next = append(analysisStep.Next, executionStep.ID)
	plan.AddStep(executionStep)

	reviewCandidates := filterTargetsByCapability(agentNames, agentRegistry, []agent.Capability{
		agent.CapabilityManagement,
		agent.CapabilityPlanning,
		agent.CapabilityAnalysis,
	})
	if len(reviewCandidates) > 0 {
		reviewStep := &coordination.Step{
			ID:          "auto-review",
			Name:        "复盘总结",
			Description: "校验结果并总结要点",
			Metadata: map[string]interface{}{
				"phase":                "review",
				"preferred_capability": string(agent.CapabilityPlanning),
			},
			Candidates: appendTargetsUnique([]coordination.Target(nil), reviewCandidates),
		}
		ensureDefaultTarget(reviewStep, agentTargets)
		executionStep.Next = append(executionStep.Next, reviewStep.ID)
		plan.AddStep(reviewStep)
	}

	return plan, nil
}

// defaultRouter supplies a minimal routing strategy.
func (o *BaseOrchestrator) defaultRouter() coordination.Router {
	return coordination.RouterFunc(func(ctx runtime.Context, req coordination.Request, plan *coordination.Plan, step *coordination.Step) (coordination.Target, error) {
		if step == nil {
			return coordination.Target{}, fmt.Errorf("orchestrator: step is nil")
		}
		if value := ctx.GetStateValue(handoffNextTargetKey); value != nil {
			if target, ok := asCoordinationTarget(value); ok && target.Name != "" {
				ctx.SetStateValue(handoffNextTargetKey, nil)
				return target, nil
			}
		}
		target := coordination.MustTarget(step)
		if target.Name == "" {
			return coordination.Target{}, coordination.ErrNoTarget
		}
		return target, nil
	})
}

func (r executionRunner) applyHandoff(ctx runtime.Context, handoff *schema.Handoff) {
	if handoff == nil || handoff.Target == "" {
		r.clearNextHandoff(ctx)
		ctx.SetStateValue(schema.HandoffPendingStateKey, nil)
		return
	}

	target, ok := r.orchestrator.resolveCoordinationTarget(handoff.Target)
	if !ok {
		r.clearNextHandoff(ctx)
		ctx.SetStateValue(schema.HandoffPendingStateKey, nil)
		return
	}

	ctx.SetStateValue(handoffNextTargetKey, target)
	ctx.SetStateValue(schema.HandoffPendingStateKey, handoff)
}

func (r executionRunner) clearNextHandoff(ctx runtime.Context) {
	ctx.SetStateValue(handoffNextTargetKey, nil)
}

func makeTargets(names []string, targetType coordination.TargetType) []coordination.Target {
	targets := make([]coordination.Target, 0, len(names))
	for _, name := range names {
		targets = append(targets, coordination.Target{Name: name, Type: targetType})
	}
	return targets
}

func (o *BaseOrchestrator) resolveCoordinationTarget(name string) (coordination.Target, bool) {
	if name == "" {
		return coordination.Target{}, false
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	if _, ok := o.agents[name]; ok {
		return coordination.Target{Name: name, Type: coordination.TargetAgent}, true
	}
	for key, ag := range o.agents {
		if ag.ID() == name {
			return coordination.Target{Name: key, Type: coordination.TargetAgent}, true
		}
	}
	if _, ok := o.workflows[name]; ok {
		return coordination.Target{Name: name, Type: coordination.TargetWorkflow}, true
	}
	return coordination.Target{}, false
}

func asCoordinationTarget(value interface{}) (coordination.Target, bool) {
	switch v := value.(type) {
	case coordination.Target:
		if v.Name != "" {
			return v, true
		}
	case *coordination.Target:
		if v != nil && v.Name != "" {
			return *v, true
		}
	}
	return coordination.Target{}, false
}

func filterTargetsByCapability(agentNames []string, agents map[string]agent.Agent, capabilities []agent.Capability) []coordination.Target {
	if len(capabilities) == 0 {
		return nil
	}
	targets := make([]coordination.Target, 0, len(agentNames))
	for _, name := range agentNames {
		ag := agents[name]
		if ag == nil {
			continue
		}
		if cap, ok := firstCapabilityMatch(ag, capabilities); ok {
			target := coordination.Target{Name: name, Type: coordination.TargetAgent}
			target.Metadata = map[string]interface{}{"match_capability": string(cap)}
			targets = append(targets, target)
		}
	}
	return targets
}

func firstCapabilityMatch(ag agent.Agent, capabilities []agent.Capability) (agent.Capability, bool) {
	for _, cap := range capabilities {
		if agentHasCapability(ag, cap) {
			return cap, true
		}
	}
	return "", false
}

func agentHasCapability(ag agent.Agent, capability agent.Capability) bool {
	if capability == "" {
		return true
	}
	for _, cap := range ag.Capabilities() {
		if cap == capability {
			return true
		}
	}
	if details := ag.GetCapabilities(); details != nil {
		for _, cap := range details.CoreCapabilities {
			if cap == capability {
				return true
			}
		}
		for _, tag := range details.CustomTags {
			if tag == string(capability) {
				return true
			}
		}
	}
	return false
}

func ensureDefaultTarget(step *coordination.Step, fallback []coordination.Target) {
	if step == nil || step.DefaultTarget != nil {
		return
	}
	if len(step.Candidates) > 0 {
		target := step.Candidates[0]
		step.DefaultTarget = &target
		return
	}
	if len(fallback) > 0 {
		target := fallback[0]
		step.DefaultTarget = &target
	}
}

func appendTargetsUnique(base []coordination.Target, additions []coordination.Target) []coordination.Target {
	if len(additions) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base))
	for _, target := range base {
		seen[targetKey(target)] = struct{}{}
	}
	for _, target := range additions {
		key := targetKey(target)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		base = append(base, target)
	}
	return base
}

func targetKey(target coordination.Target) string {
	return string(target.Type) + ":" + target.Name
}
