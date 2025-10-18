package hitl

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// Example demonstrates how to use HITL in a multi-agent system
func Example() {
	// 1. Create stores
	approvalStore := NewMemoryApprovalStore()
	feedbackStore := NewMemoryFeedbackStore()
	checkpointer := runtime.NewMemoryCheckpointer()

	// 2. Create HITL manager
	hitlMgr := NewManager(checkpointer, approvalStore, feedbackStore)

	// 3. Register approval policies
	// Policy 1: High-risk tools need approval
	hitlMgr.RegisterPolicy(PolicyConfig{
		Trigger: TriggerBeforeToolCall,
		Condition: func(ctx runtime.Context, data interface{}) bool {
			if toolCall, ok := data.(schema.ToolCall); ok {
				// Require approval for delete/write operations
				return toolCall.Name == "delete_file" || toolCall.Name == "write_database"
			}
			return false
		},
		Priority:  9,
		Timeout:   5 * time.Minute,
		Approvers: []string{"admin", "security"},
	})

	// Policy 2: Cost threshold
	hitlMgr.RegisterPolicy(CostThresholdPolicy(100.0, 10*time.Minute))

	// 4. Create context
	ctx := runtime.NewContext(context.Background(), "session-1", "trace-1")

	// 5. Simulate a tool call that needs approval
	toolCall := schema.ToolCall{
		ID:   "tool-1",
		Name: "delete_file",
		Args: []byte(`{"path": "/important/file.txt"}`),
	}

	// 6. Check if approval is needed
	approval, err := hitlMgr.CheckApproval(ctx, TriggerBeforeToolCall, toolCall)
	if err != nil {
		fmt.Printf("Error checking approval: %v\n", err)
		return
	}

	if approval != nil {
		fmt.Printf("Approval needed for: %s\n", approval.Action)
		fmt.Printf("Request ID: %s\n", approval.ID)
		fmt.Printf("Priority: %d\n", approval.Priority)

		// Simulate human decision (in real scenario, this comes from API/UI)
		go func() {
			time.Sleep(100 * time.Millisecond)
			decision := &ApprovalDecision{
				RequestID:    approval.ID,
				DecisionType: DecisionApprove,
				ApprovedBy:   "admin@example.com",
				Reason:       "Verified the file is safe to delete",
				Feedback:     "Good decision by the agent",
			}
			hitlMgr.SubmitDecision(decision)
		}()

		// Wait for decision
		decision, err := hitlMgr.WaitForDecision(approval.ID, 1*time.Second)
		if err != nil {
			fmt.Printf("Error waiting for decision: %v\n", err)
			return
		}

		fmt.Printf("Decision: %s by %s\n", decision.DecisionType, decision.ApprovedBy)
		fmt.Printf("Reason: %s\n", decision.Reason)
	}

	// 7. Record feedback
	err = hitlMgr.RecordFeedback("session-1", "agent-1", "Agent made good decisions", 5)
	if err != nil {
		fmt.Printf("Error recording feedback: %v\n", err)
		return
	}

	fmt.Println("HITL workflow completed successfully")
}

// TestHITLApprovalFlow tests the complete approval flow
func TestHITLApprovalFlow(t *testing.T) {
	approvalStore := NewMemoryApprovalStore()
	feedbackStore := NewMemoryFeedbackStore()
	checkpointer := runtime.NewMemoryCheckpointer()

	hitlMgr := NewManager(checkpointer, approvalStore, feedbackStore)

	// Register a simple policy
	hitlMgr.RegisterPolicy(PolicyConfig{
		Trigger: TriggerBeforeToolCall,
		Condition: func(ctx runtime.Context, data interface{}) bool {
			return true // Always require approval
		},
		Priority:  5,
		Timeout:   1 * time.Second,
		Approvers: []string{"admin"},
	})

	ctx := runtime.NewContext(context.Background(), "session-1", "trace-1")

	// Check approval
	approval, err := hitlMgr.CheckApproval(ctx, TriggerBeforeToolCall, "test_data")
	if err != nil {
		t.Fatalf("CheckApproval failed: %v", err)
	}

	if approval == nil {
		t.Fatal("Expected approval request, got nil")
	}

	// Submit decision
	decision := &ApprovalDecision{
		RequestID:    approval.ID,
		DecisionType: DecisionApprove,
		ApprovedBy:   "admin",
		Reason:       "Test approval",
	}

	if err := hitlMgr.SubmitDecision(decision); err != nil {
		t.Fatalf("SubmitDecision failed: %v", err)
	}

	// Wait for decision
	result, err := hitlMgr.WaitForDecision(approval.ID, 1*time.Second)
	if err != nil {
		t.Fatalf("WaitForDecision failed: %v", err)
	}

	if result.DecisionType != DecisionApprove {
		t.Fatalf("Expected DecisionApprove, got %s", result.DecisionType)
	}

	// Verify pending approvals
	pending, err := hitlMgr.GetPendingApprovals("session-1")
	if err != nil {
		t.Fatalf("GetPendingApprovals failed: %v", err)
	}

	if len(pending) != 0 {
		t.Fatalf("Expected 0 pending approvals, got %d", len(pending))
	}
}

// TestHITLRejection tests rejection flow
func TestHITLRejection(t *testing.T) {
	approvalStore := NewMemoryApprovalStore()
	feedbackStore := NewMemoryFeedbackStore()
	checkpointer := runtime.NewMemoryCheckpointer()

	hitlMgr := NewManager(checkpointer, approvalStore, feedbackStore)

	hitlMgr.RegisterPolicy(PolicyConfig{
		Trigger: TriggerBeforeToolCall,
		Condition: func(ctx runtime.Context, data interface{}) bool {
			return true
		},
		Priority:  5,
		Timeout:   1 * time.Second,
		Approvers: []string{"admin"},
	})

	ctx := runtime.NewContext(context.Background(), "session-1", "trace-1")

	approval, err := hitlMgr.CheckApproval(ctx, TriggerBeforeToolCall, "test_data")
	if err != nil {
		t.Fatalf("CheckApproval failed: %v", err)
	}

	if approval == nil {
		t.Fatal("Expected approval request, got nil")
	}

	// Submit rejection
	decision := &ApprovalDecision{
		RequestID:    approval.ID,
		DecisionType: DecisionReject,
		ApprovedBy:   "admin",
		Reason:       "Too risky",
	}

	if err := hitlMgr.SubmitDecision(decision); err != nil {
		t.Fatalf("SubmitDecision failed: %v", err)
	}

	result, err := hitlMgr.WaitForDecision(approval.ID, 1*time.Second)
	if err != nil {
		t.Fatalf("WaitForDecision failed: %v", err)
	}

	if result.DecisionType != DecisionReject {
		t.Fatalf("Expected DecisionReject, got %s", result.DecisionType)
	}
}
