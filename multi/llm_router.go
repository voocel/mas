package multi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/schema"
)

// agentSelection is the structured output for LLM agent selection.
type agentSelection struct {
	AgentName string `json:"agent_name"`
	Reason    string `json:"reason"`
}

// LLMRouter uses an LLM to intelligently select the best agent for a task.
type LLMRouter struct {
	Model   llm.ChatModel
	Default string // Fallback agent name if LLM selection fails
	// OnFallback is called when falling back to default agent. Optional.
	OnFallback func(err error, defaultAgent string)
}

// NewLLMRouter creates a new LLM-driven router.
func NewLLMRouter(model llm.ChatModel, defaultAgent string) *LLMRouter {
	return &LLMRouter{Model: model, Default: defaultAgent}
}

// Select uses the LLM to choose the most appropriate agent.
func (r *LLMRouter) Select(ctx context.Context, input schema.Message, team *Team) (*agent.Agent, error) {
	if r.Model == nil {
		return nil, fmt.Errorf("llm_router: model is nil")
	}

	agents := r.buildAgentList(team)
	if len(agents) == 0 {
		return nil, fmt.Errorf("llm_router: no agents in team")
	}

	if len(agents) == 1 {
		for name := range agents {
			return team.Route(name)
		}
	}

	// Call LLM for selection
	agentName, err := r.selectViaLLM(ctx, input.Content, agents)
	if err != nil {
		return r.fallback(team, err)
	}

	if _, ok := agents[agentName]; !ok {
		return r.fallback(team, fmt.Errorf("selected agent %q not found", agentName))
	}

	return team.Route(agentName)
}

func (r *LLMRouter) fallback(team *Team, originalErr error) (*agent.Agent, error) {
	if r.Default != "" {
		if r.OnFallback != nil {
			r.OnFallback(originalErr, r.Default)
		}
		return team.Route(r.Default)
	}
	return nil, fmt.Errorf("llm_router: %w", originalErr)
}

func (r *LLMRouter) selectViaLLM(ctx context.Context, userInput string, agents map[string]string) (string, error) {
	prompt := r.buildPrompt(userInput, agents)
	req := &llm.Request{
		Messages:       []schema.Message{{Role: schema.RoleUser, Content: prompt}},
		ResponseFormat: &llm.ResponseFormat{Type: "json_object"},
	}

	resp, err := r.Model.Generate(ctx, req)
	if err != nil {
		return "", err
	}

	var selection agentSelection
	if err := json.Unmarshal([]byte(resp.Message.Content), &selection); err != nil {
		return "", err
	}
	if selection.AgentName == "" {
		return "", fmt.Errorf("empty agent_name in response")
	}
	return selection.AgentName, nil
}

func (r *LLMRouter) buildAgentList(team *Team) map[string]string {
	agents := make(map[string]string)
	for _, name := range team.List() {
		ag, ok := team.Get(name)
		if !ok {
			continue
		}
		desc := ag.HandoffDescription()
		if desc == "" {
			desc = ag.SystemPrompt()
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
		}
		if desc == "" {
			desc = ag.Name()
		}
		agents[name] = desc
	}
	return agents
}

func (r *LLMRouter) buildPrompt(userInput string, agents map[string]string) string {
	var sb strings.Builder
	sb.WriteString("Select the best agent for this request.\n\nAgents:\n")

	names := make([]string, 0, len(agents))
	for name := range agents {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", name, agents[name]))
	}

	sb.WriteString("\nRequest: ")
	sb.WriteString(userInput)
	sb.WriteString("\n\nRespond: {\"agent_name\": \"<name>\", \"reason\": \"<why>\"}")
	return sb.String()
}
