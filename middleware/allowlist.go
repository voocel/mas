package middleware

import (
	"context"
	"fmt"

	"github.com/voocel/mas/runner"
)

// ToolAllowlist allows only specified tools to execute.
type ToolAllowlist struct {
	Allowed map[string]struct{}
}

// NewToolAllowlist creates a tool allowlist.
func NewToolAllowlist(names ...string) *ToolAllowlist {
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name != "" {
			allowed[name] = struct{}{}
		}
	}
	return &ToolAllowlist{Allowed: allowed}
}

func (m *ToolAllowlist) BeforeTool(_ context.Context, state *runner.ToolState) error {
	if m == nil || len(m.Allowed) == 0 {
		return nil
	}
	if state == nil || state.Call == nil {
		return nil
	}
	if _, ok := m.Allowed[state.Call.Name]; !ok {
		return fmt.Errorf("tool not allowed: %s", state.Call.Name)
	}
	return nil
}

var _ runner.BeforeTool = (*ToolAllowlist)(nil)
