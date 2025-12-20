package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
)

type HITLAction int

const (
	HITLAllow HITLAction = iota
	HITLInterrupt
)

type HITLDecision struct {
	Action HITLAction
	Reason string
}

func Allow() HITLDecision {
	return HITLDecision{Action: HITLAllow}
}

func Interrupt(reason string) HITLDecision {
	return HITLDecision{Action: HITLInterrupt, Reason: reason}
}

type HITLApprover interface {
	ApproveLLM(ctx context.Context, state *runner.State, req *llm.Request) HITLDecision
	ApproveTool(ctx context.Context, state *runner.ToolState) HITLDecision
}

type HITLFunc struct {
	LLM  func(ctx context.Context, state *runner.State, req *llm.Request) HITLDecision
	Tool func(ctx context.Context, state *runner.ToolState) HITLDecision
}

func (f HITLFunc) ApproveLLM(ctx context.Context, state *runner.State, req *llm.Request) HITLDecision {
	if f.LLM == nil {
		return Allow()
	}
	return f.LLM(ctx, state, req)
}

func (f HITLFunc) ApproveTool(ctx context.Context, state *runner.ToolState) HITLDecision {
	if f.Tool == nil {
		return Allow()
	}
	return f.Tool(ctx, state)
}

type HITLError struct {
	Stage   string
	RunID   schema.RunID
	StepID  schema.StepID
	SpanID  schema.SpanID
	AgentID string
	Tool    string
	Reason  string
}

func (e *HITLError) Error() string {
	if e == nil {
		return "hitl: interrupt"
	}
	msg := "hitl: interrupt"
	if e.Stage != "" {
		msg += fmt.Sprintf(" stage=%s", e.Stage)
	}
	if e.Tool != "" {
		msg += fmt.Sprintf(" tool=%s", e.Tool)
	}
	if e.RunID != "" {
		msg += fmt.Sprintf(" run=%s", e.RunID)
	}
	if e.StepID != "" {
		msg += fmt.Sprintf(" step=%s", e.StepID)
	}
	if e.Reason != "" {
		msg += fmt.Sprintf(" reason=%s", e.Reason)
	}
	return msg
}

func IsHITLInterrupt(err error) bool {
	var hitlErr *HITLError
	return errors.As(err, &hitlErr)
}

type HITLMiddleware struct {
	Approver HITLApprover
}

func (m *HITLMiddleware) HandleLLM(ctx context.Context, state *runner.State, req *llm.Request, next runner.LLMHandler) (*llm.Response, error) {
	if m == nil || m.Approver == nil {
		return next(ctx, req)
	}
	decision := m.Approver.ApproveLLM(ctx, state, req)
	if decision.Action == HITLAllow {
		return next(ctx, req)
	}
	return nil, m.newError("llm", state, nil, decision.Reason)
}

func (m *HITLMiddleware) BeforeTool(ctx context.Context, state *runner.ToolState) error {
	if m == nil || m.Approver == nil {
		return nil
	}
	decision := m.Approver.ApproveTool(ctx, state)
	if decision.Action == HITLAllow {
		return nil
	}
	return m.newError("tool", nil, state, decision.Reason)
}

func (m *HITLMiddleware) newError(stage string, state *runner.State, toolState *runner.ToolState, reason string) error {
	err := &HITLError{
		Stage:  stage,
		Reason: reason,
	}
	if state != nil {
		err.RunID = state.RunID
		err.StepID = state.StepID
		err.SpanID = state.SpanID
		if state.Agent != nil {
			err.AgentID = state.Agent.ID()
		}
	}
	if toolState != nil {
		err.RunID = toolState.RunID
		err.StepID = toolState.StepID
		err.SpanID = toolState.SpanID
		if toolState.Agent != nil {
			err.AgentID = toolState.Agent.ID()
		}
		if toolState.Call != nil {
			err.Tool = toolState.Call.Name
		}
	}
	return err
}

var _ runner.LLMMiddleware = (*HITLMiddleware)(nil)
var _ runner.BeforeTool = (*HITLMiddleware)(nil)
