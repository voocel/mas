package agentcore

// emit sends an event to the channel (non-blocking if channel is full, drops event).
func emit(ch chan<- Event, ev Event) {
	select {
	case ch <- ev:
	default:
	}
}

// emitError sends an error event followed by agent_end.
func emitError(ch chan<- Event, err error) {
	emit(ch, Event{Type: EventError, Err: err})
	emit(ch, Event{Type: EventAgentEnd, Err: err})
}

// DefaultConvertToLLM filters AgentMessages to LLM-compatible Messages.
// Custom message types are dropped; only user/assistant/system/tool messages pass through.
func DefaultConvertToLLM(msgs []AgentMessage) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if msg, ok := m.(Message); ok {
			out = append(out, msg)
		}
	}
	return out
}

// RepairMessageSequence ensures tool call / tool result pairs are complete.
// Orphaned tool calls (no matching result) get a synthetic error result inserted.
// Orphaned tool results (no matching call) are removed.
// This prevents LLM providers from rejecting malformed message sequences.
func RepairMessageSequence(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))

	for i, msg := range msgs {
		out = append(out, msg)

		if msg.Role != RoleAssistant {
			continue
		}
		calls := msg.ToolCalls()
		if len(calls) == 0 {
			continue
		}

		// Collect tool result IDs that follow this assistant message
		answered := make(map[string]bool, len(calls))
		for j := i + 1; j < len(msgs); j++ {
			next := msgs[j]
			if next.Role == RoleTool {
				if id, ok := next.Metadata["tool_call_id"].(string); ok {
					answered[id] = true
				}
				continue
			}
			break // stop at first non-tool message
		}

		// Insert synthetic results for unanswered tool calls
		for _, call := range calls {
			if !answered[call.ID] {
				out = append(out, ToolResultMsg(call.ID, []byte(`"Tool result missing (conversation was truncated or interrupted)."`), true))
			}
		}
	}

	// Remove orphaned tool results (no matching call)
	callIDs := make(map[string]bool)
	for _, msg := range out {
		for _, call := range msg.ToolCalls() {
			callIDs[call.ID] = true
		}
	}

	cleaned := make([]Message, 0, len(out))
	for _, msg := range out {
		if msg.Role == RoleTool {
			if id, ok := msg.Metadata["tool_call_id"].(string); ok && !callIDs[id] {
				continue
			}
		}
		cleaned = append(cleaned, msg)
	}

	return cleaned
}

// dequeue removes messages from a queue according to the given mode.
// QueueModeAll drains everything; QueueModeOneAtATime takes only the first message.
func dequeue(queue *[]AgentMessage, mode QueueMode) []AgentMessage {
	if len(*queue) == 0 {
		return nil
	}
	if mode == QueueModeOneAtATime {
		first := (*queue)[0]
		*queue = (*queue)[1:]
		return []AgentMessage{first}
	}
	// QueueModeAll: drain everything
	result := *queue
	*queue = nil
	return result
}
