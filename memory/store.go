package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/voocel/mas/schema"
)

type ConversationStore interface {
	Add(ctx context.Context, message schema.Message) error
	GetConversationContext(ctx context.Context) ([]schema.Message, error)
	Clone() ConversationStore
}

type Summarizer interface {
	Summarize(ctx context.Context, history []schema.Message) (string, error)
}

// Store is the default in-memory implementation of ConversationStore.
type Store struct {
	mu         sync.RWMutex
	window     int
	messages   []schema.Message
	summarizer Summarizer
}

type Option func(*Store)

// WithWindow limits how many recent messages are retained; non-positive keeps the full history.
func WithWindow(window int) Option {
	return func(store *Store) {
		if window > 0 {
			store.window = window
		}
	}
}

func WithSummarizer(s Summarizer) Option {
	return func(store *Store) {
		store.summarizer = s
	}
}

// NewStore constructs the in-memory conversation store.
func NewStore(opts ...Option) *Store {
	store := &Store{
		messages: make([]schema.Message, 0),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(store)
		}
	}
	return store
}

// Add appends a new message to the store.
func (s *Store) Add(_ context.Context, message schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	s.messages = append(s.messages, *message.Clone())
	s.trimWindow()
	return nil
}

// GetConversationContext returns a copy of the current conversation history.
func (s *Store) GetConversationContext(context.Context) ([]schema.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := make([]schema.Message, len(s.messages))
	for i, msg := range s.messages {
		history[i] = *msg.Clone()
	}
	return history, nil
}

// Clone returns a deep copy so runtime.Context cloning has isolated history.
func (s *Store) Clone() ConversationStore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &Store{
		window:     s.window,
		summarizer: s.summarizer,
		messages:   make([]schema.Message, len(s.messages)),
	}
	for i, msg := range s.messages {
		clone.messages[i] = *msg.Clone()
	}
	return clone
}

// Summarize uses the configured Summarizer to generate a conversation summary.
func (s *Store) Summarize(ctx context.Context) (string, error) {
	s.mu.RLock()
	summarizer := s.summarizer
	history := make([]schema.Message, len(s.messages))
	for i, msg := range s.messages {
		history[i] = *msg.Clone()
	}
	s.mu.RUnlock()

	if summarizer == nil {
		return "", errors.New("memory: summarizer not configured")
	}
	return summarizer.Summarize(ctx, history)
}

func (s *Store) trimWindow() {
	if s.window <= 0 {
		return
	}
	if len(s.messages) <= s.window {
		return
	}
	s.messages = append([]schema.Message(nil), s.messages[len(s.messages)-s.window:]...)
}

var _ ConversationStore = (*Store)(nil)
