package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas"
	"github.com/voocel/mas/skills"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	fmt.Println("=== MAS Dynamic Topology Demo ===")

	// Demo 1: Basic topology setup
	basicTopologyDemo(apiKey)

	// Demo 2: Star topology collaboration
	starTopologyDemo(apiKey)

	// Demo 3: Dynamic topology adaptation
	adaptiveTopologyDemo(apiKey)

	// Demo 4: Task distribution and load balancing
	taskDistributionDemo(apiKey)

	// Demo 5: Complete topology lifecycle
	topologyLifecycleDemo(apiKey)
}

// basicTopologyDemo shows basic topology setup
func basicTopologyDemo(apiKey string) {
	fmt.Println("\n1. Basic Topology Setup:")

	// Create dynamic topology
	topology := mas.NewDynamicTopology(mas.StarTopology, mas.CooperativeMode)

	// Create agent nodes with different roles
	coordinator := createAgentNode(apiKey, "coordinator", mas.CoordinatorRole, []string{"planning", "coordination"})
	worker1 := createAgentNode(apiKey, "worker1", mas.WorkerRole, []string{"math", "analysis"})
	worker2 := createAgentNode(apiKey, "worker2", mas.WorkerRole, []string{"text", "writing"})
	specialist := createAgentNode(apiKey, "specialist", mas.SpecialistRole, []string{"research", "analysis"})

	// Add nodes to topology
	topology.AddNode(coordinator)
	topology.AddNode(worker1)
	topology.AddNode(worker2)
	topology.AddNode(specialist)

	// Check topology metrics
	metrics := topology.GetMetrics()
	fmt.Printf("Topology created with %d nodes and %d edges\n", metrics.TotalNodes, metrics.TotalEdges)
	fmt.Printf("Topology Type: %s\n", topology.GetTopologyType())
	fmt.Printf("Collaboration Mode: %s\n", topology.GetCollaborationMode())
	fmt.Printf("Average Load: %.2f\n", metrics.AverageLoad)
	fmt.Printf("Network Efficiency: %.2f\n", metrics.NetworkEfficiency)
}

// starTopologyDemo shows star topology collaboration
func starTopologyDemo(apiKey string) {
	fmt.Println("\n2. Star Topology Collaboration:")

	topology := mas.NewDynamicTopology(mas.StarTopology, mas.DelegationMode)

	// Create nodes with varying capabilities
	coordinator := createAgentNode(apiKey, "central_coordinator", mas.CoordinatorRole,
		[]string{"planning", "coordination", "decision_making"})

	mathWorker := createAgentNode(apiKey, "math_worker", mas.WorkerRole,
		[]string{"mathematics", "calculation"})

	textWorker := createAgentNode(apiKey, "text_worker", mas.WorkerRole,
		[]string{"text_analysis", "writing"})

	researchSpecialist := createAgentNode(apiKey, "research_specialist", mas.SpecialistRole,
		[]string{"research", "data_analysis"})

	// Add nodes
	topology.AddNode(coordinator)
	topology.AddNode(mathWorker)
	topology.AddNode(textWorker)
	topology.AddNode(researchSpecialist)

	ctx := context.Background()

	// Create and distribute tasks
	tasks := []*mas.CollaborationTask{
		mas.NewCollaborationTask("data_analysis", 3, []string{"mathematics", "research"}),
		mas.NewCollaborationTask("report_writing", 2, []string{"text_analysis", "writing"}),
		mas.NewCollaborationTask("calculation_task", 1, []string{"mathematics"}),
	}

	fmt.Printf("Distributing %d tasks across topology...\n", len(tasks))

	for _, task := range tasks {
		assignment, err := topology.DistributeTask(ctx, task)
		if err != nil {
			log.Printf("Failed to distribute task %s: %v", task.ID, err)
			continue
		}

		fmt.Printf("Task %s (%s) assigned to %v (Coordinator: %s, Confidence: %.2f)\n",
			task.ID, task.Type, assignment.AssignedTo, assignment.Coordinator, assignment.Confidence)
	}

	// Check performance
	analysis, err := topology.AnalyzePerformance(ctx)
	if err == nil {
		fmt.Printf("Topology Analysis:\n")
		fmt.Printf("- Overall Health: %.2f\n", analysis.OverallHealth)
		fmt.Printf("- Efficiency Score: %.2f\n", analysis.EfficiencyScore)
		fmt.Printf("- Scalability Score: %.2f\n", analysis.ScalabilityScore)
		fmt.Printf("- Resilience Score: %.2f\n", analysis.ResilienceScore)
		if len(analysis.Recommendations) > 0 {
			fmt.Printf("- Recommendations: %v\n", analysis.Recommendations)
		}
	}
}

// adaptiveTopologyDemo shows dynamic topology adaptation
func adaptiveTopologyDemo(apiKey string) {
	fmt.Println("\n3. Dynamic Topology Adaptation:")

	topology := mas.NewDynamicTopology(mas.AdaptiveTopology, mas.SwarmMode)

	// Create diverse agent network
	agents := []*mas.TopologyNode{
		createAgentNode(apiKey, "leader", mas.CoordinatorRole, []string{"leadership", "planning"}),
		createAgentNode(apiKey, "analyst1", mas.SpecialistRole, []string{"data_analysis", "statistics"}),
		createAgentNode(apiKey, "analyst2", mas.SpecialistRole, []string{"text_analysis", "research"}),
		createAgentNode(apiKey, "worker1", mas.WorkerRole, []string{"calculation", "processing"}),
		createAgentNode(apiKey, "worker2", mas.WorkerRole, []string{"writing", "formatting"}),
		createAgentNode(apiKey, "broker", mas.BrokerRole, []string{"communication", "routing"}),
	}

	for _, agent := range agents {
		topology.AddNode(agent)
	}

	ctx := context.Background()

	fmt.Printf("Initial topology: %s\n", topology.GetTopologyType())

	// Test different workload patterns and adaptation
	workloadPatterns := []*mas.WorkloadPattern{
		{
			TaskTypes:        []string{"analysis", "calculation"},
			IntensityProfile: map[string]float64{"analysis": 0.9, "calculation": 0.5},
			TimePattern:      "peak",
			Duration:         time.Hour,
		},
		{
			TaskTypes:        []string{"communication", "routing"},
			IntensityProfile: map[string]float64{"communication": 0.8, "routing": 0.7},
			TimePattern:      "burst",
			Duration:         time.Minute * 30,
		},
		{
			TaskTypes:        []string{"processing", "writing"},
			IntensityProfile: map[string]float64{"processing": 0.6, "writing": 0.4},
			TimePattern:      "constant",
			Duration:         time.Hour * 2,
		},
	}

	for i, workload := range workloadPatterns {
		fmt.Printf("\nAdapting to workload pattern %d (%s):\n", i+1, workload.TimePattern)

		err := topology.AdaptToWorkload(ctx, workload)
		if err != nil {
			log.Printf("Failed to adapt to workload: %v", err)
			continue
		}

		fmt.Printf("Adapted topology type: %s\n", topology.GetTopologyType())

		// Check metrics after adaptation
		metrics := topology.GetMetrics()
		fmt.Printf("Post-adaptation metrics: Load Balance: %.2f, Efficiency: %.2f\n",
			metrics.LoadBalance, metrics.NetworkEfficiency)
	}

	// Test topology optimization
	fmt.Printf("\nOptimizing topology...\n")
	err := topology.OptimizeTopology(ctx)
	if err == nil {
		fmt.Printf("Topology optimized successfully\n")
		finalMetrics := topology.GetMetrics()
		fmt.Printf("Final metrics: Load Balance: %.2f, Efficiency: %.2f\n",
			finalMetrics.LoadBalance, finalMetrics.NetworkEfficiency)
	}
}

// taskDistributionDemo shows intelligent task distribution
func taskDistributionDemo(apiKey string) {
	fmt.Println("\n4. Intelligent Task Distribution:")

	topology := mas.NewDynamicTopology(mas.HubTopology, mas.SpecializationMode)

	// Create specialized nodes
	nodes := []*mas.TopologyNode{
		createAgentNode(apiKey, "math_hub", mas.CoordinatorRole, []string{"mathematics", "statistics", "calculation"}),
		createAgentNode(apiKey, "text_hub", mas.CoordinatorRole, []string{"text_analysis", "writing", "research"}),
		createAgentNode(apiKey, "math_worker1", mas.WorkerRole, []string{"mathematics", "calculation"}),
		createAgentNode(apiKey, "math_worker2", mas.WorkerRole, []string{"mathematics", "statistics"}),
		createAgentNode(apiKey, "text_worker1", mas.WorkerRole, []string{"text_analysis", "writing"}),
		createAgentNode(apiKey, "text_worker2", mas.WorkerRole, []string{"research", "analysis"}),
	}

	for _, node := range nodes {
		topology.AddNode(node)
	}

	// Simulate different load levels
	topology.UpdateNodeStatus("math_worker1", 0.9, 0.8)  // High load, good performance
	topology.UpdateNodeStatus("math_worker2", 0.3, 0.9)  // Low load, excellent performance
	topology.UpdateNodeStatus("text_worker1", 0.7, 0.7)  // Medium load and performance
	topology.UpdateNodeStatus("text_worker2", 0.2, 0.95) // Low load, excellent performance

	ctx := context.Background()

	// Create diverse tasks
	complexTasks := []*mas.CollaborationTask{
		mas.NewCollaborationTask("complex_calculation", 5, []string{"mathematics", "statistics"}),
		mas.NewCollaborationTask("research_analysis", 4, []string{"research", "text_analysis"}),
		mas.NewCollaborationTask("simple_math", 2, []string{"mathematics"}),
		mas.NewCollaborationTask("document_writing", 3, []string{"writing", "text_analysis"}),
		mas.NewCollaborationTask("statistical_report", 4, []string{"statistics", "writing"}),
	}

	fmt.Printf("Distributing %d tasks with load balancing...\n", len(complexTasks))

	for _, task := range complexTasks {
		assignment, err := topology.DistributeTask(ctx, task)
		if err != nil {
			log.Printf("Failed to distribute task %s: %v", task.ID, err)
			continue
		}

		fmt.Printf("Task: %s (Priority: %d)\n", task.Type, task.Priority)
		fmt.Printf("  → Assigned to: %v\n", assignment.AssignedTo)
		fmt.Printf("  → Coordinator: %s\n", assignment.Coordinator)
		fmt.Printf("  → Confidence: %.2f\n", assignment.Confidence)
		fmt.Printf("  → Estimated Time: %v\n", assignment.EstimatedTime)
	}

	// Predict bottlenecks
	predictions, err := topology.PredictBottlenecks(ctx)
	if err == nil && len(predictions) > 0 {
		fmt.Printf("\nBottleneck Predictions:\n")
		for _, pred := range predictions {
			fmt.Printf("- Node %s: Load %.2f → %.2f (in %v, Confidence: %.2f)\n",
				pred.NodeID, topology.GetMetrics().AverageLoad, pred.PredictedLoad,
				pred.TimeToBottleneck, pred.Confidence)
			fmt.Printf("  Mitigation: %v\n", pred.Mitigation)
		}
	}
}

// topologyLifecycleDemo shows complete topology management lifecycle
func topologyLifecycleDemo(apiKey string) {
	fmt.Println("\n5. Complete Topology Lifecycle:")

	// Start with simple topology
	topology := mas.NewDynamicTopology(mas.ChainTopology, mas.CooperativeMode)

	// Subscribe to topology events
	topology.Subscribe("node_added", func(event *mas.TopologyEvent) {
		fmt.Printf("Event: Node %s added\n", event.NodeID)
	})

	topology.Subscribe("topology_type_changed", func(event *mas.TopologyEvent) {
		fmt.Printf("Event: Topology changed to %s\n", event.Data["new_type"])
	})

	topology.Subscribe("topology_optimized", func(event *mas.TopologyEvent) {
		fmt.Printf("Event: Topology optimized (Efficiency: %.2f)\n", event.Data["efficiency_score"])
	})

	ctx := context.Background()

	// Phase 1: Build initial topology
	fmt.Printf("\nPhase 1: Building initial topology\n")
	initialNodes := []*mas.TopologyNode{
		createAgentNode(apiKey, "node1", mas.WorkerRole, []string{"task1"}),
		createAgentNode(apiKey, "node2", mas.WorkerRole, []string{"task2"}),
		createAgentNode(apiKey, "node3", mas.WorkerRole, []string{"task3"}),
	}

	for _, node := range initialNodes {
		topology.AddNode(node)
	}

	// Phase 2: Scale up
	fmt.Printf("\nPhase 2: Scaling up topology\n")
	for i := 4; i <= 8; i++ {
		node := createAgentNode(apiKey, fmt.Sprintf("node%d", i), mas.WorkerRole,
			[]string{fmt.Sprintf("task%d", i%4+1)})
		topology.AddNode(node)
	}

	// Phase 3: Add coordinator and reorganize
	fmt.Printf("\nPhase 3: Adding coordination layer\n")
	coordinator := createAgentNode(apiKey, "main_coordinator", mas.CoordinatorRole,
		[]string{"coordination", "planning"})
	topology.AddNode(coordinator)

	// Switch to star topology for better coordination
	topology.SetTopologyType(mas.StarTopology)

	// Phase 4: Optimize for performance
	fmt.Printf("\nPhase 4: Performance optimization\n")

	// Test different optimization criteria
	optimizationTests := []mas.OptimizationCriteria{
		{
			BalanceLoad:        true,
			WeightBalance:      0.8,
			MinimizeLatency:    false,
			MaximizeThroughput: false,
		},
		{
			MinimizeLatency: true,
			WeightLatency:   0.7,
			BalanceLoad:     true,
			WeightBalance:   0.3,
		},
		{
			MaximizeThroughput: true,
			WeightThroughput:   0.9,
			MinimizeLatency:    false,
			BalanceLoad:        false,
		},
	}

	for i, criteria := range optimizationTests {
		fmt.Printf("\nOptimization test %d:\n", i+1)

		err := topology.ReorganizeTopology(ctx, criteria)
		if err == nil {
			metrics := topology.GetMetrics()
			fmt.Printf("- Topology: %s\n", topology.GetTopologyType())
			fmt.Printf("- Load Balance: %.2f\n", metrics.LoadBalance)
			fmt.Printf("- Network Efficiency: %.2f\n", metrics.NetworkEfficiency)
		}
	}

	// Phase 5: Final analysis and snapshot
	fmt.Printf("\nPhase 5: Final analysis\n")

	finalAnalysis, err := topology.AnalyzePerformance(ctx)
	if err == nil {
		fmt.Printf("Final Topology Analysis:\n")
		fmt.Printf("- Overall Health: %.2f\n", finalAnalysis.OverallHealth)
		fmt.Printf("- Efficiency Score: %.2f\n", finalAnalysis.EfficiencyScore)
		fmt.Printf("- Scalability Score: %.2f\n", finalAnalysis.ScalabilityScore)
		fmt.Printf("- Resilience Score: %.2f\n", finalAnalysis.ResilienceScore)
		fmt.Printf("- Optimal Topology: %s\n", finalAnalysis.OptimalTopology)

		if len(finalAnalysis.Recommendations) > 0 {
			fmt.Printf("- Recommendations: %v\n", finalAnalysis.Recommendations)
		}
	}

	// Take topology snapshot
	snapshot := topology.GetTopologySnapshot()
	fmt.Printf("\nTopology Snapshot:\n")
	fmt.Printf("- Timestamp: %s\n", snapshot.Timestamp.Format("15:04:05"))
	fmt.Printf("- Type: %s\n", snapshot.Type)
	fmt.Printf("- Mode: %s\n", snapshot.Mode)
	fmt.Printf("- Nodes: %d\n", len(snapshot.Nodes))
	fmt.Printf("- Edges: %d\n", len(snapshot.Edges))
	fmt.Printf("- Metrics: Load Balance %.2f, Efficiency %.2f\n",
		snapshot.Metrics.LoadBalance, snapshot.Metrics.NetworkEfficiency)

	fmt.Printf("\nTopology lifecycle completed successfully!\n")
}

// Helper function to create agent nodes
func createAgentNode(apiKey, name string, role mas.AgentRole, capabilities []string) *mas.TopologyNode {
	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt(fmt.Sprintf("You are a %s agent with capabilities: %v", role, capabilities))

	// Add appropriate skills based on capabilities
	for _, capability := range capabilities {
		switch capability {
		case "mathematics", "calculation", "statistics":
			agent = agent.WithSkills(skills.MathSkill())
		case "text_analysis", "writing", "research":
			agent = agent.WithSkills(skills.TextAnalysisSkill())
		case "planning", "coordination":
			agent = agent.WithSkills(skills.PlanningSkill())
		}
	}

	return mas.NewTopologyNode(agent, role, capabilities)
}
