package shared

import (
	"context"
	"testing"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

func TestInMemorySharedContext(t *testing.T) {
	sharedCtx := NewInMemorySharedContext()
	ctx := context.Background()

	// Test agent registration
	metadata := AgentMetadata{
		ID:           "agent_1",
		Name:         "Test Agent 1",
		Type:         "test",
		Capabilities: []string{"chat", "analysis"},
		Status:       AgentStatusActive,
		Metadata:     make(map[string]interface{}),
	}

	err := sharedCtx.RegisterAgent(ctx, "agent_1", metadata)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// Test getting agent metadata
	retrievedMetadata, err := sharedCtx.GetAgentMetadata(ctx, "agent_1")
	if err != nil {
		t.Fatalf("Failed to get agent metadata: %v", err)
	}

	if retrievedMetadata.ID != "agent_1" {
		t.Errorf("Expected agent ID 'agent_1', got %s", retrievedMetadata.ID)
	}

	// Test getting active agents
	activeAgents, err := sharedCtx.GetActiveAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get active agents: %v", err)
	}

	if len(activeAgents) != 1 {
		t.Errorf("Expected 1 active agent, got %d", len(activeAgents))
	}

	// Test global state
	err = sharedCtx.SetGlobalState(ctx, "test_key", "test_value")
	if err != nil {
		t.Fatalf("Failed to set global state: %v", err)
	}

	value, err := sharedCtx.GetGlobalState(ctx, "test_key")
	if err != nil {
		t.Fatalf("Failed to get global state: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected global state value 'test_value', got %v", value)
	}

	// Test context sharing
	shareData := map[string]interface{}{
		"shared_info": "important data",
		"timestamp":   time.Now(),
	}

	err = sharedCtx.ShareContext(ctx, "agent_1", "agent_2", shareData)
	if err != nil {
		t.Fatalf("Failed to share context: %v", err)
	}

	sharedData, err := sharedCtx.GetSharedContext(ctx, "agent_2")
	if err != nil {
		t.Fatalf("Failed to get shared context: %v", err)
	}

	if sharedData["agent_1:shared_info"] != "important data" {
		t.Errorf("Expected shared data not found")
	}

	// Test context synchronization
	state := contextpkg.NewContextState("thread_1", "agent_1")
	state.Messages = append(state.Messages, contextpkg.NewMessage("user", "test message"))

	err = sharedCtx.SyncContext(ctx, "agent_1", state)
	if err != nil {
		t.Fatalf("Failed to sync context: %v", err)
	}

	syncedState, err := sharedCtx.GetSyncedContext(ctx, "agent_1")
	if err != nil {
		t.Fatalf("Failed to get synced context: %v", err)
	}

	if len(syncedState.Messages) != 1 {
		t.Errorf("Expected 1 synced message, got %d", len(syncedState.Messages))
	}

	// Test event subscription
	eventChan, err := sharedCtx.SubscribeToEvents(ctx, "agent_1", []string{"test_event"})
	if err != nil {
		t.Fatalf("Failed to subscribe to events: %v", err)
	}

	// Test event broadcasting
	event := &SharedEvent{
		Type:   "test_event",
		Source: "test",
		Data:   map[string]interface{}{"test": "data"},
	}

	err = sharedCtx.BroadcastEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to broadcast event: %v", err)
	}

	// Check if event was received
	select {
	case receivedEvent := <-eventChan:
		if receivedEvent.Type != "test_event" {
			t.Errorf("Expected event type 'test_event', got %s", receivedEvent.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Event not received within timeout")
	}

	// Test agent unregistration
	err = sharedCtx.UnregisterAgent(ctx, "agent_1")
	if err != nil {
		t.Fatalf("Failed to unregister agent: %v", err)
	}

	activeAgents, err = sharedCtx.GetActiveAgents(ctx)
	if err != nil {
		t.Fatalf("Failed to get active agents after unregistration: %v", err)
	}

	if len(activeAgents) != 0 {
		t.Errorf("Expected 0 active agents after unregistration, got %d", len(activeAgents))
	}
}

func TestCoordinator(t *testing.T) {
	sharedCtx := NewInMemorySharedContext()
	config := CoordinatorConfig{
		MaxConcurrentTasks:  10,
		TaskTimeout:         30 * time.Second,
		HealthCheckInterval: 5 * time.Second,
		EnableLoadBalancing: true,
		EnableFailover:      true,
	}

	coordinator := NewCoordinator(sharedCtx, config)

	// Test agent registration
	metadata := AgentMetadata{
		ID:           "agent_1",
		Name:         "Test Agent 1",
		Type:         "worker",
		Capabilities: []string{"data_processing", "analysis"},
		Status:       AgentStatusActive,
	}

	err := coordinator.RegisterAgent(context.Background(), "agent_1", metadata)
	if err != nil {
		t.Fatalf("Failed to register agent with coordinator: %v", err)
	}

	// Test task submission
	task := &Task{
		Type:     "data_processing",
		Priority: 5,
		Data: map[string]interface{}{
			"input": "test data",
		},
		MaxRetries: 3,
	}

	err = coordinator.SubmitTask(context.Background(), task)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Test getting optimal agent
	agentID, err := coordinator.GetOptimalAgent(context.Background(), task)
	if err != nil {
		t.Fatalf("Failed to get optimal agent: %v", err)
	}

	if agentID != "agent_1" {
		t.Errorf("Expected optimal agent 'agent_1', got %s", agentID)
	}

	// Test task assignment
	err = coordinator.AssignTask(context.Background(), task.ID, agentID)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Test task completion
	result := map[string]interface{}{
		"output": "processed data",
		"status": "success",
	}

	err = coordinator.CompleteTask(context.Background(), task.ID, result)
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	// Test coordinator stats
	stats := coordinator.GetCoordinatorStats()
	if stats.TotalAgents != 1 {
		t.Errorf("Expected 1 total agent, got %d", stats.TotalAgents)
	}

	if stats.CompletedTasks != 1 {
		t.Errorf("Expected 1 completed task, got %d", stats.CompletedTasks)
	}
}

func TestCommunicationManager(t *testing.T) {
	sharedCtx := NewInMemorySharedContext()
	config := CommunicationConfig{
		MaxChannels:       100,
		MessageTimeout:    30 * time.Second,
		MaxMessageSize:    1024 * 1024, // 1MB
		EnableEncryption:  false,
		EnableCompression: false,
		RetryAttempts:     3,
		HeartbeatInterval: 10 * time.Second,
	}

	commManager := NewCommunicationManager(sharedCtx, config)

	// Test channel creation
	participants := []string{"agent_1", "agent_2"}
	channel, err := commManager.CreateChannel(context.Background(), DirectChannel, participants)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	if channel.Type != DirectChannel {
		t.Errorf("Expected channel type 'direct', got %s", channel.Type)
	}

	if len(channel.Participants) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(channel.Participants))
	}

	// Test message sending
	message := &Message{
		From:     "agent_1",
		To:       []string{"agent_2"},
		Type:     TextMessage,
		Content:  "Hello, agent_2!",
		Priority: NormalPriority,
		Data:     make(map[string]interface{}),
		Metadata: make(map[string]interface{}),
	}

	err = commManager.SendMessage(context.Background(), message)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Give some time for message processing
	time.Sleep(100 * time.Millisecond)

	// Test getting messages
	messages, err := commManager.GetMessages(context.Background(), channel.ID, 10)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "Hello, agent_2!" {
		t.Errorf("Expected message content 'Hello, agent_2!', got %s", messages[0].Content)
	}

	// Test marking message as read
	err = commManager.MarkMessageAsRead(context.Background(), messages[0].ID, "agent_2")
	if err != nil {
		t.Fatalf("Failed to mark message as read: %v", err)
	}

	// Test getting channels for agent
	channels, err := commManager.GetChannelsForAgent(context.Background(), "agent_1")
	if err != nil {
		t.Fatalf("Failed to get channels for agent: %v", err)
	}

	if len(channels) != 1 {
		t.Errorf("Expected 1 channel for agent_1, got %d", len(channels))
	}

	// Test closing channel
	err = commManager.CloseChannel(context.Background(), channel.ID)
	if err != nil {
		t.Fatalf("Failed to close channel: %v", err)
	}

	// Verify channel is closed
	closedChannel, err := commManager.GetChannel(context.Background(), channel.ID)
	if err != nil {
		t.Fatalf("Failed to get channel after closing: %v", err)
	}

	if closedChannel.Status != ChannelClosed {
		t.Errorf("Expected channel status 'closed', got %s", closedChannel.Status)
	}
}
