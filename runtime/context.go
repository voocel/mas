package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/voocel/mas/schema"
)

// Context extends the standard context with MAS-specific capabilities
type Context interface {
	context.Context
	SessionID() string
	TraceID() string
	State() State
	SetState(State)
	AddEvent(schema.StreamEvent)
	Events() <-chan schema.StreamEvent
	Conversation() ConversationStore
	Clone() Context

	SetStateValue(key string, value interface{}) error
	GetStateValue(key string) interface{}
	HasStateValue(key string) bool
}

// ConversationStore manages conversation history for a context
type ConversationStore interface {
	Add(ctx context.Context, message schema.Message) error
	GetConversationContext(ctx context.Context) ([]schema.Message, error)
}

type cloneableConversation interface {
	CloneConversation() ConversationStore
}

// ContextOption configures Context creation
type ContextOption func(*contextOptions)

type contextOptions struct {
	conversation ConversationStore
	eventBuffer  int
}

// WithConversation injects a custom conversation store implementation
func WithConversation(store ConversationStore) ContextOption {
	return func(opts *contextOptions) {
		opts.conversation = store
	}
}

// WithEventBufferSize configures the event channel buffer size (defaults to 100)
func WithEventBufferSize(size int) ContextOption {
	return func(opts *contextOptions) {
		if size > 0 {
			opts.eventBuffer = size
		}
	}
}

// State represents the execution state for a context
type State interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	Delete(key string) error
	Keys() []string
	Clone() State
}

type masContext struct {
	context.Context
	sessionID    string
	traceID      string
	state        State
	events       chan schema.StreamEvent
	conversation ConversationStore
	mu           sync.RWMutex
}

const defaultEventBuffer = 100

// NewContext creates a new MAS execution context
func NewContext(parent context.Context, sessionID, traceID string, options ...ContextOption) Context {
	if parent == nil {
		parent = context.Background()
	}

	opts := applyOptions(options...)

	conversation := opts.conversation
	if conversation == nil {
		conversation = newInMemoryConversation()
	}

	bufferSize := opts.eventBuffer
	if bufferSize <= 0 {
		bufferSize = defaultEventBuffer
	}

	return &masContext{
		Context:      parent,
		sessionID:    sessionID,
		traceID:      traceID,
		state:        NewMemoryState(),
		events:       make(chan schema.StreamEvent, bufferSize),
		conversation: conversation,
	}
}

// NewContextWithTimeout creates a context with a timeout
func NewContextWithTimeout(parent context.Context, sessionID, traceID string, timeout time.Duration, options ...ContextOption) (Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	opts := applyOptions(options...)

	conversation := opts.conversation
	if conversation == nil {
		conversation = newInMemoryConversation()
	}

	bufferSize := opts.eventBuffer
	if bufferSize <= 0 {
		bufferSize = defaultEventBuffer
	}

	masCtx := &masContext{
		Context:      ctx,
		sessionID:    sessionID,
		traceID:      traceID,
		state:        NewMemoryState(),
		events:       make(chan schema.StreamEvent, bufferSize),
		conversation: conversation,
	}

	return masCtx, cancel
}

func applyOptions(options ...ContextOption) contextOptions {
	opts := contextOptions{}
	for _, opt := range options {
		if opt != nil {
			opt(&opts)
		}
	}
	return opts
}

func (c *masContext) SessionID() string {
	return c.sessionID
}

func (c *masContext) TraceID() string {
	return c.traceID
}

func (c *masContext) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *masContext) SetState(state State) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

func (c *masContext) AddEvent(event schema.StreamEvent) {
	select {
	case c.events <- event:
	default:
		select {
		case <-c.events:
		default:
		}
		c.events <- event
	}
}

func (c *masContext) Events() <-chan schema.StreamEvent {
	return c.events
}

func (c *masContext) Conversation() ConversationStore {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conversation
}

func (c *masContext) Clone() Context {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var clonedState State
	if c.state != nil {
		clonedState = c.state.Clone()
	} else {
		clonedState = NewMemoryState()
	}

	clonedConversation := cloneConversationStore(c.conversation)

	bufferSize := cap(c.events)
	if bufferSize <= 0 {
		bufferSize = defaultEventBuffer
	}

	clone := &masContext{
		Context:      c.Context,
		sessionID:    c.sessionID,
		traceID:      c.traceID,
		state:        clonedState,
		events:       make(chan schema.StreamEvent, bufferSize),
		conversation: clonedConversation,
	}

	return clone
}

func cloneConversationStore(store ConversationStore) ConversationStore {
	if store == nil {
		return newInMemoryConversation()
	}
	if cloneable, ok := store.(cloneableConversation); ok {
		return cloneable.CloneConversation()
	}
	// fallback: new empty store
	return newInMemoryConversation()
}

func (c *masContext) SetStateValue(key string, value interface{}) error {
	return c.state.Set(key, value)
}

func (c *masContext) GetStateValue(key string) interface{} {
	value, _ := c.state.Get(key)
	return value
}

func (c *masContext) HasStateValue(key string) bool {
	_, exists := c.state.Get(key)
	return exists
}

// memoryState implements the State interface in memory
type memoryState struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewMemoryState creates a new in-memory state store
func NewMemoryState() State {
	return &memoryState{
		data: make(map[string]interface{}),
	}
}

func (s *memoryState) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data[key]
	return value, exists
}

func (s *memoryState) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *memoryState) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *memoryState) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

func (s *memoryState) Clone() State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := NewMemoryState()
	for k, v := range s.data {
		clone.Set(k, v)
	}
	return clone
}

// In-memory conversation store
type inMemoryConversation struct {
	mu       sync.RWMutex
	messages []schema.Message
}

func newInMemoryConversation() ConversationStore {
	return &inMemoryConversation{
		messages: make([]schema.Message, 0),
	}
}

func (c *inMemoryConversation) Add(ctx context.Context, message schema.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	c.messages = append(c.messages, *message.Clone())
	return nil
}

func (c *inMemoryConversation) GetConversationContext(ctx context.Context) ([]schema.Message, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	history := make([]schema.Message, len(c.messages))
	for i, msg := range c.messages {
		history[i] = *msg.Clone()
	}
	return history, nil
}

func (c *inMemoryConversation) CloneConversation() ConversationStore {
	c.mu.RLock()
	defer c.mu.RUnlock()

	clone := &inMemoryConversation{
		messages: make([]schema.Message, len(c.messages)),
	}
	for i, msg := range c.messages {
		clone.messages[i] = *msg.Clone()
	}
	return clone
}
