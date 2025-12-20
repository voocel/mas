package multi

import (
	"fmt"
	"strings"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/schema"
)

// Router selects an agent based on input.
type Router interface {
	Select(input schema.Message, team *Team) (*agent.Agent, error)
}

// FixedRouter always returns a fixed agent.
type FixedRouter struct {
	Name string
}

func (r *FixedRouter) Select(_ schema.Message, team *Team) (*agent.Agent, error) {
	return team.Route(r.Name)
}

// KeywordRouter routes based on keywords.
type KeywordRouter struct {
	Rules         map[string]string
	Default       string
	CaseSensitive bool
}

func (r *KeywordRouter) Select(input schema.Message, team *Team) (*agent.Agent, error) {
	content := input.Content
	if !r.CaseSensitive {
		content = strings.ToLower(content)
	}

	for keyword, name := range r.Rules {
		kw := keyword
		if !r.CaseSensitive {
			kw = strings.ToLower(keyword)
		}
		if strings.Contains(content, kw) {
			return team.Route(name)
		}
	}

	if r.Default != "" {
		return team.Route(r.Default)
	}
	return nil, fmt.Errorf("router: no match and no default")
}
