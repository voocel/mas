package schema

import (
	"encoding/json"
	"strings"
	"time"
)

// Handoff represents a control transfer between agents
// This is the core primitive for multi-agent collaboration and handoffs
type Handoff struct {
	// Target is the destination agent or node name
	Target string `json:"target"`

	// Reason explains why the handoff happens
	Reason string `json:"reason,omitempty"`

	// Message is the input passed to the next agent
	Message string `json:"message,omitempty"`

	// Payload carries the data delivered to the target
	Payload map[string]interface{} `json:"payload,omitempty"`

	// Context carries additional contextual information
	Context map[string]interface{} `json:"context,omitempty"`

	// Priority indicates urgency (1-10, 10 is highest)
	Priority int `json:"priority,omitempty"`

	// Timeout sets the expiration window
	Timeout time.Duration `json:"timeout,omitempty"`

	// Metadata stores arbitrary metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// State keys shared across the framework for storing handoff metadata.
const (
	HandoffPendingStateKey    = "handoff.pending"
	HandoffNextTargetStateKey = "handoff.next_target"
)

// HandoffType enumerates the supported handoff types
type HandoffType string

const (
	// TransferToolPrefix is the standard prefix for handoff tools.
	TransferToolPrefix = "transfer_to_"

	HandoffTypeDelegate    HandoffType = "delegate"    // Delegate: assign the task to another agent
	HandoffTypeCollaborate HandoffType = "collaborate" // Collaborate: work with another agent to complete the task
	HandoffTypeEscalate    HandoffType = "escalate"    // Escalate: forward the task to a higher-tier agent
	HandoffTypeRoute       HandoffType = "route"       // Route: forward the task to another agent based on routing conditions
)

// HandoffRequest describes a handoff request
type HandoffRequest struct {
	Type     HandoffType            `json:"type"`
	From     string                 `json:"from"`
	To       string                 `json:"to"`
	Message  Message                `json:"message"`
	Context  map[string]interface{} `json:"context,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HandoffResponse describes a handoff response
type HandoffResponse struct {
	Success  bool                   `json:"success"`
	Message  Message                `json:"message,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewHandoff creates a new handoff instance
func NewHandoff(target string) *Handoff {
	return &Handoff{
		Target:   target,
		Payload:  make(map[string]interface{}),
		Context:  make(map[string]interface{}),
		Priority: 5, // Default medium priority
		Metadata: make(map[string]interface{}),
	}
}

// WithPayload sets a payload entry
func (h *Handoff) WithPayload(key string, value interface{}) *Handoff {
	h.Payload[key] = value
	return h
}

// WithContext sets a context entry
func (h *Handoff) WithContext(key string, value interface{}) *Handoff {
	h.Context[key] = value
	return h
}

// WithPriority updates the priority
func (h *Handoff) WithPriority(priority int) *Handoff {
	if priority < 1 {
		priority = 1
	}
	if priority > 10 {
		priority = 10
	}
	h.Priority = priority
	return h
}

// WithTimeout sets the timeout
func (h *Handoff) WithTimeout(timeout time.Duration) *Handoff {
	h.Timeout = timeout
	return h
}

// WithMetadata sets a metadata entry
func (h *Handoff) WithMetadata(key string, value interface{}) *Handoff {
	h.Metadata[key] = value
	return h
}

// GetPayload reads a payload entry
func (h *Handoff) GetPayload(key string) (interface{}, bool) {
	value, exists := h.Payload[key]
	return value, exists
}

// GetContext reads a context entry
func (h *Handoff) GetContext(key string) (interface{}, bool) {
	value, exists := h.Context[key]
	return value, exists
}

// GetMetadata reads a metadata entry
func (h *Handoff) GetMetadata(key string) (interface{}, bool) {
	value, exists := h.Metadata[key]
	return value, exists
}

// IsValid validates the handoff
func (h *Handoff) IsValid() bool {
	if h.Target == "" {
		return false
	}
	if h.Priority == 0 {
		return true
	}
	return h.Priority >= 1 && h.Priority <= 10
}

// HandoffManager defines the handoff manager interface
type HandoffManager interface {
	// RegisterHandler registers a handoff handler
	RegisterHandler(target string, handler HandoffHandler) error

	// Execute runs the handoff
	Execute(request HandoffRequest) (HandoffResponse, error)

	// CanHandle checks whether a target can be handled
	CanHandle(target string) bool

	// ListTargets lists all available handoff targets
	ListTargets() []string
}

// HandoffHandler processes handoff requests
type HandoffHandler interface {
	// Handle processes the handoff request
	Handle(request HandoffRequest) (HandoffResponse, error)

	// CanHandle determines whether the handler supports the request
	CanHandle(request HandoffRequest) bool
}

// HandoffEvent records a handoff event
type HandoffEvent struct {
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// HandoffEventType enumerates handoff event types
const (
	HandoffEventStarted   = "handoff_started"
	HandoffEventCompleted = "handoff_completed"
	HandoffEventFailed    = "handoff_failed"
	HandoffEventTimeout   = "handoff_timeout"
)

// NewHandoffEvent constructs a handoff event
func NewHandoffEvent(eventType, from, to string, success bool) *HandoffEvent {
	return &HandoffEvent{
		Type:      eventType,
		From:      from,
		To:        to,
		Success:   success,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// HandoffFromInterface attempts to convert an arbitrary value into a *Handoff.
func HandoffFromInterface(value interface{}) *Handoff {
	switch v := value.(type) {
	case *Handoff:
		if v != nil && v.Target != "" {
			return v
		}
	case Handoff:
		h := v
		if h.Target != "" {
			return &h
		}
	case map[string]interface{}:
		if data, err := json.Marshal(v); err == nil {
			var h Handoff
			if err := json.Unmarshal(data, &h); err == nil && h.Target != "" {
				return &h
			}
		}
	}
	return nil
}

// ParseHandoff parses a handoff directive from content.
// Accepted formats:
// - {"handoff": {...}}
// - {"target": "...", "reason": "...", "message": "..."}
// - "HANDOFF: {...}"
func ParseHandoff(content string) *Handoff {
	raw := strings.TrimSpace(content)
	if raw == "" {
		return nil
	}
	upper := strings.ToUpper(raw)
	if strings.HasPrefix(upper, "HANDOFF:") {
		raw = strings.TrimSpace(raw[len("HANDOFF:"):])
	}

	type wrapper struct {
		Handoff *Handoff `json:"handoff"`
	}
	var w wrapper
	if err := json.Unmarshal([]byte(raw), &w); err == nil && w.Handoff != nil {
		if w.Handoff.Priority == 0 {
			w.Handoff.Priority = 5
		}
		if w.Handoff.IsValid() {
			return w.Handoff
		}
	}

	var h Handoff
	if err := json.Unmarshal([]byte(raw), &h); err == nil {
		if h.Priority == 0 {
			h.Priority = 5
		}
		if h.IsValid() {
			return &h
		}
	}
	return nil
}
