package mas

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const defaultMaxTurns = 10

// AgentLoop starts an agent loop with new prompt messages.
// Prompts are added to context and events are emitted for them.
func AgentLoop(ctx context.Context, prompts []AgentMessage, agentCtx AgentContext, config LoopConfig) <-chan Event {
	ch := make(chan Event, 128)

	go func() {
		defer close(ch)

		newMessages := make([]AgentMessage, len(prompts))
		copy(newMessages, prompts)

		currentCtx := AgentContext{
			SystemPrompt: agentCtx.SystemPrompt,
			Messages:     append(copyMessages(agentCtx.Messages), prompts...),
			Tools:        agentCtx.Tools,
		}

		emit(ch, Event{Type: EventAgentStart})
		emit(ch, Event{Type: EventTurnStart})

		for _, p := range prompts {
			emit(ch, Event{Type: EventMessageStart, Message: p})
			emit(ch, Event{Type: EventMessageEnd, Message: p})
		}

		runLoop(ctx, &currentCtx, &newMessages, config, ch)
	}()

	return ch
}

// AgentLoopContinue continues from existing context without adding new messages.
// The last message in context must convert to user or tool role via ConvertToLLM.
func AgentLoopContinue(ctx context.Context, agentCtx AgentContext, config LoopConfig) <-chan Event {
	ch := make(chan Event, 128)

	if len(agentCtx.Messages) == 0 {
		go func() {
			defer close(ch)
			emitError(ch, fmt.Errorf("cannot continue: no messages in context"))
		}()
		return ch
	}

	go func() {
		defer close(ch)

		var newMessages []AgentMessage
		currentCtx := AgentContext{
			SystemPrompt: agentCtx.SystemPrompt,
			Messages:     copyMessages(agentCtx.Messages),
			Tools:        agentCtx.Tools,
		}

		emit(ch, Event{Type: EventAgentStart})
		emit(ch, Event{Type: EventTurnStart})

		runLoop(ctx, &currentCtx, &newMessages, config, ch)
	}()

	return ch
}

// runLoop is the main double-loop logic shared by AgentLoop and AgentLoopContinue.
func runLoop(ctx context.Context, currentCtx *AgentContext, newMessages *[]AgentMessage, config LoopConfig, ch chan<- Event) {
	maxTurns := config.MaxTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}

	firstTurn := true
	turnCount := 0

	// Check for steering messages at start
	var pendingMessages []AgentMessage
	if config.GetSteeringMessages != nil {
		pendingMessages = config.GetSteeringMessages()
	}

	// Outer loop: continues when follow-up messages arrive after agent would stop
	for {
		hasMoreToolCalls := true
		var steeringAfterTools []AgentMessage

		// Inner loop: process tool calls and steering messages
		for hasMoreToolCalls || len(pendingMessages) > 0 {
			// Check for context cancellation (Abort)
			if ctx.Err() != nil {
				emit(ch, Event{Type: EventError, Err: ctx.Err()})
				emit(ch, Event{Type: EventAgentEnd, Data: *newMessages})
				return
			}

			if turnCount >= maxTurns {
				emit(ch, Event{Type: EventError, Err: fmt.Errorf("max turns (%d) reached", maxTurns)})
				emit(ch, Event{Type: EventAgentEnd, Data: *newMessages})
				return
			}

			if !firstTurn {
				emit(ch, Event{Type: EventTurnStart})
			} else {
				firstTurn = false
			}

			// Process pending messages (inject before next LLM call)
			if len(pendingMessages) > 0 {
				for _, msg := range pendingMessages {
					emit(ch, Event{Type: EventMessageStart, Message: msg})
					emit(ch, Event{Type: EventMessageEnd, Message: msg})
					currentCtx.Messages = append(currentCtx.Messages, msg)
					*newMessages = append(*newMessages, msg)
				}
				pendingMessages = nil
			}

			// Call LLM (streaming: events emitted inside callLLM)
			assistantMsg, err := callLLM(ctx, currentCtx, config, ch)
			if err != nil {
				emitError(ch, fmt.Errorf("llm call failed: %w", err))
				return
			}

			currentCtx.Messages = append(currentCtx.Messages, assistantMsg)
			*newMessages = append(*newMessages, assistantMsg)

			// Check stop reason — terminate early on error/aborted
			if assistantMsg.StopReason == "error" || assistantMsg.StopReason == "aborted" {
				emit(ch, Event{Type: EventTurnEnd, Message: assistantMsg})
				emit(ch, Event{Type: EventAgentEnd, Data: *newMessages})
				return
			}

			// Check for tool calls
			toolCalls := assistantMsg.ToolCalls
			hasMoreToolCalls = len(toolCalls) > 0

			var turnToolResults []ToolResult
			if hasMoreToolCalls {
				var steering []AgentMessage
				turnToolResults, steering = executeToolCalls(ctx, currentCtx.Tools, toolCalls, config, ch)

				for _, tr := range turnToolResults {
					resultMsg := toolResultToMessage(tr)
					emit(ch, Event{Type: EventMessageStart, Message: resultMsg})
					emit(ch, Event{Type: EventMessageEnd, Message: resultMsg})
					currentCtx.Messages = append(currentCtx.Messages, resultMsg)
					*newMessages = append(*newMessages, resultMsg)
				}

				steeringAfterTools = steering
			}

			emit(ch, Event{Type: EventTurnEnd, Message: assistantMsg, ToolResults: turnToolResults})
			turnCount++

			// Get steering messages after turn completes
			if len(steeringAfterTools) > 0 {
				pendingMessages = steeringAfterTools
				steeringAfterTools = nil
			} else if config.GetSteeringMessages != nil {
				pendingMessages = config.GetSteeringMessages()
			}
		}

		// Agent would stop here. Check for follow-up messages.
		if config.GetFollowUpMessages != nil {
			followUp := config.GetFollowUpMessages()
			if len(followUp) > 0 {
				pendingMessages = followUp
				continue
			}
		}

		break
	}

	emit(ch, Event{Type: EventAgentEnd, Data: *newMessages})
}

// callLLM applies the two-stage pipeline and calls the model.
// It streams assistant responses via message_start/update/end events
func callLLM(ctx context.Context, agentCtx *AgentContext, config LoopConfig, ch chan<- Event) (Message, error) {
	messages := agentCtx.Messages

	// Stage 1: TransformContext (AgentMessage[] → AgentMessage[])
	if config.TransformContext != nil {
		var err error
		messages, err = config.TransformContext(ctx, messages)
		if err != nil {
			return Message{}, fmt.Errorf("transform context: %w", err)
		}
	}

	// Stage 2: ConvertToLLM (AgentMessage[] → Message[])
	convertFn := config.ConvertToLLM
	if convertFn == nil {
		convertFn = DefaultConvertToLLM
	}
	llmMessages := convertFn(messages)

	// Build tool specs
	toolSpecs := buildToolSpecs(agentCtx.Tools)

	// Prepend system prompt as first message if set
	if agentCtx.SystemPrompt != "" {
		systemMsg := Message{
			Role:    RoleSystem,
			Content: agentCtx.SystemPrompt,
		}
		llmMessages = append([]Message{systemMsg}, llmMessages...)
	}

	// Call via StreamFn (non-streaming shortcut, e.g. mock/proxy)
	if config.StreamFn != nil {
		resp, err := config.StreamFn(ctx, &LLMRequest{
			Messages: llmMessages,
			Tools:    toolSpecs,
		})
		if err != nil {
			return Message{}, err
		}
		resp.Message.Timestamp = time.Now()
		msg := resp.Message
		emit(ch, Event{Type: EventMessageStart, Message: msg})
		emit(ch, Event{Type: EventMessageEnd, Message: msg})
		return msg, nil
	}

	if config.Model == nil {
		return Message{}, fmt.Errorf("no model configured")
	}

	// Use streaming for real-time token deltas (drives TUI/UI)
	return callLLMStream(ctx, config.Model, llmMessages, toolSpecs, ch)
}

// callLLMStream uses GenerateStream and emits real-time message_start/update/end events.
func callLLMStream(ctx context.Context, model ChatModel, messages []Message, tools []ToolSpec, ch chan<- Event) (Message, error) {
	streamCh, err := model.GenerateStream(ctx, messages, tools)
	if err != nil {
		// Fallback to non-streaming
		resp, err := model.Generate(ctx, messages, tools)
		if err != nil {
			return Message{}, err
		}
		resp.Message.Timestamp = time.Now()
		return resp.Message, nil
	}

	var (
		started bool
		partial = Message{Role: RoleAssistant, Timestamp: time.Now()}
	)

	for ev := range streamCh {
		switch ev.Type {
		case StreamEventToken:
			if !started {
				started = true
				emit(ch, Event{Type: EventMessageStart, Message: partial})
			}
			partial.Content += ev.Delta
			emit(ch, Event{Type: EventMessageUpdate, Message: partial, Delta: ev.Delta})

		case StreamEventDone:
			finalMsg := ev.Message
			finalMsg.Timestamp = time.Now()
			if !started {
				emit(ch, Event{Type: EventMessageStart, Message: finalMsg})
			}
			emit(ch, Event{Type: EventMessageEnd, Message: finalMsg})
			return finalMsg, nil

		case StreamEventError:
			return Message{}, ev.Err
		}
	}

	// Stream closed without done event — use partial
	partial.Timestamp = time.Now()
	if !started {
		emit(ch, Event{Type: EventMessageStart, Message: partial})
	}
	emit(ch, Event{Type: EventMessageEnd, Message: partial})
	return partial, nil
}

// executeToolCalls runs tool calls sequentially, checking steering after each.
func executeToolCalls(ctx context.Context, tools []Tool, calls []ToolCall, config LoopConfig, ch chan<- Event) ([]ToolResult, []AgentMessage) {
	results := make([]ToolResult, 0, len(calls))

	for i, call := range calls {
		tool := findTool(tools, call.Name)
		label := toolLabel(tool)

		emit(ch, Event{
			Type:      EventToolExecStart,
			ToolID:    call.ID,
			Tool:      call.Name,
			ToolLabel: label,
			Args:      call.Args,
		})

		var result ToolResult

		if tool == nil {
			errContent, _ := json.Marshal(fmt.Sprintf("tool %q not found", call.Name))
			result = ToolResult{
				ToolCallID: call.ID,
				Content:    errContent,
				IsError:    true,
			}
		} else {
			// Inject progress callback so tools can report partial results
			progressCtx := WithToolProgress(ctx, func(partial json.RawMessage) {
				emit(ch, Event{
					Type:      EventToolExecUpdate,
					ToolID:    call.ID,
					Tool:      call.Name,
					ToolLabel: label,
					Args:      call.Args,
					Result:    partial,
				})
			})

			output, err := tool.Execute(progressCtx, call.Args)
			if err != nil {
				errContent, _ := json.Marshal(err.Error())
				result = ToolResult{
					ToolCallID: call.ID,
					Content:    errContent,
					IsError:    true,
				}
			} else {
				result = ToolResult{
					ToolCallID: call.ID,
					Content:    output,
				}
			}
		}

		emit(ch, Event{
			Type:      EventToolExecEnd,
			ToolID:    call.ID,
			Tool:      call.Name,
			ToolLabel: label,
			Result:    result.Content,
			IsError:   result.IsError,
		})

		results = append(results, result)

		// Check for steering messages — skip remaining tools if user interrupted
		if config.GetSteeringMessages != nil {
			steering := config.GetSteeringMessages()
			if len(steering) > 0 {
				// Skip remaining tool calls
				for _, skipped := range calls[i+1:] {
					results = append(results, skipToolCall(skipped, tools, ch))
				}
				return results, steering
			}
		}
	}

	return results, nil
}

// skipToolCall creates a skipped result for an interrupted tool call.
func skipToolCall(call ToolCall, tools []Tool, ch chan<- Event) ToolResult {
	label := toolLabel(findTool(tools, call.Name))

	emit(ch, Event{
		Type:      EventToolExecStart,
		ToolID:    call.ID,
		Tool:      call.Name,
		ToolLabel: label,
		Args:      call.Args,
	})

	content, _ := json.Marshal("Skipped due to queued user message.")
	result := ToolResult{
		ToolCallID: call.ID,
		Content:    content,
		IsError:    true,
	}

	emit(ch, Event{
		Type:      EventToolExecEnd,
		ToolID:    call.ID,
		Tool:      call.Name,
		ToolLabel: label,
		Result:    result.Content,
		IsError:   true,
	})

	return result
}

// toolLabel returns the human-readable label for a tool, or empty string if not available.
func toolLabel(tool Tool) string {
	if tool == nil {
		return ""
	}
	if labeler, ok := tool.(ToolLabeler); ok {
		return labeler.Label()
	}
	return ""
}

// toolResultToMessage converts a ToolResult into a Message for the context.
func toolResultToMessage(tr ToolResult) Message {
	return Message{
		Role:    RoleTool,
		Content: string(tr.Content),
		Metadata: map[string]any{
			"tool_call_id": tr.ToolCallID,
			"is_error":     tr.IsError,
		},
		Timestamp: time.Now(),
	}
}

// buildToolSpecs converts Tool interfaces to ToolSpec for the LLM.
func buildToolSpecs(tools []Tool) []ToolSpec {
	if len(tools) == 0 {
		return nil
	}
	specs := make([]ToolSpec, 0, len(tools))
	for _, t := range tools {
		specs = append(specs, ToolSpec{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		})
	}
	return specs
}

func findTool(tools []Tool, name string) Tool {
	for _, t := range tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

func copyMessages(msgs []AgentMessage) []AgentMessage {
	out := make([]AgentMessage, len(msgs))
	copy(out, msgs)
	return out
}
