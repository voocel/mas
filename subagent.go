package agentcore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/voocel/agentcore/schema"
)

const (
	maxParallelTasks = 8
	maxConcurrency   = 4
)

// SubAgentConfig defines a sub-agent's identity and capabilities.
type SubAgentConfig struct {
	Name         string
	Description  string
	Model        ChatModel
	SystemPrompt string
	Tools        []Tool
	StreamFn     StreamFn
	MaxTurns     int
}

// subagentParams is the JSON schema input for the subagent tool.
// Three mutually exclusive modes:
//   - Single: Agent + Task
//   - Parallel: Tasks array
//   - Chain: Chain array with {previous} placeholder
type subagentParams struct {
	Agent string          `json:"agent,omitempty"`
	Task  string          `json:"task,omitempty"`
	Tasks []subagentTask  `json:"tasks,omitempty"`
	Chain []subagentChain `json:"chain,omitempty"`
}

type subagentTask struct {
	Agent string `json:"agent"`
	Task  string `json:"task"`
}

type subagentChain struct {
	Agent string `json:"agent"`
	Task  string `json:"task"`
}

// subagentResult captures one sub-agent's execution outcome.
type subagentResult struct {
	Agent   string `json:"agent"`
	Task    string `json:"task"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error,omitempty"`
	Step    int    `json:"step,omitempty"`
}

// SubAgentTool implements the Tool interface.
// The main agent calls this tool to delegate tasks to specialized sub-agents
// with isolated contexts
type SubAgentTool struct {
	agents map[string]SubAgentConfig
}

// NewSubAgentTool creates a subagent tool from a set of agent configs.
func NewSubAgentTool(agents ...SubAgentConfig) *SubAgentTool {
	m := make(map[string]SubAgentConfig, len(agents))
	for _, a := range agents {
		m[a.Name] = a
	}
	return &SubAgentTool{agents: m}
}

func (t *SubAgentTool) Name() string  { return "subagent" }
func (t *SubAgentTool) Label() string { return "Delegate to SubAgent" }

func (t *SubAgentTool) Description() string {
	names := make([]string, 0, len(t.agents))
	for _, a := range t.agents {
		names = append(names, fmt.Sprintf("%s (%s)", a.Name, a.Description))
	}
	return fmt.Sprintf(
		"Delegate tasks to specialized subagents with isolated context. "+
			"Modes: single (agent + task), parallel (tasks array), chain (sequential with {previous} placeholder). "+
			"Available agents: %s",
		strings.Join(names, ", "),
	)
}

func (t *SubAgentTool) Schema() map[string]any {
	agentNames := make([]string, 0, len(t.agents))
	for name := range t.agents {
		agentNames = append(agentNames, name)
	}
	taskItem := schema.Object(
		schema.Property("agent", schema.Enum("Agent name", agentNames...)).Required(),
		schema.Property("task", schema.String("Task description")).Required(),
	)
	return schema.Object(
		schema.Property("agent", schema.Enum("Name of the agent to invoke (single mode)", agentNames...)),
		schema.Property("task", schema.String("Task to delegate (single mode)")),
		schema.Property("tasks", schema.Array("Array of {agent, task} for parallel execution", taskItem)),
		schema.Property("chain", schema.Array("Array of {agent, task} for sequential execution. Use {previous} in task to reference prior output.", taskItem)),
	)
}

func (t *SubAgentTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var params subagentParams
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid subagent params: %w", err)
	}

	hasChain := len(params.Chain) > 0
	hasParallel := len(params.Tasks) > 0
	hasSingle := params.Agent != "" && params.Task != ""
	modeCount := boolToInt(hasChain) + boolToInt(hasParallel) + boolToInt(hasSingle)

	if modeCount != 1 {
		return json.Marshal("Invalid parameters: provide exactly one mode (agent+task, tasks, or chain)")
	}

	switch {
	case hasChain:
		return t.executeChain(ctx, params.Chain)
	case hasParallel:
		return t.executeParallel(ctx, params.Tasks)
	default:
		return t.executeSingle(ctx, params.Agent, params.Task)
	}
}

// executeSingle runs one sub-agent with an isolated context.
func (t *SubAgentTool) executeSingle(ctx context.Context, agentName, task string) (json.RawMessage, error) {
	output, err := t.runAgent(ctx, agentName, task)
	if err != nil {
		return json.Marshal(fmt.Sprintf("Agent %q failed: %v", agentName, err))
	}
	return json.Marshal(output)
}

// executeChain runs sub-agents sequentially, passing each output to the next via {previous}.
func (t *SubAgentTool) executeChain(ctx context.Context, chain []subagentChain) (json.RawMessage, error) {
	var previous string
	results := make([]subagentResult, 0, len(chain))

	for i, step := range chain {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		task := strings.ReplaceAll(step.Task, "{previous}", previous)
		output, err := t.runAgent(ctx, step.Agent, task)

		result := subagentResult{
			Agent: step.Agent,
			Task:  task,
			Step:  i + 1,
		}

		if err != nil {
			result.Output = err.Error()
			result.IsError = true
			results = append(results, result)
			return json.Marshal(map[string]any{
				"error":   fmt.Sprintf("Chain stopped at step %d (%s): %v", i+1, step.Agent, err),
				"results": results,
			})
		}

		result.Output = output
		results = append(results, result)
		previous = output
	}

	return json.Marshal(map[string]any{
		"output":  previous,
		"results": results,
	})
}

// executeParallel runs multiple sub-agents concurrently with bounded concurrency.
func (t *SubAgentTool) executeParallel(ctx context.Context, tasks []subagentTask) (json.RawMessage, error) {
	if len(tasks) > maxParallelTasks {
		return json.Marshal(fmt.Sprintf("Too many parallel tasks (%d). Max is %d.", len(tasks), maxParallelTasks))
	}

	results := make([]subagentResult, len(tasks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, st subagentTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			output, err := t.runAgent(ctx, st.Agent, st.Task)
			result := subagentResult{
				Agent: st.Agent,
				Task:  st.Task,
			}
			if err != nil {
				result.Output = err.Error()
				result.IsError = true
			} else {
				result.Output = output
			}
			results[idx] = result
		}(i, task)
	}

	wg.Wait()

	successCount := 0
	for _, r := range results {
		if !r.IsError {
			successCount++
		}
	}

	return json.Marshal(map[string]any{
		"summary": fmt.Sprintf("%d/%d succeeded", successCount, len(results)),
		"results": results,
	})
}

// runAgent executes an isolated agent loop for the given agent config and task.
// Returns the final assistant output text.
func (t *SubAgentTool) runAgent(ctx context.Context, agentName, task string) (string, error) {
	cfg, ok := t.agents[agentName]
	if !ok {
		available := make([]string, 0, len(t.agents))
		for name := range t.agents {
			available = append(available, name)
		}
		return "", fmt.Errorf("unknown agent %q, available: %s", agentName, strings.Join(available, ", "))
	}

	userMsg := UserMsg(task)

	agentCtx := AgentContext{
		SystemPrompt: cfg.SystemPrompt,
		Tools:        cfg.Tools,
	}

	loopCfg := LoopConfig{
		Model:    cfg.Model,
		StreamFn: cfg.StreamFn,
		MaxTurns: cfg.MaxTurns,
	}
	if loopCfg.MaxTurns <= 0 {
		loopCfg.MaxTurns = defaultMaxTurns
	}

	events := AgentLoop(ctx, []AgentMessage{userMsg}, agentCtx, loopCfg)

	// Consume events, extract final assistant output
	var lastAssistantContent string
	var lastErr error

	for ev := range events {
		switch ev.Type {
		case EventMessageEnd:
			if ev.Message != nil && ev.Message.GetRole() == RoleAssistant {
				lastAssistantContent = ev.Message.TextContent()
			}
		case EventError:
			if ev.Err != nil {
				lastErr = ev.Err
			}
		}
	}

	if lastErr != nil && lastAssistantContent == "" {
		return "", lastErr
	}

	if lastAssistantContent == "" {
		return "(no output)", nil
	}

	return lastAssistantContent, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
