package sandbox

import "errors"

const (
	CodePolicyDenied   = "policy_denied"
	CodeInvalidRequest = "invalid_request"
	CodeToolFailed     = "tool_failed"
	CodeSandboxCrash   = "sandbox_crash"
	CodeInternal       = "internal"
)

var (
	ErrPolicyDenied   = errors.New("sandbox policy denied")
	ErrInvalidRequest = errors.New("sandbox invalid request")
	ErrToolFailed     = errors.New("sandbox tool failed")
	ErrSandboxCrash   = errors.New("sandbox crashed")
	ErrInternal       = errors.New("sandbox internal error")
)

func ErrorFromDetail(detail *ErrorDetail) error {
	if detail == nil || detail.Code == "" {
		return nil
	}
	switch detail.Code {
	case CodePolicyDenied:
		return ErrPolicyDenied
	case CodeInvalidRequest:
		return ErrInvalidRequest
	case CodeToolFailed:
		return ErrToolFailed
	case CodeSandboxCrash:
		return ErrSandboxCrash
	default:
		return ErrInternal
	}
}
