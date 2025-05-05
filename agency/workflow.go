package agency

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkflowStatus defines the workflow status
type WorkflowStatus string

const (
	// WorkflowStatusPending workflow is pending
	WorkflowStatusPending WorkflowStatus = "pending"
	// WorkflowStatusRunning workflow is running
	WorkflowStatusRunning WorkflowStatus = "running"
	// WorkflowStatusCompleted workflow is completed
	WorkflowStatusCompleted WorkflowStatus = "completed"
	// WorkflowStatusFailed workflow has failed
	WorkflowStatusFailed WorkflowStatus = "failed"
	// WorkflowStatusCancelled workflow is cancelled
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// WorkflowStep defines a workflow step
type WorkflowStep struct {
	ID          string
	Name        string
	Description string
	AgentID     string
	InputFrom   []string // List of IDs from previous steps

	// Transform function converts outputs from previous steps to input for current step
	Transform func(ctx context.Context, inputs map[string]interface{}) (interface{}, error)

	// Output processor function for the step
	OutputProcessor func(ctx context.Context, output interface{}) (interface{}, error)

	// Condition function determines whether to execute this step
	Condition func(ctx context.Context, inputs map[string]interface{}) (bool, error)
}

// Workflow defines a workflow
type Workflow struct {
	ID          string
	Name        string
	Description string
	Steps       map[string]*WorkflowStep
	Order       []string // Step execution order

	// Runtime status
	Status        WorkflowStatus
	StepStatus    map[string]WorkflowStatus
	StepOutputs   map[string]interface{}
	CurrentStepID string
	Error         string

	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time

	mu sync.RWMutex
}

// NewWorkflow creates a new workflow
func NewWorkflow(name, description string) *Workflow {
	return &Workflow{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Steps:       make(map[string]*WorkflowStep),
		Order:       make([]string, 0),
		Status:      WorkflowStatusPending,
		StepStatus:  make(map[string]WorkflowStatus),
		StepOutputs: make(map[string]interface{}),
		CreatedAt:   time.Now(),
	}
}

// AddStep adds a workflow step
func (w *Workflow) AddStep(step *WorkflowStep) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if step.ID == "" {
		step.ID = uuid.New().String()
	}

	if _, exists := w.Steps[step.ID]; exists {
		return fmt.Errorf("step with ID %s already exists", step.ID)
	}

	w.Steps[step.ID] = step
	w.Order = append(w.Order, step.ID)
	w.StepStatus[step.ID] = WorkflowStatusPending

	return nil
}

// SetStepOrder sets the step execution order
func (w *Workflow) SetStepOrder(order []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Verify all steps exist
	for _, stepID := range order {
		if _, exists := w.Steps[stepID]; !exists {
			return fmt.Errorf("step with ID %s does not exist", stepID)
		}
	}

	// Verify all steps are included in the order
	if len(order) != len(w.Steps) {
		return fmt.Errorf("order must include all steps (found %d, expected %d)", len(order), len(w.Steps))
	}

	w.Order = order
	return nil
}

// GetStep gets a workflow step
func (w *Workflow) GetStep(stepID string) (*WorkflowStep, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	step, exists := w.Steps[stepID]
	if !exists {
		return nil, fmt.Errorf("step with ID %s not found", stepID)
	}

	return step, nil
}

// Execute executes the workflow
func (w *Workflow) Execute(ctx context.Context, agency *Agency, input interface{}) (interface{}, error) {
	w.mu.Lock()
	w.Status = WorkflowStatusRunning
	now := time.Now()
	w.StartedAt = &now
	w.mu.Unlock()

	// Initialize input for the first step
	initialInputs := map[string]interface{}{
		"input": input,
	}

	var finalOutput interface{}

	// Execute each step in order
	for _, stepID := range w.Order {
		w.mu.Lock()
		w.CurrentStepID = stepID
		w.StepStatus[stepID] = WorkflowStatusRunning
		w.mu.Unlock()

		step, err := w.GetStep(stepID)
		if err != nil {
			w.setError(err.Error())
			return nil, err
		}

		// Collect inputs for this step
		stepInputs := make(map[string]interface{})
		for k, v := range initialInputs {
			stepInputs[k] = v
		}

		// Add outputs from all previous steps as inputs
		for _, inputStepID := range step.InputFrom {
			w.mu.RLock()
			output, exists := w.StepOutputs[inputStepID]
			w.mu.RUnlock()

			if !exists {
				err := fmt.Errorf("required input from step %s not available", inputStepID)
				w.setError(err.Error())
				return nil, err
			}

			stepInputs[inputStepID] = output
		}

		// Check if conditions are met
		if step.Condition != nil {
			shouldRun, err := step.Condition(ctx, stepInputs)
			if err != nil {
				w.setStepStatus(stepID, WorkflowStatusFailed)
				w.setError(fmt.Sprintf("condition check failed for step %s: %v", stepID, err))
				return nil, err
			}

			if !shouldRun {
				w.setStepStatus(stepID, WorkflowStatusCancelled)
				continue // Skip this step
			}
		}

		// Transform input
		var stepInput interface{} = stepInputs
		if step.Transform != nil {
			stepInput, err = step.Transform(ctx, stepInputs)
			if err != nil {
				w.setStepStatus(stepID, WorkflowStatusFailed)
				w.setError(fmt.Sprintf("input transformation failed for step %s: %v", stepID, err))
				return nil, err
			}
		}

		// Get the agent
		agent, err := agency.GetAgent(step.AgentID)
		if err != nil {
			w.setStepStatus(stepID, WorkflowStatusFailed)
			w.setError(fmt.Sprintf("agent %s not found for step %s", step.AgentID, stepID))
			return nil, err
		}

		// Execute agent task
		output, err := agent.Process(ctx, stepInput)
		if err != nil {
			w.setStepStatus(stepID, WorkflowStatusFailed)
			w.setError(fmt.Sprintf("execution failed for step %s: %v", stepID, err))
			return nil, err
		}

		// Process output
		if step.OutputProcessor != nil {
			output, err = step.OutputProcessor(ctx, output)
			if err != nil {
				w.setStepStatus(stepID, WorkflowStatusFailed)
				w.setError(fmt.Sprintf("output processing failed for step %s: %v", stepID, err))
				return nil, err
			}
		}

		// Save step output
		w.mu.Lock()
		w.StepOutputs[stepID] = output
		w.StepStatus[stepID] = WorkflowStatusCompleted
		w.mu.Unlock()

		// Record the output of the last step as the workflow result
		finalOutput = output
	}

	// Workflow completed successfully
	w.mu.Lock()
	w.Status = WorkflowStatusCompleted
	now = time.Now()
	w.FinishedAt = &now
	w.mu.Unlock()

	return finalOutput, nil
}

// setError sets the workflow error
func (w *Workflow) setError(errMsg string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Status = WorkflowStatusFailed
	w.Error = errMsg
	now := time.Now()
	w.FinishedAt = &now
}

// setStepStatus sets the step status
func (w *Workflow) setStepStatus(stepID string, status WorkflowStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.StepStatus[stepID] = status
}

// GetStatus gets the workflow status
func (w *Workflow) GetStatus() WorkflowStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.Status
}

// GetStepStatus gets the step status
func (w *Workflow) GetStepStatus(stepID string) (WorkflowStatus, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	status, exists := w.StepStatus[stepID]
	if !exists {
		return "", fmt.Errorf("step with ID %s not found", stepID)
	}

	return status, nil
}

// GetStepOutput gets the step output
func (w *Workflow) GetStepOutput(stepID string) (interface{}, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	output, exists := w.StepOutputs[stepID]
	if !exists {
		return nil, fmt.Errorf("output for step with ID %s not found", stepID)
	}

	return output, nil
}
