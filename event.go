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
