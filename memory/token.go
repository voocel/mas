package memory

import "github.com/voocel/mas"

// EstimateTokens estimates the token count for a single message.
// Uses chars/4 approximation (conservative overestimate).
func EstimateTokens(msg mas.AgentMessage) int {
	var chars int

	switch v := msg.(type) {
	case mas.Message:
		for _, b := range v.Content {
			switch b.Type {
			case mas.ContentText:
				chars += len(b.Text)
			case mas.ContentThinking:
				chars += len(b.Thinking)
			case mas.ContentToolCall:
				if b.ToolCall != nil {
					chars += len(b.ToolCall.Name) + len(b.ToolCall.Args)
				}
			}
		}
	case CompactionSummary:
		chars = len(v.Summary)
	default:
		return 0
	}

	return max((chars+3)/4, 1) // ceil(chars/4), at least 1
}

// EstimateTotal estimates the total token count for a message list.
func EstimateTotal(msgs []mas.AgentMessage) int {
	total := 0
	for _, m := range msgs {
		total += EstimateTokens(m)
	}
	return total
}
