package observer

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
)

// JSONObserver outputs structured JSON logs.
type JSONObserver struct {
	logger *log.Logger
}

// NewJSONObserver creates a JSONObserver.
func NewJSONObserver(out io.Writer) *JSONObserver {
	if out == nil {
		out = io.Discard
	}
	return &JSONObserver{logger: log.New(out, "", 0)}
}

func (o *JSONObserver) OnLLMStart(ctx context.Context, state *runner.State, req *llm.Request) {
	o.log("llm_start", map[string]interface{}{
		"agent_id":   state.Agent.ID(),
		"agent_name": state.Agent.Name(),
		"turn":       state.Turn,
		"messages":   len(state.Messages),
	})
}

func (o *JSONObserver) OnLLMEnd(ctx context.Context, state *runner.State, resp *llm.Response, err error) {
	fields := map[string]interface{}{
		"agent_id":   state.Agent.ID(),
		"agent_name": state.Agent.Name(),
		"turn":       state.Turn,
	}
	if err != nil {
		fields["error"] = err.Error()
		o.log("llm_error", fields)
		return
	}
	if resp != nil {
		fields["content_len"] = len(resp.Message.Content)
		fields["tool_calls"] = len(resp.Message.ToolCalls)
	}
	o.log("llm_end", fields)
}

func (o *JSONObserver) OnToolCall(ctx context.Context, state *runner.ToolState) {
	if state == nil || state.Call == nil {
		return
	}
	o.log("tool_call", map[string]interface{}{
		"tool": state.Call.Name,
		"id":   state.Call.ID,
	})
}

func (o *JSONObserver) OnToolResult(ctx context.Context, state *runner.ToolState) {
	if state == nil || state.Result == nil {
		return
	}
	fields := map[string]interface{}{
		"id": state.Result.ID,
	}
	if state.Result.Error != "" {
		fields["error"] = state.Result.Error
		o.log("tool_error", fields)
		return
	}
	fields["size"] = len(state.Result.Result)
	o.log("tool_result", fields)
}

func (o *JSONObserver) OnError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	o.log("error", map[string]interface{}{
		"error": err.Error(),
	})
}

func (o *JSONObserver) log(event string, fields map[string]interface{}) {
	payload := map[string]interface{}{
		"ts":    time.Now().Format(time.RFC3339Nano),
		"event": event,
	}
	for k, v := range fields {
		payload[k] = v
	}
	data, err := json.Marshal(payload)
	if err != nil {
		o.logger.Printf("{\"event\":\"error\",\"error\":\"%s\"}", err.Error())
		return
	}
	o.logger.Print(string(data))
}

var _ runner.Observer = (*JSONObserver)(nil)
