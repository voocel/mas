package tools

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/utils"
)

// Executor coordinates tool execution
type Executor struct {
	registry *Registry
	sandbox  *Sandbox
	pool     *workerPool
	config   *ExecutorConfig
	mu       sync.RWMutex
}

// ExecutorConfig configures the executor
type ExecutorConfig struct {
	MaxConcurrency int                `json:"max_concurrency"`
	DefaultTimeout time.Duration      `json:"default_timeout"`
	EnableSandbox  bool               `json:"enable_sandbox"`
	EnableRetry    bool               `json:"enable_retry"`
	RetryConfig    *utils.RetryConfig `json:"retry_config"`
}

// DefaultExecutorConfig provides sensible defaults
var DefaultExecutorConfig = &ExecutorConfig{
	MaxConcurrency: 10,
	DefaultTimeout: 30 * time.Second,
	EnableSandbox:  true,
	EnableRetry:    true,
	RetryConfig: &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
		Jitter:      true,
	},
}

// NewExecutor constructs a tool executor
func NewExecutor(registry *Registry, config *ExecutorConfig) *Executor {
	if registry == nil {
		registry = NewRegistry()
	}
	if config == nil {
		config = DefaultExecutorConfig
	}

	executor := &Executor{
		registry: registry,
		sandbox:  NewSandbox(nil, nil),
		config:   config,
		pool:     newWorkerPool(config.MaxConcurrency),
	}

	if !config.EnableSandbox {
		executor.sandbox.Disable()
	}

	return executor
}

// Execute runs a single tool
func (e *Executor) Execute(ctx runtime.Context, toolCall schema.ToolCall) (schema.ToolResult, error) {
	tool, exists := e.registry.Get(toolCall.Name)
	if !exists {
		return schema.ToolResult{}, schema.NewToolError(toolCall.Name, "execute", schema.ErrToolNotFound)
	}

	// Emit tool call event
	ctx.AddEvent(schema.NewToolCallEvent(toolCall, ""))

	var result json.RawMessage
	var err error

	if e.config.EnableRetry {
		err = e.config.RetryConfig.Execute(ctx, func() error {
			result, err = e.executeTool(tool, ctx, toolCall.Args)
			return err
		})
	} else {
		result, err = e.executeTool(tool, ctx, toolCall.Args)
	}

	toolResult := schema.ToolResult{
		ID: toolCall.ID,
	}

	if err != nil {
		toolResult.Error = err.Error()
	} else {
		toolResult.Result = result
	}

	// Emit tool result event
	ctx.AddEvent(schema.NewToolResultEvent(toolResult, ""))

	return toolResult, err
}

// ExecuteParallel runs multiple tools concurrently
func (e *Executor) ExecuteParallel(ctx runtime.Context, toolCalls []schema.ToolCall) ([]schema.ToolResult, error) {
	if len(toolCalls) == 0 {
		return []schema.ToolResult{}, nil
	}

	results := make([]schema.ToolResult, len(toolCalls))
	errChan := make(chan error, len(toolCalls))
	var wg sync.WaitGroup

	for i, toolCall := range toolCalls {
		wg.Add(1)
		go func(idx int, tc schema.ToolCall) {
			defer wg.Done()

			result, err := e.Execute(ctx, tc)
			results[idx] = result

			if err != nil {
				errChan <- err
			}
		}(i, toolCall)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return results, errors[0]
	}

	return results, nil
}

// ExecuteAsync executes a tool asynchronously
func (e *Executor) ExecuteAsync(ctx runtime.Context, toolCall schema.ToolCall) (<-chan ExecutionResult, error) {
	resultChan := make(chan ExecutionResult, 1)

	task := &executionTask{
		executor: e,
		ctx:      ctx,
		toolCall: toolCall,
		result:   resultChan,
	}

	if err := e.pool.Submit(task); err != nil {
		close(resultChan)
		return nil, err
	}

	return resultChan, nil
}

// executeTool runs the tool with validation and sandboxing
func (e *Executor) executeTool(tool Tool, ctx runtime.Context, input json.RawMessage) (json.RawMessage, error) {
	if baseTool, ok := tool.(*BaseTool); ok {
		if err := baseTool.ValidateInput(input); err != nil {
			return nil, err
		}
	}

	// Execute inside the sandbox
	if e.sandbox.IsEnabled() {
		return e.sandbox.Execute(tool, ctx, input)
	}

	// Execute directly
	return tool.Execute(ctx, input)
}

// GetRegistry returns the registry
func (e *Executor) GetRegistry() *Registry {
	return e.registry
}

// GetSandbox returns the sandbox
func (e *Executor) GetSandbox() *Sandbox {
	return e.sandbox
}

// SetSandbox updates the sandbox
func (e *Executor) SetSandbox(sandbox *Sandbox) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sandbox = sandbox
}

// ExecutionResult captures execution outcome
type ExecutionResult struct {
	ToolResult schema.ToolResult
	Error      error
}

// executionTask represents a queued execution
type executionTask struct {
	executor *Executor
	ctx      runtime.Context
	toolCall schema.ToolCall
	result   chan<- ExecutionResult
}

// Execute runs the task
func (t *executionTask) Execute() {
	defer close(t.result)

	toolResult, err := t.executor.Execute(t.ctx, t.toolCall)
	t.result <- ExecutionResult{
		ToolResult: toolResult,
		Error:      err,
	}
}

// workerPool manages background workers
type workerPool struct {
	workers    int
	taskQueue  chan Task
	workerQuit chan struct{}
	quit       chan struct{}
	wg         sync.WaitGroup
}

// Task represents a unit of work
type Task interface {
	Execute()
}

// newWorkerPool constructs a worker pool
func newWorkerPool(workers int) *workerPool {
	pool := &workerPool{
		workers:    workers,
		taskQueue:  make(chan Task, workers*2),
		workerQuit: make(chan struct{}),
		quit:       make(chan struct{}),
	}

	pool.start()
	return pool
}

// start launches workers
func (p *workerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// worker consumes tasks
func (p *workerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case task := <-p.taskQueue:
			if task != nil {
				task.Execute()
			}
		case <-p.quit:
			return
		}
	}
}

// Submit enqueues a task
func (p *workerPool) Submit(task Task) error {
	select {
	case p.taskQueue <- task:
		return nil
	case <-p.quit:
		return schema.ErrContextCancelled
	default:
		return schema.ErrTimeout
	}
}

// Stop shuts down the worker pool
func (p *workerPool) Stop() {
	close(p.quit)
	p.wg.Wait()
}
