package manager

import (
	"context"
	"errors"

	"github.com/voocel/mas/executor/sandbox"
	"github.com/voocel/mas/executor/sandbox/audit"
	"github.com/voocel/mas/executor/sandbox/auth"
	"github.com/voocel/mas/executor/sandbox/policy"
	"github.com/voocel/mas/executor/sandbox/runtime"
)

type Service struct {
	Evaluator policy.Evaluator
	Runtime   runtime.Runtime
	Auditor   audit.Logger
	Auth      auth.Authenticator
}

func (s *Service) CreateSandbox(ctx context.Context, req sandbox.CreateSandboxRequest) (*sandbox.CreateSandboxResponse, error) {
	if s == nil || s.Runtime == nil {
		return nil, errors.New("runtime is nil")
	}
	return s.Runtime.CreateSandbox(ctx, req)
}

func (s *Service) ExecuteTool(ctx context.Context, req sandbox.ExecuteToolRequest) (*sandbox.ExecuteToolResponse, error) {
	if s == nil || s.Runtime == nil {
		return nil, errors.New("runtime is nil")
	}

	if s.Evaluator != nil {
		decision, err := s.Evaluator.Evaluate(ctx, req)
		if err != nil {
			return nil, err
		}
		if !decision.Allowed {
			resp := &sandbox.ExecuteToolResponse{
				ToolCallID: req.ToolCallID,
				Status:     sandbox.StatusError,
				Error: &sandbox.ErrorDetail{
					Code:    sandbox.CodePolicyDenied,
					Message: decision.Reason,
				},
				ExitCode: 1,
			}
			s.record(ctx, req, "deny", resp.Status, resp.Error.Message)
			return resp, nil
		}
	}

	resp, err := s.Runtime.ExecuteTool(ctx, req)
	if resp == nil {
		resp = &sandbox.ExecuteToolResponse{
			ToolCallID: req.ToolCallID,
			Status:     sandbox.StatusError,
			Error:      &sandbox.ErrorDetail{Code: sandbox.CodeInternal, Message: "runtime error"},
			ExitCode:   1,
		}
	}
	s.record(ctx, req, "allow", resp.Status, errorMessage(resp.Error))
	return resp, err
}

func (s *Service) DestroySandbox(ctx context.Context, req sandbox.DestroySandboxRequest) (*sandbox.DestroySandboxResponse, error) {
	if s == nil || s.Runtime == nil {
		return nil, errors.New("runtime is nil")
	}
	return s.Runtime.DestroySandbox(ctx, req)
}

func (s *Service) Health(ctx context.Context) (*sandbox.HealthResponse, error) {
	return &sandbox.HealthResponse{Status: sandbox.StatusOK, Version: "v0"}, nil
}

func (s *Service) record(ctx context.Context, req sandbox.ExecuteToolRequest, decision, status, errMsg string) {
	if s == nil || s.Auditor == nil {
		return
	}
	s.Auditor.Record(ctx, audit.Record{
		RunID:    req.Trace.RunID,
		Tool:     req.Tool.Name,
		Decision: decision,
		Status:   status,
		Error:    errMsg,
	})
}

func errorMessage(detail *sandbox.ErrorDetail) string {
	if detail == nil {
		return ""
	}
	return detail.Message
}
