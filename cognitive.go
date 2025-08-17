package mas

import (
	"context"
	"fmt"
	"time"
)

// CognitiveLayers represents different cognitive processing levels
type CognitiveLayer int

const (
	ReflexLayer     CognitiveLayer = iota // Reflex layer: immediate responses
	CerebellumLayer                       // Cerebellum layer: skill execution
	CortexLayer                           // Cortex layer: reasoning and decision-making
	MetaLayer                             // Meta layer: planning and monitoring
)

// Plan represents a high-level execution plan
type Plan struct {
	ID                string                 `json:"id"`
	Goal              string                 `json:"goal"`
	Steps             []PlanStep             `json:"steps"`
	Context           map[string]interface{} `json:"context"`
	CreatedAt         time.Time              `json:"created_at"`
	EstimatedDuration time.Duration          `json:"estimated_duration"`
}

// PlanStep represents a single step in a plan
type PlanStep struct {
	ID           string                 `json:"id"`
	Action       string                 `json:"action"`
	Layer        CognitiveLayer         `json:"layer"`
	Parameters   map[string]interface{} `json:"parameters"`
	Dependencies []string               `json:"dependencies"`
	Completed    bool                   `json:"completed"`
}

// Situation represents current contextual information
type Situation struct {
	Context     map[string]interface{} `json:"context"`
	Inputs      []string               `json:"inputs"`
	Constraints []string               `json:"constraints"`
	Goals       []string               `json:"goals"`
	Timestamp   time.Time              `json:"timestamp"`
}

// Decision represents the result of reasoning
type Decision struct {
	Action     string                 `json:"action"`
	Layer      CognitiveLayer         `json:"layer"`
	Confidence float64                `json:"confidence"`
	Reasoning  string                 `json:"reasoning"`
	Parameters map[string]interface{} `json:"parameters"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Stimulus represents input that triggers reactive behavior
type Stimulus struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Urgency   float64                `json:"urgency"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
}

// Action represents an action to be executed
type Action struct {
	Type       string                 `json:"type"`
	Layer      CognitiveLayer         `json:"layer"`
	Parameters map[string]interface{} `json:"parameters"`
	Priority   int                    `json:"priority"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Skill represents a reusable capability
type Skill interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	Layer() CognitiveLayer
	Prerequisites() []string
}

// SkillLibrary manages available skills
type SkillLibrary interface {
	RegisterSkill(skill Skill) error
	GetSkill(name string) (Skill, error)
	ListSkills() []Skill
	FindSkillsByLayer(layer CognitiveLayer) []Skill
}

// Note: CognitiveAgent capabilities are now integrated directly into the Agent interface

// CognitiveMode controls how the agent processes information
type CognitiveMode int

const (
	AutomaticMode CognitiveMode = iota // Automatically select the most suitable layer
	ReflexMode                         // Only use reflex layer
	SkillMode                          // Prioritize skill layer
	ReasoningMode                      // Prioritize reasoning layer
	PlanningMode                       // Prioritize planning layer
)

// CognitiveState represents the current cognitive state
type CognitiveState struct {
	CurrentLayer    CognitiveLayer `json:"current_layer"`
	Mode            CognitiveMode  `json:"mode"`
	ActivePlan      *Plan          `json:"active_plan,omitempty"`
	LoadedSkills    []string       `json:"loaded_skills"`
	RecentDecisions []*Decision    `json:"recent_decisions"`
	LastUpdate      time.Time      `json:"last_update"`
}

// skillLibrary implements SkillLibrary
type skillLibrary struct {
	skills map[string]Skill
}

// NewSkillLibrary creates a new skill library
func NewSkillLibrary() SkillLibrary {
	return &skillLibrary{
		skills: make(map[string]Skill),
	}
}

func (sl *skillLibrary) RegisterSkill(skill Skill) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}

	name := skill.Name()
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	sl.skills[name] = skill
	return nil
}

func (sl *skillLibrary) GetSkill(name string) (Skill, error) {
	skill, exists := sl.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}
	return skill, nil
}

func (sl *skillLibrary) ListSkills() []Skill {
	skills := make([]Skill, 0, len(sl.skills))
	for _, skill := range sl.skills {
		skills = append(skills, skill)
	}
	return skills
}

func (sl *skillLibrary) FindSkillsByLayer(layer CognitiveLayer) []Skill {
	var skills []Skill
	for _, skill := range sl.skills {
		if skill.Layer() == layer {
			skills = append(skills, skill)
		}
	}
	return skills
}

// basicSkill provides a simple skill implementation
type basicSkill struct {
	name          string
	description   string
	layer         CognitiveLayer
	prerequisites []string
	executor      func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// NewSkill creates a new skill
func NewSkill(name, description string, layer CognitiveLayer, executor func(ctx context.Context, params map[string]interface{}) (interface{}, error)) Skill {
	return &basicSkill{
		name:          name,
		description:   description,
		layer:         layer,
		prerequisites: make([]string, 0),
		executor:      executor,
	}
}

func (s *basicSkill) Name() string {
	return s.name
}

func (s *basicSkill) Description() string {
	return s.description
}

func (s *basicSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return s.executor(ctx, params)
}

func (s *basicSkill) Layer() CognitiveLayer {
	return s.layer
}

func (s *basicSkill) Prerequisites() []string {
	return s.prerequisites
}

// Utility functions

// NewPlan creates a new plan
func NewPlan(goal string) *Plan {
	return &Plan{
		ID:        generatePlanID(),
		Goal:      goal,
		Steps:     make([]PlanStep, 0),
		Context:   make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
}

// NewSituation creates a new situation
func NewSituation(context map[string]interface{}, inputs []string) *Situation {
	return &Situation{
		Context:     context,
		Inputs:      inputs,
		Constraints: make([]string, 0),
		Goals:       make([]string, 0),
		Timestamp:   time.Now(),
	}
}

// NewStimulus creates a new stimulus
func NewStimulus(stimulusType string, data map[string]interface{}, urgency float64) *Stimulus {
	return &Stimulus{
		Type:      stimulusType,
		Data:      data,
		Urgency:   urgency,
		Timestamp: time.Now(),
	}
}

// Helper functions
func generatePlanID() string {
	return fmt.Sprintf("plan_%d", time.Now().UnixNano())
}

// LayerName returns the string name of a cognitive layer
func (cl CognitiveLayer) String() string {
	switch cl {
	case ReflexLayer:
		return "reflex"
	case CerebellumLayer:
		return "cerebellum"
	case CortexLayer:
		return "cortex"
	case MetaLayer:
		return "meta"
	default:
		return "unknown"
	}
}

// ModeName returns the string name of a cognitive mode
func (cm CognitiveMode) String() string {
	switch cm {
	case AutomaticMode:
		return "automatic"
	case ReflexMode:
		return "reflex"
	case SkillMode:
		return "skill"
	case ReasoningMode:
		return "reasoning"
	case PlanningMode:
		return "planning"
	default:
		return "unknown"
	}
}
