package workflow

import (
	"context"
	"fmt"
)

// executeFrom executes workflow starting from a specific node
func (b *WorkflowBuilder) executeFrom(ctx context.Context, nodeID string, wfCtx *WorkflowContext) (*WorkflowContext, error) {
	visited := make(map[string]bool)
	queue := []string{nodeID}
	var completedNodes []string

	for len(queue) > 0 {
		currentNodeID := queue[0]
		queue = queue[1:]

		if visited[currentNodeID] {
			continue // Avoid cycles
		}
		visited[currentNodeID] = true

		select {
		case <-ctx.Done():
			return wfCtx, ctx.Err()
		default:
		}

		// Get and validate node
		node, exists := b.nodes[currentNodeID]
		if !exists {
			return wfCtx, fmt.Errorf("node %s not found", currentNodeID)
		}

		// Save checkpoint before node execution if enabled
		if b.enableCheckpoints && b.checkpointer != nil && b.checkpointConfig.SaveBeforeNode {
			// Save before-node checkpoint
			// This would use the actual checkpoint saving logic
		}

		// Execute the node
		if err := node.Execute(ctx, wfCtx); err != nil {
			// Save checkpoint on error if enabled
			if b.enableCheckpoints && b.checkpointer != nil {
				// Save error checkpoint
			}
			return wfCtx, fmt.Errorf("node %s execution failed: %w", currentNodeID, err)
		}

		// Mark node as completed
		completedNodes = append(completedNodes, currentNodeID)

		// Save checkpoint after node execution if enabled
		if b.enableCheckpoints && b.checkpointer != nil && b.checkpointConfig.SaveAfterNode {
			// Save after-node checkpoint
		}

		// Check if node specified next node (for conditional routing)
		if nextNode := wfCtx.Get("next_node"); nextNode != nil {
			if nextNodeStr, ok := nextNode.(string); ok && nextNodeStr != "" {
				wfCtx.Set("next_node", nil) // Clear for next iteration
				queue = append(queue, nextNodeStr)
				continue
			}
		}

		// Add next nodes to queue based on edges
		if nextNodes, exists := b.edges[currentNodeID]; exists {
			queue = append(queue, nextNodes...)
		}
	}

	// Save final checkpoint if enabled
	if b.enableCheckpoints && b.checkpointer != nil && b.checkpointConfig.AutoSave {
		// Save final checkpoint
	}

	return wfCtx, nil
}

// executeFromWithSkip executes workflow starting from a specific node, skipping completed nodes
func (b *WorkflowBuilder) executeFromWithSkip(ctx context.Context, startNodeID string, wfCtx *WorkflowContext, completedNodes []string) (*WorkflowContext, error) {
	if startNodeID == "" {
		// All nodes completed
		return wfCtx, nil
	}

	visited := make(map[string]bool)
	queue := []string{startNodeID}
	
	// Mark completed nodes as visited to skip them
	completed := make(map[string]bool)
	for _, nodeID := range completedNodes {
		completed[nodeID] = true
	}

	// Track newly completed nodes
	newlyCompleted := make([]string, len(completedNodes))
	copy(newlyCompleted, completedNodes)

	for len(queue) > 0 {
		currentNodeID := queue[0]
		queue = queue[1:]

		if visited[currentNodeID] {
			continue // Avoid cycles
		}
		visited[currentNodeID] = true

		// Skip if already completed
		if completed[currentNodeID] {
			fmt.Printf("Skipping already completed node: %s\n", currentNodeID)
			
			// Add next nodes to queue
			if nextNodes, exists := b.edges[currentNodeID]; exists {
				queue = append(queue, nextNodes...)
			}
			continue
		}

		select {
		case <-ctx.Done():
			return wfCtx, ctx.Err()
		default:
		}

		// Get and validate node
		node, exists := b.nodes[currentNodeID]
		if !exists {
			return wfCtx, fmt.Errorf("node %s not found", currentNodeID)
		}

		fmt.Printf("Executing node: %s\n", currentNodeID)

		// Execute the node
		if err := node.Execute(ctx, wfCtx); err != nil {
			return wfCtx, fmt.Errorf("node %s execution failed: %w", currentNodeID, err)
		}

		// Mark node as completed
		newlyCompleted = append(newlyCompleted, currentNodeID)
		completed[currentNodeID] = true

		// Check if node specified next node (for conditional routing)
		if nextNode := wfCtx.Get("next_node"); nextNode != nil {
			if nextNodeStr, ok := nextNode.(string); ok && nextNodeStr != "" {
				wfCtx.Set("next_node", nil) // Clear for next iteration
				queue = append(queue, nextNodeStr)
				continue
			}
		}

		// Add next nodes to queue based on edges
		if nextNodes, exists := b.edges[currentNodeID]; exists {
			queue = append(queue, nextNodes...)
		}
	}

	return wfCtx, nil
}

// findNextNode finds the next node to execute based on completed nodes
func (b *WorkflowBuilder) findNextNode(lastNode string, completedNodes []string) string {
	completed := make(map[string]bool)
	for _, nodeID := range completedNodes {
		completed[nodeID] = true
	}

	// If we have a last node, check its edges first
	if lastNode != "" {
		if edges, exists := b.edges[lastNode]; exists {
			for _, nextNodeID := range edges {
				if !completed[nextNodeID] {
					return nextNodeID
				}
			}
		}
	}

	// Find any unfinished node
	for nodeID := range b.nodes {
		if !completed[nodeID] {
			return nodeID
		}
	}

	// All nodes completed or no nodes found, return empty
	return ""
}