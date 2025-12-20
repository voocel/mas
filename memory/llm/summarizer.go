package llm

import (
	"context"
	"fmt"
	"strings"

	masllm "github.com/voocel/mas/llm"
	"github.com/voocel/mas/schema"
)

type Summarizer struct {
	Model         masllm.ChatModel
	SystemPrompt  string
	SummaryLength int
}

// NewSummarizer creates an LLM-based summarizer.
func NewSummarizer(model masllm.ChatModel) *Summarizer {
	return &Summarizer{
		Model:         model,
		SystemPrompt:  "You are a summarization assistant. Compress the conversation into a concise summary.",
		SummaryLength: 200,
	}
}

// Summarize sends conversation history to the model and returns a summary.
func (s *Summarizer) Summarize(ctx context.Context, history []schema.Message) (string, error) {
	if s == nil || s.Model == nil {
		return "", fmt.Errorf("memory/llm: summarizer missing model")
	}

	prompt := buildSummaryPrompt(history, s.SummaryLength)
	messages := []schema.Message{
		{Role: schema.RoleSystem, Content: s.SystemPrompt},
		{Role: schema.RoleUser, Content: prompt},
	}

	resp, err := s.Model.Generate(ctx, &masllm.Request{Messages: messages})
	if err != nil {
		return "", err
	}
	return resp.Message.Content, nil
}

func buildSummaryPrompt(history []schema.Message, limit int) string {
	var builder strings.Builder
	builder.WriteString("Please summarize the following conversation. Keep the summary under ")
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

// Summarizer provides lightweight LLM summarization.
