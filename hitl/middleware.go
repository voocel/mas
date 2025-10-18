package hitl

import (
	"fmt"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// ToolCallMiddleware wraps agent execution with HITL approval for tool calls
type ToolCallMiddleware struct {
	manager *Manager
	timeout time.Duration
}

// NewToolCallMiddleware creates a new tool call middleware
func NewToolCallMiddleware(manager *Manager, timeout time.Duration) *ToolCallMiddleware {
	return &ToolCallMiddleware{
		manager: manager,
		timeout: timeout,
	}
}

// WrapAgent wraps an agent with HITL approval for tool calls
func (m *ToolCallMiddleware) WrapAgent(ag agent.Agent) agent.Agent {
	return &hitlWrappedAgent{
		agent:      ag,
		middleware: m,
	}
}

// hitlWrappedAgent wraps an agent with HITL approval
type hitlWrappedAgent struct {
	agent      agent.Agent
	middleware *ToolCallMiddleware
}

func (w *hitlWrappedAgent) ID() string {
	return w.agent.ID()
}

func (w *hitlWrappedAgent) Name() string {
	return w.agent.Name()
}

func (w *hitlWrappedAgent) Execute(ctx runtime.Context, input schema.Message) (schema.Message, error) {
	// Check if tool calls need approval
	response, err := w.agent.Execute(ctx, input)
	if err != nil {
		return response, err
	}

	// Check for tool calls that need approval
	if response.HasToolCalls() {
		for _, toolCall := range response.ToolCalls {
			// Check if this tool call needs approval
			approval, err := w.middleware.manager.CheckApproval(
				ctx,
				TriggerBeforeToolCall,
				toolCall,
			)
			if err != nil {
				return schema.Message{}, fmt.Errorf("failed to check approval: %w", err)
			}

			if approval != nil {
				// Wait for human decision
				decision, err := w.middleware.manager.WaitForDecision(
					approval.ID,
					w.middleware.timeout,
				)
				if err != nil {
					return schema.Message{}, fmt.Errorf("approval failed: %w", err)
				}

				// Handle decision
				if decision.DecisionType == DecisionReject {
					return schema.Message{
						Role:    schema.RoleAssistant,
						Content: fmt.Sprintf("Tool call rejected by %s: %s", decision.ApprovedBy, decision.Reason),
					}, nil
				}

				if decision.DecisionType == DecisionModify {
					// Apply modifications to tool call
					if modifiedArgs, ok := decision.ModifiedData["args"]; ok {
						toolCall.Args = modifiedArgs.([]byte)
					}
				}
			}
		}
	}

	return response, nil
}

func (w *hitlWrappedAgent) ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error) {
	return w.agent.ExecuteStream(ctx, input)
}

func (w *hitlWrappedAgent) ExecuteWithHandoff(ctx runtime.Context, input schema.Message) (schema.Message, *schema.Handoff, error) {
	response, handoff, err := w.agent.ExecuteWithHandoff(ctx, input)
	if err != nil {
		return response, handoff, err
	}

	// Check if handoff needs approval
	if handoff != nil && handoff.Target != "" {
		approval, err := w.middleware.manager.CheckApproval(
			ctx,
			TriggerBeforeHandoff,
			handoff,
		)
		if err != nil {
			return schema.Message{}, nil, fmt.Errorf("failed to check handoff approval: %w", err)
		}

		if approval != nil {
			decision, err := w.middleware.manager.WaitForDecision(
				approval.ID,
				w.middleware.timeout,
			)
			if err != nil {
				return schema.Message{}, nil, fmt.Errorf("handoff approval failed: %w", err)
			}

			if decision.DecisionType == DecisionReject {
				return response, nil, nil // Cancel handoff
			}

			if decision.DecisionType == DecisionModify {
				if newTarget, ok := decision.ModifiedData["target"].(string); ok {
					handoff.Target = newTarget
				}
			}
		}
	}

	return response, handoff, nil
}

func (w *hitlWrappedAgent) Tools() []tools.Tool {
	return w.agent.Tools()
}

func (w *hitlWrappedAgent) Capabilities() []agent.Capability {
	return w.agent.Capabilities()
}

func (w *hitlWrappedAgent) GetCapabilities() *agent.AgentCapabilities {
	return w.agent.GetCapabilities()
}

func (w *hitlWrappedAgent) GetModel() llm.ChatModel {
	return w.agent.GetModel()
}

func (w *hitlWrappedAgent) GetSystemPrompt() string {
	return w.agent.GetSystemPrompt()
}

func (w *hitlWrappedAgent) CanHandoff() bool {
	return w.agent.CanHandoff()
}

// CostThresholdPolicy creates a policy that triggers approval when cost exceeds threshold
func CostThresholdPolicy(threshold float64, timeout time.Duration) PolicyConfig {
	return PolicyConfig{
		Trigger: TriggerCostThreshold,
		Condition: func(ctx runtime.Context, data interface{}) bool {
			if cost, ok := data.(float64); ok {
				return cost > threshold
			}
			return false
		},
		Priority:  8,
		Timeout:   timeout,
		Approvers: []string{"admin"},
	}
}

// HighRiskToolPolicy creates a policy for high-risk tools
func HighRiskToolPolicy(riskTools []string, timeout time.Duration) PolicyConfig {
	return PolicyConfig{
		Trigger: TriggerBeforeToolCall,
		Condition: func(ctx runtime.Context, data interface{}) bool {
			if toolCall, ok := data.(schema.ToolCall); ok {
				for _, riskTool := range riskTools {
					if toolCall.Name == riskTool {
						return true
					}
				}
			}
			return false
		},
		Priority:  9,
		Timeout:   timeout,
		Approvers: []string{"security_team"},
	}
}
