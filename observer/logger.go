package observer

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
)

// LoggerObserver provides basic log output.
type LoggerObserver struct {
	logger *log.Logger
}

// NewLoggerObserver creates a LoggerObserver.
func NewLoggerObserver(out io.Writer) *LoggerObserver {
	if out == nil {
		out = io.Discard
	}
	return &LoggerObserver{
		logger: log.New(out, "mas ", log.LstdFlags|log.Lmicroseconds),
	}
}

func (o *LoggerObserver) OnLLMStart(ctx context.Context, state *runner.State, req *llm.Request) {
	o.logger.Printf(
		"llm start agent=%s turn=%d messages=%d run_id=%s step_id=%s span_id=%s",
		state.Agent.Name(),
		state.Turn,
		len(state.Messages),
		state.RunID,
		state.StepID,
		state.SpanID,
	)
}

func (o *LoggerObserver) OnLLMEnd(ctx context.Context, state *runner.State, resp *llm.Response, err error) {
	if err != nil {
		o.logger.Printf(
			"llm error agent=%s turn=%d run_id=%s step_id=%s span_id=%s err=%v",
			state.Agent.Name(),
			state.Turn,
			state.RunID,
			state.StepID,
			state.SpanID,
			err,
		)
		return
	}
	contentLen := 0
	if resp != nil {
		contentLen = len(resp.Message.Content)
	}
	o.logger.Printf(
		"llm end agent=%s turn=%d content_len=%d run_id=%s step_id=%s span_id=%s",
		state.Agent.Name(),
		state.Turn,
		contentLen,
		state.RunID,
		state.StepID,
		state.SpanID,
	)
}

func (o *LoggerObserver) OnToolCall(ctx context.Context, state *runner.ToolState) {
	if state == nil || state.Call == nil {
		return
	}
	o.logger.Printf(
		"tool call name=%s id=%s run_id=%s step_id=%s span_id=%s",
		state.Call.Name,
		state.Call.ID,
		state.RunID,
		state.StepID,
		state.SpanID,
	)
}

func (o *LoggerObserver) OnToolResult(ctx context.Context, state *runner.ToolState) {
	if state == nil || state.Result == nil {
		return
	}
	err := state.Result.Error
	if err != "" {
		o.logger.Printf(
			"tool result id=%s run_id=%s step_id=%s span_id=%s err=%s",
			state.Result.ID,
			state.RunID,
			state.StepID,
			state.SpanID,
			err,
		)
		return
	}
	o.logger.Printf(
		"tool result id=%s size=%d run_id=%s step_id=%s span_id=%s",
		state.Result.ID,
		len(state.Result.Result),
		state.RunID,
		state.StepID,
		state.SpanID,
	)
}

func (o *LoggerObserver) OnError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	o.logger.Printf("error %v", err)
}

var _ runner.Observer = (*LoggerObserver)(nil)

// SimpleTimerTracer provides minimal duration tracing.
type SimpleTimerTracer struct {
	logger *log.Logger
}

// NewSimpleTimerTracer creates a tracer.
func NewSimpleTimerTracer(out io.Writer) *SimpleTimerTracer {
	if out == nil {
		out = io.Discard
	}
	return &SimpleTimerTracer{
		logger: log.New(out, "mas ", log.LstdFlags|log.Lmicroseconds),
	}
}

func (t *SimpleTimerTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, func(error)) {
	start := time.Now()
	attrText := ""
	if len(attrs) > 0 {
		attrText = fmt.Sprintf(" attrs=%v", attrs)
	}
	t.logger.Printf("span start %s%s", name, attrText)
	return ctx, func(err error) {
		if err != nil {
			t.logger.Printf("span end %s err=%v duration=%s", name, err, time.Since(start))
			return
		}
		t.logger.Printf("span end %s duration=%s", name, time.Since(start))
	}
}

var _ runner.Tracer = (*SimpleTimerTracer)(nil)
