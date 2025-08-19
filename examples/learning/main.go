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

	fmt.Println("=== MAS Learning and Adaptation Demo ===")

	// Demo 1: Basic learning setup
	basicLearningDemo(apiKey)

	// Demo 2: Experience recording and pattern discovery
	experienceAndPatternsDemo(apiKey)

	// Demo 3: Self-reflection and adaptation
	selfReflectionDemo(apiKey)

	// Demo 4: Performance prediction and optimization
	performanceOptimizationDemo(apiKey)

	// Demo 5: Complete learning agent lifecycle
	learningLifecycleDemo(apiKey)
}

// basicLearningDemo shows basic learning engine setup
func basicLearningDemo(apiKey string) {
	fmt.Println("\n1. Basic Learning Engine Setup:")

	// Create agent with learning capabilities
	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a learning intelligent agent.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
		)

	// Create and attach learning engine
	learningEngine := mas.NewLearningEngine(agent)
	agent = agent.WithLearningEngine(learningEngine)

	fmt.Printf("Learning engine attached to agent: %s\n", agent.Name())

	// Get initial metrics
	metrics := agent.GetLearningMetrics()
	fmt.Printf("Initial learning metrics:\n")
	fmt.Printf("- Total Experiences: %d\n", metrics.TotalExperiences)
	fmt.Printf("- Learning Rate: %.3f\n", metrics.LearningRate)
	fmt.Printf("- Adaptation Rate: %.3f\n", metrics.AdaptationRate)
}

// experienceAndPatternsDemo shows experience recording and pattern discovery
func experienceAndPatternsDemo(apiKey string) {
	fmt.Println("\n2. Experience Recording and Pattern Discovery:")

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a pattern-learning agent.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
		).
		WithLearningEngine(mas.NewLearningEngine(nil))

	ctx := context.Background()

	// Simulate various experiences
	experiences := []*mas.Experience{
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.9),
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.85),
		mas.NewExperience(mas.SkillExperience, "math_calculation", false, 0.2),
		mas.NewExperience(mas.SkillExperience, "text_analysis", true, 0.8),
		mas.NewExperience(mas.SkillExperience, "text_analysis", true, 0.75),
		mas.NewExperience(mas.ChatExperience, "simple_chat", true, 0.7),
		mas.NewExperience(mas.ChatExperience, "complex_chat", false, 0.3),
		mas.NewExperience(mas.DecisionExperience, "priority_decision", true, 0.85),
	}

	// Add context to some experiences
	experiences[0].Context["difficulty"] = "easy"
	experiences[1].Context["difficulty"] = "medium"
	experiences[2].Context["difficulty"] = "hard"
	experiences[3].Context["text_type"] = "technical"
	experiences[4].Context["text_type"] = "casual"

	// Record all experiences
	fmt.Printf("Recording %d experiences...\n", len(experiences))
	for i, exp := range experiences {
		err := agent.RecordExperience(ctx, exp)
		if err != nil {
			log.Printf("Failed to record experience %d: %v", i, err)
		}
	}

	// Check updated metrics
	metrics := agent.GetLearningMetrics()
	fmt.Printf("Updated learning metrics:\n")
	fmt.Printf("- Total Experiences: %d\n", metrics.TotalExperiences)

	// Get learning engine to check patterns
	if learningEngine := agent.GetLearningEngine(); learningEngine != nil {
		// Analyze experiences
		analysis, err := learningEngine.AnalyzeExperiences(ctx)
		if err == nil {
			fmt.Printf("Learning Analysis:\n")
			fmt.Printf("- Success Rate: %.2f%%\n", analysis.SuccessRate*100)
			fmt.Printf("- Average Performance: %.2f\n", analysis.AveragePerformance)
			fmt.Printf("- Performance Trend: %s\n", analysis.PerformanceTrend)
			fmt.Printf("- Weak Areas: %v\n", analysis.WeakAreas)
			fmt.Printf("- Recommendations: %v\n", analysis.Recommendations)
		}

		// Discover patterns
		patterns, err := learningEngine.DiscoverPatterns(ctx)
		if err == nil {
			fmt.Printf("Discovered Patterns: %d\n", len(patterns))
			for _, pattern := range patterns {
				fmt.Printf("- %s: Success Rate %.2f, Confidence %.2f\n",
					pattern.Name, pattern.SuccessRate, pattern.Confidence)
			}
		}
	}
}

// selfReflectionDemo shows self-reflection capabilities
func selfReflectionDemo(apiKey string) {
	fmt.Println("\n3. Self-Reflection and Adaptation:")

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a self-reflective learning agent.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
			skills.PlanningSkill(),
		).
		WithLearningEngine(mas.NewLearningEngine(nil))

	ctx := context.Background()

	// Record some diverse experiences to enable meaningful reflection
	experiences := []*mas.Experience{
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.95),
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.88),
		mas.NewExperience(mas.SkillExperience, "text_analysis", true, 0.82),
		mas.NewExperience(mas.SkillExperience, "text_analysis", false, 0.3),
		mas.NewExperience(mas.PlanExperience, "task_planning", true, 0.75),
		mas.NewExperience(mas.DecisionExperience, "complex_decision", false, 0.25),
		mas.NewExperience(mas.ChatExperience, "helpful_response", true, 0.9),
		mas.NewExperience(mas.ChatExperience, "complex_query", true, 0.7),
	}

	for _, exp := range experiences {
		agent.RecordExperience(ctx, exp)
	}

	// Perform self-reflection
	fmt.Printf("Performing self-reflection...\n")
	reflection, err := agent.SelfReflect(ctx)
	if err != nil {
		log.Printf("Self-reflection failed: %v", err)
		return
	}

	fmt.Printf("Self-Reflection Results:\n")
	fmt.Printf("- Overall Assessment: %s\n", reflection.OverallAssessment)
	fmt.Printf("- Learning Progress: %.2f\n", reflection.LearningProgress)
	fmt.Printf("- Self-Confidence: %.2f\n", reflection.SelfConfidence)
	fmt.Printf("- Goals Alignment: %.2f\n", reflection.GoalsAlignment)
	fmt.Printf("- Adaptation Needed: %t\n", reflection.AdaptationNeeded)
	fmt.Printf("- Reflection Depth: %s\n", reflection.ReflectionDepth)
	fmt.Printf("- Strengths: %v\n", reflection.Strengths)
	fmt.Printf("- Weaknesses: %v\n", reflection.Weaknesses)
	fmt.Printf("- Recommended Actions: %v\n", reflection.RecommendedActions)

	// Test adaptation based on reflection
	if learningEngine := agent.GetLearningEngine(); learningEngine != nil {
		analysis, _ := learningEngine.AnalyzeExperiences(ctx)
		if analysis != nil {
			fmt.Printf("\nAdapting behavior based on insights...\n")
			err := learningEngine.AdaptBehavior(ctx, analysis)
			if err == nil {
				fmt.Printf("Behavior adaptation completed\n")
			}
		}
	}
}

// performanceOptimizationDemo shows performance prediction and optimization
func performanceOptimizationDemo(apiKey string) {
	fmt.Println("\n4. Performance Prediction and Optimization:")

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a performance-optimizing agent.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
		).
		WithLearningEngine(mas.NewLearningEngine(nil))

	ctx := context.Background()

	// Record domain-specific experiences
	mathExperiences := []*mas.Experience{
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.9),
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.85),
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.92),
		mas.NewExperience(mas.SkillExperience, "math_calculation", false, 0.4),
	}

	for _, exp := range mathExperiences {
		exp.Context["domain"] = "mathematics"
		agent.RecordExperience(ctx, exp)
	}

	learningEngine := agent.GetLearningEngine()
	if learningEngine == nil {
		fmt.Printf("Learning engine not available\n")
		return
	}

	// Predict performance for math tasks
	predictedPerf, err := learningEngine.PredictPerformance(ctx, "math_calculation", map[string]interface{}{
		"domain": "mathematics",
	})
	if err == nil {
		fmt.Printf("Predicted performance for math_calculation: %.2f\n", predictedPerf)
	}

	// Optimize strategy for mathematics domain
	optimization, err := learningEngine.OptimizeStrategy(ctx, "mathematics")
	if err == nil {
		fmt.Printf("Strategy Optimization for Mathematics:\n")
		fmt.Printf("- Current Performance: %.2f\n", optimization.CurrentPerformance)
		fmt.Printf("- Optimized Strategy: %s\n", optimization.OptimizedStrategy)
		fmt.Printf("- Expected Improvement: %.2f\n", optimization.ExpectedImprovement)
		fmt.Printf("- Confidence: %.2f\n", optimization.Confidence)
		fmt.Printf("- Recommendations: %v\n", optimization.Recommendations)
	}

	// Test pattern application
	patterns, err := learningEngine.GetPatterns(mas.PatternFilter{})
	if err == nil && len(patterns) > 0 {
		fmt.Printf("\nApplying discovered pattern...\n")
		application, err := learningEngine.ApplyPattern(ctx, patterns[0].ID, map[string]interface{}{
			"domain": "mathematics",
		})
		if err == nil {
			fmt.Printf("Pattern Application:\n")
			fmt.Printf("- Recommended Action: %s\n", application.RecommendedAction)
			fmt.Printf("- Expected Outcome: %s\n", application.ExpectedOutcome)
			fmt.Printf("- Risk Assessment: %s\n", application.RiskAssessment)
			fmt.Printf("- Confidence: %.2f\n", application.Confidence)
		}
	}
}

// learningLifecycleDemo shows complete learning agent lifecycle
func learningLifecycleDemo(apiKey string) {
	fmt.Println("\n5. Complete Learning Agent Lifecycle:")

	// Create learning agent with all capabilities
	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a comprehensive learning agent.").
		WithSkills(
			skills.MathSkill(),
			skills.TextAnalysisSkill(),
			skills.PlanningSkill(),
			skills.QuickResponseSkill(),
		).
		WithLearningEngine(mas.NewLearningEngine(nil))

	ctx := context.Background()

	fmt.Printf("Starting comprehensive learning lifecycle...\n")

	// Phase 1: Initial experiences
	fmt.Printf("\nPhase 1: Gathering initial experiences\n")
	initialExperiences := []*mas.Experience{
		mas.NewExperience(mas.ChatExperience, "greeting", true, 0.8),
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.85),
		mas.NewExperience(mas.SkillExperience, "text_analysis", false, 0.4),
		mas.NewExperience(mas.DecisionExperience, "simple_decision", true, 0.7),
	}

	for _, exp := range initialExperiences {
		agent.RecordExperience(ctx, exp)
	}

	// Phase 2: Learning and pattern discovery
	fmt.Printf("\nPhase 2: Learning and pattern discovery\n")
	learningEngine := agent.GetLearningEngine()
	patterns, _ := learningEngine.DiscoverPatterns(ctx)
	fmt.Printf("Discovered %d initial patterns\n", len(patterns))

	// Phase 3: Self-reflection
	fmt.Printf("\nPhase 3: Self-reflection\n")
	reflection, err := agent.SelfReflect(ctx)
	if err == nil {
		fmt.Printf("Self-confidence: %.2f\n", reflection.SelfConfidence)
		fmt.Printf("Adaptation needed: %t\n", reflection.AdaptationNeeded)
	}

	// Phase 4: More experiences and adaptation
	fmt.Printf("\nPhase 4: Continued learning and adaptation\n")
	additionalExperiences := []*mas.Experience{
		mas.NewExperience(mas.SkillExperience, "math_calculation", true, 0.9),
		mas.NewExperience(mas.SkillExperience, "text_analysis", true, 0.85),
		mas.NewExperience(mas.PlanExperience, "complex_planning", true, 0.8),
		mas.NewExperience(mas.ChatExperience, "complex_query", true, 0.75),
	}

	for _, exp := range additionalExperiences {
		agent.RecordExperience(ctx, exp)
	}

	// Final analysis
	fmt.Printf("\nPhase 5: Final analysis\n")
	analysis, err := learningEngine.AnalyzeExperiences(ctx)
	if err == nil {
		fmt.Printf("Final Success Rate: %.2f%%\n", analysis.SuccessRate*100)
		fmt.Printf("Performance Trend: %s\n", analysis.PerformanceTrend)
		fmt.Printf("Confidence Score: %.2f\n", analysis.ConfidenceScore)
	}

	// Final metrics
	metrics := agent.GetLearningMetrics()
	fmt.Printf("\nFinal Learning Metrics:\n")
	fmt.Printf("- Total Experiences: %d\n", metrics.TotalExperiences)
	fmt.Printf("- Learning Rate: %.3f\n", metrics.LearningRate)
	fmt.Printf("- Pattern Discovery Rate: %.3f\n", metrics.PatternDiscoveryRate)

	// Final self-reflection
	finalReflection, err := agent.SelfReflect(ctx)
	if err == nil {
		fmt.Printf("\nFinal Self-Assessment:\n")
		fmt.Printf("- Learning Progress: %.2f\n", finalReflection.LearningProgress)
		fmt.Printf("- Self-Confidence: %.2f\n", finalReflection.SelfConfidence)
		fmt.Printf("- Key Strengths: %v\n", finalReflection.Strengths)

		if len(finalReflection.RecommendedActions) > 0 {
			fmt.Printf("- Next Steps: %v\n", finalReflection.RecommendedActions)
		}
	}

	fmt.Printf("\nLearning lifecycle completed successfully!\n")
}
