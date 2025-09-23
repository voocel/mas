package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	masllm "github.com/voocel/mas/llm"
	masmemory "github.com/voocel/mas/memory"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

type Summarizer struct {
	Model         masllm.ChatModel
	SystemPrompt  string
	SummaryLength int
}

// NewSummarizer constructs an LLM-backed summarizer.
func NewSummarizer(model masllm.ChatModel) *Summarizer {
	return &Summarizer{
		Model:         model,
		SystemPrompt:  "You are an assistant that distills user conversations into concise summaries.",
		SummaryLength: 200,
	}
}

// Summarize sends the conversation history to the model and returns the summary text.
func (s *Summarizer) Summarize(ctx context.Context, history []schema.Message) (string, error) {
	if s == nil || s.Model == nil {
		return "", fmt.Errorf("memory/llm: summarizer missing model")
	}

	prompt := buildSummaryPrompt(history, s.SummaryLength)
	messages := []schema.Message{
		{Role: schema.RoleSystem, Content: s.SystemPrompt},
		{Role: schema.RoleUser, Content: prompt},
	}

	runtimeCtx := runtime.NewContext(ctx, "memory-summary", fmt.Sprintf("summary-%d", time.Now().UnixNano()))
	resp, err := s.Model.Generate(runtimeCtx, &masllm.Request{Messages: messages})
	if err != nil {
		return "", err
	}
	return resp.Message.Content, nil
}

func buildSummaryPrompt(history []schema.Message, limit int) string {
	var builder strings.Builder
	builder.WriteString("Summarize the following conversation. Keep the summary under ")
	builder.WriteString(fmt.Sprintf("%d words.\n\n", limit))
	for _, msg := range history {
		if msg.Role == schema.RoleSystem {
			continue
		}
		role := string(msg.Role)
		if len(role) > 0 {
			role = strings.ToUpper(role[:1]) + role[1:]
		}
		builder.WriteString(role)
		builder.WriteString(": ")
		builder.WriteString(msg.Content)
		builder.WriteString("\n")
	}
	builder.WriteString("\nSummary:")
	return builder.String()
}

var _ masmemory.Summarizer = (*Summarizer)(nil)
