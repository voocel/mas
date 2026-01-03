package multi

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

const transferToolPrefix = "transfer_to_"

type transferArgs struct {
	Reason  string                 `json:"reason,omitempty"`
	Message string                 `json:"message,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type TransferTool struct {
	*tools.BaseTool
	Target string
}

func NewTransferTool(target string) *TransferTool {
	name := transferToolName(target)
	schemaDef := tools.CreateToolSchema(
		fmt.Sprintf("Transfer control to agent %s", target),
		map[string]interface{}{
			"reason":  tools.StringProperty("Reason for transferring to this agent"),
			"message": tools.StringProperty("Message for the target agent"),
			"payload": tools.ObjectProperty("Additional payload data", map[string]interface{}{}),
		},
		nil,
	)
	baseTool := tools.NewBaseTool(name, "Agent handoff tool", schemaDef)
	cfg := *tools.DefaultToolConfig
	cfg.Sandbox = false
	baseTool.SetConfig(&cfg)
	return &TransferTool{BaseTool: baseTool, Target: target}
}

func (t *TransferTool) Execute(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args transferArgs
	_ = json.Unmarshal(input, &args)
	h := &schema.Handoff{
		Target:  t.Target,
		Reason:  args.Reason,
		Message: args.Message,
		Payload: args.Payload,
	}
	return json.Marshal(map[string]interface{}{"handoff": h})
}

func buildTransferTools(team *Team, excludeName string) ([]tools.Tool, map[string]string) {
	if team == nil {
		return nil, nil
	}
	toolsList := make([]tools.Tool, 0)
	toolMap := make(map[string]string)
	excludeName = strings.TrimSpace(excludeName)
	for _, name := range team.List() {
		if excludeName != "" && name == excludeName {
			continue
		}
		toolName := transferToolName(name)
		if _, exists := toolMap[toolName]; exists {
			continue
		}
		toolMap[toolName] = name
		toolsList = append(toolsList, NewTransferTool(name))
	}
	return toolsList, toolMap
}

func transferToolName(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return transferToolPrefix + "unknown"
	}
	normalized := strings.ToLower(target)
	normalized = invalidToolChars.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		normalized = "agent"
	}
	return transferToolPrefix + normalized
}

var invalidToolChars = regexp.MustCompile(`[^a-z0-9_-]+`)

func parseTransferCall(call schema.ToolCall, target string) *schema.Handoff {
	if target == "" {
		return nil
	}
	var args transferArgs
	_ = json.Unmarshal(call.Args, &args)
	h := &schema.Handoff{
		Target:  target,
		Reason:  args.Reason,
		Message: args.Message,
		Payload: args.Payload,
	}
	if h.Priority == 0 {
		h.Priority = 5
	}
	if h.IsValid() {
		return h
	}
	return nil
}
