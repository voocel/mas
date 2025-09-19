package llm

import (
	"context"
	"os"
	"testing"

	"github.com/voocel/litellm"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

func TestLiteLLMAdapter_Creation(t *testing.T) {
	adapter := NewOpenAIModel("gpt-4.1-mini", os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_BASE_URL"))
	if adapter.Info().Name != "gpt-4.1-mini" {
		t.Errorf("Expected model name 'gpt-4.1-mini', got %s", adapter.Info().Name)
	}

	if adapter.Info().Provider != "openai" {
		t.Errorf("Expected provider 'openai', got %s", adapter.Info().Provider)
	}

	// Check capabilities
	capabilities := adapter.Info().Capabilities
	expectedCaps := []string{"chat", "completion", "streaming"}
	for _, expected := range expectedCaps {
		found := false
		for _, cap := range capabilities {
			if cap == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected capability '%s' not found", expected)
		}
	}
}

func TestLiteLLMAdapter_Anthropic(t *testing.T) {
	adapter := NewAnthropicModel("claude-4-sonnet", os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("ANTHROPIC_BASE_URL"))

	if adapter.Info().Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %s", adapter.Info().Provider)
	}
}

func TestLiteLLMAdapter_Gemini(t *testing.T) {
	adapter := NewGeminiModel("gemini-2.5-flash", os.Getenv("GEMINI_API_KEY"), os.Getenv("GEMINI_BASE_URL"))
	if adapter.Info().Provider != "google" {
		t.Errorf("Expected provider 'google', got %s", adapter.Info().Provider)
	}
}

func TestLiteLLMAdapter_MessageConversion(t *testing.T) {
	adapter := NewOpenAIModel("gpt-4.1-mini", os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_BASE_URL"))
	messages := []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: "Hello",
		},
		{
			Role:    schema.RoleAssistant,
			Content: "Hi there!",
		},
	}

	llmMessages, err := adapter.convertMessages(messages)
	if err != nil {
		t.Fatalf("Failed to convert messages: %v", err)
	}

	if len(llmMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(llmMessages))
	}

	if llmMessages[0].Role != "user" {
		t.Errorf("Expected role 'user', got %s", llmMessages[0].Role)
	}

	if llmMessages[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %s", llmMessages[0].Content)
	}
}

func TestLiteLLMAdapter_ToolCallConversion(t *testing.T) {
	adapter := NewLiteLLMAdapter("gpt-4.1-mini")

	toolCallArgs := []byte(`{"expression":"1+1"}`)
	messages := []schema.Message{
		{
			Role: schema.RoleAssistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call_1",
					Name: "calculator",
					Args: toolCallArgs,
				},
			},
		},
		{
			Role:    schema.RoleTool,
			ID:      "call_1",
			Content: "2",
		},
	}

	llmMessages, err := adapter.convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	if len(llmMessages[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(llmMessages[0].ToolCalls))
	}

	if llmMessages[0].ToolCalls[0].Function.Name != "calculator" {
		t.Errorf("unexpected tool call name: %s", llmMessages[0].ToolCalls[0].Function.Name)
	}

	if llmMessages[1].ToolCallID != "call_1" {
		t.Errorf("expected tool call ID 'call_1', got %s", llmMessages[1].ToolCallID)
	}

	response := &litellm.Response{
		Content: "",
		ToolCalls: []litellm.ToolCall{
			{
				ID: "call_1",
				Function: litellm.FunctionCall{
					Name:      "calculator",
					Arguments: "{\"value\":2}",
				},
			},
		},
		FinishReason: "tool_calls",
		Model:        "gpt-test",
		Provider:     "openai",
		Usage: litellm.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	msg, usage := adapter.convertResponse(response)

	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call in response, got %d", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].Name != "calculator" {
		t.Errorf("unexpected response tool call name: %s", msg.ToolCalls[0].Name)
	}
	if usage.TotalTokens == 0 {
		t.Errorf("expected token usage to be populated")
	}
}

// Streaming test
func TestLiteLLMAdapter_StreamAPI(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping streaming API test")
	}
	apiBaseUrl := os.Getenv("OPENAI_BASE_URL")
	if apiBaseUrl == "" {
		apiBaseUrl = "https://api.openai.com/v1"
	}

	adapter := NewOpenAIModel("gpt-4.1-mini", apiKey, apiBaseUrl)

	ctx := runtime.NewContext(context.Background(), "test-session", "test-trace")
	eventChan, err := adapter.GenerateStream(ctx, &Request{Messages: []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: "Count from 1 to 5, one number per line.",
		},
	}})
	if err != nil {
		t.Fatalf("Failed to start stream: %v", err)
	}

	var events []schema.StreamEvent
	var finalContent string

	// Collect all events
	for event := range eventChan {
		events = append(events, event)

		switch event.Type {
		case schema.EventStart:
			t.Log("Stream started")
		case schema.EventToken:
			if tokenEvent, ok := event.Data.(schema.TokenEvent); ok {
				t.Logf("Token: %s", tokenEvent.Delta)
			}
		case schema.EventEnd:
			if msg, ok := event.Data.(schema.Message); ok {
				finalContent = msg.Content
			}
			t.Log("Stream ended")
		case schema.EventError:
			t.Errorf("Stream error: %v", event.Error)
		}
	}

	if len(events) == 0 {
		t.Error("Expected at least one event")
	}

	// Check for start event
	if events[0].Type != schema.EventStart {
		t.Errorf("Expected first event to be start, got %s", events[0].Type)
	}

	// Check for end event
	if events[len(events)-1].Type != schema.EventEnd {
		t.Errorf("Expected last event to be end, got %s", events[len(events)-1].Type)
	}

	if finalContent == "" {
		t.Error("Expected non-empty final content")
	}

	t.Logf("Final content: %s", finalContent)
}

func TestToolCallingSupport(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"gpt-4.1", true},
		{"gpt-5", true},
		{"claude-4-sonnet", true},
		{"gemini-2.5-pro", true},
		{"unknown-model", false},
	}

	for _, test := range tests {
		result := supportsToolCalling(test.model)
		if result != test.expected {
			t.Errorf("supportsToolCalling(%s) = %t, expected %t", test.model, result, test.expected)
		}
	}
}
