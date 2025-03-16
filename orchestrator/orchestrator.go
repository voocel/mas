package orchestrator

import (
	"context"
	"sync"
	"time"

	"github.com/voocel/mas/communication"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/voocel/mas/agent"
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
	// TaskStatusPending 待处理
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusRunning 运行中
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusPaused 已暂停
	TaskStatusPaused TaskStatus = "paused"
	// TaskStatusCompleted 已完成
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed 失败
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled 已取消
	TaskStatusCancelled TaskStatus = "cancelled"
)

type Options struct {
	Bus          communication.Bus
	DefaultTTL   time.Duration
	PollInterval time.Duration
}

// Orchestrator 定义了多智能体系统的编排器
type Orchestrator interface {
	// RegisterAgent 注册一个智能体
	RegisterAgent(agent agent.Agent) error

	// GetAgent 获取指定ID的智能体
	GetAgent(id string) (agent.Agent, error)

	// ListAgents 列出所有注册的智能体
	ListAgents() []agent.Agent

	// SubmitTask 提交一个任务
	SubmitTask(ctx context.Context, task Task) (string, error)

	// GetTask 获取指定ID的任务信息
	GetTask(id string) (Task, error)

	// CancelTask 取消一个任务
	CancelTask(ctx context.Context, id string) error

	// Start 启动编排器
	Start() error

	// Stop 停止编排器
	Stop() error
}

// BasicOrchestrator 实现了基础的编排器功能
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

// NewBasicOrchestrator 创建一个新的基础编排器
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

// RegisterAgent 注册一个智能体
func (o *BasicOrchestrator) RegisterAgent(a agent.Agent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 使用智能体的名称作为ID
	agentID := a.Name()
	o.agents[agentID] = a
	return nil
}

// GetAgent 获取指定ID的智能体
func (o *BasicOrchestrator) GetAgent(id string) (agent.Agent, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	a, ok := o.agents[id]
	if !ok {
		return nil, errors.New("agent not found")
	}

	return a, nil
}

// ListAgents 列出所有注册的智能体
func (o *BasicOrchestrator) ListAgents() []agent.Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agents := make([]agent.Agent, 0, len(o.agents))
	for _, a := range o.agents {
		agents = append(agents, a)
	}

	return agents
}

// SubmitTask 提交一个任务
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

	// 启动异步处理任务
	go o.processTask(task.ID)

	return task.ID, nil
}

// GetTask 获取指定ID的任务信息
func (o *BasicOrchestrator) GetTask(id string) (Task, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	task, ok := o.tasks[id]
	if !ok {
		return Task{}, errors.New("task not found")
	}

	return task, nil
}

// CancelTask 取消一个任务
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

// Start 启动编排器
func (o *BasicOrchestrator) Start() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return errors.New("orchestrator already running")
	}

	o.running = true

	return nil
}

// Stop 停止编排器
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

// processTask 异步处理任务
func (o *BasicOrchestrator) processTask(taskID string) {
	o.mu.Lock()
	task, ok := o.tasks[taskID]
	if !ok || task.Status != TaskStatusPending {
		o.mu.Unlock()
		return
	}

	// 更新任务状态为运行中
	now := time.Now()
	task.Status = TaskStatusRunning
	task.UpdatedAt = now
	task.StartedAt = &now
	o.tasks[taskID] = task
	o.mu.Unlock()

	ctx, cancel := context.WithTimeout(o.ctx, o.ttl)
	defer cancel()

	// todo 需要更复杂的协作逻辑
	var result interface{}
	var taskErr error

	// 顺序执行每个智能体的处理
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
