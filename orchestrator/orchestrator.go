package orchestrator

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/communication"
)

type Task struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      TaskStatus             `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	FinishedAt  *time.Time             `json:"finished_at,omitempty"`
	AgentIDs    []string               `json:"agent_ids"`
	Input       interface{}            `json:"input"`
	Output      interface{}            `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type TaskStatus string

const (
	// TaskStatusPending pending
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusRunning running
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusPaused paused
	TaskStatusPaused TaskStatus = "paused"
	// TaskStatusCompleted completed
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed failed
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled cancelled
	TaskStatusCancelled TaskStatus = "cancelled"
)

type Options struct {
	Bus          communication.Bus
	DefaultTTL   time.Duration
	PollInterval time.Duration
}

// Orchestrator defines an orchestrator for multi-agent systems
type Orchestrator interface {
	// RegisterAgent registers an agent
	RegisterAgent(agent agent.Agent) error

	// GetAgent gets an agent by ID
	GetAgent(id string) (agent.Agent, error)

	// ListAgents lists all registered agents
	ListAgents() []agent.Agent

	// SubmitTask submits a task
	SubmitTask(ctx context.Context, task Task) (string, error)

	// GetTask gets task information by ID
	GetTask(id string) (Task, error)

	// CancelTask cancels a task
	CancelTask(ctx context.Context, id string) error

	// Start starts the orchestrator
	Start() error

	// Stop stops the orchestrator
	Stop() error
}

// BasicOrchestrator implements basic orchestrator functionality
type BasicOrchestrator struct {
	agents    map[string]agent.Agent
	tasks     map[string]Task
	bus       communication.Bus
	ttl       time.Duration
	pollInt   time.Duration
	running   bool
	ctx       context.Context
	cancelCtx context.CancelFunc
	mu        sync.RWMutex
}

// NewBasicOrchestrator creates a new basic orchestrator
func NewBasicOrchestrator(opts Options) *BasicOrchestrator {
	ttl := 1 * time.Hour
	if opts.DefaultTTL > 0 {
		ttl = opts.DefaultTTL
	}

	pollInt := 100 * time.Millisecond
	if opts.PollInterval > 0 {
		pollInt = opts.PollInterval
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BasicOrchestrator{
		agents:    make(map[string]agent.Agent),
		tasks:     make(map[string]Task),
		bus:       opts.Bus,
		ttl:       ttl,
		pollInt:   pollInt,
		running:   false,
		ctx:       ctx,
		cancelCtx: cancel,
		mu:        sync.RWMutex{},
	}
}

// RegisterAgent registers an agent
func (o *BasicOrchestrator) RegisterAgent(a agent.Agent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Use the agent's name as ID
	agentID := a.Name()
	o.agents[agentID] = a
	return nil
}

// GetAgent gets an agent by ID
func (o *BasicOrchestrator) GetAgent(id string) (agent.Agent, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	a, ok := o.agents[id]
	if !ok {
		return nil, errors.New("agent not found")
	}

	return a, nil
}

// ListAgents lists all registered agents
func (o *BasicOrchestrator) ListAgents() []agent.Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agents := make([]agent.Agent, 0, len(o.agents))
	for _, a := range o.agents {
		agents = append(agents, a)
	}

	return agents
}

// SubmitTask submits a task
func (o *BasicOrchestrator) SubmitTask(ctx context.Context, task Task) (string, error) {
	if !o.running {
		return "", errors.New("orchestrator not running")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	task.Status = TaskStatusPending

	for _, agentID := range task.AgentIDs {
		if _, ok := o.agents[agentID]; !ok {
			return "", errors.Errorf("agent with ID %s not found", agentID)
		}
	}

	o.tasks[task.ID] = task

	// Start asynchronous task processing
	go o.processTask(task.ID)

	return task.ID, nil
}

// GetTask gets task information by ID
func (o *BasicOrchestrator) GetTask(id string) (Task, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	task, ok := o.tasks[id]
	if !ok {
		return Task{}, errors.New("task not found")
	}

	return task, nil
}

// CancelTask cancels a task
func (o *BasicOrchestrator) CancelTask(ctx context.Context, id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	task, ok := o.tasks[id]
	if !ok {
		return errors.New("task not found")
	}

	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed || task.Status == TaskStatusCancelled {
		return errors.New("cannot cancel a finished task")
	}

	task.Status = TaskStatusCancelled
	task.UpdatedAt = time.Now()
	o.tasks[id] = task

	return nil
}

// Start starts the orchestrator
func (o *BasicOrchestrator) Start() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return errors.New("orchestrator already running")
	}

	o.running = true

	return nil
}

// Stop stops the orchestrator
func (o *BasicOrchestrator) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.running {
		return errors.New("orchestrator not running")
	}

	o.cancelCtx()
	o.running = false

	return nil
}

// processTask processes task asynchronously
func (o *BasicOrchestrator) processTask(taskID string) {
	o.mu.Lock()
	task, ok := o.tasks[taskID]
	if !ok || task.Status != TaskStatusPending {
		o.mu.Unlock()
		return
	}

	// Update task status to running
	now := time.Now()
	task.Status = TaskStatusRunning
	task.UpdatedAt = now
	task.StartedAt = &now
	o.tasks[taskID] = task
	o.mu.Unlock()

	ctx, cancel := context.WithTimeout(o.ctx, o.ttl)
	defer cancel()

	// todo need more complex collaboration logic
	var result interface{}
	var taskErr error

	// Process with each agent sequentially
	for _, agentID := range task.AgentIDs {
		o.mu.RLock()
		agent, ok := o.agents[agentID]
		o.mu.RUnlock()

		if !ok {
			taskErr = errors.Errorf("agent %s not found", agentID)
			break
		}

		result, taskErr = agent.Process(ctx, result)
		if taskErr != nil {
			break
		}
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	task, ok = o.tasks[taskID]
	if !ok {
		return
	}

	now = time.Now()
	task.UpdatedAt = now
	task.FinishedAt = &now

	if taskErr != nil {
		task.Status = TaskStatusFailed
		task.Error = taskErr.Error()
	} else {
		task.Status = TaskStatusCompleted
		task.Output = result
	}

	o.tasks[taskID] = task
}
