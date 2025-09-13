package shared

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CommunicationManager manages communication between agents
type CommunicationManager struct {
	sharedContext SharedContext
	channels      map[string]*CommunicationChannel
	messageQueue  *MessageQueue
	router        *MessageRouter
	mutex         sync.RWMutex
	config        CommunicationConfig
}

// CommunicationConfig configures the communication manager
type CommunicationConfig struct {
	MaxChannels       int           `json:"max_channels"`
	MessageTimeout    time.Duration `json:"message_timeout"`
	MaxMessageSize    int           `json:"max_message_size"`
	EnableEncryption  bool          `json:"enable_encryption"`
	EnableCompression bool          `json:"enable_compression"`
	RetryAttempts     int           `json:"retry_attempts"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

// CommunicationChannel represents a communication channel between agents
type CommunicationChannel struct {
	ID           string                 `json:"id"`
	Participants []string               `json:"participants"`
	Type         ChannelType            `json:"type"`
	Status       ChannelStatus          `json:"status"`
	Messages     []*Message             `json:"messages"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	Metadata     map[string]interface{} `json:"metadata"`
	mutex        sync.RWMutex
}

// ChannelType represents the type of communication channel
type ChannelType string

const (
	DirectChannel    ChannelType = "direct"    // 1-to-1 communication
	GroupChannel     ChannelType = "group"     // 1-to-many communication
	BroadcastChannel ChannelType = "broadcast" // 1-to-all communication
	TopicChannel     ChannelType = "topic"     // Topic-based communication
)

// ChannelStatus represents the status of a communication channel
type ChannelStatus string

const (
	ChannelActive   ChannelStatus = "active"
	ChannelInactive ChannelStatus = "inactive"
	ChannelClosed   ChannelStatus = "closed"
)

// Message represents a message between agents
type Message struct {
	ID          string                 `json:"id"`
	From        string                 `json:"from"`
	To          []string               `json:"to"`
	ChannelID   string                 `json:"channel_id"`
	Type        MessageType            `json:"type"`
	Content     string                 `json:"content"`
	Data        map[string]interface{} `json:"data"`
	Priority    MessagePriority        `json:"priority"`
	Status      MessageStatus          `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
	ReadAt      *time.Time             `json:"read_at,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// MessageType represents the type of message
type MessageType string

const (
	TextMessage     MessageType = "text"
	DataMessage     MessageType = "data"
	CommandMessage  MessageType = "command"
	EventMessage    MessageType = "event"
	RequestMessage  MessageType = "request"
	ResponseMessage MessageType = "response"
)

// MessagePriority represents the priority of a message
type MessagePriority int

const (
	LowPriority    MessagePriority = 1
	NormalPriority MessagePriority = 5
	HighPriority   MessagePriority = 10
	UrgentPriority MessagePriority = 15
)

// MessageStatus represents the status of a message
type MessageStatus string

const (
	MessagePending   MessageStatus = "pending"
	MessageSent      MessageStatus = "sent"
	MessageDelivered MessageStatus = "delivered"
	MessageRead      MessageStatus = "read"
	MessageFailed    MessageStatus = "failed"
	MessageExpired   MessageStatus = "expired"
)

// MessageQueue manages a queue of messages
type MessageQueue struct {
	messages []*Message
	mutex    sync.RWMutex
	notify   chan struct{}
}

// MessageRouter routes messages to appropriate channels
type MessageRouter struct {
	routes map[string]string // agentID -> channelID
	mutex  sync.RWMutex
}

// NewCommunicationManager creates a new communication manager
func NewCommunicationManager(sharedContext SharedContext, config CommunicationConfig) *CommunicationManager {
	cm := &CommunicationManager{
		sharedContext: sharedContext,
		channels:      make(map[string]*CommunicationChannel),
		messageQueue:  NewMessageQueue(),
		router:        NewMessageRouter(),
		config:        config,
	}

	// Start background processes
	go cm.messageProcessingLoop()
	go cm.heartbeatLoop()

	return cm
}

// NewMessageQueue creates a new message queue
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		messages: make([]*Message, 0),
		notify:   make(chan struct{}, 1),
	}
}

// NewMessageRouter creates a new message router
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		routes: make(map[string]string),
	}
}

// CreateChannel creates a new communication channel
func (cm *CommunicationManager) CreateChannel(ctx context.Context, channelType ChannelType, participants []string) (*CommunicationChannel, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if len(cm.channels) >= cm.config.MaxChannels {
		return nil, fmt.Errorf("maximum number of channels reached")
	}

	channel := &CommunicationChannel{
		ID:           generateChannelID(),
		Participants: participants,
		Type:         channelType,
		Status:       ChannelActive,
		Messages:     make([]*Message, 0),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	cm.channels[channel.ID] = channel

	// Update router
	for _, participant := range participants {
		cm.router.AddRoute(participant, channel.ID)
	}

	// Broadcast channel creation event
	event := &SharedEvent{
		Type:   "channel.created",
		Source: "communication_manager",
		Data: map[string]interface{}{
			"channel_id":   channel.ID,
			"channel_type": channelType,
			"participants": participants,
		},
		Timestamp: time.Now(),
	}

	cm.sharedContext.BroadcastEvent(ctx, event)

	return channel, nil
}

// SendMessage sends a message through the communication system
func (cm *CommunicationManager) SendMessage(ctx context.Context, message *Message) error {
	if message.ID == "" {
		message.ID = generateMessageID()
	}

	message.CreatedAt = time.Now()
	message.Status = MessagePending

	// Validate message size
	if len(message.Content) > cm.config.MaxMessageSize {
		return fmt.Errorf("message size exceeds maximum allowed size")
	}

	// Set expiration if not set
	if message.ExpiresAt == nil && cm.config.MessageTimeout > 0 {
		expiresAt := time.Now().Add(cm.config.MessageTimeout)
		message.ExpiresAt = &expiresAt
	}

	// Add to message queue
	return cm.messageQueue.Enqueue(message)
}

// GetChannel retrieves a communication channel by ID
func (cm *CommunicationManager) GetChannel(ctx context.Context, channelID string) (*CommunicationChannel, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	channel, exists := cm.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel %s not found", channelID)
	}

	return channel, nil
}

// GetChannelsForAgent retrieves all channels for a specific agent
func (cm *CommunicationManager) GetChannelsForAgent(ctx context.Context, agentID string) ([]*CommunicationChannel, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var channels []*CommunicationChannel
	for _, channel := range cm.channels {
		for _, participant := range channel.Participants {
			if participant == agentID {
				channels = append(channels, channel)
				break
			}
		}
	}

	return channels, nil
}

// CloseChannel closes a communication channel
func (cm *CommunicationManager) CloseChannel(ctx context.Context, channelID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	channel, exists := cm.channels[channelID]
	if !exists {
		return fmt.Errorf("channel %s not found", channelID)
	}

	channel.Status = ChannelClosed

	// Remove routes
	for _, participant := range channel.Participants {
		cm.router.RemoveRoute(participant, channelID)
	}

	// Broadcast channel closure event
	event := &SharedEvent{
		Type:   "channel.closed",
		Source: "communication_manager",
		Data: map[string]interface{}{
			"channel_id": channelID,
		},
		Timestamp: time.Now(),
	}

	cm.sharedContext.BroadcastEvent(ctx, event)

	return nil
}

// GetMessages retrieves messages from a channel
func (cm *CommunicationManager) GetMessages(ctx context.Context, channelID string, limit int) ([]*Message, error) {
	channel, err := cm.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	channel.mutex.RLock()
	defer channel.mutex.RUnlock()

	messages := channel.Messages
	if limit > 0 && len(messages) > limit {
		// Return the most recent messages
		start := len(messages) - limit
		messages = messages[start:]
	}

	return messages, nil
}

// MarkMessageAsRead marks a message as read
func (cm *CommunicationManager) MarkMessageAsRead(ctx context.Context, messageID, agentID string) error {
	// Find the message across all channels
	for _, channel := range cm.channels {
		channel.mutex.Lock()
		for _, message := range channel.Messages {
			if message.ID == messageID {
				// Check if agent is authorized to read this message
				authorized := false
				for _, recipient := range message.To {
					if recipient == agentID {
						authorized = true
						break
					}
				}

				if authorized {
					now := time.Now()
					message.ReadAt = &now
					message.Status = MessageRead
				}

				channel.mutex.Unlock()
				return nil
			}
		}
		channel.mutex.Unlock()
	}

	return fmt.Errorf("message %s not found", messageID)
}

// Background processes
func (cm *CommunicationManager) messageProcessingLoop() {
	for {
		select {
		case <-cm.messageQueue.notify:
			cm.processMessages()
		}
	}
}

func (cm *CommunicationManager) heartbeatLoop() {
	ticker := time.NewTicker(cm.config.HeartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		cm.sendHeartbeat()
	}
}

func (cm *CommunicationManager) processMessages() {
	messages := cm.messageQueue.DequeueAll()

	for _, message := range messages {
		err := cm.deliverMessage(message)
		if err != nil {
			// Handle delivery failure
			message.Status = MessageFailed
			// Could implement retry logic here
		}
	}
}

func (cm *CommunicationManager) deliverMessage(message *Message) error {
	// Find appropriate channel
	channelID := cm.router.GetRoute(message.From)
	if channelID == "" {
		return fmt.Errorf("no route found for agent %s", message.From)
	}

	channel, err := cm.GetChannel(context.Background(), channelID)
	if err != nil {
		return err
	}

	// Add message to channel
	channel.mutex.Lock()
	defer channel.mutex.Unlock()

	channel.Messages = append(channel.Messages, message)
	channel.LastActivity = time.Now()

	// Update message status
	message.Status = MessageDelivered
	now := time.Now()
	message.DeliveredAt = &now

	return nil
}

func (cm *CommunicationManager) sendHeartbeat() {
	// Send heartbeat to all active channels
	// This could be used for connection health monitoring
}

// MessageQueue methods
func (mq *MessageQueue) Enqueue(message *Message) error {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	mq.messages = append(mq.messages, message)

	// Notify processing loop
	select {
	case mq.notify <- struct{}{}:
	default:
	}

	return nil
}

func (mq *MessageQueue) DequeueAll() []*Message {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	messages := mq.messages
	mq.messages = make([]*Message, 0)
	return messages
}

// MessageRouter methods
func (mr *MessageRouter) AddRoute(agentID, channelID string) {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()
	mr.routes[agentID] = channelID
}

func (mr *MessageRouter) RemoveRoute(agentID, channelID string) {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()
	if mr.routes[agentID] == channelID {
		delete(mr.routes, agentID)
	}
}

func (mr *MessageRouter) GetRoute(agentID string) string {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()
	return mr.routes[agentID]
}

// Helper functions
func generateChannelID() string {
	return "channel_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func generateMessageID() string {
	return "message_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}
