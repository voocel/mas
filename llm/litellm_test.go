package llm

import (
	"context"
	"os"
	"testing"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

func TestLiteLLMAdapter_Creation(t *testing.T) {
	adapter := NewOpenAIModel("gpt-4.1-mini", os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_BASE_URL"))
	if adapter.info.Name != "gpt-4.1-mini" {
		t.Errorf("Expected model name 'gpt-4.1-mini', got %s", adapter.info.Name)
	}

	if adapter.info.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got %s", adapter.info.Provider)
	}

	// Check capabilities
	capabilities := adapter.info.Capabilities
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

	if adapter.info.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %s", adapter.info.Provider)
	}
}

func TestLiteLLMAdapter_Gemini(t *testing.T) {
	adapter := NewGeminiModel("gemini-2.5-flash", os.Getenv("GEMINI_API_KEY"), os.Getenv("GEMINI_BASE_URL"))
	if adapter.info.Provider != "google" {
		t.Errorf("Expected provider 'google', got %s", adapter.info.Provider)
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

// Streaming test
func TestLiteLLMAdapter_StreamAPI(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("Please set environment variable OPENAI_API_KEY")
	}
	apiBaseUrl := os.Getenv("OPENAI_BASE_URL")
	if apiBaseUrl == "" {
		apiBaseUrl = "https://api.openai.com/v1"
	}

	adapter := NewOpenAIModel("gpt-4.1-mini", apiKey, apiBaseUrl)

	ctx := runtime.NewContext(context.Background(), "test-session", "test-trace")
	eventChan, err := adapter.GenerateStream(ctx, []schema.Message{
		{
			Role:    schema.RoleUser,
			Content: "Count from 1 to 5, one number per line.",
		},
	})
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
