package memory

import "github.com/voocel/agentcore"

// EstimateTokens estimates the token count for a single message.
// Uses chars/4 approximation (conservative overestimate).
func EstimateTokens(msg agentcore.AgentMessage) int {
	var chars int

	switch v := msg.(type) {
	case agentcore.Message:
		for _, b := range v.Content {
			switch b.Type {
			case agentcore.ContentText:
				chars += len(b.Text)
			case agentcore.ContentThinking:
				chars += len(b.Thinking)
			case agentcore.ContentToolCall:
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
func EstimateTotal(msgs []agentcore.AgentMessage) int {
	total := 0
	for _, m := range msgs {
		total += EstimateTokens(m)
	}
	return total
}

// ---------------------------------------------------------------------------
// Hybrid context token estimation (mirrors pi-mono estimateContextTokens)
// ---------------------------------------------------------------------------

// ContextUsageEstimate holds the hybrid token estimation result.
// It combines LLM-reported Usage data with chars/4 estimation for trailing messages.
type ContextUsageEstimate struct {
	// Tokens is the total estimated context tokens (UsageTokens + TrailingTokens).
	Tokens int
	// UsageTokens is the token count derived from the last LLM-reported Usage.
	UsageTokens int
	// TrailingTokens is the chars/4 estimate for messages after the last Usage.
	TrailingTokens int
	// LastUsageIndex is the index of the last assistant message with Usage, or -1 if none.
	LastUsageIndex int
}

// calculateContextTokens computes total context tokens from LLM-reported Usage.
// Prefers TotalTokens; falls back to summing Input + Output + CacheRead + CacheWrite.
func calculateContextTokens(u *agentcore.Usage) int {
	if u.TotalTokens > 0 {
		return u.TotalTokens
	}
	return u.Input + u.Output + u.CacheRead + u.CacheWrite
}

// EstimateContextTokens uses a hybrid approach: actual Usage from the last
// non-aborted assistant message, plus chars/4 estimates for trailing messages.
// This approximates the current context window occupancy.
func EstimateContextTokens(msgs []agentcore.AgentMessage) ContextUsageEstimate {
	// Walk backwards to find last assistant Message with valid Usage
	lastIdx := -1
	var lastUsage *agentcore.Usage
	for i := len(msgs) - 1; i >= 0; i-- {
		msg, ok := msgs[i].(agentcore.Message)
		if !ok || msg.Role != agentcore.RoleAssistant {
			continue
		}
		if msg.StopReason == agentcore.StopReasonAborted || msg.StopReason == agentcore.StopReasonError {
			continue
		}
		if msg.Usage != nil {
			lastIdx = i
			lastUsage = msg.Usage
			break
		}
	}

	// No Usage found — fall back to pure chars/4 estimation
	if lastIdx < 0 {
		total := EstimateTotal(msgs)
		return ContextUsageEstimate{
			Tokens:         total,
			TrailingTokens: total,
			LastUsageIndex: -1,
		}
	}

	usageTokens := calculateContextTokens(lastUsage)

	// Estimate trailing messages after the last usage point
	var trailing int
	for i := lastIdx + 1; i < len(msgs); i++ {
		trailing += EstimateTokens(msgs[i])
	}

	return ContextUsageEstimate{
		Tokens:         usageTokens + trailing,
		UsageTokens:    usageTokens,
		TrailingTokens: trailing,
		LastUsageIndex: lastIdx,
	}
}

// ContextEstimateAdapter adapts EstimateContextTokens to the agentcore.ContextEstimateFn signature.
func ContextEstimateAdapter(msgs []agentcore.AgentMessage) (tokens, usageTokens, trailingTokens int) {
	e := EstimateContextTokens(msgs)
	return e.Tokens, e.UsageTokens, e.TrailingTokens
}
