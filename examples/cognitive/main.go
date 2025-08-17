package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/voocel/mas"
	"github.com/voocel/mas/skills"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	fmt.Println("=== MAS Cognitive Architecture Demo ===")

	// Demo 1: Basic cognitive agent
	basicCognitiveDemo(apiKey)

	// Demo 2: Skill execution
	skillExecutionDemo(apiKey)

	// Demo 3: Planning and reasoning
	planningReasoningDemo(apiKey)

	// Demo 4: Reactive behavior
	reactiveBehaviorDemo(apiKey)

	// Demo 5: Cognitive layers in action
	cognitiveLayers(apiKey)
}

// basicCognitiveDemo shows basic cognitive capabilities
func basicCognitiveDemo(apiKey string) {
	fmt.Println("\n1. Basic Cognitive Agent:")

	// Create agent with cognitive capabilities
	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are an intelligent assistant with cognitive capabilities.")

	// Check cognitive state
	state := agent.GetCognitiveState()
	fmt.Printf("Initial cognitive state: Layer=%s, Mode=%s\n",
		state.CurrentLayer, state.Mode)

	// Basic chat - automatically uses appropriate layer
	response, err := agent.Chat(context.Background(), "Hello! How are you today?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent response: %s\n", response)

	// Check updated state
	state = agent.GetCognitiveState()
	fmt.Printf("Updated cognitive state: Layer=%s\n", state.CurrentLayer)
}

// skillExecutionDemo shows skill-based execution
func skillExecutionDemo(apiKey string) {
	fmt.Println("\n2. Skill Execution Demo:")

	// Create agent with skills
	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a skilled assistant.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
			skills.QuickResponseSkill(),
		)

	ctx := context.Background()

	// Execute math skill
	fmt.Println("Executing math skill:")
	result, err := agent.ExecuteSkill(ctx, "math_calculation", map[string]interface{}{
		"expression": "15 + 25 * 2",
	})
	if err != nil {
		log.Printf("Math skill error: %v", err)
	} else {
		fmt.Printf("Math result: %v\n", result)
	}

	// Execute text analysis skill
	fmt.Println("\nExecuting text analysis skill:")
	result, err = agent.ExecuteSkill(ctx, "text_analysis", map[string]interface{}{
		"text": "This is a wonderful day! I feel great and excited about the amazing possibilities ahead.",
	})
	if err != nil {
		log.Printf("Text analysis error: %v", err)
	} else {
		fmt.Printf("Text analysis: %v\n", result)
	}

	// Show loaded skills
	skillLib := agent.GetSkillLibrary()
	allSkills := skillLib.ListSkills()
	fmt.Printf("\nLoaded skills: ")
	for _, skill := range allSkills {
		fmt.Printf("%s (%s) ", skill.Name(), skill.Layer())
	}
	fmt.Println()
}

// planningReasoningDemo shows high-level cognitive functions
func planningReasoningDemo(apiKey string) {
	fmt.Println("\n3. Planning and Reasoning Demo:")

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a strategic planner and analyst.").
		WithSkills(skills.PlanningSkill())

	ctx := context.Background()

	// Create a plan
	fmt.Println("Creating a plan:")
	plan, err := agent.Plan(ctx, "Organize a team meeting to discuss project progress")
	if err != nil {
		log.Printf("Planning error: %v", err)
	} else {
		fmt.Printf("Plan created: %s\n", plan.Goal)
		fmt.Printf("Plan ID: %s\n", plan.ID)
		fmt.Printf("LLM Response: %s\n", plan.Context["llm_response"])
	}

	// Reasoning about a situation
	fmt.Println("\nReasoning about situation:")
	situation := mas.NewSituation(
		map[string]interface{}{
			"project_status": "behind_schedule",
			"team_morale":    "medium",
			"deadline":       "2 weeks",
		},
		[]string{"team meeting requested", "project delays reported"},
	)
	situation.Constraints = []string{"limited budget", "team availability"}
	situation.Goals = []string{"get back on schedule", "improve team coordination"}

	decision, err := agent.Reason(ctx, situation)
	if err != nil {
		log.Printf("Reasoning error: %v", err)
	} else {
		fmt.Printf("Decision: %s\n", decision.Action)
		fmt.Printf("Confidence: %.2f\n", decision.Confidence)
		fmt.Printf("Layer: %s\n", decision.Layer)
		fmt.Printf("Reasoning: %s\n", decision.Reasoning)
	}
}

// reactiveBehaviorDemo shows reactive capabilities
func reactiveBehaviorDemo(apiKey string) {
	fmt.Println("\n4. Reactive Behavior Demo:")

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a responsive assistant.").
		WithSkills(skills.QuickResponseSkill()).
		SetCognitiveMode(mas.ReflexMode)

	ctx := context.Background()

	// High urgency stimulus
	fmt.Println("High urgency stimulus:")
	highUrgencyStimulus := mas.NewStimulus("emergency", map[string]interface{}{
		"message":  "System failure detected",
		"severity": "critical",
	}, 0.9)

	action, err := agent.React(ctx, highUrgencyStimulus)
	if err != nil {
		log.Printf("Reaction error: %v", err)
	} else {
		fmt.Printf("Reaction: %s (Layer: %s, Priority: %d)\n",
			action.Type, action.Layer, action.Priority)
	}

	// Low urgency stimulus
	fmt.Println("\nLow urgency stimulus:")
	lowUrgencyStimulus := mas.NewStimulus("info_request", map[string]interface{}{
		"question": "What's the weather like?",
	}, 0.3)

	action, err = agent.React(ctx, lowUrgencyStimulus)
	if err != nil {
		log.Printf("Reaction error: %v", err)
	} else {
		fmt.Printf("Reaction: %s (Layer: %s, Priority: %d)\n",
			action.Type, action.Layer, action.Priority)
	}
}

// cognitiveLayers shows different cognitive processing modes
func cognitiveLayers(apiKey string) {
	fmt.Println("\n5. Cognitive Layers in Action:")

	baseAgent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are an adaptive cognitive agent.").
		WithSkills(
			skills.QuickResponseSkill(),
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
			skills.PlanningSkill(),
		)

	ctx := context.Background()

	// Test different cognitive modes
	modes := []mas.CognitiveMode{
		mas.ReflexMode,
		mas.SkillMode,
		mas.ReasoningMode,
		mas.PlanningMode,
		mas.AutomaticMode,
	}

	for _, mode := range modes {
		fmt.Printf("\nTesting %s mode:\n", mode)

		agent := baseAgent.SetCognitiveMode(mode)

		response, err := agent.Chat(ctx, "I need to calculate 25 * 4 and then analyze the result")
		if err != nil {
			log.Printf("Error in %s mode: %v", mode, err)
			continue
		}

		state := agent.GetCognitiveState()
		fmt.Printf("Mode: %s, Current Layer: %s\n", state.Mode, state.CurrentLayer)
		fmt.Printf("Response: %s\n", response)
	}
}
