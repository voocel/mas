package multi

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

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
	return NewTransferToolWithName(target, transferToolName(target))
}

func NewTransferToolWithName(target, name string) *TransferTool {
	if strings.TrimSpace(name) == "" {
		name = transferToolName(target)
	}
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
	names := team.List()
	counts := make(map[string]int, len(names))
	for _, name := range names {
		if excludeName != "" && name == excludeName {
			continue
		}
		base := transferToolBaseName(name)
		counts[base]++
	}
	for _, name := range names {
		if excludeName != "" && name == excludeName {
			continue
		}
		base := transferToolBaseName(name)
		toolName := base
		if counts[base] > 1 {
			toolName = base + "_" + shortHash(name)
		}
		if _, exists := toolMap[toolName]; exists {
			continue
		}
		toolMap[toolName] = name
		toolsList = append(toolsList, NewTransferToolWithName(name, toolName))
	}
	return toolsList, toolMap
}

func transferToolName(target string) string {
	return transferToolBaseName(target)
}

func transferToolBaseName(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return schema.TransferToolPrefix + "unknown"
	}
	normalized := strings.ToLower(target)
	normalized = invalidToolChars.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		normalized = "agent"
	}
	return schema.TransferToolPrefix + normalized
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

func shortHash(value string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return fmt.Sprintf("%08x", h.Sum32())
}
