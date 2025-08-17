package mas

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GoalPriority represents the priority level of a goal
type GoalPriority int

const (
	LowPriority      GoalPriority = iota // Low priority goal
	MediumPriority                       // Medium priority goal
	HighPriority                         // High priority goal
	CriticalPriority                     // Critical priority goal
)

// GoalStatus represents the current status of a goal
type GoalStatus int

const (
	GoalPending   GoalStatus = iota // Goal is waiting to be started
	GoalActive                      // Goal is currently being pursued
	GoalCompleted                   // Goal has been successfully completed
	GoalFailed                      // Goal has failed
	GoalPaused                      // Goal is temporarily paused
	GoalCancelled                   // Goal has been cancelled
)

// Goal represents an autonomous goal for the agent
type Goal struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Priority     GoalPriority           `json:"priority"`
	Status       GoalStatus             `json:"status"`
	Progress     float64                `json:"progress"` // 0.0 to 1.0
	Deadline     *time.Time             `json:"deadline,omitempty"`
	SubGoals     []*Goal                `json:"sub_goals,omitempty"`
	ParentGoalID string                 `json:"parent_goal_id,omitempty"`
	Context      map[string]interface{} `json:"context"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// AutonomousAction represents an action the agent can take
type AutonomousAction struct {
	ID             string                 `json:"id"`
	GoalID         string                 `json:"goal_id"`
	Type           string                 `json:"type"` // "skill", "plan", "reason", "react"
	Name           string                 `json:"name"`
	Parameters     map[string]interface{} `json:"parameters"`
	Conditions     []string               `json:"conditions"` // Prerequisites for this action
	ExpectedResult string                 `json:"expected_result"`
	Priority       int                    `json:"priority"`
	CreatedAt      time.Time              `json:"created_at"`
}

// ExecutionResult represents the result of an autonomous action
type ExecutionResult struct {
	ActionID    string                 `json:"action_id"`
	GoalID      string                 `json:"goal_id"`
	Success     bool                   `json:"success"`
	Result      interface{}            `json:"result"`
	Error       string                 `json:"error,omitempty"`
	Feedback    string                 `json:"feedback"`
	Progress    float64                `json:"progress"`
	NextActions []*AutonomousAction    `json:"next_actions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	ExecutedAt  time.Time              `json:"executed_at"`
}

// AutonomousStrategy defines how the agent pursues goals
type AutonomousStrategy int

const (
	SequentialStrategy AutonomousStrategy = iota // Execute goals one by one
	ParallelStrategy                             // Execute multiple goals simultaneously
	PriorityStrategy                             // Execute by priority order
	AdaptiveStrategy                             // Dynamically adapt strategy based on context
)

// GoalManager manages the agent's goals and autonomous behavior
type GoalManager interface {
	// Goal management
	AddGoal(ctx context.Context, goal *Goal) error
	GetGoal(goalID string) (*Goal, error)
	UpdateGoal(ctx context.Context, goal *Goal) error
	RemoveGoal(ctx context.Context, goalID string) error
	ListGoals(filter GoalFilter) ([]*Goal, error)

	// Autonomous execution
	StartAutonomousMode(ctx context.Context, strategy AutonomousStrategy) error
	StopAutonomousMode(ctx context.Context) error
	IsAutonomous() bool

	// Progress tracking
	GetProgress(goalID string) (float64, error)
	GetOverallProgress() *ProgressSummary

	// Action planning and execution
	PlanActions(ctx context.Context, goalID string) ([]*AutonomousAction, error)
	ExecuteAction(ctx context.Context, action *AutonomousAction) (*ExecutionResult, error)

	// Learning and adaptation
	RecordResult(result *ExecutionResult) error
	GetLearnings() *LearningInsights
	AdaptStrategy(insights *LearningInsights) error
}

// GoalFilter defines filtering criteria for goals
type GoalFilter struct {
	Status       []GoalStatus   `json:"status,omitempty"`
	Priority     []GoalPriority `json:"priority,omitempty"`
	HasDeadline  bool           `json:"has_deadline,omitempty"`
	ParentGoalID string         `json:"parent_goal_id,omitempty"`
}

// ProgressSummary provides an overview of all goals progress
type ProgressSummary struct {
	TotalGoals                int            `json:"total_goals"`
	CompletedGoals            int            `json:"completed_goals"`
	ActiveGoals               int            `json:"active_goals"`
	FailedGoals               int            `json:"failed_goals"`
	OverallProgress           float64        `json:"overall_progress"`
	EstimatedTimeToCompletion *time.Duration `json:"estimated_completion,omitempty"`
}

// LearningInsights represents learned patterns and improvements
type LearningInsights struct {
	SuccessfulPatterns []ActionPattern     `json:"successful_patterns"`
	FailurePatterns    []ActionPattern     `json:"failure_patterns"`
	OptimalStrategies  map[string]float64  `json:"optimal_strategies"`
	PerformanceMetrics *PerformanceMetrics `json:"performance_metrics"`
	Recommendations    []string            `json:"recommendations"`
	LastUpdated        time.Time           `json:"last_updated"`
}

// ActionPattern represents a learned pattern of actions
type ActionPattern struct {
	Context     map[string]interface{} `json:"context"`
	Actions     []string               `json:"actions"`
	SuccessRate float64                `json:"success_rate"`
	Frequency   int                    `json:"frequency"`
}

// PerformanceMetrics tracks agent performance
type PerformanceMetrics struct {
	AverageCompletionTime time.Duration `json:"average_completion_time"`
	SuccessRate           float64       `json:"success_rate"`
	EfficiencyScore       float64       `json:"efficiency_score"`
	AdaptabilityScore     float64       `json:"adaptability_score"`
}

// basicGoalManager implements GoalManager
type basicGoalManager struct {
	goals      map[string]*Goal
	actions    map[string]*AutonomousAction
	results    []*ExecutionResult
	insights   *LearningInsights
	strategy   AutonomousStrategy
	autonomous bool
	agent      Agent
	mu         sync.RWMutex
	stopChan   chan struct{}
}

// NewGoalManager creates a new goal manager for autonomous behavior
func NewGoalManager(agent Agent) GoalManager {
	return &basicGoalManager{
		goals:   make(map[string]*Goal),
		actions: make(map[string]*AutonomousAction),
		results: make([]*ExecutionResult, 0),
		insights: &LearningInsights{
			SuccessfulPatterns: make([]ActionPattern, 0),
			FailurePatterns:    make([]ActionPattern, 0),
			OptimalStrategies:  make(map[string]float64),
			PerformanceMetrics: &PerformanceMetrics{},
			Recommendations:    make([]string, 0),
			LastUpdated:        time.Now(),
		},
		strategy:   AdaptiveStrategy,
		autonomous: false,
		agent:      agent,
		stopChan:   make(chan struct{}),
	}
}

// AddGoal implements GoalManager.AddGoal
func (gm *basicGoalManager) AddGoal(ctx context.Context, goal *Goal) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if goal.ID == "" {
		goal.ID = generateGoalID()
	}

	goal.CreatedAt = time.Now()
	goal.UpdatedAt = time.Now()
	goal.Status = GoalPending
	goal.Progress = 0.0

	if goal.Context == nil {
		goal.Context = make(map[string]interface{})
	}

	gm.goals[goal.ID] = goal

	// Emit goal creation event
	if gm.agent.GetEventBus() != nil {
		gm.agent.PublishEvent(ctx, EventType("goal.created"), EventData(
			"goal_id", goal.ID,
			"title", goal.Title,
			"priority", goal.Priority,
		))
	}

	return nil
}

// GetGoal implements GoalManager.GetGoal
func (gm *basicGoalManager) GetGoal(goalID string) (*Goal, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	goal, exists := gm.goals[goalID]
	if !exists {
		return nil, fmt.Errorf("goal not found: %s", goalID)
	}

	// Return a copy to prevent external modification
	goalCopy := *goal
	return &goalCopy, nil
}

// UpdateGoal implements GoalManager.UpdateGoal
func (gm *basicGoalManager) UpdateGoal(ctx context.Context, goal *Goal) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if _, exists := gm.goals[goal.ID]; !exists {
		return fmt.Errorf("goal not found: %s", goal.ID)
	}

	goal.UpdatedAt = time.Now()

	// Mark as completed if progress reaches 100%
	if goal.Progress >= 1.0 && goal.Status != GoalCompleted {
		goal.Status = GoalCompleted
		completedAt := time.Now()
		goal.CompletedAt = &completedAt
	}

	gm.goals[goal.ID] = goal

	// Emit goal update event
	if gm.agent.GetEventBus() != nil {
		gm.agent.PublishEvent(ctx, EventType("goal.updated"), EventData(
			"goal_id", goal.ID,
			"status", goal.Status,
			"progress", goal.Progress,
		))
	}

	return nil
}

// RemoveGoal implements GoalManager.RemoveGoal
func (gm *basicGoalManager) RemoveGoal(ctx context.Context, goalID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if _, exists := gm.goals[goalID]; !exists {
		return fmt.Errorf("goal not found: %s", goalID)
	}

	delete(gm.goals, goalID)

	// Emit goal removal event
	if gm.agent.GetEventBus() != nil {
		gm.agent.PublishEvent(ctx, EventType("goal.removed"), EventData(
			"goal_id", goalID,
		))
	}

	return nil
}

// ListGoals implements GoalManager.ListGoals
func (gm *basicGoalManager) ListGoals(filter GoalFilter) ([]*Goal, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	var goals []*Goal

	for _, goal := range gm.goals {
		if gm.matchesFilter(goal, filter) {
			goalCopy := *goal
			goals = append(goals, &goalCopy)
		}
	}

	return goals, nil
}

// matchesFilter checks if a goal matches the given filter
func (gm *basicGoalManager) matchesFilter(goal *Goal, filter GoalFilter) bool {
	// Check status filter
	if len(filter.Status) > 0 {
		matched := false
		for _, status := range filter.Status {
			if goal.Status == status {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check priority filter
	if len(filter.Priority) > 0 {
		matched := false
		for _, priority := range filter.Priority {
			if goal.Priority == priority {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check deadline filter
	if filter.HasDeadline {
		if goal.Deadline == nil {
			return false
		}
	}

	// Check parent goal filter
	if filter.ParentGoalID != "" {
		if goal.ParentGoalID != filter.ParentGoalID {
			return false
		}
	}

	return true
}

// StartAutonomousMode implements GoalManager.StartAutonomousMode
func (gm *basicGoalManager) StartAutonomousMode(ctx context.Context, strategy AutonomousStrategy) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.autonomous {
		return fmt.Errorf("autonomous mode is already active")
	}

	gm.autonomous = true
	gm.strategy = strategy
	gm.stopChan = make(chan struct{})

	// Start autonomous execution loop
	go gm.autonomousLoop(ctx)

	// Emit autonomous mode start event
	if gm.agent.GetEventBus() != nil {
		gm.agent.PublishEvent(ctx, EventType("autonomous.started"), EventData(
			"strategy", strategy,
		))
	}

	return nil
}

// StopAutonomousMode implements GoalManager.StopAutonomousMode
func (gm *basicGoalManager) StopAutonomousMode(ctx context.Context) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if !gm.autonomous {
		return fmt.Errorf("autonomous mode is not active")
	}

	gm.autonomous = false
	close(gm.stopChan)

	// Emit autonomous mode stop event
	if gm.agent.GetEventBus() != nil {
		gm.agent.PublishEvent(ctx, EventType("autonomous.stopped"), EventData())
	}

	return nil
}

// IsAutonomous implements GoalManager.IsAutonomous
func (gm *basicGoalManager) IsAutonomous() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.autonomous
}

// Helper functions

// generateGoalID generates a unique ID for a goal
func generateGoalID() string {
	return fmt.Sprintf("goal_%d", time.Now().UnixNano())
}

// Priority returns string representation of GoalPriority
func (gp GoalPriority) String() string {
	switch gp {
	case LowPriority:
		return "low"
	case MediumPriority:
		return "medium"
	case HighPriority:
		return "high"
	case CriticalPriority:
		return "critical"
	default:
		return "unknown"
	}
}

// Status returns string representation of GoalStatus
func (gs GoalStatus) String() string {
	switch gs {
	case GoalPending:
		return "pending"
	case GoalActive:
		return "active"
	case GoalCompleted:
		return "completed"
	case GoalFailed:
		return "failed"
	case GoalPaused:
		return "paused"
	case GoalCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// autonomousLoop runs the main autonomous execution loop
func (gm *basicGoalManager) autonomousLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-gm.stopChan:
			return
		case <-ticker.C:
			gm.executeAutonomousCycle(ctx)
		}
	}
}

// executeAutonomousCycle executes one cycle of autonomous behavior
func (gm *basicGoalManager) executeAutonomousCycle(ctx context.Context) {
	// Get active and pending goals
	activeGoals, err := gm.ListGoals(GoalFilter{
		Status: []GoalStatus{GoalActive, GoalPending},
	})
	if err != nil {
		return
	}

	if len(activeGoals) == 0 {
		return
	}

	// Select goal based on strategy
	goalToWork := gm.selectGoalByStrategy(activeGoals)
	if goalToWork == nil {
		return
	}

	// Activate goal if pending
	if goalToWork.Status == GoalPending {
		goalToWork.Status = GoalActive
		gm.UpdateGoal(ctx, goalToWork)
	}

	// Plan and execute actions for the goal
	actions, err := gm.PlanActions(ctx, goalToWork.ID)
	if err != nil {
		return
	}

	// Execute the highest priority action
	if len(actions) > 0 {
		result, err := gm.ExecuteAction(ctx, actions[0])
		if err == nil {
			gm.RecordResult(result)
			gm.updateGoalProgress(ctx, goalToWork, result)
		}
	}
}

// selectGoalByStrategy selects the next goal to work on based on strategy
func (gm *basicGoalManager) selectGoalByStrategy(goals []*Goal) *Goal {
	if len(goals) == 0 {
		return nil
	}

	switch gm.strategy {
	case SequentialStrategy:
		// Select oldest pending goal, or first active
		for _, goal := range goals {
			if goal.Status == GoalPending {
				return goal
			}
		}
		for _, goal := range goals {
			if goal.Status == GoalActive {
				return goal
			}
		}

	case PriorityStrategy:
		// Select highest priority goal
		var selected *Goal
		for _, goal := range goals {
			if selected == nil || goal.Priority > selected.Priority {
				selected = goal
			}
		}
		return selected

	case ParallelStrategy:
		// In parallel mode, select any active goal or activate a pending one
		for _, goal := range goals {
			if goal.Status == GoalActive {
				return goal
			}
		}
		for _, goal := range goals {
			if goal.Status == GoalPending {
				return goal
			}
		}

	case AdaptiveStrategy:
		// Use learning insights to make optimal choice
		return gm.selectOptimalGoal(goals)
	}

	return goals[0] // fallback
}

// selectOptimalGoal uses learning insights for adaptive selection
func (gm *basicGoalManager) selectOptimalGoal(goals []*Goal) *Goal {
	// Simple heuristic: balance priority and success probability
	var bestGoal *Goal
	bestScore := -1.0

	for _, goal := range goals {
		// Calculate score based on priority and historical success
		priorityWeight := float64(goal.Priority) / float64(CriticalPriority)

		// Check historical success rate for similar goals
		successRate := gm.getHistoricalSuccessRate(goal)

		// Simple scoring formula
		score := (priorityWeight * 0.6) + (successRate * 0.4)

		if score > bestScore {
			bestScore = score
			bestGoal = goal
		}
	}

	return bestGoal
}

// getHistoricalSuccessRate estimates success rate based on past results
func (gm *basicGoalManager) getHistoricalSuccessRate(goal *Goal) float64 {
	// Simple implementation: look for similar goal patterns
	if gm.insights.PerformanceMetrics != nil {
		return gm.insights.PerformanceMetrics.SuccessRate
	}
	return 0.5 // default neutral probability
}

// PlanActions implements GoalManager.PlanActions
func (gm *basicGoalManager) PlanActions(ctx context.Context, goalID string) ([]*AutonomousAction, error) {
	goal, err := gm.GetGoal(goalID)
	if err != nil {
		return nil, err
	}

	// Use agent's planning capability to generate actions
	plan, err := gm.agent.Plan(ctx, fmt.Sprintf("Create action plan for goal: %s. %s", goal.Title, goal.Description))
	if err != nil {
		return nil, fmt.Errorf("failed to plan actions: %w", err)
	}

	// Convert plan to autonomous actions
	actions := gm.convertPlanToActions(plan, goalID)

	// Store actions for tracking
	gm.mu.Lock()
	for _, action := range actions {
		gm.actions[action.ID] = action
	}
	gm.mu.Unlock()

	return actions, nil
}

// convertPlanToActions converts a Plan to AutonomousActions
func (gm *basicGoalManager) convertPlanToActions(plan *Plan, goalID string) []*AutonomousAction {
	var actions []*AutonomousAction

	// Extract action from plan context (simplified approach)
	if llmResponse, ok := plan.Context["llm_response"].(string); ok {
		action := &AutonomousAction{
			ID:             generateActionID(),
			GoalID:         goalID,
			Type:           "plan",
			Name:           "execute_plan",
			Parameters:     map[string]interface{}{"plan": llmResponse},
			Conditions:     []string{},
			ExpectedResult: "progress towards goal completion",
			Priority:       1,
			CreatedAt:      time.Now(),
		}
		actions = append(actions, action)
	}

	return actions
}

// ExecuteAction implements GoalManager.ExecuteAction
func (gm *basicGoalManager) ExecuteAction(ctx context.Context, action *AutonomousAction) (*ExecutionResult, error) {
	result := &ExecutionResult{
		ActionID:   action.ID,
		GoalID:     action.GoalID,
		Metadata:   make(map[string]interface{}),
		ExecutedAt: time.Now(),
	}

	// Emit action start event
	if gm.agent.GetEventBus() != nil {
		gm.agent.PublishEvent(ctx, EventType("action.started"), EventData(
			"action_id", action.ID,
			"goal_id", action.GoalID,
			"type", action.Type,
		))
	}

	var err error

	// Execute action based on type
	switch action.Type {
	case "skill":
		result.Result, err = gm.agent.ExecuteSkill(ctx, action.Name, action.Parameters)

	case "plan":
		// For plan actions, use the agent's reasoning capability
		goal, _ := gm.GetGoal(action.GoalID)
		situation := NewSituation(
			map[string]interface{}{
				"goal":        goal.Title,
				"description": goal.Description,
				"progress":    goal.Progress,
			},
			[]string{fmt.Sprintf("plan: %v", action.Parameters["plan"])},
		)
		decision, err := gm.agent.Reason(ctx, situation)
		if err == nil {
			result.Result = decision
		}

	case "reason":
		// Direct reasoning action
		situation := NewSituation(action.Parameters, []string{action.Name})
		result.Result, err = gm.agent.Reason(ctx, situation)

	case "react":
		// Reactive action
		stimulus := NewStimulus(action.Name, action.Parameters, 0.5)
		result.Result, err = gm.agent.React(ctx, stimulus)

	default:
		err = fmt.Errorf("unknown action type: %s", action.Type)
	}

	// Update result
	result.Success = err == nil
	if err != nil {
		result.Error = err.Error()
		result.Progress = 0.0
	} else {
		result.Progress = 0.1 // Default progress increment
		result.Feedback = "Action completed successfully"
	}

	// Emit action completion event
	eventType := EventType("action.completed")
	if !result.Success {
		eventType = EventType("action.failed")
	}

	if gm.agent.GetEventBus() != nil {
		gm.agent.PublishEvent(ctx, eventType, EventData(
			"action_id", action.ID,
			"goal_id", action.GoalID,
			"success", result.Success,
			"progress", result.Progress,
		))
	}

	return result, nil
}

// updateGoalProgress updates goal progress based on execution result
func (gm *basicGoalManager) updateGoalProgress(ctx context.Context, goal *Goal, result *ExecutionResult) {
	if result.Success {
		// Increment progress
		newProgress := goal.Progress + result.Progress
		if newProgress > 1.0 {
			newProgress = 1.0
		}
		goal.Progress = newProgress

		// Update goal
		gm.UpdateGoal(ctx, goal)
	}
}

// RecordResult implements GoalManager.RecordResult
func (gm *basicGoalManager) RecordResult(result *ExecutionResult) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.results = append(gm.results, result)

	// Update learning insights
	gm.updateInsights(result)

	return nil
}

// updateInsights updates learning insights based on execution results
func (gm *basicGoalManager) updateInsights(result *ExecutionResult) {
	// Update success rate
	totalResults := len(gm.results)
	successCount := 0

	for _, r := range gm.results {
		if r.Success {
			successCount++
		}
	}

	if totalResults > 0 {
		gm.insights.PerformanceMetrics.SuccessRate = float64(successCount) / float64(totalResults)
	}

	// Calculate efficiency score (simplified)
	gm.insights.PerformanceMetrics.EfficiencyScore = gm.insights.PerformanceMetrics.SuccessRate

	gm.insights.LastUpdated = time.Now()
}

// GetProgress implements GoalManager.GetProgress
func (gm *basicGoalManager) GetProgress(goalID string) (float64, error) {
	goal, err := gm.GetGoal(goalID)
	if err != nil {
		return 0, err
	}
	return goal.Progress, nil
}

// GetOverallProgress implements GoalManager.GetOverallProgress
func (gm *basicGoalManager) GetOverallProgress() *ProgressSummary {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	summary := &ProgressSummary{}

	for _, goal := range gm.goals {
		summary.TotalGoals++

		switch goal.Status {
		case GoalCompleted:
			summary.CompletedGoals++
		case GoalActive:
			summary.ActiveGoals++
		case GoalFailed:
			summary.FailedGoals++
		}
	}

	if summary.TotalGoals > 0 {
		summary.OverallProgress = float64(summary.CompletedGoals) / float64(summary.TotalGoals)
	}

	return summary
}

// GetLearnings implements GoalManager.GetLearnings
func (gm *basicGoalManager) GetLearnings() *LearningInsights {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Return a copy to prevent external modification
	insights := *gm.insights
	return &insights
}

// AdaptStrategy implements GoalManager.AdaptStrategy
func (gm *basicGoalManager) AdaptStrategy(insights *LearningInsights) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Simple adaptation logic based on success rate
	if insights.PerformanceMetrics.SuccessRate > 0.8 {
		gm.strategy = ParallelStrategy // High success rate, try parallel
	} else if insights.PerformanceMetrics.SuccessRate < 0.4 {
		gm.strategy = SequentialStrategy // Low success rate, be more conservative
	} else {
		gm.strategy = PriorityStrategy // Medium success rate, focus on priorities
	}

	return nil
}

// Helper functions

// generateActionID generates a unique ID for an action
func generateActionID() string {
	return fmt.Sprintf("action_%d", time.Now().UnixNano())
}

// NewGoal creates a new goal with default values
func NewGoal(title, description string, priority GoalPriority) *Goal {
	return &Goal{
		ID:          generateGoalID(),
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      GoalPending,
		Progress:    0.0,
		Context:     make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
