package hitl

import (
	"time"

	"github.com/voocel/mas/runtime"
)

// ApprovalTrigger defines when to trigger approval
type ApprovalTrigger string

const (
	TriggerBeforeToolCall ApprovalTrigger = "before_tool_call"
	TriggerAfterToolCall  ApprovalTrigger = "after_tool_call"
	TriggerBeforeHandoff  ApprovalTrigger = "before_handoff"
	TriggerHighRiskAction ApprovalTrigger = "high_risk_action"
	TriggerCostThreshold  ApprovalTrigger = "cost_threshold"
	TriggerCustom         ApprovalTrigger = "custom"
)

// DecisionType defines the type of approval decision
type DecisionType string

const (
	DecisionApprove DecisionType = "approve"
	DecisionReject  DecisionType = "reject"
	DecisionModify  DecisionType = "modify"
)

// ApprovalRequest represents a request for human approval
type ApprovalRequest struct {
	ID         string                 `json:"id"`
	SessionID  string                 `json:"session_id"`
	ThreadID   string                 `json:"thread_id"`
	Trigger    ApprovalTrigger        `json:"trigger"`
	AgentID    string                 `json:"agent_id"`
	Action     string                 `json:"action"`
	Context    map[string]interface{} `json:"context"`
	Checkpoint *runtime.StateSnapshot `json:"checkpoint"`
	CreatedAt  time.Time              `json:"created_at"`
	ExpiresAt  time.Time              `json:"expires_at"`
	Priority   int                    `json:"priority"` // 1-10
	Approvers  []string               `json:"approvers"`
	Status     string                 `json:"status"` // pending, approved, rejected, expired
}

// ApprovalDecision represents a human's decision on an approval request
type ApprovalDecision struct {
	ID           string                 `json:"id"`
	RequestID    string                 `json:"request_id"`
	DecisionType DecisionType           `json:"decision_type"`
	ApprovedBy   string                 `json:"approved_by"`
	Reason       string                 `json:"reason"`
	ModifiedData map[string]interface{} `json:"modified_data"`
	Feedback     string                 `json:"feedback"`
	CreatedAt    time.Time              `json:"created_at"`
}

// PolicyConfig defines an approval policy
type PolicyConfig struct {
	Trigger   ApprovalTrigger
	Condition func(ctx runtime.Context, data interface{}) bool
	Priority  int
	Timeout   time.Duration
	Approvers []string
}

// ApprovalStore defines the interface for storing approval requests and decisions
type ApprovalStore interface {
	// SaveRequest stores an approval request
	SaveRequest(req *ApprovalRequest) error

	// GetRequest retrieves an approval request
	GetRequest(requestID string) (*ApprovalRequest, error)

	// ListPending lists all pending approval requests
	ListPending(sessionID string) ([]*ApprovalRequest, error)

	// SaveDecision stores an approval decision
	SaveDecision(decision *ApprovalDecision) error

	// GetDecision retrieves an approval decision
	GetDecision(requestID string) (*ApprovalDecision, error)

	// UpdateRequestStatus updates the status of an approval request
	UpdateRequestStatus(requestID string, status string) error
}

// Checkpointer interface for saving/restoring state
type Checkpointer interface {
	Put(config map[string]interface{}, snapshot *runtime.StateSnapshot, metadata map[string]interface{}) error
	GetTuple(config map[string]interface{}) (*runtime.CheckpointTuple, error)
}

// ApprovalHandler handles approval logic
type ApprovalHandler interface {
	// ShouldApprove determines if approval is needed
	ShouldApprove(ctx runtime.Context, trigger ApprovalTrigger, data interface{}) bool

	// CreateRequest creates an approval request
	CreateRequest(ctx runtime.Context, trigger ApprovalTrigger, data interface{}) (*ApprovalRequest, error)

	// WaitForDecision waits for a human decision
	WaitForDecision(requestID string, timeout time.Duration) (*ApprovalDecision, error)

	// ApplyDecision applies the human's decision
	ApplyDecision(ctx runtime.Context, decision *ApprovalDecision) error
}

// FeedbackRecord records human feedback
type FeedbackRecord struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	RequestID string                 `json:"request_id"`
	AgentID   string                 `json:"agent_id"`
	Feedback  string                 `json:"feedback"`
	Rating    int                    `json:"rating"` // 1-5
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

// FeedbackStore stores feedback records
type FeedbackStore interface {
	SaveFeedback(feedback *FeedbackRecord) error
	GetFeedback(sessionID string) ([]*FeedbackRecord, error)
	GetAgentFeedback(agentID string) ([]*FeedbackRecord, error)
}
