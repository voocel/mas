package coordination

import (
	"fmt"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// RequestType enumerates coordination request types used during planning and routing.
type RequestType string

const (
	RequestTypeAgent    RequestType = "agent"
	RequestTypeWorkflow RequestType = "workflow"
	RequestTypeAuto     RequestType = "auto"
)

// TargetType describes the type of execution target.
type TargetType string

const (
	TargetAgent    TargetType = "agent"
	TargetWorkflow TargetType = "workflow"
)

// Request captures caller intent as the input to planning.
type Request struct {
	Input      schema.Message         `json:"input"`
	Target     string                 `json:"target"`
	Type       RequestType            `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Target describes a concrete execution target.
type Target struct {
	Name       string                 `json:"name"`
	Type       TargetType             `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Step represents a single node within a plan.
type Step struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Candidates    []Target               `json:"candidates,omitempty"`
	DefaultTarget *Target                `json:"default_target,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Next          []string               `json:"next,omitempty"`
}

// Plan stores the whole execution blueprint, potentially linear or graph-based.
type Plan struct {
	Steps    map[string]*Step       `json:"steps"`
	Entry    []string               `json:"entry"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewPlan creates an empty plan instance.
func NewPlan() *Plan {
	return &Plan{
		Steps: make(map[string]*Step),
		Entry: make([]string, 0),
	}
}

// AddStep registers a step into the plan.
func (p *Plan) AddStep(step *Step) {
	if step == nil || step.ID == "" {
		return
	}
	if p.Steps == nil {
		p.Steps = make(map[string]*Step)
	}
	p.Steps[step.ID] = step
	if len(p.Entry) == 0 {
		p.Entry = append(p.Entry, step.ID)
	}
}

// Clone produces a deep copy of the plan for branching scenarios.
func (p *Plan) Clone() *Plan {
	if p == nil {
		return nil
	}
	clone := &Plan{
		Steps:    make(map[string]*Step, len(p.Steps)),
		Entry:    append([]string(nil), p.Entry...),
		Metadata: cloneMap(p.Metadata),
	}
	for id, step := range p.Steps {
		clone.Steps[id] = step.Clone()
	}
	return clone
}

// OrderedSteps returns steps in topological order for acyclic plans.
func (p *Plan) OrderedSteps() ([]*Step, error) {
	if p == nil {
		return nil, fmt.Errorf("coordination: plan is nil")
	}
	inDegree := make(map[string]int)
	for id := range p.Steps {
		inDegree[id] = 0
	}
	for _, step := range p.Steps {
		for _, next := range step.Next {
			inDegree[next]++
		}
	}
	queue := make([]string, 0)
	for _, id := range p.Entry {
		if _, ok := inDegree[id]; ok {
			queue = append(queue, id)
		}
	}
	ordered := make([]*Step, 0, len(p.Steps))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		step, ok := p.Steps[id]
		if !ok {
			return nil, fmt.Errorf("coordination: step %s not found", id)
		}
		ordered = append(ordered, step)
		for _, next := range step.Next {
			if deg, ok := inDegree[next]; ok {
				inDegree[next] = deg - 1
				if inDegree[next] == 0 {
					queue = append(queue, next)
				}
			}
		}
	}
	if len(ordered) != len(p.Steps) {
		return nil, fmt.Errorf("coordination: plan contains cycle or disconnected steps")
	}
	return ordered, nil
}

// Validate performs basic integrity checks on the plan definition.
func (p *Plan) Validate() error {
	if p == nil {
		return fmt.Errorf("coordination: plan is nil")
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("coordination: plan has no steps")
	}
	if len(p.Entry) == 0 {
		return fmt.Errorf("coordination: plan has no entry")
	}
	for _, id := range p.Entry {
		if _, ok := p.Steps[id]; !ok {
			return fmt.Errorf("coordination: entry step %s missing", id)
		}
	}
	return nil
}

// Clone creates a deep copy of the step.
func (s *Step) Clone() *Step {
	if s == nil {
		return nil
	}
	clone := &Step{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Metadata:    cloneMap(s.Metadata),
		Next:        append([]string(nil), s.Next...),
	}
	if len(s.Candidates) > 0 {
		clone.Candidates = make([]Target, len(s.Candidates))
		copy(clone.Candidates, s.Candidates)
	}
	if s.DefaultTarget != nil {
		target := *s.DefaultTarget
		target.Metadata = cloneMap(target.Metadata)
		target.Parameters = cloneMap(target.Parameters)
		clone.DefaultTarget = &target
	}
	return clone
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// Planner defines how to produce a plan from an execution request.
type Planner interface {
	Plan(ctx runtime.Context, req Request) (*Plan, error)
}

// PlannerFunc allows plain functions to satisfy the Planner interface.
type PlannerFunc func(ctx runtime.Context, req Request) (*Plan, error)

func (f PlannerFunc) Plan(ctx runtime.Context, req Request) (*Plan, error) {
	return f(ctx, req)
}

// Router selects the concrete target for a given plan step.
type Router interface {
	Route(ctx runtime.Context, req Request, plan *Plan, step *Step) (Target, error)
}

// RouterFunc allows plain functions to satisfy the Router interface.
type RouterFunc func(ctx runtime.Context, req Request, plan *Plan, step *Step) (Target, error)

func (f RouterFunc) Route(ctx runtime.Context, req Request, plan *Plan, step *Step) (Target, error) {
	return f(ctx, req, plan, step)
}

// BuildLinearPlan builds a simple linear plan from the provided steps.
func BuildLinearPlan(steps ...*Step) *Plan {
	plan := NewPlan()
	var prev *Step
	for _, step := range steps {
		if step == nil {
			continue
		}
		plan.AddStep(step)
		if prev != nil {
			prev.Next = append(prev.Next, step.ID)
		}
		prev = step
	}
	return plan
}

// SingleStep returns a plan consisting of a single step.
func SingleStep(id string, target Target) *Plan {
	step := &Step{
		ID:            id,
		Name:          id,
		DefaultTarget: &target,
	}
	return BuildLinearPlan(step)
}

// MustTarget returns the fallback target for a step.
func MustTarget(step *Step) Target {
	if step == nil {
		return Target{}
	}
	if step.DefaultTarget != nil {
		return *step.DefaultTarget
	}
	if len(step.Candidates) > 0 {
		return step.Candidates[0]
	}
	return Target{}
}

// ErrNoTarget is returned when no executable target exists for a step.
var ErrNoTarget = fmt.Errorf("coordination: no target available for step")

// EnsureTarget validates that the target can be executed.
func EnsureTarget(target Target) error {
	if target.Name == "" {
		return ErrNoTarget
	}
	if target.Type != TargetAgent && target.Type != TargetWorkflow {
		return fmt.Errorf("coordination: unsupported target type %s", target.Type)
	}
	return nil
}
