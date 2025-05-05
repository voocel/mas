package agency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/orchestrator"
)

// Tool represents a tool that agents can use
type Tool struct {
	Name        string
	Description string
	Handler     ToolHandler
}

// ToolHandler is the handler function type for tools
type ToolHandler func(ctx context.Context, params map[string]interface{}, agent *agent.Agent) (string, error)

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content string `json:"content"`
}

// UnmarshalParams unmarshals a JSON string into a parameter map
func UnmarshalParams(paramsJSON string) (map[string]interface{}, error) {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, err
	}
	return params, nil
}

// SendMessageTool defines an inter-agent communication tool
type SendMessageTool struct {
	agency *Agency
	sender agent.Agent
}

// NewSendMessageTool creates a send message tool
func NewSendMessageTool(agency *Agency, sender agent.Agent) *SendMessageTool {
	return &SendMessageTool{
		agency: agency,
		sender: sender,
	}
}

// Name returns the tool name
func (t *SendMessageTool) Name() string {
	return "send_message"
}

// Description returns the tool description
func (t *SendMessageTool) Description() string {
	return "Send a message to other agents"
}

// SendMessageParams parameters for the send message tool
type SendMessageParams struct {
	Recipient string `json:"recipient" binding:"required"`
	Content   string `json:"content" binding:"required"`
	WaitReply bool   `json:"wait_reply,omitempty"`
}

// Execute executes the tool
func (t *SendMessageTool) Execute(ctx context.Context, paramsJSON string) (string, error) {
	var params SendMessageParams
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return "", err
	}

	// Verify recipient exists
	_, err := t.agency.GetAgent(params.Recipient)
	if err != nil {
		return "", err
	}

	// Check communication permission
	if !t.agency.FlowChart.CanCommunicate(t.sender.Name(), params.Recipient) {
		return "", fmt.Errorf("agent %s is not allowed to communicate with %s", t.sender.Name(), params.Recipient)
	}

	// Create task
	task := orchestrator.Task{
		Name:        fmt.Sprintf("Process message from %s", t.sender.Name()),
		Description: "Process inter-agent message",
		AgentIDs:    []string{params.Recipient},
		Input:       params.Content,
		Metadata: map[string]interface{}{
			"sender": t.sender.Name(),
		},
	}

	// Submit task
	taskID, err := t.agency.Orchestrator.SubmitTask(ctx, task)
	if err != nil {
		return "", err
	}

	if !params.WaitReply {
		return fmt.Sprintf("Message sent to %s, task ID: %s", params.Recipient, taskID), nil
	}

	// Wait for reply
	for {
		task, err := t.agency.Orchestrator.GetTask(taskID)
		if err != nil {
			return "", err
		}

		if task.Status == orchestrator.TaskStatusCompleted {
			result, ok := task.Output.(string)
			if !ok {
				return "", fmt.Errorf("task output is not a string")
			}
			return fmt.Sprintf("Reply from %s: %s", params.Recipient, result), nil
		} else if task.Status == orchestrator.TaskStatusFailed {
			return "", fmt.Errorf("task failed: %s", task.Error)
		}

		// Wait for a while before checking again
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue checking
		}
	}
}

// GetAvailableRecipientsTool tool for getting available communication recipients
type GetAvailableRecipientsTool struct {
	agency *Agency
	sender agent.Agent
}

// NewGetAvailableRecipientsTool creates a tool for getting available communication recipients
func NewGetAvailableRecipientsTool(agency *Agency, sender agent.Agent) *GetAvailableRecipientsTool {
	return &GetAvailableRecipientsTool{
		agency: agency,
		sender: sender,
	}
}

// Name returns the tool name
func (t *GetAvailableRecipientsTool) Name() string {
	return "get_available_recipients"
}

// Description returns the tool description
func (t *GetAvailableRecipientsTool) Description() string {
	return "Get all other agents the current agent can communicate with"
}

// Execute executes the tool
func (t *GetAvailableRecipientsTool) Execute(ctx context.Context, paramsJSON string) (string, error) {
	recipients := t.agency.FlowChart.GetReceivers(t.sender.Name())

	if len(recipients) == 0 {
		return "No agents available for communication", nil
	}

	result := "Available agents for communication:\n"
	for _, recipient := range recipients {
		_, err := t.agency.GetAgent(recipient)
		if err == nil {
			result += fmt.Sprintf("- %s\n", recipient)
		}
	}

	return result, nil
}

// SendMessage sends a message to the specified agent
func (a *Agency) SendMessage(ctx context.Context, agentID string, content string) (string, error) {
	// Check if the agent exists
	_, err := a.GetAgent(agentID)
	if err != nil {
		return "", err
	}

	// Create a task to send a message
	task := orchestrator.Task{
		Name:        "Send Message",
		Description: fmt.Sprintf("Send message to agent %s", agentID),
		AgentIDs:    []string{agentID},
		Input:       content,
	}

	// Submit task to orchestrator
	taskID, err := a.Orchestrator.SubmitTask(ctx, task)
	if err != nil {
		return "", err
	}

	// Wait for task completion
	for {
		task, err := a.Orchestrator.GetTask(taskID)
		if err != nil {
			return "", err
		}

		if task.Status == orchestrator.TaskStatusCompleted {
			result, ok := task.Output.(string)
			if !ok {
				return "", fmt.Errorf("task output is not a string")
			}
			return result, nil
		} else if task.Status == orchestrator.TaskStatusFailed {
			return "", fmt.Errorf("task failed: %s", task.Error)
		}

		// Wait for a while before checking again
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue checking
		}
	}
}

// GetAgentInfo gets information about an agent
func (a *Agency) GetAgentInfo(ctx context.Context, agentID string) (string, error) {
	agent, err := a.GetAgent(agentID)
	if err != nil {
		return "", err
	}

	info := map[string]interface{}{
		"id":   agentID,
		"name": agent.Name(),
	}

	jsonData, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}
