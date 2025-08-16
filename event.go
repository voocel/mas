package mas

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

// Standard event types
const (
	// Agent events
	EventAgentStarted   EventType = "agent.started"
	EventAgentStopped   EventType = "agent.stopped"
	EventAgentChatStart EventType = "agent.chat.start"
	EventAgentChatEnd   EventType = "agent.chat.end"
	EventAgentError     EventType = "agent.error"

	// Tool events
	EventToolStart EventType = "tool.start"
	EventToolEnd   EventType = "tool.end"
	EventToolError EventType = "tool.error"

	// Workflow events
	EventWorkflowStart EventType = "workflow.start"
	EventWorkflowEnd   EventType = "workflow.end"
	EventWorkflowError EventType = "workflow.error"
	EventNodeStart     EventType = "node.start"
	EventNodeEnd       EventType = "node.end"
	EventNodeError     EventType = "node.error"

	// Memory events
	EventMemoryAdd   EventType = "memory.add"
	EventMemoryClear EventType = "memory.clear"
)

// Event represents a system event
type Event struct {
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	ID        string                 `json:"id"`
}

// EventHandler handles events
type EventHandler func(ctx context.Context, event Event) error

// EventBus manages event distribution
type EventBus interface {
	// Subscribe to specific event types
	Subscribe(eventType EventType, handler EventHandler) error

	// Unsubscribe from event types
	Unsubscribe(eventType EventType, handler EventHandler) error

	// Publish an event
	Publish(ctx context.Context, event Event) error

	// Close the event bus
	Close() error
}

// StreamEventBus extends EventBus with streaming capabilities
type StreamEventBus interface {
	EventBus

	// Stream returns a channel for receiving events
	Stream(ctx context.Context, eventTypes ...EventType) (<-chan Event, error)
}

// eventBus is the default implementation
type eventBus struct {
	handlers map[EventType][]EventHandler
	streams  map[string]chan Event
	mu       sync.RWMutex
	closed   bool
}

// NewEventBus creates a new event bus
func NewEventBus() StreamEventBus {
	return &eventBus{
		handlers: make(map[EventType][]EventHandler),
		streams:  make(map[string]chan Event),
	}
}

// Subscribe implements EventBus
func (eb *eventBus) Subscribe(eventType EventType, handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	return nil
}

// Unsubscribe implements EventBus
func (eb *eventBus) Unsubscribe(eventType EventType, handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.handlers[eventType]
	for i, h := range handlers {
		// Compare function pointers (note: this is a simplified approach)
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
	return nil
}

// Publish implements EventBus
func (eb *eventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	// Set timestamp and ID if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ID == "" {
		event.ID = generateEventID()
	}

	// Send to handlers
	if handlers, exists := eb.handlers[event.Type]; exists {
		for _, handler := range handlers {
			go func(h EventHandler) {
				if err := h(ctx, event); err != nil {
					// Log error but don't block other handlers
					// In a production system, you might want proper logging
					fmt.Printf("Event handler error: %v\n", err)
				}
			}(handler)
		}
	}

	// Send to streams
	for _, stream := range eb.streams {
		select {
		case stream <- event:
		default:
			// Drop event if channel is full (non-blocking)
		}
	}

	return nil
}

// Stream implements StreamEventBus
func (eb *eventBus) Stream(ctx context.Context, eventTypes ...EventType) (<-chan Event, error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return nil, fmt.Errorf("event bus is closed")
	}

	// Create buffered channel
	stream := make(chan Event, 100)
	streamID := generateStreamID()
	eb.streams[streamID] = stream

	// Filter events if specific types requested
	var eventTypeMap map[EventType]bool
	if len(eventTypes) > 0 {
		eventTypeMap = make(map[EventType]bool)
		for _, et := range eventTypes {
			eventTypeMap[et] = true
		}
	}

	// Create filtered channel
	filtered := make(chan Event, 100)

	go func() {
		defer func() {
			eb.mu.Lock()
			delete(eb.streams, streamID)
			close(stream)
			eb.mu.Unlock()
			close(filtered)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-stream:
				if !ok {
					return
				}

				// Filter events if needed
				if eventTypeMap != nil && !eventTypeMap[event.Type] {
					continue
				}

				select {
				case filtered <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return filtered, nil
}

// Close implements EventBus
func (eb *eventBus) Close() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return nil
	}

	eb.closed = true

	// Close all streams
	for _, stream := range eb.streams {
		close(stream)
	}
	eb.streams = make(map[string]chan Event)
	eb.handlers = make(map[EventType][]EventHandler)

	return nil
}

// Helper functions
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

func generateStreamID() string {
	return fmt.Sprintf("stream_%d", time.Now().UnixNano())
}

// NewEvent creates a new event with common fields
func NewEvent(eventType EventType, source string, data map[string]interface{}) Event {
	return Event{
		Type:      eventType,
		Source:    source,
		Data:      data,
		Timestamp: time.Now(),
		ID:        generateEventID(),
	}
}

// EventData helpers for type-safe event data creation
func EventData(kv ...interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			if key, ok := kv[i].(string); ok {
				data[key] = kv[i+1]
			}
		}
	}
	return data
}
