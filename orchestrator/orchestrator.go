package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

type Orchestrator struct {
	cfg config
}

type StepResult struct {
	Step   string `json:"step"`
	Output string `json:"output"`
}

type Reflection struct {
	Done      bool     `json:"done"`
	Reason    string   `json:"reason,omitempty"`
	NextSteps []string `json:"next_steps,omitempty"`
}

type Result struct {
	Goal       string       `json:"goal"`
	Plan       []string     `json:"plan"`
	Steps      []StepResult `json:"steps"`
	Reflection Reflection   `json:"reflection"`
	Iterations int          `json:"iterations"`
}

type config struct {
	model                 llm.ChatModel
	planModel             llm.ChatModel
	reflectModel          llm.ChatModel
	runner                *runner.Runner
	agent                 *agent.Agent
	tools                 []tools.Tool
	agentID               string
	agentName             string
	agentSystemPrompt     string
	plannerSystemPrompt   string
	reflectorSystemPrompt string
	maxPlanSteps          int
	maxIterations         int
	maxStepResultChars    int
}

type Option func(*config)

func WithRunner(r *runner.Runner) Option {
	return func(cfg *config) {
		cfg.runner = r
	}
}

func WithAgent(ag *agent.Agent) Option {
	return func(cfg *config) {
		cfg.agent = ag
	}
}

func WithTools(toolList ...tools.Tool) Option {
	return func(cfg *config) {
		cfg.tools = append(cfg.tools, toolList...)
	}
}

func WithAgentID(id string) Option {
	return func(cfg *config) {
		cfg.agentID = id
	}
}

func WithAgentName(name string) Option {
	return func(cfg *config) {
		cfg.agentName = name
	}
}

func WithAgentSystemPrompt(prompt string) Option {
	return func(cfg *config) {
		cfg.agentSystemPrompt = prompt
	}
}

func WithPlannerSystemPrompt(prompt string) Option {
	return func(cfg *config) {
		cfg.plannerSystemPrompt = prompt
	}
}

func WithReflectorSystemPrompt(prompt string) Option {
	return func(cfg *config) {
		cfg.reflectorSystemPrompt = prompt
	}
}

func WithPlannerModel(model llm.ChatModel) Option {
	return func(cfg *config) {
		cfg.planModel = model
	}
}

func WithReflectorModel(model llm.ChatModel) Option {
	return func(cfg *config) {
		cfg.reflectModel = model
	}
}

func WithMaxPlanSteps(n int) Option {
	return func(cfg *config) {
		cfg.maxPlanSteps = n
	}
}

func WithMaxIterations(n int) Option {
	return func(cfg *config) {
		cfg.maxIterations = n
	}
}

func WithMaxStepResultChars(n int) Option {
	return func(cfg *config) {
		cfg.maxStepResultChars = n
	}
}

func New(model llm.ChatModel, opts ...Option) (*Orchestrator, error) {
	if model == nil {
		return nil, errors.New("orchestrator: model is nil")
	}
	cfg := config{
		model:                 model,
		plannerSystemPrompt:   defaultPlannerSystemPrompt(),
		reflectorSystemPrompt: defaultReflectorSystemPrompt(),
		maxPlanSteps:          6,
		maxIterations:         2,
		maxStepResultChars:    2000,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.planModel == nil {
		cfg.planModel = cfg.model
	}
	if cfg.reflectModel == nil {
		cfg.reflectModel = cfg.model
	}
	if cfg.agent == nil {
		agentID := cfg.agentID
		if agentID == "" {
			agentID = "assistant"
		}
		agentName := cfg.agentName
		if agentName == "" {
			agentName = "assistant"
		}
		cfg.agent = agent.New(
			agentID,
			agentName,
			agent.WithSystemPrompt(cfg.agentSystemPrompt),
			agent.WithTools(cfg.tools...),
		)
	}
	if cfg.runner == nil {
		cfg.runner = runner.New(runner.Config{Model: cfg.model})
	}
	return &Orchestrator{cfg: cfg}, nil
}

func (o *Orchestrator) Run(ctx context.Context, goal string) (Result, error) {
	if o == nil {
		return Result{}, errors.New("orchestrator: nil receiver")
	}
	if strings.TrimSpace(goal) == "" {
		return Result{}, errors.New("orchestrator: goal is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	plan, err := o.plan(ctx, goal, "")
	if err != nil {
		return Result{}, err
	}

	var reflection Reflection
	var stepResults []StepResult

	for iter := 0; iter < o.cfg.maxIterations; iter++ {
		stepResults = stepResults[:0]
		for i, step := range plan {
			input := o.buildStepInput(goal, plan, i, step)
			msg, err := o.cfg.runner.Run(ctx, o.cfg.agent, schema.Message{
				Role:    schema.RoleUser,
				Content: input,
			})
			if err != nil {
				return Result{
					Goal:       goal,
					Plan:       plan,
					Steps:      stepResults,
					Reflection: reflection,
					Iterations: iter + 1,
				}, err
			}
			stepResults = append(stepResults, StepResult{
				Step:   step,
				Output: truncate(msg.Content, o.cfg.maxStepResultChars),
			})
		}

		reflection, err = o.reflect(ctx, goal, plan, stepResults)
		if err != nil {
			return Result{
				Goal:       goal,
				Plan:       plan,
				Steps:      stepResults,
				Reflection: reflection,
				Iterations: iter + 1,
			}, err
		}
		if reflection.Done {
			return Result{
				Goal:       goal,
				Plan:       plan,
				Steps:      stepResults,
				Reflection: reflection,
				Iterations: iter + 1,
			}, nil
		}

		if len(reflection.NextSteps) > 0 {
			plan = normalizeSteps(reflection.NextSteps, o.cfg.maxPlanSteps)
			if len(plan) == 0 {
				return Result{
					Goal:       goal,
					Plan:       plan,
					Steps:      stepResults,
					Reflection: reflection,
					Iterations: iter + 1,
				}, errors.New("orchestrator: empty plan after reflection")
			}
			continue
		}

		plan, err = o.plan(ctx, goal, reflection.Reason)
		if err != nil {
			return Result{
				Goal:       goal,
				Plan:       plan,
				Steps:      stepResults,
				Reflection: reflection,
				Iterations: iter + 1,
			}, err
		}
	}

	return Result{
		Goal:       goal,
		Plan:       plan,
		Steps:      stepResults,
		Reflection: reflection,
		Iterations: o.cfg.maxIterations,
	}, nil
}

func (o *Orchestrator) plan(ctx context.Context, goal string, feedback string) ([]string, error) {
	messages := []schema.Message{
		{Role: schema.RoleSystem, Content: o.cfg.plannerSystemPrompt},
		{Role: schema.RoleUser, Content: o.buildPlanPrompt(goal, feedback)},
	}
	resp, err := o.cfg.planModel.Generate(ctx, &llm.Request{Messages: messages})
	if err != nil {
		return nil, err
	}
	steps := parsePlan(resp.Message.Content, o.cfg.maxPlanSteps)
	if len(steps) == 0 {
		return nil, errors.New("orchestrator: empty plan")
	}
	return steps, nil
}

func (o *Orchestrator) reflect(ctx context.Context, goal string, plan []string, results []StepResult) (Reflection, error) {
	messages := []schema.Message{
		{Role: schema.RoleSystem, Content: o.cfg.reflectorSystemPrompt},
		{Role: schema.RoleUser, Content: o.buildReflectionPrompt(goal, plan, results)},
	}
	resp, err := o.cfg.reflectModel.Generate(ctx, &llm.Request{Messages: messages})
	if err != nil {
		return Reflection{}, err
	}
	reflection, ok := parseReflection(resp.Message.Content)
	if !ok {
		return Reflection{
			Done:   true,
			Reason: "reflection parse failed",
		}, nil
	}
	reflection.NextSteps = normalizeSteps(reflection.NextSteps, o.cfg.maxPlanSteps)
	return reflection, nil
}

func (o *Orchestrator) buildPlanPrompt(goal string, feedback string) string {
	builder := strings.Builder{}
	builder.WriteString("Goal: ")
	builder.WriteString(strings.TrimSpace(goal))
	builder.WriteString("\n")
	if strings.TrimSpace(feedback) != "" {
		builder.WriteString("Feedback: ")
		builder.WriteString(strings.TrimSpace(feedback))
		builder.WriteString("\n")
	}
	builder.WriteString("Break the goal into executable steps and output JSON: {\"steps\":[\"...\"]}.\n")
	builder.WriteString(fmt.Sprintf("Requirements: no more than %d steps, each step must be concrete and actionable.\n", o.cfg.maxPlanSteps))
	builder.WriteString("Output JSON only, no extra text.")
	return builder.String()
}

func (o *Orchestrator) buildReflectionPrompt(goal string, plan []string, results []StepResult) string {
	builder := strings.Builder{}
	builder.WriteString("Goal: ")
	builder.WriteString(strings.TrimSpace(goal))
	builder.WriteString("\n")
	builder.WriteString("Plan:\n")
	for i, step := range plan {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}
	builder.WriteString("Execution results:\n")
	for i, res := range results {
		builder.WriteString(fmt.Sprintf("%d. Step: %s\n", i+1, res.Step))
		builder.WriteString(fmt.Sprintf("   Output: %s\n", strings.TrimSpace(res.Output)))
	}
	builder.WriteString("Decide whether the goal is completed and output JSON: {\"done\":true|false,\"reason\":\"...\",\"next_steps\":[\"...\"]}\n")
	builder.WriteString(fmt.Sprintf("Requirements: if not done, next_steps must be no more than %d.\n", o.cfg.maxPlanSteps))
	builder.WriteString("Output JSON only, no extra text.")
	return builder.String()
}

func (o *Orchestrator) buildStepInput(goal string, plan []string, index int, step string) string {
	builder := strings.Builder{}
	builder.WriteString("Goal: ")
	builder.WriteString(strings.TrimSpace(goal))
	builder.WriteString("\n")
	builder.WriteString("Plan:\n")
	for i, s := range plan {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("Current step (%d/%d): %s\n", index+1, len(plan), strings.TrimSpace(step)))
	builder.WriteString("Execute only the current step, provide verifiable results, and call tools if needed.")
	return builder.String()
}

type planEnvelope struct {
	Steps []string `json:"steps"`
}

type reflectionEnvelope struct {
	Done      bool     `json:"done"`
	Reason    string   `json:"reason"`
	NextSteps []string `json:"next_steps"`
}

func parsePlan(content string, maxSteps int) []string {
	content = stripCodeFence(content)
	if steps, ok := parsePlanJSON(content); ok {
		return normalizeSteps(steps, maxSteps)
	}
	lines := parseStepsFromLines(content)
	return normalizeSteps(lines, maxSteps)
}

func parsePlanJSON(content string) ([]string, bool) {
	if steps, ok := decodePlan(content); ok {
		return steps, true
	}
	if jsonCandidate := extractJSONCandidate(content, '{', '}'); jsonCandidate != "" {
		if steps, ok := decodePlan(jsonCandidate); ok {
			return steps, true
		}
	}
	if jsonCandidate := extractJSONCandidate(content, '[', ']'); jsonCandidate != "" {
		if steps, ok := decodePlan(jsonCandidate); ok {
			return steps, true
		}
	}
	return nil, false
}

func decodePlan(content string) ([]string, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || !json.Valid([]byte(trimmed)) {
		return nil, false
	}
	var envelope planEnvelope
	if err := json.Unmarshal([]byte(trimmed), &envelope); err == nil && len(envelope.Steps) > 0 {
		return envelope.Steps, true
	}
	var list []string
	if err := json.Unmarshal([]byte(trimmed), &list); err == nil && len(list) > 0 {
		return list, true
	}
	return nil, false
}

func parseReflection(content string) (Reflection, bool) {
	content = stripCodeFence(content)
	if reflection, ok := decodeReflection(content); ok {
		return reflection, true
	}
	if jsonCandidate := extractJSONCandidate(content, '{', '}'); jsonCandidate != "" {
		if reflection, ok := decodeReflection(jsonCandidate); ok {
			return reflection, true
		}
	}
	return Reflection{}, false
}

func decodeReflection(content string) (Reflection, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || !json.Valid([]byte(trimmed)) {
		return Reflection{}, false
	}
	var envelope reflectionEnvelope
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return Reflection{}, false
	}
	return Reflection{
		Done:      envelope.Done,
		Reason:    strings.TrimSpace(envelope.Reason),
		NextSteps: envelope.NextSteps,
	}, true
}

func normalizeSteps(steps []string, maxSteps int) []string {
	if len(steps) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(steps))
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		clean := strings.TrimSpace(step)
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
		if maxSteps > 0 && len(out) >= maxSteps {
			break
		}
	}
	return out
}

func parseStepsFromLines(content string) []string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		clean := stripLeadingBullet(line)
		if clean == "" {
			continue
		}
		out = append(out, clean)
	}
	return out
}

func stripLeadingBullet(line string) string {
	s := strings.TrimSpace(line)
	if s == "" {
		return ""
	}
	for len(s) > 0 {
		switch s[0] {
		case '-', '*':
			s = strings.TrimSpace(s[1:])
			continue
		default:
		}
		break
	}
	// 去掉前置序号，如 "1." 或 "2)"
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) {
		if s[i] == '.' || s[i] == ')' {
			s = strings.TrimSpace(s[i+1:])
		}
	}
	return strings.TrimSpace(s)
}

func stripCodeFence(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	last := strings.LastIndex(trimmed, "```")
	if last <= 0 {
		return trimmed
	}
	inner := strings.TrimSpace(trimmed[3:last])
	if nl := strings.Index(inner, "\n"); nl != -1 {
		firstLine := strings.TrimSpace(inner[:nl])
		if isSimpleLangLabel(firstLine) {
			inner = strings.TrimSpace(inner[nl+1:])
		}
	}
	return inner
}

func isSimpleLangLabel(text string) bool {
	if text == "" || len(text) > 16 {
		return false
	}
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			continue
		}
		return false
	}
	return true
}

func extractJSONCandidate(content string, open, close byte) string {
	start := strings.IndexByte(content, open)
	end := strings.LastIndexByte(content, close)
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(content[start : end+1])
}

func truncate(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

func defaultPlannerSystemPrompt() string {
	return "You are a planning assistant who breaks goals into clear, actionable steps."
}

func defaultReflectorSystemPrompt() string {
	return "You are a reflection assistant who decides if the goal is complete and suggests improvements."
}
