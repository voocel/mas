package shared

import (
	"context"
	"fmt"
	"sync"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// Coordinator manages coordination between multiple agents
type Coordinator struct {
	sharedContext SharedContext
	agents        map[string]*ManagedAgent
	taskQueue     *TaskQueue
	loadBalancer  *LoadBalancer
	mutex         sync.RWMutex
	config        CoordinatorConfig
}

// CoordinatorConfig configures the coordinator
type CoordinatorConfig struct {
	MaxConcurrentTasks  int           `json:"max_concurrent_tasks"`
	TaskTimeout         time.Duration `json:"task_timeout"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	EnableLoadBalancing bool          `json:"enable_load_balancing"`
	EnableFailover      bool          `json:"enable_failover"`
}

// ManagedAgent represents an agent managed by the coordinator
type ManagedAgent struct {
	ID           string                   `json:"id"`
	Metadata     *AgentMetadata           `json:"metadata"`
	CurrentTasks []string                 `json:"current_tasks"`
	LoadScore    float64                  `json:"load_score"`
	LastPing     time.Time                `json:"last_ping"`
	IsHealthy    bool                     `json:"is_healthy"`
	Context      *contextpkg.ContextState `json:"context"`
}

// Task represents a task that can be assigned to agents
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	Status      TaskStatus             `json:"status"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusAssigned  TaskStatus = "assigned"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskQueue manages a queue of tasks
type TaskQueue struct {
	tasks  []*Task
	mutex  sync.RWMutex
	notify chan struct{}
}

// LoadBalancer manages load balancing across agents
type LoadBalancer struct {
	strategy LoadBalancingStrategy
}

// LoadBalancingStrategy defines different load balancing strategies
type LoadBalancingStrategy string

const (
	RoundRobin      LoadBalancingStrategy = "round_robin"
	LeastLoaded     LoadBalancingStrategy = "least_loaded"
	CapabilityBased LoadBalancingStrategy = "capability_based"
)

// NewCoordinator creates a new coordinator
func NewCoordinator(sharedContext SharedContext, config CoordinatorConfig) *Coordinator {
	coordinator := &Coordinator{
		sharedContext: sharedContext,
		agents:        make(map[string]*ManagedAgent),
		taskQueue:     NewTaskQueue(),
		loadBalancer:  NewLoadBalancer(LeastLoaded),
		config:        config,
	}

	// Start background processes
	go coordinator.healthCheckLoop()
	go coordinator.taskProcessingLoop()

	return coordinator
}

// NewTaskQueue creates a new task queue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks:  make([]*Task, 0),
		notify: make(chan struct{}, 1),
	}
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(strategy LoadBalancingStrategy) *LoadBalancer {
	return &LoadBalancer{
		strategy: strategy,
	}
}

// RegisterAgent registers an agent with the coordinator
func (c *Coordinator) RegisterAgent(ctx context.Context, agentID string, metadata AgentMetadata) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Register with shared context
	err := c.sharedContext.RegisterAgent(ctx, agentID, metadata)
	if err != nil {
		return fmt.Errorf("failed to register agent with shared context: %w", err)
	}

	// Create managed agent
	managedAgent := &ManagedAgent{
		ID:           agentID,
		Metadata:     &metadata,
		CurrentTasks: make([]string, 0),
		LoadScore:    0.0,
		LastPing:     time.Now(),
		IsHealthy:    true,
	}

	c.agents[agentID] = managedAgent

	return nil
}

// UnregisterAgent unregisters an agent from the coordinator
func (c *Coordinator) UnregisterAgent(ctx context.Context, agentID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Unregister from shared context
	err := c.sharedContext.UnregisterAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to unregister agent from shared context: %w", err)
	}

	// Remove managed agent
	delete(c.agents, agentID)

	return nil
}

// SubmitTask submits a task to the coordinator
func (c *Coordinator) SubmitTask(ctx context.Context, task *Task) error {
	if task.ID == "" {
		task.ID = generateTaskID()
	}

	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()

	if task.Timeout == 0 {
		task.Timeout = c.config.TaskTimeout
	}

	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	return c.taskQueue.Enqueue(task)
}

// AssignTask assigns a task to a specific agent
func (c *Coordinator) AssignTask(ctx context.Context, taskID, agentID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Find the task
	task := c.taskQueue.FindTask(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Check if agent exists and is healthy
	agent, exists := c.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	if !agent.IsHealthy {
		return fmt.Errorf("agent %s is not healthy", agentID)
	}

	// Assign the task
	task.AssignedTo = agentID
	task.Status = TaskStatusAssigned
	now := time.Now()
	task.StartedAt = &now

	// Add to agent's current tasks
	agent.CurrentTasks = append(agent.CurrentTasks, taskID)

	// Update load score
	agent.LoadScore = c.calculateLoadScore(agent)

	return nil
}

// GetOptimalAgent finds the optimal agent for a task
func (c *Coordinator) GetOptimalAgent(ctx context.Context, task *Task) (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.loadBalancer.SelectAgent(c.agents, task)
}

// UpdateAgentStatus updates the status of an agent
func (c *Coordinator) UpdateAgentStatus(ctx context.Context, agentID string, status AgentStatus) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	agent, exists := c.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	agent.Metadata.Status = status
	agent.LastPing = time.Now()

	// Update in shared context
	return c.sharedContext.RegisterAgent(ctx, agentID, *agent.Metadata)
}

// CompleteTask marks a task as completed
func (c *Coordinator) CompleteTask(ctx context.Context, taskID string, result map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	task := c.taskQueue.FindTask(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Status = TaskStatusCompleted
	now := time.Now()
	task.CompletedAt = &now

	// Remove from agent's current tasks
	if task.AssignedTo != "" {
		if agent, exists := c.agents[task.AssignedTo]; exists {
			agent.CurrentTasks = removeTaskFromList(agent.CurrentTasks, taskID)
			agent.LoadScore = c.calculateLoadScore(agent)
		}
	}

	// Store result in task data
	task.Data["result"] = result

	return nil
}

// FailTask marks a task as failed
func (c *Coordinator) FailTask(ctx context.Context, taskID string, reason string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	task := c.taskQueue.FindTask(taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Retries++

	// Check if we should retry
	if task.Retries < task.MaxRetries {
		task.Status = TaskStatusPending
		task.AssignedTo = ""
		task.StartedAt = nil
	} else {
		task.Status = TaskStatusFailed
		task.Data["failure_reason"] = reason
	}

	// Remove from agent's current tasks
	if task.AssignedTo != "" {
		if agent, exists := c.agents[task.AssignedTo]; exists {
			agent.CurrentTasks = removeTaskFromList(agent.CurrentTasks, taskID)
			agent.LoadScore = c.calculateLoadScore(agent)
		}
	}

	return nil
}

// GetCoordinatorStats returns statistics about the coordinator
func (c *Coordinator) GetCoordinatorStats() CoordinatorStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := CoordinatorStats{
		TotalAgents:    len(c.agents),
		HealthyAgents:  0,
		PendingTasks:   0,
		RunningTasks:   0,
		CompletedTasks: 0,
		FailedTasks:    0,
	}

	// Count healthy agents
	for _, agent := range c.agents {
		if agent.IsHealthy {
			stats.HealthyAgents++
		}
	}

	// Count tasks by status
	for _, task := range c.taskQueue.tasks {
		switch task.Status {
		case TaskStatusPending:
			stats.PendingTasks++
		case TaskStatusRunning:
			stats.RunningTasks++
		case TaskStatusCompleted:
			stats.CompletedTasks++
		case TaskStatusFailed:
			stats.FailedTasks++
		}
	}

	return stats
}

// CoordinatorStats represents statistics about the coordinator
type CoordinatorStats struct {
	TotalAgents    int `json:"total_agents"`
	HealthyAgents  int `json:"healthy_agents"`
	PendingTasks   int `json:"pending_tasks"`
	RunningTasks   int `json:"running_tasks"`
	CompletedTasks int `json:"completed_tasks"`
	FailedTasks    int `json:"failed_tasks"`
}

// Background processes
func (c *Coordinator) healthCheckLoop() {
	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.performHealthCheck()
	}
}

func (c *Coordinator) taskProcessingLoop() {
	for {
		select {
		case <-c.taskQueue.notify:
			c.processPendingTasks()
		}
	}
}

func (c *Coordinator) performHealthCheck() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for _, agent := range c.agents {
		// Check if agent has been inactive for too long
		if now.Sub(agent.LastPing) > c.config.HealthCheckInterval*3 {
			agent.IsHealthy = false
			agent.Metadata.Status = AgentStatusOffline
		}
	}
}

func (c *Coordinator) processPendingTasks() {
	if !c.config.EnableLoadBalancing {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, task := range c.taskQueue.tasks {
		if task.Status == TaskStatusPending {
			selectedAgentID, err := c.loadBalancer.SelectAgent(c.agents, task)
			if err == nil && selectedAgentID != "" {
				c.AssignTask(context.Background(), task.ID, selectedAgentID)
			}
		}
	}
}

func (c *Coordinator) calculateLoadScore(agent *ManagedAgent) float64 {
	// Simple load calculation based on number of current tasks
	return float64(len(agent.CurrentTasks))
}

// Helper functions
func generateTaskID() string {
	return "task_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func removeTaskFromList(tasks []string, taskID string) []string {
	for i, id := range tasks {
		if id == taskID {
			return append(tasks[:i], tasks[i+1:]...)
		}
	}
	return tasks
}

// TaskQueue methods
func (tq *TaskQueue) Enqueue(task *Task) error {
	tq.mutex.Lock()
	defer tq.mutex.Unlock()

	tq.tasks = append(tq.tasks, task)

	// Notify task processing loop
	select {
	case tq.notify <- struct{}{}:
	default:
	}

	return nil
}

func (tq *TaskQueue) FindTask(taskID string) *Task {
	tq.mutex.RLock()
	defer tq.mutex.RUnlock()

	for _, task := range tq.tasks {
		if task.ID == taskID {
			return task
		}
	}
	return nil
}

// LoadBalancer methods
func (lb *LoadBalancer) SelectAgent(agents map[string]*ManagedAgent, task *Task) (string, error) {
	switch lb.strategy {
	case RoundRobin:
		return lb.selectRoundRobin(agents)
	case LeastLoaded:
		return lb.selectLeastLoaded(agents)
	case CapabilityBased:
		return lb.selectByCapability(agents, task)
	default:
		return lb.selectLeastLoaded(agents)
	}
}

func (lb *LoadBalancer) selectRoundRobin(agents map[string]*ManagedAgent) (string, error) {
	// Simple round-robin implementation
	for agentID, agent := range agents {
		if agent.IsHealthy && agent.Metadata.Status == AgentStatusActive {
			return agentID, nil
		}
	}
	return "", fmt.Errorf("no healthy agents available")
}

func (lb *LoadBalancer) selectLeastLoaded(agents map[string]*ManagedAgent) (string, error) {
	var bestAgent string
	var minLoad float64 = -1

	for agentID, agent := range agents {
		if agent.IsHealthy && agent.Metadata.Status == AgentStatusActive {
			if minLoad < 0 || agent.LoadScore < minLoad {
				minLoad = agent.LoadScore
				bestAgent = agentID
			}
		}
	}

	if bestAgent == "" {
		return "", fmt.Errorf("no healthy agents available")
	}

	return bestAgent, nil
}

func (lb *LoadBalancer) selectByCapability(agents map[string]*ManagedAgent, task *Task) (string, error) {
	// Select agent based on required capabilities
	requiredCapabilities, ok := task.Data["required_capabilities"].([]string)
	if !ok {
		// Fall back to least loaded if no capabilities specified
		return lb.selectLeastLoaded(agents)
	}

	var bestAgent string
	var minLoad float64 = -1

	for agentID, agent := range agents {
		if agent.IsHealthy && agent.Metadata.Status == AgentStatusActive {
			if lb.hasRequiredCapabilities(agent.Metadata.Capabilities, requiredCapabilities) {
				if minLoad < 0 || agent.LoadScore < minLoad {
					minLoad = agent.LoadScore
					bestAgent = agentID
				}
			}
		}
	}

	if bestAgent == "" {
		return "", fmt.Errorf("no agents with required capabilities available")
	}

	return bestAgent, nil
}

func (lb *LoadBalancer) hasRequiredCapabilities(agentCapabilities, requiredCapabilities []string) bool {
	capabilitySet := make(map[string]bool)
	for _, cap := range agentCapabilities {
		capabilitySet[cap] = true
	}

	for _, required := range requiredCapabilities {
		if !capabilitySet[required] {
			return false
		}
	}

	return true
}
