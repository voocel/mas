# MAS Hierarchical Cognitive Architecture

Demonstrates the hierarchical cognitive capabilities of the MAS framework, implementing the Brain-Cerebellum pattern.

## 🧠 Design Features

✅ **No Wrapper Design**: Directly extends Agent interface without adapter pattern
✅ **Four-Layer Cognitive Architecture**: Reflex → Cerebellum → Cortex → Meta
✅ **Skill Library System**: Pluggable cognitive skill modules
✅ **Automatic Layer Selection**: Agent automatically selects the most suitable cognitive layer
✅ **Real-time State Tracking**: Complete cognitive state observability

## 🎯 Cognitive Layers

| Layer | Purpose | Characteristics | Examples |
|-------|---------|----------------|----------|
| **Reflex** | Reflex Layer | Immediate response, no thinking required | Emergency handling |
| **Cerebellum** | Cerebellum Layer | Skilled actions, automatic execution | Math calculations, text analysis |
| **Cortex** | Cortex Layer | Reasoning analysis, complex thinking | Decision making, problem solving |
| **Meta** | Meta-cognitive Layer | Planning monitoring, strategy adjustment | Plan formulation, goal management |

## 🚀 Usage

### 1. Basic Cognitive Capabilities
```go
// Create an agent with cognitive capabilities
agent := mas.NewAgent("gpt-4", apiKey).
    WithSystemPrompt("You are an intelligent assistant.")

// Check cognitive state
state := agent.GetCognitiveState()
fmt.Printf("Layer: %s, Mode: %s\n", state.CurrentLayer, state.Mode)
```

### 2. Skill Execution
```go
// Add skills
agent := mas.NewAgent("gpt-4", apiKey).
    WithSkills(
        skills.MathSkill(),        // Cerebellum layer
        skills.TextAnalysisSkill(), // Cortex layer
        skills.QuickResponseSkill(), // Reflex layer
        skills.PlanningSkill(),     // Meta-cognitive layer
    )

// Execute skill
result, _ := agent.ExecuteSkill(ctx, "math_calculation", map[string]interface{}{
    "expression": "15 + 25 * 2",
})
```

### 3. High-Level Cognitive Functions
```go
// Planning capability
plan, _ := agent.Plan(ctx, "Organize a team meeting")

// Reasoning capability
situation := mas.NewSituation(context, inputs)
decision, _ := agent.Reason(ctx, situation)

// Reaction capability
stimulus := mas.NewStimulus("emergency", data, 0.9)
action, _ := agent.React(ctx, stimulus)
```

### 4. Cognitive Mode Control
```go
// Set cognitive modes
reflexAgent := agent.SetCognitiveMode(mas.ReflexMode)      // Reflex only
skillAgent := agent.SetCognitiveMode(mas.SkillMode)        // Skill priority
reasoningAgent := agent.SetCognitiveMode(mas.ReasoningMode) // Reasoning priority
autoAgent := agent.SetCognitiveMode(mas.AutomaticMode)      // Automatic selection
```

## 🛠️ Built-in Skills

### Math Skill (Cerebellum Layer)
```go
skills.MathSkill() // Mathematical calculations and analysis
```

### Text Analysis Skill (Cortex Layer)
```go
skills.TextAnalysisSkill() // Sentiment analysis, keyword extraction
```

### Quick Response Skill (Reflex Layer)
```go
skills.QuickResponseSkill() // Emergency immediate response
```

### Planning Skill (Meta Layer)
```go
skills.PlanningSkill() // Task decomposition and plan formulation
```

## 🔄 Automatic Layer Selection

The agent automatically selects cognitive layers based on task complexity:

- **Simple Queries** → Reflex layer for quick response
- **Calculation Tasks** → Cerebellum layer for skill execution
- **Complex Analysis** → Cortex layer for deep reasoning
- **Planning Tasks** → Meta layer for strategic thinking

## 📊 Cognitive State Monitoring

```go
state := agent.GetCognitiveState()

// Cognitive state information
fmt.Printf("Current Layer: %s\n", state.CurrentLayer)
fmt.Printf("Working Mode: %s\n", state.Mode)
fmt.Printf("Active Plan: %v\n", state.ActivePlan)
fmt.Printf("Loaded Skills: %v\n", state.LoadedSkills)
fmt.Printf("Recent Decisions: %v\n", state.RecentDecisions)
```

## 🎨 Complete Example

```go
func cognitiveDemo() {
    // Create cognitive agent
    agent := mas.NewAgent("gpt-4", apiKey).
        WithSkills(
            skills.MathSkill(),
            skills.TextAnalysisSkill(),
            skills.PlanningSkill(),
        ).
        SetCognitiveMode(mas.AutomaticMode)

    // High-level planning
    plan, _ := agent.Plan(ctx, "Complete project analysis report")

    // Skill execution
    mathResult, _ := agent.ExecuteSkill(ctx, "math_calculation", params)

    // Reasoning decision
    decision, _ := agent.Reason(ctx, situation)

    // Reaction response
    action, _ := agent.React(ctx, stimulus)

    // Monitor state
    state := agent.GetCognitiveState()
    fmt.Printf("Cognitive State: %+v\n", state)
}
```

## ✨ Design Advantages

1. **🎯 Simple and Direct**: No wrapper, directly extends Agent interface
2. **🧠 Human-like Cognition**: Simulates human hierarchical thinking patterns
3. **⚡ Auto-Adaptive**: Automatically selects optimal layer based on tasks
4. **🔧 Skill-Oriented**: Pluggable cognitive skill system
5. **📊 Fully Observable**: Real-time cognitive state monitoring
6. **🚀 Production Ready**: Event integration, performance optimization

This cognitive architecture gives MAS Agents **human-like hierarchical thinking capabilities**, from simple reflexes to complex planning, achieving truly intelligent behavior!
