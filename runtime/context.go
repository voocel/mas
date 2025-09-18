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
}

// State represents the execution state
type State interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	Delete(key string) error
	Keys() []string
	Clone() State
}

type masContext struct {
	context.Context
	sessionID string
	traceID   string
	state     State
	events    chan schema.StreamEvent
	mu        sync.RWMutex
}

// NewContext creates a new MAS execution context
func NewContext(parent context.Context, sessionID, traceID string) Context {
	if parent == nil {
		parent = context.Background()
	}

	return &masContext{
		Context:   parent,
		sessionID: sessionID,
		traceID:   traceID,
		state:     NewMemoryState(),
		events:    make(chan schema.StreamEvent, 100), // Buffer 100 events
	}
}

// NewContextWithTimeout creates a context with a timeout
func NewContextWithTimeout(parent context.Context, sessionID, traceID string, timeout time.Duration) (Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	masCtx := &masContext{
		Context:   ctx,
		sessionID: sessionID,
		traceID:   traceID,
		state:     NewMemoryState(),
		events:    make(chan schema.StreamEvent, 100),
	}

	return masCtx, cancel
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
		// If the channel is full, drop the oldest event
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

	clone := &memoryState{
		data: make(map[string]interface{}),
	}

	for k, v := range s.data {
		clone.data[k] = v
	}

	return clone
}

// WithState stores a key/value pair in the context state
func WithState(ctx Context, key string, value interface{}) Context {
	ctx.State().Set(key, value)
	return ctx
}

// GetState retrieves a value from the context state
func GetState(ctx Context, key string) (interface{}, bool) {
	return ctx.State().Get(key)
}

// GetStateString retrieves a string value from the context state
func GetStateString(ctx Context, key string) (string, bool) {
	value, exists := ctx.State().Get(key)
	if !exists {
		return "", false
	}

	if str, ok := value.(string); ok {
		return str, true
	}
	return "", false
}

// GetStateInt retrieves an integer value from the context state
func GetStateInt(ctx Context, key string) (int, bool) {
	value, exists := ctx.State().Get(key)
	if !exists {
		return 0, false
	}

	if i, ok := value.(int); ok {
		return i, true
	}
	return 0, false
}
