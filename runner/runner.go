package runner

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/guardrail"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/memory"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// Config controls Runner behavior.
type Config struct {
	Model          llm.ChatModel
	Memory         memory.Store
	ToolInvoker    tools.Invoker
	Middlewares    []Middleware
	Observer       Observer
	Tracer         Tracer
	RunIDGenerator func() string
	Checkpointer   Checkpointer
	ResponseFormat *llm.ResponseFormat
	MaxTurns       int
	HistoryWindow  int
}

// Runner executes an agent run loop.
type Runner struct {
	config Config
}

var runCounter uint64

func defaultRunIDGenerator() string {
	seq := atomic.AddUint64(&runCounter, 1)
	return fmt.Sprintf("run_%d_%d", time.Now().UnixNano(), seq)
}

// New creates a Runner and fills default config.
func New(cfg Config) *Runner {
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = 4
	}
	if cfg.HistoryWindow == 0 {
		cfg.HistoryWindow = 20
	}
	if cfg.Memory == nil {
		cfg.Memory = memory.NewBuffer(cfg.HistoryWindow)
	}
	if cfg.ToolInvoker == nil {
		cfg.ToolInvoker = tools.NewSerialInvoker()
	}
	if cfg.Observer == nil {
		cfg.Observer = &NoopObserver{}
	}
	if cfg.Tracer == nil {
		cfg.Tracer = &NoopTracer{}
	}
	if cfg.RunIDGenerator == nil {
		cfg.RunIDGenerator = defaultRunIDGenerator
	}
	return &Runner{config: cfg}
}

// WithMemory returns a Runner using the provided memory.
func (r *Runner) WithMemory(store memory.Store) *Runner {
	if r == nil {
		return New(Config{Memory: store})
	}
	cfg := r.config
	cfg.Memory = store
	return New(cfg)
}

// GetMemory returns the current memory.
func (r *Runner) GetMemory() memory.Store {
	if r == nil {
		return nil
	}
	return r.config.Memory
}

// State describes the context of a run turn.
type State struct {
	Context  context.Context
	Agent    *agent.Agent
	Input    schema.Message
	Messages []schema.Message
	Response schema.Message
	Turn     int
	RunID    schema.RunID
	StepID   schema.StepID
	SpanID   schema.SpanID
}

// ToolState describes tool call context.
type ToolState struct {
	Context context.Context
	Agent   *agent.Agent
	Call    *schema.ToolCall
	Result  *schema.ToolResult
	RunID   schema.RunID
	StepID  schema.StepID
	SpanID  schema.SpanID
}

// RunResult carries full execution results.
type RunResult struct {
	Message     schema.Message
	Usage       llm.TokenUsage
	ToolCalls   []schema.ToolCall
	ToolResults []schema.ToolResult
}

// Middleware is a marker interface for optional hooks.
type Middleware interface{}

type BeforeLLM interface {
	BeforeLLM(ctx context.Context, state *State) error
}

type AfterLLM interface {
	AfterLLM(ctx context.Context, state *State) error
}

type BeforeTool interface {
	BeforeTool(ctx context.Context, state *ToolState) error
}

type AfterTool interface {
	AfterTool(ctx context.Context, state *ToolState) error
}

// LLMHandler defines the LLM call signature.
type LLMHandler func(ctx context.Context, req *llm.Request) (*llm.Response, error)

// LLMMiddleware allows wrapping LLM calls.
type LLMMiddleware interface {
	HandleLLM(ctx context.Context, state *State, req *llm.Request, next LLMHandler) (*llm.Response, error)
}

// ContextMiddleware allows setting context for LLM/Tool.
type ContextMiddleware interface {
	LLMContext(ctx context.Context, state *State) (context.Context, context.CancelFunc)
	ToolContext(ctx context.Context, state *ToolState) (context.Context, context.CancelFunc)
}

// Observer provides observability callbacks.
type Observer interface {
	OnLLMStart(ctx context.Context, state *State, req *llm.Request)
	OnLLMEnd(ctx context.Context, state *State, resp *llm.Response, err error)
	OnToolCall(ctx context.Context, state *ToolState)
	OnToolResult(ctx context.Context, state *ToolState)
	OnError(ctx context.Context, err error)
}

// NoopObserver is a default no-op implementation.
type NoopObserver struct{}

func (o *NoopObserver) OnLLMStart(ctx context.Context, state *State, req *llm.Request) {}
func (o *NoopObserver) OnLLMEnd(ctx context.Context, state *State, resp *llm.Response, err error) {
}
func (o *NoopObserver) OnToolCall(ctx context.Context, state *ToolState)   {}
func (o *NoopObserver) OnToolResult(ctx context.Context, state *ToolState) {}
func (o *NoopObserver) OnError(ctx context.Context, err error)             {}

// Tracer provides a lightweight tracing interface.
type Tracer interface {
	StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, func(error))
}

// NoopTracer is a default no-op implementation.
type NoopTracer struct{}

func (t *NoopTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, func(error)) {
	return ctx, func(error) {}
}

// Run executes one message input (tool-call loop supported).
func (r *Runner) Run(ctx context.Context, ag *agent.Agent, input schema.Message) (schema.Message, error) {
	result, err := r.RunWithResult(ctx, ag, input)
	if err != nil {
		return schema.Message{}, err
	}
	return result.Message, nil
}

// RunWithResult executes and returns richer results.
func (r *Runner) RunWithResult(ctx context.Context, ag *agent.Agent, input schema.Message) (RunResult, error) {
	if ag == nil {
		return RunResult{}, fmt.Errorf("runner: agent is nil")
	}
	if r.config.Model == nil {
		return RunResult{}, fmt.Errorf("runner: model is nil")
	}

	// Run input guardrails
	if err := r.runInputGuardrails(ctx, ag, &input); err != nil {
		r.config.Observer.OnError(ctx, err)
		return RunResult{}, err
	}

	runID := schema.RunID(r.config.RunIDGenerator())
	if runID == "" {
		runID = schema.RunID(defaultRunIDGenerator())
	}

	messages := r.buildInitialMessages(ctx, ag, input)
	if err := r.config.Memory.Add(ctx, input); err != nil {
		return RunResult{}, err
	}

	return r.runWithState(ctx, ag, input, runID, messages, 1, nil, nil)
}

func (r *Runner) RunFromCheckpoint(ctx context.Context, ag *agent.Agent, checkpoint *Checkpoint) (RunResult, error) {
	if ag == nil {
		return RunResult{}, fmt.Errorf("runner: agent is nil")
	}
	if r.config.Model == nil {
		return RunResult{}, fmt.Errorf("runner: model is nil")
	}
	if checkpoint == nil {
		return RunResult{}, fmt.Errorf("runner: checkpoint is nil")
	}
	if checkpoint.RunID == "" {
		return RunResult{}, fmt.Errorf("runner: checkpoint run id is empty")
	}

	store := memory.NewBuffer(r.config.HistoryWindow)
	filtered := make([]schema.Message, 0, len(checkpoint.Messages))
	for _, msg := range checkpoint.Messages {
		if msg.Role == schema.RoleSystem {
			continue
		}
		filtered = append(filtered, msg)
	}
	if err := store.AddBatch(ctx, filtered); err != nil {
		return RunResult{}, err
	}

	run := r.WithMemory(store)
	return run.runWithState(
		ctx,
		ag,
		checkpoint.Input,
		checkpoint.RunID,
		cloneMessages(checkpoint.Messages),
		checkpoint.Turn+1,
		cloneToolCalls(checkpoint.ToolCalls),
		cloneToolResults(checkpoint.ToolResults),
	)
}

func (r *Runner) runWithState(
	ctx context.Context,
	ag *agent.Agent,
	input schema.Message,
	runID schema.RunID,
	messages []schema.Message,
	startTurn int,
	toolCalls []schema.ToolCall,
	toolResults []schema.ToolResult,
) (RunResult, error) {
	registry := tools.NewRegistry()
	for _, t := range ag.Tools() {
		if err := registry.Register(t); err != nil {
			return RunResult{}, err
		}
	}

	var lastUsage llm.TokenUsage

	for turn := startTurn; turn <= r.config.MaxTurns; turn++ {
		stepID := schema.StepID(fmt.Sprintf("%s.step.%d", runID, turn))
		llmSpanID := schema.SpanID(fmt.Sprintf("%s.llm", stepID))
		state := &State{
			Context:  ctx,
			Agent:    ag,
			Input:    input,
			Messages: messages,
			Turn:     turn,
			RunID:    runID,
			StepID:   stepID,
			SpanID:   llmSpanID,
		}
		if err := r.runBeforeLLM(ctx, state); err != nil {
			r.config.Observer.OnError(ctx, err)
			return RunResult{}, err
		}

		req := r.buildRequest(messages, ag)
		llmCtx, cancels := r.applyLLMContext(ctx, state)
		state.Context = llmCtx

		spanCtx, endSpan := r.config.Tracer.StartSpan(llmCtx, "llm.generate", map[string]string{
			"agent_id":   ag.ID(),
			"agent_name": ag.Name(),
			"turn":       fmt.Sprintf("%d", turn),
			"run_id":     string(runID),
			"step_id":    string(stepID),
			"span_id":    string(llmSpanID),
		})
		state.Context = spanCtx
		r.config.Observer.OnLLMStart(llmCtx, state, req)
		resp, err := r.callLLM(spanCtx, state, req)
		endSpan(err)
		runCancels(cancels)
		r.config.Observer.OnLLMEnd(llmCtx, state, resp, err)
		if err != nil {
			r.config.Observer.OnError(llmCtx, err)
			return RunResult{}, err
		}
		lastUsage = resp.Usage

		state.Response = resp.Message
		if err := r.runAfterLLM(ctx, state); err != nil {
			r.config.Observer.OnError(ctx, err)
			return RunResult{}, err
		}

		messages = append(messages, resp.Message)
		if err := r.config.Memory.Add(ctx, resp.Message); err != nil {
			return RunResult{}, err
		}

		if !resp.Message.HasToolCalls() {
			// Run output guardrails before returning
			if err := r.runOutputGuardrails(ctx, ag, &resp.Message); err != nil {
				r.config.Observer.OnError(ctx, err)
				return RunResult{}, err
			}
			if err := r.saveCheckpoint(ctx, runID, turn, input, messages, toolCalls, toolResults); err != nil {
				return RunResult{}, err
			}
			return RunResult{
				Message:     resp.Message,
				Usage:       lastUsage,
				ToolCalls:   toolCalls,
				ToolResults: toolResults,
			}, nil
		}

		toolCalls = append(toolCalls, resp.Message.ToolCalls...)
		toolMessages, results, err := r.executeTools(ctx, registry, ag, resp.Message.ToolCalls, runID, stepID)
		if err != nil {
			r.config.Observer.OnError(ctx, err)
			return RunResult{}, err
		}
		toolResults = append(toolResults, results...)

		for _, toolMsg := range toolMessages {
			messages = append(messages, toolMsg)
			if err := r.config.Memory.Add(ctx, toolMsg); err != nil {
				return RunResult{}, err
			}
		}

		if err := r.saveCheckpoint(ctx, runID, turn, input, messages, toolCalls, toolResults); err != nil {
			return RunResult{}, err
		}
	}

	return RunResult{}, fmt.Errorf("runner: exceeded max turns %d", r.config.MaxTurns)
}

func (r *Runner) saveCheckpoint(
	ctx context.Context,
	runID schema.RunID,
	turn int,
	input schema.Message,
	messages []schema.Message,
	toolCalls []schema.ToolCall,
	toolResults []schema.ToolResult,
) error {
	if r.config.Checkpointer == nil {
		return nil
	}
	checkpoint := Checkpoint{
		RunID:       runID,
		Turn:        turn,
		Input:       *input.Clone(),
		Messages:    cloneMessages(messages),
		ToolCalls:   cloneToolCalls(toolCalls),
		ToolResults: cloneToolResults(toolResults),
		UpdatedAt:   time.Now(),
	}
	return r.config.Checkpointer.Save(ctx, checkpoint)
}

// RunStream executes with streaming events.
func (r *Runner) RunStream(ctx context.Context, ag *agent.Agent, input schema.Message) (<-chan schema.StreamEvent, error) {
	if ag == nil {
		return nil, fmt.Errorf("runner: agent is nil")
	}
	if r.config.Model == nil {
		return nil, fmt.Errorf("runner: model is nil")
	}
	if !r.config.Model.SupportsStreaming() {
		return nil, fmt.Errorf("runner: model does not support streaming")
	}

	// Run input guardrails before streaming
	if err := r.runInputGuardrails(ctx, ag, &input); err != nil {
		r.config.Observer.OnError(ctx, err)
		return nil, err
	}

	runID := schema.RunID(r.config.RunIDGenerator())
	if runID == "" {
		runID = schema.RunID(defaultRunIDGenerator())
	}

	out := make(chan schema.StreamEvent, 128)
	go func() {
		defer close(out)

		messages := r.buildInitialMessages(ctx, ag, input)
		if err := r.config.Memory.Add(ctx, input); err != nil {
			out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, "", "")
			return
		}

		var toolCalls []schema.ToolCall
		var toolResults []schema.ToolResult

		registry := tools.NewRegistry()
		for _, t := range ag.Tools() {
			if err := registry.Register(t); err != nil {
				out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, "", "")
				return
			}
		}

		for turn := 1; turn <= r.config.MaxTurns; turn++ {
			stepID := schema.StepID(fmt.Sprintf("%s.step.%d", runID, turn))
			llmSpanID := schema.SpanID(fmt.Sprintf("%s.llm", stepID))
			state := &State{
				Context:  ctx,
				Agent:    ag,
				Input:    input,
				Messages: messages,
				Turn:     turn,
				RunID:    runID,
				StepID:   stepID,
				SpanID:   llmSpanID,
			}
			if err := r.runBeforeLLM(ctx, state); err != nil {
				r.config.Observer.OnError(ctx, err)
				out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, llmSpanID)
				return
			}

			req := r.buildRequest(messages, ag)
			llmCtx, cancels := r.applyLLMContext(ctx, state)
			state.Context = llmCtx

			spanCtx, endSpan := r.config.Tracer.StartSpan(llmCtx, "llm.stream", map[string]string{
				"agent_id":   ag.ID(),
				"agent_name": ag.Name(),
				"turn":       fmt.Sprintf("%d", turn),
				"run_id":     string(runID),
				"step_id":    string(stepID),
				"span_id":    string(llmSpanID),
			})
			state.Context = spanCtx
			r.config.Observer.OnLLMStart(llmCtx, state, req)
			stream, err := r.config.Model.GenerateStream(spanCtx, req)
			if err != nil {
				r.config.Observer.OnLLMEnd(llmCtx, state, nil, err)
				r.config.Observer.OnError(llmCtx, err)
				runCancels(cancels)
				endSpan(err)
				out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, llmSpanID)
				return
			}

			var finalMessage schema.Message
			for event := range stream {
				if event.AgentID == "" {
					event.AgentID = ag.ID()
				}
				out <- event.WithIDs(runID, stepID, llmSpanID)
				if event.Type == schema.EventEnd {
					if msg, ok := event.Data.(schema.Message); ok {
						finalMessage = msg
					}
				}
				if event.Type == schema.EventError {
					r.config.Observer.OnLLMEnd(llmCtx, state, nil, event.Error)
					runCancels(cancels)
					endSpan(event.Error)
					return
				}
			}

			if finalMessage.Role == "" {
				err := fmt.Errorf("runner: missing final message")
				r.config.Observer.OnLLMEnd(llmCtx, state, nil, err)
				runCancels(cancels)
				endSpan(err)
				out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, llmSpanID)
				return
			}

			state.Response = finalMessage
			if err := r.runAfterLLM(ctx, state); err != nil {
				r.config.Observer.OnError(ctx, err)
				runCancels(cancels)
				endSpan(err)
				out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, llmSpanID)
				return
			}
			r.config.Observer.OnLLMEnd(llmCtx, state, &llm.Response{Message: finalMessage}, nil)
			endSpan(nil)
			runCancels(cancels)

			messages = append(messages, finalMessage)
			if err := r.config.Memory.Add(ctx, finalMessage); err != nil {
				out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, llmSpanID)
				return
			}

			if !finalMessage.HasToolCalls() {
				// Run output guardrails before completing
				if err := r.runOutputGuardrails(ctx, ag, &finalMessage); err != nil {
					out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, llmSpanID)
					return
				}
				if err := r.saveCheckpoint(ctx, runID, turn, input, messages, toolCalls, toolResults); err != nil {
					out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, llmSpanID)
				}
				return
			}

			toolCalls = append(toolCalls, finalMessage.ToolCalls...)
			for i, call := range finalMessage.ToolCalls {
				spanID := schema.SpanID(fmt.Sprintf("%s.tool.%d", stepID, i))
				out <- schema.NewToolCallEvent(call, ag.ID()).WithIDs(runID, stepID, spanID)
			}

			toolMessages, results, err := r.executeTools(ctx, registry, ag, finalMessage.ToolCalls, runID, stepID)
			if err != nil {
				r.config.Observer.OnError(ctx, err)
				out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, "")
				return
			}
			toolResults = append(toolResults, results...)

			for i, toolMsg := range toolMessages {
				if i < len(results) {
					spanID := schema.SpanID(fmt.Sprintf("%s.tool.%d", stepID, i))
					out <- schema.NewToolResultEvent(results[i], ag.ID()).WithIDs(runID, stepID, spanID)
				}
				messages = append(messages, toolMsg)
				if err := r.config.Memory.Add(ctx, toolMsg); err != nil {
					out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, "")
					return
				}
			}

			if err := r.saveCheckpoint(ctx, runID, turn, input, messages, toolCalls, toolResults); err != nil {
				out <- schema.NewErrorEvent(err, ag.ID()).WithIDs(runID, stepID, "")
				return
			}
		}

		out <- schema.NewErrorEvent(fmt.Errorf("runner: exceeded max turns %d", r.config.MaxTurns), ag.ID()).WithIDs(runID, "", "")
	}()

	return out, nil
}

func (r *Runner) buildInitialMessages(ctx context.Context, ag *agent.Agent, input schema.Message) []schema.Message {
	history, err := r.config.Memory.History(ctx)
	if err != nil {
		r.config.Observer.OnError(ctx, fmt.Errorf("failed to load history: %w", err))
		history = nil // Continue with empty history
	}
	history = trimHistory(history, r.config.HistoryWindow)

	messages := make([]schema.Message, 0, len(history)+2)
	if sys := ag.SystemPrompt(); sys != "" {
		messages = append(messages, schema.Message{Role: schema.RoleSystem, Content: sys})
	}
	messages = append(messages, history...)
	messages = append(messages, input)
	return messages
}

func (r *Runner) buildRequest(messages []schema.Message, ag *agent.Agent) *llm.Request {
	req := &llm.Request{Messages: messages}
	if r.config.Model.SupportsTools() && len(ag.Tools()) > 0 {
		req.Tools = collectToolSpecs(ag.Tools())
		req.ToolChoice = &llm.ToolChoiceOption{Type: "auto"}
	}
	if r.config.ResponseFormat != nil {
		req.ResponseFormat = r.config.ResponseFormat
	}
	return req
}

func collectToolSpecs(toolList []tools.Tool) []llm.ToolSpec {
	specs := make([]llm.ToolSpec, 0, len(toolList))
	for _, t := range toolList {
		if t == nil || t.Schema() == nil {
			continue
		}
		params := map[string]interface{}{"type": "object"}
		if t.Schema().Type != "" {
			params["type"] = t.Schema().Type
		}
		if len(t.Schema().Properties) > 0 {
			params["properties"] = t.Schema().Properties
		}
		if len(t.Schema().Required) > 0 {
			params["required"] = t.Schema().Required
		}
		specs = append(specs, llm.ToolSpec{Name: t.Name(), Description: t.Description(), Parameters: params})
	}
	return specs
}

func trimHistory(history []schema.Message, window int) []schema.Message {
	if window <= 0 || len(history) <= window {
		return history
	}
	return history[len(history)-window:]
}

func (r *Runner) executeTools(ctx context.Context, registry *tools.Registry, ag *agent.Agent, calls []schema.ToolCall, runID schema.RunID, stepID schema.StepID) ([]schema.Message, []schema.ToolResult, error) {
	spanCtx, endSpan := r.config.Tracer.StartSpan(ctx, "tool.invoke", map[string]string{
		"agent_id":   ag.ID(),
		"agent_name": ag.Name(),
		"tool_count": fmt.Sprintf("%d", len(calls)),
		"run_id":     string(runID),
		"step_id":    string(stepID),
	})
	ctx = spanCtx

	for idx := range calls {
		spanID := schema.SpanID(fmt.Sprintf("%s.tool.%d", stepID, idx))
		state := &ToolState{Agent: ag, Call: &calls[idx], RunID: runID, StepID: stepID, SpanID: spanID}
		r.config.Observer.OnToolCall(ctx, state)
		if err := r.runBeforeTool(ctx, state); err != nil {
			endSpan(err)
			return nil, nil, err
		}
	}

	toolState := &ToolState{Agent: ag, Context: ctx, RunID: runID, StepID: stepID}
	toolCtx, cancels := r.applyToolContext(ctx, toolState)
	toolState.Context = toolCtx
	defer runCancels(cancels)

	results, err := r.config.ToolInvoker.Invoke(toolCtx, registry, calls)
	if err != nil {
		endSpan(err)
		return nil, nil, err
	}

	toolMessages := make([]schema.Message, 0, len(results))
	for idx := range results {
		spanID := schema.SpanID(fmt.Sprintf("%s.tool.%d", stepID, idx))
		state := &ToolState{Agent: ag, Result: &results[idx], RunID: runID, StepID: stepID, SpanID: spanID}
		if err := r.runAfterTool(ctx, state); err != nil {
			endSpan(err)
			return nil, nil, err
		}
		r.config.Observer.OnToolResult(ctx, state)

		msg := schema.Message{
			ID:      results[idx].ID,
			Role:    schema.RoleTool,
			Content: string(results[idx].Result),
		}
		if results[idx].Error != "" {
			msg.Content = results[idx].Error
			msg.SetMetadata("error", results[idx].Error)
		}
		toolMessages = append(toolMessages, msg)
	}
	endSpan(nil)
	return toolMessages, results, nil
}

func (r *Runner) callLLM(ctx context.Context, state *State, req *llm.Request) (*llm.Response, error) {
	handler := func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
		return r.config.Model.Generate(ctx, req)
	}

	for i := len(r.config.Middlewares) - 1; i >= 0; i-- {
		mw := r.config.Middlewares[i]
		llmMw, ok := mw.(LLMMiddleware)
		if !ok {
			continue
		}
		next := handler
		handler = func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
			return llmMw.HandleLLM(ctx, state, req, next)
		}
	}

	return handler(ctx, req)
}

func (r *Runner) applyLLMContext(ctx context.Context, state *State) (context.Context, []context.CancelFunc) {
	current := ctx
	cancels := make([]context.CancelFunc, 0)
	for _, mw := range r.config.Middlewares {
		cm, ok := mw.(ContextMiddleware)
		if !ok {
			continue
		}
		updated, cancel := cm.LLMContext(current, state)
		if updated != nil {
			current = updated
		}
		if cancel != nil {
			cancels = append(cancels, cancel)
		}
	}
	return current, cancels
}

func (r *Runner) applyToolContext(ctx context.Context, state *ToolState) (context.Context, []context.CancelFunc) {
	current := ctx
	cancels := make([]context.CancelFunc, 0)
	for _, mw := range r.config.Middlewares {
		cm, ok := mw.(ContextMiddleware)
		if !ok {
			continue
		}
		updated, cancel := cm.ToolContext(current, state)
		if updated != nil {
			current = updated
		}
		if cancel != nil {
			cancels = append(cancels, cancel)
		}
	}
	return current, cancels
}

func runCancels(cancels []context.CancelFunc) {
	for i := len(cancels) - 1; i >= 0; i-- {
		if cancels[i] != nil {
			cancels[i]()
		}
	}
}

func (r *Runner) runBeforeLLM(ctx context.Context, state *State) error {
	for _, mw := range r.config.Middlewares {
		if hook, ok := mw.(BeforeLLM); ok {
			if err := hook.BeforeLLM(ctx, state); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) runAfterLLM(ctx context.Context, state *State) error {
	for _, mw := range r.config.Middlewares {
		if hook, ok := mw.(AfterLLM); ok {
			if err := hook.AfterLLM(ctx, state); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) runBeforeTool(ctx context.Context, state *ToolState) error {
	for _, mw := range r.config.Middlewares {
		if hook, ok := mw.(BeforeTool); ok {
			if err := hook.BeforeTool(ctx, state); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) runAfterTool(ctx context.Context, state *ToolState) error {
	for _, mw := range r.config.Middlewares {
		if hook, ok := mw.(AfterTool); ok {
			if err := hook.AfterTool(ctx, state); err != nil {
				return err
			}
		}
	}
	return nil
}

// runInputGuardrails executes all input guardrails for the agent.
func (r *Runner) runInputGuardrails(ctx context.Context, ag *agent.Agent, input *schema.Message) error {
	for _, g := range ag.InputGuardrails() {
		result := g.ValidateInput(ctx, input)
		if !result.Passed {
			return &guardrail.GuardrailError{
				GuardrailName: g.Name(),
				Type:          "input",
				Reason:        result.Reason,
				Details:       result.Details,
			}
		}
	}
	return nil
}

// runOutputGuardrails executes all output guardrails for the agent.
func (r *Runner) runOutputGuardrails(ctx context.Context, ag *agent.Agent, output *schema.Message) error {
	for _, g := range ag.OutputGuardrails() {
		result := g.ValidateOutput(ctx, output)
		if !result.Passed {
			return &guardrail.GuardrailError{
				GuardrailName: g.Name(),
				Type:          "output",
				Reason:        result.Reason,
				Details:       result.Details,
			}
		}
	}
	return nil
}
