package agentcore

import (
	"context"
	"encoding/json"
	"time"
)

// ProxyEventType identifies proxy streaming event types.
// Proxy events are bandwidth-optimized: they carry only deltas,
// not the full partial message on each event.
type ProxyEventType string

const (
	ProxyEventTextDelta     ProxyEventType = "text_delta"
	ProxyEventThinkingDelta ProxyEventType = "thinking_delta"
	ProxyEventToolCallStart ProxyEventType = "toolcall_start"
	ProxyEventToolCallDelta ProxyEventType = "toolcall_delta"
	ProxyEventDone          ProxyEventType = "done"
	ProxyEventError         ProxyEventType = "error"
)

// ProxyEvent is a bandwidth-optimized event from a remote proxy server.
// The client reconstructs the full message incrementally from these deltas.
type ProxyEvent struct {
	Type       ProxyEventType `json:"type"`
	Delta      string         `json:"delta,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`
	StopReason StopReason     `json:"stop_reason,omitempty"`
	Usage      *Usage         `json:"usage,omitempty"`
	Err        error          `json:"-"`
}

// ProxyStreamFn makes an LLM call through a remote proxy
// and returns a channel of bandwidth-optimized ProxyEvents.
type ProxyStreamFn func(ctx context.Context, req *LLMRequest) (<-chan ProxyEvent, error)

// ProxyModel implements ChatModel by forwarding to a remote proxy server.
// It reconstructs streaming events from bandwidth-optimized ProxyEvents.
//
// Usage:
//
//	proxy := agentcore.NewProxyModel(myProxyFn)
//	agent := agentcore.NewAgent(agentcore.WithModel(proxy))
type ProxyModel struct {
	streamFn ProxyStreamFn
}

// NewProxyModel creates a ChatModel that delegates to a proxy stream function.
func NewProxyModel(fn ProxyStreamFn) *ProxyModel {
	return &ProxyModel{streamFn: fn}
}

// Generate collects the full streamed response synchronously.
func (p *ProxyModel) Generate(ctx context.Context, messages []Message, tools []ToolSpec, opts ...CallOption) (*LLMResponse, error) {
	ch, err := p.GenerateStream(ctx, messages, tools, opts...)
	if err != nil {
		return nil, err
	}
	var final Message
	for ev := range ch {
		switch ev.Type {
		case StreamEventDone:
			final = ev.Message
		case StreamEventError:
			return nil, ev.Err
		}
	}
	return &LLMResponse{Message: final}, nil
}

// GenerateStream converts proxy events into standard StreamEvents.
func (p *ProxyModel) GenerateStream(ctx context.Context, messages []Message, tools []ToolSpec, opts ...CallOption) (<-chan StreamEvent, error) {
	proxyCh, err := p.streamFn(ctx, &LLMRequest{Messages: messages, Tools: tools})
	if err != nil {
		return nil, err
	}

	eventCh := make(chan StreamEvent, 100)
	go func() {
		defer close(eventCh)

		var (
			partial      = Message{Role: RoleAssistant}
			textStarted  bool
			thinkStarted bool
		)

		for ev := range proxyCh {
			switch ev.Type {
			case ProxyEventTextDelta:
				idx := proxyFindOrCreate(&partial.Content, ContentText)
				partial.Content[idx].Text += ev.Delta
				if !textStarted {
					textStarted = true
					eventCh <- StreamEvent{Type: StreamEventTextStart, ContentIndex: idx, Message: partial}
				}
				eventCh <- StreamEvent{Type: StreamEventTextDelta, ContentIndex: idx, Delta: ev.Delta, Message: partial}

			case ProxyEventThinkingDelta:
				idx := proxyFindOrCreate(&partial.Content, ContentThinking)
				partial.Content[idx].Thinking += ev.Delta
				if !thinkStarted {
					thinkStarted = true
					eventCh <- StreamEvent{Type: StreamEventThinkingStart, ContentIndex: idx, Message: partial}
				}
				eventCh <- StreamEvent{Type: StreamEventThinkingDelta, ContentIndex: idx, Delta: ev.Delta, Message: partial}

			case ProxyEventToolCallStart:
				partial.Content = append(partial.Content, ToolCallBlock(ToolCall{
					ID:   ev.ToolCallID,
					Name: ev.ToolName,
				}))
				eventCh <- StreamEvent{Type: StreamEventToolCallStart, Message: partial}

			case ProxyEventToolCallDelta:
				if idx := proxyLastToolCall(partial.Content); idx >= 0 && partial.Content[idx].ToolCall != nil {
					partial.Content[idx].ToolCall.Args = append(partial.Content[idx].ToolCall.Args, json.RawMessage(ev.Delta)...)
				}
				eventCh <- StreamEvent{Type: StreamEventToolCallDelta, Delta: ev.Delta, Message: partial}

			case ProxyEventDone:
				partial.StopReason = ev.StopReason
				partial.Usage = ev.Usage
				partial.Timestamp = time.Now()
				eventCh <- StreamEvent{Type: StreamEventDone, Message: partial, StopReason: ev.StopReason}

			case ProxyEventError:
				eventCh <- StreamEvent{Type: StreamEventError, Err: ev.Err}
				return
			}
		}
	}()

	return eventCh, nil
}

// SupportsTools reports that the proxy can handle tool calls.
func (p *ProxyModel) SupportsTools() bool { return true }

// proxyFindOrCreate returns the index of the last block of the given type,
// or appends a new empty block and returns its index.
func proxyFindOrCreate(blocks *[]ContentBlock, ct ContentType) int {
	for i := len(*blocks) - 1; i >= 0; i-- {
		if (*blocks)[i].Type == ct {
			return i
		}
	}
	switch ct {
	case ContentText:
		*blocks = append(*blocks, TextBlock(""))
	case ContentThinking:
		*blocks = append(*blocks, ThinkingBlock(""))
	}
	return len(*blocks) - 1
}

// proxyLastToolCall returns the index of the last tool call block.
func proxyLastToolCall(blocks []ContentBlock) int {
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].Type == ContentToolCall {
			return i
		}
	}
	return -1
}
