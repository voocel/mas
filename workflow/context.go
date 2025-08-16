package workflow

import (
	"sync"
	"time"
)

// WorkflowContext represents the execution context for a workflow
type WorkflowContext struct {
	ID       string         `json:"id"`
	Data     map[string]any `json:"data"`
	Messages []Message      `json:"messages"`
	mutex    sync.RWMutex
}

// Message represents a message in the workflow context
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Get safely retrieves a value from context
func (c *WorkflowContext) Get(key string) any {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Data[key]
}

// Set safely sets a value in context
func (c *WorkflowContext) Set(key string, value any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Data[key] = value
}

// AddMessage adds a message to the context
func (c *WorkflowContext) AddMessage(role, content string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Messages = append(c.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// GetMessages returns a copy of all messages
func (c *WorkflowContext) GetMessages() []Message {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	messages := make([]Message, len(c.Messages))
	copy(messages, c.Messages)
	return messages
}

// GetData returns a copy of the data map
func (c *WorkflowContext) GetData() map[string]any {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	data := make(map[string]any)
	for k, v := range c.Data {
		data[k] = v
	}
	return data
}

// NewWorkflowContext creates a new workflow context
func NewWorkflowContext(id string, initialData map[string]any) *WorkflowContext {
	data := make(map[string]any)
	if initialData != nil {
		for k, v := range initialData {
			data[k] = v
		}
	}
	
	return &WorkflowContext{
		ID:       id,
		Data:     data,
		Messages: make([]Message, 0),
	}
}