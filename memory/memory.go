package memory

import (
	"time"

	"github.com/voocel/agentcore"
)

// CompactionSummary is a compacted context summary message.
// It implements AgentMessage but is NOT a Message, so DefaultConvertToLLM
// will filter it out. Use CompactionConvertToLLM to handle it.
type CompactionSummary struct {
	Summary       string
	TokensBefore  int
	ReadFiles     []string
	ModifiedFiles []string
	Timestamp     time.Time
}

func (c CompactionSummary) GetRole() agentcore.Role        { return agentcore.RoleUser }
func (c CompactionSummary) GetTimestamp() time.Time   { return c.Timestamp }

// CompactionConvertToLLM converts AgentMessages to LLM Messages,
// handling CompactionSummary by wrapping it as a user message with XML tags.
// For all other message types, it delegates to DefaultConvertToLLM behavior.
func CompactionConvertToLLM(msgs []agentcore.AgentMessage) []agentcore.Message {
	out := make([]agentcore.Message, 0, len(msgs))
	for _, m := range msgs {
		switch v := m.(type) {
		case CompactionSummary:
			out = append(out, agentcore.Message{
				Role:    agentcore.RoleUser,
				Content: []agentcore.ContentBlock{agentcore.TextBlock("<context-summary>\n" + v.Summary + "\n</context-summary>")},
				Metadata: map[string]any{
					"type":           "compaction_summary",
					"tokens_before":  v.TokensBefore,
					"read_files":     v.ReadFiles,
					"modified_files": v.ModifiedFiles,
				},
				Timestamp: v.Timestamp,
			})
		case agentcore.Message:
			out = append(out, v)
		}
	}
	return out
}
