package policy

import (
	"context"

	"github.com/voocel/mas/executor/sandbox"
)

type Decision struct {
	Allowed bool
	Reason  string
}

type Evaluator interface {
	Evaluate(ctx context.Context, req sandbox.ExecuteToolRequest) (Decision, error)
}
