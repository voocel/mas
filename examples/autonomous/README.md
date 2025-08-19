# MAS Autonomous and Goal-Oriented Capabilities

Demonstrates the autonomous agent capabilities of the MAS framework, implementing true goal-oriented and autonomous decision-making systems.

## 🎯 Design Features

✅ **Goal-Oriented System**: Define, decompose, track, and complete goals
✅ **Autonomous Decision Making**: Agents proactively formulate and execute action plans
✅ **Multi-Strategy Support**: Sequential, parallel, priority, and adaptive strategies
✅ **Real-time Progress Tracking**: Complete goal execution status monitoring
✅ **Learning and Adaptation**: Learn from execution results to optimize decisions
✅ **Event-Driven**: Full event recording and observability

## 🚀 Core Capabilities

### 🎯 **Goal Management System**
```go
// Create a goal
goal := mas.NewGoal(
    "Data Analysis Task",
    "Analyze customer feedback and extract insights",
    mas.HighPriority,
)

// Set deadline
deadline := time.Now().Add(1 * time.Hour)
goal.Deadline = &deadline

// Add to agent
agent.AddGoal(ctx, goal)
```

### 🤖 **Autonomous Execution Mode**
```go
// Start autonomous mode
agent.StartAutonomous(ctx, mas.PriorityStrategy)

// Check status
fmt.Printf("Agent is autonomous: %t\n", agent.IsAutonomous())

// Stop autonomous mode
agent.StopAutonomous(ctx)
```

### 📊 **Progress Monitoring**
```go
// Get overall progress
progress := goalManager.GetOverallProgress()
fmt.Printf("Overall progress: %.2f%% (%d/%d goals)\n",
    progress.OverallProgress*100,
    progress.CompletedGoals,
    progress.TotalGoals)
```

### 📈 **Learning and Adaptation**
```go
// Get learning insights
insights := goalManager.GetLearnings()
fmt.Printf("Success Rate: %.2f%%\n", insights.PerformanceMetrics.SuccessRate*100)

// Adapt strategy based on learning results
goalManager.AdaptStrategy(insights)
```

## 🔄 **Autonomous Execution Strategies**

| Strategy | Characteristics | Use Cases |
|----------|----------------|-----------|
| **Sequential** | Execute goals sequentially | When tasks have dependencies |
| **Priority** | Execute by priority | When urgent tasks need priority |
| **Parallel** | Process multiple goals in parallel | When independent tasks can run simultaneously |
| **Adaptive** | Intelligent adaptive selection | Complex dynamic environments |

## 🏗️ **Goal State Management**

```go
// Goal state transitions
const (
    GoalPending     // Waiting to start
    GoalActive      // Currently executing
    GoalCompleted   // Completed
    GoalFailed      // Execution failed
    GoalPaused      // Temporarily paused
    GoalCancelled   // Cancelled
)

// Goal priorities
const (
    LowPriority     // Low priority
    MediumPriority  // Medium priority
    HighPriority    // High priority
    CriticalPriority // Critical priority
)
```

## 🛠️ **Complete Usage Examples**

### 1. Basic Autonomous Agent Setup
```go
func basicAutonomousDemo() {
    // Create an agent with autonomous capabilities
    agent := mas.NewAgent("gpt-4", apiKey).
        WithSystemPrompt("You are an autonomous assistant.").
        WithSkills(
            skills.MathSkill(),
            skills.PlanningSkill(),
        )

    // Create goal manager
    goalManager := mas.NewGoalManager(agent)
    agent = agent.WithGoalManager(goalManager)

    // Add goal
    goal := mas.NewGoal(
        "Daily Tasks",
        "Complete routine analysis tasks",
        mas.MediumPriority,
    )
    agent.AddGoal(ctx, goal)
}
```

### 2. Goal-Oriented Autonomous Behavior
```go
func goalOrientedDemo() {
    // Add multiple goals with different priorities
    goals := []*mas.Goal{
        mas.NewGoal("Data Analysis", "Analyze customer data", mas.HighPriority),
        mas.NewGoal("Report Generation", "Generate weekly report", mas.MediumPriority),
        mas.NewGoal("System Monitoring", "Monitor alerts", mas.CriticalPriority),
    }

    // Start autonomous mode with priority strategy
    agent.StartAutonomous(ctx, mas.PriorityStrategy)

    // Monitor progress
    progress := goalManager.GetOverallProgress()
    fmt.Printf("Progress: %.2f%%\n", progress.OverallProgress*100)
}
```

### 3. Multi-Strategy Intelligent Selection
```go
func multiStrategyDemo() {
    strategies := []mas.AutonomousStrategy{
        mas.SequentialStrategy,  // Sequential execution
        mas.PriorityStrategy,    // Priority-based
        mas.ParallelStrategy,    // Parallel execution
        mas.AdaptiveStrategy,    // Intelligent adaptation
    }

    for _, strategy := range strategies {
        agent.StartAutonomous(ctx, strategy)
        // Run and evaluate effectiveness
        time.Sleep(5 * time.Second)
        agent.StopAutonomous(ctx)
    }
}
```

### 4. Learning and Adaptation Capabilities
```go
func learningDemo() {
    // Start adaptive strategy
    agent.StartAutonomous(ctx, mas.AdaptiveStrategy)

    // Run for a while to let agent learn
    time.Sleep(10 * time.Second)

    // Get learning insights
    insights := goalManager.GetLearnings()
    fmt.Printf("Success Rate: %.2f%%\n", insights.PerformanceMetrics.SuccessRate*100)
    fmt.Printf("Efficiency Score: %.2f\n", insights.PerformanceMetrics.EfficiencyScore)

    // Automatically adjust strategy based on learning results
    goalManager.AdaptStrategy(insights)
}
```

## 🔍 **Event Monitoring**

Autonomous agents emit rich events during execution:

```go
// Goal-related events
EventType("goal.created")      // Goal created
EventType("goal.updated")      // Goal updated
EventType("goal.removed")      // Goal removed

// Autonomous mode events
EventType("autonomous.started") // Autonomous mode started
EventType("autonomous.stopped") // Autonomous mode stopped

// Action execution events
EventType("action.started")     // Action started
EventType("action.completed")   // Action completed
EventType("action.failed")      // Action failed

// Agent autonomous events
EventType("agent.autonomous.start") // Agent autonomous started
EventType("agent.autonomous.stop")  // Agent autonomous stopped
```

## 📊 **Performance Metrics**

The system automatically tracks key performance indicators:

- **Success Rate**: Goal completion success rate
- **Efficiency Score**: Execution efficiency rating
- **Adaptability Score**: Environmental adaptation capability
- **Average Completion Time**: Average goal execution time

## 🎨 **Architecture Advantages**

1. **🎯 Truly Autonomous**: Agents can proactively plan and execute tasks
2. **🧠 Intelligent Decision Making**: Combines cognitive architecture for smart choices
3. **📈 Continuous Learning**: Constantly optimizes from execution results
4. **🔄 Self-Adaptive**: Dynamically adjusts strategies based on environment
5. **📊 Fully Observable**: Complete event recording and monitoring
6. **⚡ High Performance**: Concurrent-safe goal management system

This autonomous system evolves MAS Agents from **"passive responders"** to **"proactive intelligent agents"** with true goal-oriented and autonomous decision-making capabilities!
