package memory

import (
	"context"
	"sync"
	"time"

	"github.com/voocel/mas/schema"
)

// Store defines the conversation memory interface.
type Store interface {
	Add(ctx context.Context, message schema.Message) error
	AddBatch(ctx context.Context, messages []schema.Message) error
	History(ctx context.Context) ([]schema.Message, error)
	Reset(ctx context.Context) error
	Clone() Store
}

// Buffer is a simple in-memory store with window trimming.
type Buffer struct {
	mu       sync.RWMutex
	window   int
	messages []schema.Message
}

// NewBuffer creates an in-memory store; window <= 0 means no trimming.
func NewBuffer(window int) *Buffer {
	return &Buffer{
		window:   window,
		messages: make([]schema.Message, 0),
	}
}

// Add writes a message.
func (b *Buffer) Add(_ context.Context, message schema.Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	b.messages = append(b.messages, *message.Clone())
	b.trim()
	return nil
}

func (b *Buffer) AddBatch(ctx context.Context, messages []schema.Message) error {
	if len(messages) == 0 {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, message := range messages {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		if message.Timestamp.IsZero() {
			message.Timestamp = time.Now()
		}
		b.messages = append(b.messages, *message.Clone())
	}
	b.trim()
	return nil
}

// History returns current history.
func (b *Buffer) History(context.Context) ([]schema.Message, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	history := make([]schema.Message, len(b.messages))
	for i, msg := range b.messages {
		history[i] = *msg.Clone()
	}
	return history, nil
}

// Reset clears all stored messages.
func (b *Buffer) Reset(context.Context) error {
	b.mu.Lock()
	b.messages = nil
	b.mu.Unlock()
	return nil
}

// Clone returns a copy.
func (b *Buffer) Clone() Store {
	b.mu.RLock()
	defer b.mu.RUnlock()

	clone := &Buffer{
		window:   b.window,
		messages: make([]schema.Message, len(b.messages)),
	}
	for i, msg := range b.messages {
		clone.messages[i] = *msg.Clone()
	}
	return clone
}

func (b *Buffer) trim() {
	if b.window <= 0 || len(b.messages) <= b.window {
		return
	}
	b.messages = append([]schema.Message(nil), b.messages[len(b.messages)-b.window:]...)
}
