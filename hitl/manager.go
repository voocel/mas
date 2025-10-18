package hitl

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/voocel/mas/runtime"
)

// Manager orchestrates human-in-the-loop approval workflows
type Manager struct {
	checkpointer  Checkpointer
	approvalStore ApprovalStore
	feedbackStore FeedbackStore
	policies      []PolicyConfig
	decisions     map[string]*ApprovalDecision // requestID -> decision
	decisionChans map[string]chan *ApprovalDecision
	mu            sync.RWMutex
}

// NewManager creates a new HITL manager
func NewManager(checkpointer Checkpointer, approvalStore ApprovalStore, feedbackStore FeedbackStore) *Manager {
	return &Manager{
		checkpointer:  checkpointer,
		approvalStore: approvalStore,
		feedbackStore: feedbackStore,
		policies:      make([]PolicyConfig, 0),
		decisions:     make(map[string]*ApprovalDecision),
		decisionChans: make(map[string]chan *ApprovalDecision),
	}
}

// RegisterPolicy registers an approval policy
func (m *Manager) RegisterPolicy(policy PolicyConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies = append(m.policies, policy)
}

// CheckApproval checks if approval is needed and creates a request if necessary
func (m *Manager) CheckApproval(ctx runtime.Context, trigger ApprovalTrigger, data interface{}) (*ApprovalRequest, error) {
	m.mu.RLock()
	policies := m.policies
	m.mu.RUnlock()

	// Check all policies
	for _, policy := range policies {
		if policy.Trigger != trigger {
			continue
		}

		if policy.Condition != nil && !policy.Condition(ctx, data) {
			continue
		}

		// Create approval request
		req := &ApprovalRequest{
			ID:        uuid.New().String(),
			SessionID: ctx.SessionID(),
			ThreadID:  ctx.TraceID(),
			Trigger:   trigger,
			Context:   m.extractContext(data),
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(policy.Timeout),
			Priority:  policy.Priority,
			Approvers: policy.Approvers,
			Status:    "pending",
			Action:    fmt.Sprintf("%s action", trigger),
		}

		// Save checkpoint
		if err := m.saveCheckpoint(ctx, req); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}

		// Save approval request
		if err := m.approvalStore.SaveRequest(req); err != nil {
			return nil, fmt.Errorf("failed to save approval request: %w", err)
		}

		// Create decision channel
		m.mu.Lock()
		m.decisionChans[req.ID] = make(chan *ApprovalDecision, 1)
		m.mu.Unlock()

		return req, nil
	}

	return nil, nil
}

// WaitForDecision waits for a human decision on an approval request
func (m *Manager) WaitForDecision(requestID string, timeout time.Duration) (*ApprovalDecision, error) {
	m.mu.RLock()
	// Check if decision already exists
	if decision, exists := m.decisions[requestID]; exists {
		m.mu.RUnlock()
		return decision, nil
	}

	decisionChan, exists := m.decisionChans[requestID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("approval request %s not found", requestID)
	}

	select {
	case decision := <-decisionChan:
		return decision, nil
	case <-time.After(timeout):
		// Mark as expired
		m.approvalStore.UpdateRequestStatus(requestID, "expired")
		return nil, fmt.Errorf("approval request %s expired", requestID)
	}
}

// SubmitDecision submits a human decision for an approval request
func (m *Manager) SubmitDecision(decision *ApprovalDecision) error {
	if decision.ID == "" {
		decision.ID = uuid.New().String()
	}
	decision.CreatedAt = time.Now()

	// Save decision
	if err := m.approvalStore.SaveDecision(decision); err != nil {
		return fmt.Errorf("failed to save decision: %w", err)
	}

	// Update request status
	status := "approved"
	if decision.DecisionType == DecisionReject {
		status = "rejected"
	} else if decision.DecisionType == DecisionModify {
		status = "modified"
	}
	if err := m.approvalStore.UpdateRequestStatus(decision.RequestID, status); err != nil {
		return fmt.Errorf("failed to update request status: %w", err)
	}

	// Store decision
	m.mu.Lock()
	m.decisions[decision.RequestID] = decision

	// Send decision through channel
	if decisionChan, exists := m.decisionChans[decision.RequestID]; exists {
		select {
		case decisionChan <- decision:
		default:
		}
	}
	m.mu.Unlock()

	return nil
}

// GetPendingApprovals returns all pending approval requests for a session
func (m *Manager) GetPendingApprovals(sessionID string) ([]*ApprovalRequest, error) {
	return m.approvalStore.ListPending(sessionID)
}

// RecordFeedback records human feedback
func (m *Manager) RecordFeedback(sessionID, agentID, feedback string, rating int) error {
	record := &FeedbackRecord{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		AgentID:   agentID,
		Feedback:  feedback,
		Rating:    rating,
		CreatedAt: time.Now(),
	}
	return m.feedbackStore.SaveFeedback(record)
}

// saveCheckpoint saves the current execution state
func (m *Manager) saveCheckpoint(ctx runtime.Context, req *ApprovalRequest) error {
	state := ctx.State()
	snapshot := &runtime.StateSnapshot{
		ID:       uuid.New().String(),
		ThreadID: ctx.TraceID(),
		Values:   m.stateToMap(state),
		Metadata: map[string]interface{}{
			"approval_request_id": req.ID,
			"trigger":             req.Trigger,
		},
		CreatedAt: time.Now(),
	}

	config := map[string]interface{}{
		"thread_id": ctx.TraceID(),
	}

	return m.checkpointer.Put(config, snapshot, nil)
}

// stateToMap converts state to a map
func (m *Manager) stateToMap(state runtime.State) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range state.Keys() {
		if value, ok := state.Get(key); ok {
			result[key] = value
		}
	}
	return result
}

// extractContext extracts relevant context from data
func (m *Manager) extractContext(data interface{}) map[string]interface{} {
	context := make(map[string]interface{})
	if data != nil {
		context["data"] = data
	}
	return context
}
