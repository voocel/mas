package hitl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// APIServer provides HTTP endpoints for HITL management
type APIServer struct {
	manager *Manager
	addr    string
}

// NewAPIServer creates a new HITL API server
func NewAPIServer(manager *Manager, addr string) *APIServer {
	return &APIServer{
		manager: manager,
		addr:    addr,
	}
}

// Start starts the API server
func (s *APIServer) Start() error {
	http.HandleFunc("/api/approvals/pending", s.handleGetPendingApprovals)
	http.HandleFunc("/api/approvals/submit", s.handleSubmitDecision)
	http.HandleFunc("/api/approvals/get", s.handleGetApproval)
	http.HandleFunc("/api/feedback/submit", s.handleSubmitFeedback)
	http.HandleFunc("/api/feedback/agent", s.handleGetAgentFeedback)

	return http.ListenAndServe(s.addr, nil)
}

// Request/Response types
type GetPendingApprovalsRequest struct {
	SessionID string `json:"session_id"`
}

type GetPendingApprovalsResponse struct {
	Approvals []*ApprovalRequest `json:"approvals"`
	Error     string             `json:"error,omitempty"`
}

type SubmitDecisionRequest struct {
	RequestID    string                 `json:"request_id"`
	DecisionType string                 `json:"decision_type"`
	ApprovedBy   string                 `json:"approved_by"`
	Reason       string                 `json:"reason"`
	ModifiedData map[string]interface{} `json:"modified_data,omitempty"`
	Feedback     string                 `json:"feedback,omitempty"`
}

type SubmitDecisionResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type GetApprovalRequest struct {
	RequestID string `json:"request_id"`
}

type GetApprovalResponse struct {
	Approval *ApprovalRequest  `json:"approval"`
	Decision *ApprovalDecision `json:"decision,omitempty"`
	Error    string            `json:"error,omitempty"`
}

type SubmitFeedbackRequest struct {
	SessionID string `json:"session_id"`
	AgentID   string `json:"agent_id"`
	Feedback  string `json:"feedback"`
	Rating    int    `json:"rating"`
}

type SubmitFeedbackResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type GetAgentFeedbackResponse struct {
	Feedback      []*FeedbackRecord `json:"feedback"`
	AverageRating float64           `json:"average_rating"`
	TotalFeedback int               `json:"total_feedback"`
	Error         string            `json:"error,omitempty"`
}

// Handler methods
func (s *APIServer) handleGetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetPendingApprovalsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetPendingApprovalsResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	approvals, err := s.manager.GetPendingApprovals(req.SessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetPendingApprovalsResponse{
			Error: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetPendingApprovalsResponse{
		Approvals: approvals,
	})
}

func (s *APIServer) handleSubmitDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SubmitDecisionResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	decision := &ApprovalDecision{
		ID:           uuid.New().String(),
		RequestID:    req.RequestID,
		DecisionType: DecisionType(req.DecisionType),
		ApprovedBy:   req.ApprovedBy,
		Reason:       req.Reason,
		ModifiedData: req.ModifiedData,
		Feedback:     req.Feedback,
		CreatedAt:    time.Now(),
	}

	if err := s.manager.SubmitDecision(decision); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SubmitDecisionResponse{
			Error: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SubmitDecisionResponse{
		Success: true,
	})
}

func (s *APIServer) handleGetApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetApprovalResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	approval, err := s.manager.approvalStore.GetRequest(req.RequestID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetApprovalResponse{
			Error: err.Error(),
		})
		return
	}

	decision, _ := s.manager.approvalStore.GetDecision(req.RequestID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetApprovalResponse{
		Approval: approval,
		Decision: decision,
	})
}

func (s *APIServer) handleSubmitFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SubmitFeedbackResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if err := s.manager.RecordFeedback(req.SessionID, req.AgentID, req.Feedback, req.Rating); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SubmitFeedbackResponse{
			Error: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SubmitFeedbackResponse{
		Success: true,
	})
}

func (s *APIServer) handleGetAgentFeedback(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetAgentFeedbackResponse{
			Error: "agent_id is required",
		})
		return
	}

	feedbacks, err := s.manager.feedbackStore.GetAgentFeedback(agentID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetAgentFeedbackResponse{
			Error: err.Error(),
		})
		return
	}

	// Calculate average rating
	avgRating := 0.0
	if len(feedbacks) > 0 {
		total := 0
		for _, f := range feedbacks {
			total += f.Rating
		}
		avgRating = float64(total) / float64(len(feedbacks))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetAgentFeedbackResponse{
		Feedback:      feedbacks,
		AverageRating: avgRating,
		TotalFeedback: len(feedbacks),
	})
}
