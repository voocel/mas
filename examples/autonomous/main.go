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

	fmt.Println("=== MAS Autonomous Agent Demo ===")

	// Demo 1: Basic autonomous behavior
	basicAutonomousDemo(apiKey)

	// Demo 2: Goal-oriented autonomous behavior
	goalOrientedDemo(apiKey)

	// Demo 3: Multi-goal autonomous behavior with different strategies
	multiGoalDemo(apiKey)

	// Demo 4: Learning and adaptation
	learningAdaptationDemo(apiKey)
}

// basicAutonomousDemo shows basic autonomous agent setup
func basicAutonomousDemo(apiKey string) {
	fmt.Println("\n1. Basic Autonomous Agent:")

	// Create agent with autonomous capabilities
	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are an autonomous intelligent assistant.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
			skills.PlanningSkill(),
		)

	// Create goal manager
	goalManager := mas.NewGoalManager(agent)
	agent = agent.WithGoalManager(goalManager)

	// Add a simple goal
	goal := mas.NewGoal(
		"Daily Tasks Completion",
		"Complete daily routine tasks including calculations and analysis",
		mas.MediumPriority,
	)

	ctx := context.Background()
	err := agent.AddGoal(ctx, goal)
	if err != nil {
		log.Printf("Error adding goal: %v", err)
		return
	}

	fmt.Printf("Goal added: %s\n", goal.Title)
	fmt.Printf("Goal ID: %s\n", goal.ID)
	fmt.Printf("Priority: %s\n", goal.Priority)
}

// goalOrientedDemo shows goal-oriented autonomous behavior
func goalOrientedDemo(apiKey string) {
	fmt.Println("\n2. Goal-Oriented Autonomous Behavior:")

	// Create cognitive agent with goal management
	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are an autonomous agent capable of pursuing goals.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
			skills.PlanningSkill(),
			skills.QuickResponseSkill(),
		)

	goalManager := mas.NewGoalManager(agent)
	agent = agent.WithGoalManager(goalManager)

	ctx := context.Background()

	// Add multiple goals with different priorities
	goals := []*mas.Goal{
		mas.NewGoal(
			"Data Analysis Task",
			"Analyze customer feedback data and extract insights",
			mas.HighPriority,
		),
		mas.NewGoal(
			"Report Generation",
			"Generate weekly performance report",
			mas.MediumPriority,
		),
		mas.NewGoal(
			"System Monitoring",
			"Monitor system health and respond to alerts",
			mas.CriticalPriority,
		),
	}

	// Set deadline for one goal
	deadline := time.Now().Add(1 * time.Hour)
	goals[0].Deadline = &deadline

	// Add goals
	for _, goal := range goals {
		err := agent.AddGoal(ctx, goal)
		if err != nil {
			log.Printf("Error adding goal %s: %v", goal.Title, err)
			continue
		}
		fmt.Printf("Added goal: %s (Priority: %s)\n", goal.Title, goal.Priority)
	}

	// Start autonomous mode with priority strategy
	err := agent.StartAutonomous(ctx, mas.PriorityStrategy)
	if err != nil {
		log.Printf("Error starting autonomous mode: %v", err)
		return
	}

	fmt.Printf("Autonomous mode started with PriorityStrategy\n")
	fmt.Printf("Agent is autonomous: %t\n", agent.IsAutonomous())

	// Let it run for a short time
	time.Sleep(10 * time.Second)

	// Check progress
	if goalManager := agent.GetGoalManager(); goalManager != nil {
		progress := goalManager.GetOverallProgress()
		fmt.Printf("Overall progress: %.2f%% (%d/%d goals)\n",
			progress.OverallProgress*100, progress.CompletedGoals, progress.TotalGoals)
		fmt.Printf("Active goals: %d\n", progress.ActiveGoals)
	}

	// Stop autonomous mode
	agent.StopAutonomous(ctx)
	fmt.Printf("Autonomous mode stopped\n")
}

// multiGoalDemo shows different autonomous strategies
func multiGoalDemo(apiKey string) {
	fmt.Println("\n3. Multi-Goal Autonomous Strategies:")

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a strategic autonomous agent.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
			skills.PlanningSkill(),
		)

	goalManager := mas.NewGoalManager(agent)
	agent = agent.WithGoalManager(goalManager)

	ctx := context.Background()

	// Add goals for testing different strategies
	testGoals := []*mas.Goal{
		mas.NewGoal("Quick Task A", "Simple computational task", mas.LowPriority),
		mas.NewGoal("Analysis Task B", "Complex data analysis", mas.HighPriority),
		mas.NewGoal("Planning Task C", "Strategic planning exercise", mas.MediumPriority),
		mas.NewGoal("Emergency Task D", "Critical system response", mas.CriticalPriority),
	}

	for _, goal := range testGoals {
		agent.AddGoal(ctx, goal)
	}

	// Test different strategies
	strategies := []mas.AutonomousStrategy{
		mas.SequentialStrategy,
		mas.PriorityStrategy,
		mas.ParallelStrategy,
		mas.AdaptiveStrategy,
	}

	strategyNames := []string{
		"Sequential",
		"Priority",
		"Parallel",
		"Adaptive",
	}

	for i, strategy := range strategies {
		fmt.Printf("\nTesting %s Strategy:\n", strategyNames[i])

		err := agent.StartAutonomous(ctx, strategy)
		if err != nil {
			log.Printf("Error starting %s strategy: %v", strategyNames[i], err)
			continue
		}

		// Run for a short time
		time.Sleep(5 * time.Second)

		// Check progress
		if goalManager := agent.GetGoalManager(); goalManager != nil {
			progress := goalManager.GetOverallProgress()
			fmt.Printf("Progress with %s: %.2f%%\n", strategyNames[i], progress.OverallProgress*100)
		}

		agent.StopAutonomous(ctx)
	}
}

// learningAdaptationDemo shows learning and adaptation capabilities
func learningAdaptationDemo(apiKey string) {
	fmt.Println("\n4. Learning and Adaptation:")

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a learning autonomous agent.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
			skills.PlanningSkill(),
		)

	goalManager := mas.NewGoalManager(agent)
	agent = agent.WithGoalManager(goalManager)

	ctx := context.Background()

	// Add learning goals
	learningGoals := []*mas.Goal{
		mas.NewGoal("Learning Task 1", "Learn from mathematical computations", mas.MediumPriority),
		mas.NewGoal("Learning Task 2", "Learn from text analysis patterns", mas.MediumPriority),
		mas.NewGoal("Learning Task 3", "Learn from planning optimization", mas.HighPriority),
	}

	for _, goal := range learningGoals {
		agent.AddGoal(ctx, goal)
	}

	// Start with adaptive strategy
	err := agent.StartAutonomous(ctx, mas.AdaptiveStrategy)
	if err != nil {
		log.Printf("Error starting adaptive mode: %v", err)
		return
	}

	fmt.Printf("Starting learning phase with AdaptiveStrategy\n")

	// Let it learn for a while
	time.Sleep(8 * time.Second)

	// Get learning insights
	if goalManager := agent.GetGoalManager(); goalManager != nil {
		insights := goalManager.GetLearnings()
		fmt.Printf("Learning Insights:\n")
		fmt.Printf("- Success Rate: %.2f%%\n", insights.PerformanceMetrics.SuccessRate*100)
		fmt.Printf("- Efficiency Score: %.2f\n", insights.PerformanceMetrics.EfficiencyScore)
		fmt.Printf("- Successful Patterns: %d\n", len(insights.SuccessfulPatterns))
		fmt.Printf("- Failure Patterns: %d\n", len(insights.FailurePatterns))
		fmt.Printf("- Last Updated: %s\n", insights.LastUpdated.Format("15:04:05"))

		// Demonstrate strategy adaptation
		fmt.Printf("\nAdapting strategy based on insights...\n")
		err := goalManager.AdaptStrategy(insights)
		if err != nil {
			log.Printf("Error adapting strategy: %v", err)
		} else {
			fmt.Printf("Strategy adapted successfully\n")
		}
	}

	agent.StopAutonomous(ctx)

	// Final progress summary
	if goalManager := agent.GetGoalManager(); goalManager != nil {
		progress := goalManager.GetOverallProgress()
		fmt.Printf("\nFinal Progress Summary:\n")
		fmt.Printf("- Total Goals: %d\n", progress.TotalGoals)
		fmt.Printf("- Completed: %d\n", progress.CompletedGoals)
		fmt.Printf("- Active: %d\n", progress.ActiveGoals)
		fmt.Printf("- Failed: %d\n", progress.FailedGoals)
		fmt.Printf("- Overall Progress: %.2f%%\n", progress.OverallProgress*100)
	}
}
