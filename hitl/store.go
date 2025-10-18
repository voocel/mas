package hitl

import (
	"fmt"
	"sync"
	"time"
)

// MemoryApprovalStore is an in-memory implementation of ApprovalStore
type MemoryApprovalStore struct {
	requests map[string]*ApprovalRequest
	decisions map[string]*ApprovalDecision
	mu       sync.RWMutex
}

// NewMemoryApprovalStore creates a new in-memory approval store
func NewMemoryApprovalStore() *MemoryApprovalStore {
	return &MemoryApprovalStore{
		requests: make(map[string]*ApprovalRequest),
		decisions: make(map[string]*ApprovalDecision),
	}
}

// SaveRequest saves an approval request
func (s *MemoryApprovalStore) SaveRequest(req *ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.ID == "" {
		return fmt.Errorf("approval request ID cannot be empty")
	}

	s.requests[req.ID] = req
	return nil
}

// GetRequest retrieves an approval request
func (s *MemoryApprovalStore) GetRequest(requestID string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	req, exists := s.requests[requestID]
	if !exists {
		return nil, fmt.Errorf("approval request %s not found", requestID)
	}

	return req, nil
}

// ListPending lists all pending approval requests for a session
func (s *MemoryApprovalStore) ListPending(sessionID string) ([]*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []*ApprovalRequest
	for _, req := range s.requests {
		if req.SessionID == sessionID && req.Status == "pending" {
			pending = append(pending, req)
		}
	}

	return pending, nil
}

// SaveDecision saves an approval decision
func (s *MemoryApprovalStore) SaveDecision(decision *ApprovalDecision) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if decision.ID == "" {
		return fmt.Errorf("decision ID cannot be empty")
	}

	s.decisions[decision.ID] = decision
	return nil
}

// GetDecision retrieves an approval decision
func (s *MemoryApprovalStore) GetDecision(requestID string) (*ApprovalDecision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, decision := range s.decisions {
		if decision.RequestID == requestID {
			return decision, nil
		}
	}

	return nil, fmt.Errorf("decision for request %s not found", requestID)
}

// UpdateRequestStatus updates the status of an approval request
func (s *MemoryApprovalStore) UpdateRequestStatus(requestID string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, exists := s.requests[requestID]
	if !exists {
		return fmt.Errorf("approval request %s not found", requestID)
	}

	req.Status = status
	return nil
}

// MemoryFeedbackStore is an in-memory implementation of FeedbackStore
type MemoryFeedbackStore struct {
	feedbacks map[string]*FeedbackRecord
	mu        sync.RWMutex
}

// NewMemoryFeedbackStore creates a new in-memory feedback store
func NewMemoryFeedbackStore() *MemoryFeedbackStore {
	return &MemoryFeedbackStore{
		feedbacks: make(map[string]*FeedbackRecord),
	}
}

// SaveFeedback saves a feedback record
func (s *MemoryFeedbackStore) SaveFeedback(feedback *FeedbackRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if feedback.ID == "" {
		return fmt.Errorf("feedback ID cannot be empty")
	}

	s.feedbacks[feedback.ID] = feedback
	return nil
}

// GetFeedback retrieves feedback for a session
func (s *MemoryFeedbackStore) GetFeedback(sessionID string) ([]*FeedbackRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*FeedbackRecord
	for _, feedback := range s.feedbacks {
		if feedback.SessionID == sessionID {
			result = append(result, feedback)
		}
	}

	return result, nil
}

// GetAgentFeedback retrieves feedback for an agent
func (s *MemoryFeedbackStore) GetAgentFeedback(agentID string) ([]*FeedbackRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*FeedbackRecord
	for _, feedback := range s.feedbacks {
		if feedback.AgentID == agentID {
			result = append(result, feedback)
		}
	}

	return result, nil
}

// GetAverageRating calculates average rating for an agent
func (s *MemoryFeedbackStore) GetAverageRating(agentID string) (float64, error) {
	feedbacks, err := s.GetAgentFeedback(agentID)
	if err != nil {
		return 0, err
	}

	if len(feedbacks) == 0 {
		return 0, nil
	}

	total := 0
	for _, feedback := range feedbacks {
		total += feedback.Rating
	}

	return float64(total) / float64(len(feedbacks)), nil
}

// GetRecentFeedback retrieves recent feedback within a time window
func (s *MemoryFeedbackStore) GetRecentFeedback(agentID string, duration time.Duration) ([]*FeedbackRecord, error) {
	feedbacks, err := s.GetAgentFeedback(agentID)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-duration)
	var result []*FeedbackRecord
	for _, feedback := range feedbacks {
		if feedback.CreatedAt.After(cutoff) {
			result = append(result, feedback)
		}
	}

	return result, nil
}

