package communication

import (
	"context"
	"encoding/json"
	"time"
)

// Message represents a message passed between agents
type Message struct {
	ID          string                 `json:"id"`
	SenderID    string                 `json:"sender_id"`
	ReceiverID  string                 `json:"receiver_id,omitempty"` // empty means broadcast
	Content     json.RawMessage        `json:"content"`
	ContentType string                 `json:"content_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Bus is the communication bus for agents
type Bus interface {
	// Publish publishes a message
	Publish(ctx context.Context, msg Message) error

	// Subscribe subscribes to messages from a specific sender
	Subscribe(ctx context.Context, senderID string) (<-chan Message, error)

	// SubscribeAll subscribes to all messages
	SubscribeAll(ctx context.Context) (<-chan Message, error)

	// Unsubscribe unsubscribes from messages
	Unsubscribe(ctx context.Context, ch <-chan Message) error

	// Close closes the communication bus
	Close() error
}

type Config struct {
	Type         string
	BufferSize   int
	RetryCount   int
	RetryBackoff time.Duration
	Extra        map[string]interface{}
}

// NewBus creates a new communication bus
func NewBus(config Config) (Bus, error) {
	switch config.Type {
	case "memory":
		return NewMemoryBus(config), nil
	case "redis":
		// Redis implementation not currently available, returning in-memory implementation
		return NewMemoryBus(config), nil
	default:
		return NewMemoryBus(config), nil
	}
}

// MemoryBus is an in-memory communication bus
type MemoryBus struct {
	subscribers map[string][]chan Message
	bufferSize  int
}

// NewMemoryBus creates a new in-memory bus
func NewMemoryBus(config Config) *MemoryBus {
	bufferSize := 100
	if config.BufferSize > 0 {
		bufferSize = config.BufferSize
	}

	return &MemoryBus{
		subscribers: make(map[string][]chan Message),
		bufferSize:  bufferSize,
	}
}

// Publish publishes a message
func (b *MemoryBus) Publish(ctx context.Context, msg Message) error {
	if msg.ReceiverID == "" {
		// broadcast message
		for _, channels := range b.subscribers {
			for _, ch := range channels {
				select {
				case ch <- msg:
				default:
					// if channel is full, don't wait
				}
			}
		}
	} else {
		// direct message
		channels, ok := b.subscribers[msg.ReceiverID]
		if ok {
			for _, ch := range channels {
				select {
				case ch <- msg:
				default:
					// if channel is full, don't wait
				}
			}
		}
	}
	return nil
}

// Subscribe subscribes to messages from a specific sender
func (b *MemoryBus) Subscribe(ctx context.Context, senderID string) (<-chan Message, error) {
	ch := make(chan Message, b.bufferSize)

	if _, ok := b.subscribers[senderID]; !ok {
		b.subscribers[senderID] = make([]chan Message, 0)
	}

	b.subscribers[senderID] = append(b.subscribers[senderID], ch)
	return ch, nil
}

// SubscribeAll subscribes to all messages
func (b *MemoryBus) SubscribeAll(ctx context.Context) (<-chan Message, error) {
	// special subscriber ID for receiving all messages
	return b.Subscribe(ctx, "*")
}

// Unsubscribe unsubscribes from messages
func (b *MemoryBus) Unsubscribe(ctx context.Context, ch <-chan Message) error {
	// find and remove the channel from all subscribers
	for senderID, channels := range b.subscribers {
		for i, c := range channels {
			if c == ch {
				b.subscribers[senderID] = append(channels[:i], channels[i+1:]...)
				// since ch is read-only, we can't close it directly
				// but internally we know it's actually a writable channel
				close(c)
				break
			}
		}

		// if no subscribers, remove the sender
		if len(b.subscribers[senderID]) == 0 {
			delete(b.subscribers, senderID)
		}
	}
	return nil
}

// Close closes the memory bus
func (b *MemoryBus) Close() error {
	for _, channels := range b.subscribers {
		for _, ch := range channels {
			close(ch)
		}
	}
	b.subscribers = make(map[string][]chan Message)
	return nil
}

// RedisBus is a stub for Redis communication bus
type RedisBus struct {
	// Redis client
}

var (
	ErrBusNotSupported = BusError{Code: "bus_not_supported", Message: "Communication bus type not supported"}
	ErrPublishFailed   = BusError{Code: "publish_failed", Message: "Failed to publish message"}
)

type BusError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e BusError) Error() string {
	return e.Message
}

func (e BusError) WithDetails(details string) BusError {
	e.Details = details
	return e
}
