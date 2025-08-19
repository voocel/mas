# MAS Learning and Adaptation Mechanisms

Demonstrates the intelligent learning capabilities of the MAS framework, implementing self-supervised learning, pattern recognition, and continuous adaptation in modern AI Agent systems.

## 🧠 Design Features

✅ **Experience Recording System**: Detailed recording of all interactions and execution results
✅ **Pattern Recognition Discovery**: Automatically identifies successful and failed behavior patterns
✅ **Self-Reflection Capability**: Agents can analyze their own behavior and improve
✅ **Performance Prediction**: Predicts future action success rates based on historical experience
✅ **Strategy Optimization**: Dynamically adjusts decision strategies to improve performance
✅ **Continuous Adaptation**: Continuously learns and optimizes behavior from experience

## 🎯 Core Learning Capabilities

### 📊 **Experience Recording**
```go
// Create learning experience
experience := mas.NewExperience(
    mas.SkillExperience,    // Experience type
    "math_calculation",     // Action performed
    true,                   // Whether successful
    0.85,                   // Performance score (0.0-1.0)
)

// Add context information
experience.Context["difficulty"] = "medium"
experience.Context["domain"] = "mathematics"

// Record to learning engine
agent.RecordExperience(ctx, experience)
```

### 🔍 **Pattern Discovery**
```go
// Automatically discover behavior patterns
patterns, err := learningEngine.DiscoverPatterns(ctx)

for _, pattern := range patterns {
    fmt.Printf("Pattern: %s\n", pattern.Name)
    fmt.Printf("Success Rate: %.2f\n", pattern.SuccessRate)
    fmt.Printf("Confidence: %.2f\n", pattern.Confidence)
    fmt.Printf("Performance Gain: %.2f\n", pattern.PerformanceGain)
}
```

### 🤔 **Self-Reflection**
```go
// Perform deep self-reflection
reflection, err := agent.SelfReflect(ctx)

fmt.Printf("Overall Assessment: %s\n", reflection.OverallAssessment)
fmt.Printf("Learning Progress: %.2f\n", reflection.LearningProgress)
fmt.Printf("Self-Confidence: %.2f\n", reflection.SelfConfidence)
fmt.Printf("Strengths: %v\n", reflection.Strengths)
fmt.Printf("Weaknesses: %v\n", reflection.Weaknesses)
fmt.Printf("Recommended Actions: %v\n", reflection.RecommendedActions)
```

### 📈 **Performance Prediction**
```go
// Predict success probability of specific actions
predicted, err := learningEngine.PredictPerformance(ctx,
    "math_calculation",
    map[string]interface{}{
        "difficulty": "hard",
        "domain": "mathematics",
    })

fmt.Printf("Predicted Success Rate: %.2f\n", predicted)
```

### ⚡ **Strategy Optimization**
```go
// Optimize strategy for specific domain
optimization, err := learningEngine.OptimizeStrategy(ctx, "mathematics")

fmt.Printf("Current Performance: %.2f\n", optimization.CurrentPerformance)
fmt.Printf("Optimized Strategy: %s\n", optimization.OptimizedStrategy)
fmt.Printf("Expected Improvement: %.2f\n", optimization.ExpectedImprovement)
```

## 📚 **Experience Type System**

```go
// Support multiple experience types
const (
    ChatExperience     // Conversation interaction experience
    ToolExperience     // Tool execution experience
    SkillExperience    // Skill usage experience
    GoalExperience     // Goal pursuit experience
    DecisionExperience // Decision making experience
    PlanExperience     // Planning formulation experience
)
```

## 🎯 **Learning Strategies**

| Strategy | Characteristics | Use Cases |
|----------|----------------|-----------|
| **ReinforcementLearning** | Learn based on rewards/punishments | Clear feedback environments |
| **ImitationLearning** | Imitate successful patterns | When excellent examples are available |
| **ExplorationLearning** | Learn through exploration | Unknown environment exploration |
| **ReflectionLearning** | Learn through self-reflection | When deep understanding is needed |
| **HybridLearning** | Mix multiple strategies | Complex dynamic environments |

## 🔄 **Adaptation Modes**

```go
// Ways to control adaptive behavior
const (
    ConservativeAdaptation // Conservative cautious changes
    AggressiveAdaptation   // Fast aggressive changes
    BalancedAdaptation     // Balanced moderate changes
    ContextualAdaptation   // Context-based adaptation
)
```

## 🛠️ **Complete Usage Examples**

### 1. Create Learning Agent
```go
func createLearningAgent() {
    // Create agent with learning capabilities
    agent := mas.NewAgent("gpt-4", apiKey).
        WithSystemPrompt("You are a learning agent.").
        WithSkills(
            skills.MathSkill(),
            skills.TextAnalysisSkill(),
        ).
        WithLearningEngine(mas.NewLearningEngine(agent))

    fmt.Printf("Learning agent created with capabilities\n")
}
```

### 2. Record and Analyze Experience
```go
func recordAndAnalyze() {
    // Record multiple experiences
    experiences := []*mas.Experience{
        mas.NewExperience(mas.SkillExperience, "math_calc", true, 0.9),
        mas.NewExperience(mas.SkillExperience, "text_analysis", true, 0.8),
        mas.NewExperience(mas.ChatExperience, "complex_query", false, 0.3),
    }

    for _, exp := range experiences {
        agent.RecordExperience(ctx, exp)
    }

    // Analyze learning progress
    analysis, _ := learningEngine.AnalyzeExperiences(ctx)
    fmt.Printf("Success Rate: %.2f%%\n", analysis.SuccessRate*100)
    fmt.Printf("Performance Trend: %s\n", analysis.PerformanceTrend)
}
```

### 3. Pattern Discovery and Application
```go
func discoverAndApplyPatterns() {
    // Discover behavior patterns
    patterns, _ := learningEngine.DiscoverPatterns(ctx)

    for _, pattern := range patterns {
        // Apply discovered patterns
        application, _ := learningEngine.ApplyPattern(ctx, pattern.ID,
            map[string]interface{}{
                "context": "similar_situation",
            })

        fmt.Printf("Recommended Action: %s\n", application.RecommendedAction)
        fmt.Printf("Expected Outcome: %s\n", application.ExpectedOutcome)
        fmt.Printf("Risk Assessment: %s\n", application.RiskAssessment)
    }
}
```

### 4. Self-Reflection and Adaptation
```go
func selfReflectAndAdapt() {
    // Deep self-reflection
    reflection, _ := agent.SelfReflect(ctx)

    fmt.Printf("Learning Progress: %.2f\n", reflection.LearningProgress)
    fmt.Printf("Self-Confidence: %.2f\n", reflection.SelfConfidence)

    if reflection.AdaptationNeeded {
        // Adapt based on reflection results
        analysis, _ := learningEngine.AnalyzeExperiences(ctx)
        learningEngine.AdaptBehavior(ctx, analysis)
        fmt.Printf("Behavior adapted based on reflection\n")
    }
}
```

### 5. Learning Lifecycle Management
```go
func manageLearningLifecycle() {
    // Phase 1: Initial experience collection
    collectInitialExperiences()

    // Phase 2: Pattern discovery
    patterns := discoverPatterns()

    // Phase 3: Self-reflection
    reflection := performSelfReflection()

    // Phase 4: Continuous learning
    continuousLearning()

    // Phase 5: Performance optimization
    optimizePerformance()

    fmt.Printf("Complete learning lifecycle managed\n")
}
```

## 📊 **Learning Metrics Monitoring**

The system automatically tracks key learning indicators:

```go
metrics := agent.GetLearningMetrics()

fmt.Printf("Learning Metrics:\n")
fmt.Printf("- Total Experiences: %d\n", metrics.TotalExperiences)
fmt.Printf("- Learning Rate: %.3f\n", metrics.LearningRate)
fmt.Printf("- Adaptation Rate: %.3f\n", metrics.AdaptationRate)
fmt.Printf("- Pattern Discovery Rate: %.3f\n", metrics.PatternDiscoveryRate)
fmt.Printf("- Performance Improvement: %.3f\n", metrics.PerformanceImprovement)
fmt.Printf("- Knowledge Retention: %.3f\n", metrics.KnowledgeRetention)
fmt.Printf("- Exploration Ratio: %.3f\n", metrics.ExplorationRatio)
```

## 🔍 **Event Monitoring**

Rich events emitted during the learning process:

```go
// Learning-related events
EventType("learning.experience.recorded")     // Experience recorded
EventType("learning.patterns.discovered")     // Patterns discovered
EventType("learning.analysis.completed")      // Analysis completed
EventType("learning.pattern.applied")         // Pattern applied
EventType("learning.self_reflection.completed") // Self-reflection completed
EventType("learning.adaptation.started")      // Adaptation started
EventType("learning.adaptation.completed")    // Adaptation completed

// Agent-level learning events
EventType("agent.self_reflection.start")      // Agent self-reflection started
EventType("agent.self_reflection.complete")   // Agent self-reflection completed
```

## 🎨 **Architecture Advantages**

1. **🧠 Intelligent Learning**: Automatically extracts knowledge and patterns from experience
2. **🔍 Deep Reflection**: Agents can analyze their own behavioral performance
3. **📈 Continuous Improvement**: Dynamically optimizes strategies based on learning results
4. **🎯 Accurate Prediction**: Predicts future performance based on historical data
5. **⚡ Rapid Adaptation**: Quickly adjusts behavior based on environmental changes
6. **📊 Comprehensive Monitoring**: Complete learning metrics and event tracking

This learning system evolves MAS Agents from **"static executors"** to **"intelligent learners"** with true self-improvement and continuous optimization capabilities!
