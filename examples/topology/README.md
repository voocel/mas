# Dynamic Collaboration Topology Example

This example demonstrates MAS's dynamic collaboration topology system - an intelligent multi-agent network organization system that automatically adapts to workload patterns and optimizes agent collaboration.

## Features Demonstrated

### 🕸️ **Seven Topology Types**
- **Star Topology**: Central coordinator with spoke agents
- **Chain Topology**: Sequential pipeline processing  
- **Mesh Topology**: Full interconnected network
- **Hierarchy Topology**: Tree-like hierarchical structure
- **Hub Topology**: Multiple hubs with local clusters
- **Ring Topology**: Circular communication pattern
- **Adaptive Topology**: Self-organizing optimal structure

### 🤖 **Six Collaboration Modes**
- **Competitive**: Agents compete for tasks
- **Cooperative**: Agents cooperate on shared goals
- **Delegation**: Hierarchical task delegation
- **Consensus**: Consensus-based decision making
- **Specialization**: Role-based specialization
- **Swarm**: Swarm intelligence behavior

### 🎯 **Agent Roles**
- **Coordinator**: Central coordination and planning
- **Worker**: Task execution and processing
- **Specialist**: Domain expertise and analysis
- **Broker**: Message routing and communication
- **Monitor**: Performance monitoring and analysis
- **Decider**: Decision making and strategy

### ⚡ **Intelligent Features**
- **Smart Task Distribution**: Capability and load-based assignment
- **Real-time Performance Analysis**: Network metrics and bottleneck prediction
- **Automatic Optimization**: Dynamic topology restructuring
- **Load Balancing**: Intelligent redistribution to prevent bottlenecks
- **Adaptive Reorganization**: Workload-based topology adaptation

## Running the Example

```bash
cd examples/topology
export OPENAI_API_KEY="your-api-key"
go run main.go
```

## What You'll See

### 1. Basic Topology Setup
```
Topology created with 4 nodes and 6 edges
Topology Type: star
Collaboration Mode: cooperative
Average Load: 0.00
Network Efficiency: 0.80
```

### 2. Star Topology Collaboration
```
Distributing 3 tasks across topology...
Task task_1234 (data_analysis) assigned to [node1, node2] (Coordinator: node1, Confidence: 0.85)
Task task_5678 (report_writing) assigned to [node3] (Coordinator: node3, Confidence: 0.92)

Topology Analysis:
- Overall Health: 0.87
- Efficiency Score: 0.82
- Scalability Score: 0.75
- Resilience Score: 0.68
```

### 3. Dynamic Topology Adaptation
```
Adapting to workload pattern 1 (peak):
Adapted topology type: star
Post-adaptation metrics: Load Balance: 0.85, Efficiency: 0.88

Adapting to workload pattern 2 (burst):
Adapted topology type: mesh
Post-adaptation metrics: Load Balance: 0.78, Efficiency: 0.92

Optimizing topology...
Topology optimized successfully
Final metrics: Load Balance: 0.91, Efficiency: 0.94
```

### 4. Intelligent Task Distribution
```
Distributing 5 tasks with load balancing...
Task: complex_calculation (Priority: 5)
  → Assigned to: [math_worker2]
  → Coordinator: math_worker2
  → Confidence: 0.95
  → Estimated Time: 12m0s

Bottleneck Predictions:
- Node math_worker1: Load 0.30 → 1.08 (in 12m0s, Confidence: 0.80)
  Mitigation: [redistribute_load, add_parallel_node]
```

### 5. Complete Topology Lifecycle
```
Phase 1: Building initial topology
Event: Node node1 added
Event: Node node2 added

Phase 2: Scaling up topology
Event: Node node4 added
Event: Node node5 added

Phase 3: Adding coordination layer
Event: Node main_coordinator added
Event: Topology changed to star

Phase 4: Performance optimization
Optimization test 1:
- Topology: hub
- Load Balance: 0.87
- Network Efficiency: 0.91

Final Topology Analysis:
- Overall Health: 0.89
- Efficiency Score: 0.94
- Scalability Score: 0.85
- Resilience Score: 0.77
- Optimal Topology: hub
- Recommendations: [Add more connections for better fault tolerance]

Topology Snapshot:
- Timestamp: 14:30:25
- Type: hub
- Mode: swarm
- Nodes: 9
- Edges: 24
- Metrics: Load Balance 0.87, Efficiency 0.91
```

## Key Components

### DynamicTopology Interface
```go
// Core topology management
topology := mas.NewDynamicTopology(mas.AdaptiveTopology, mas.SwarmMode)

// Node and edge management
topology.AddNode(node)
topology.AddEdge(edge)
topology.UpdateNodeStatus(nodeID, load, performance)

// Intelligent operations
assignment, _ := topology.DistributeTask(ctx, task)
topology.OptimizeTopology(ctx)
topology.AdaptToWorkload(ctx, workload)

// Analytics
metrics := topology.GetMetrics()
analysis, _ := topology.AnalyzePerformance(ctx)
predictions, _ := topology.PredictBottlenecks(ctx)
```

### Topology Nodes
```go
// Create agent nodes with roles and capabilities
coordinator := mas.NewTopologyNode(agent, mas.CoordinatorRole, 
    []string{"planning", "coordination"})
specialist := mas.NewTopologyNode(agent, mas.SpecialistRole, 
    []string{"analysis", "research"})
worker := mas.NewTopologyNode(agent, mas.WorkerRole, 
    []string{"execution", "processing"})
```

### Collaboration Tasks
```go
// Create tasks with requirements
task := mas.NewCollaborationTask("data_analysis", 3, 
    []string{"analysis", "coordination"})
task.Data["dataset"] = "sales_data.csv"
task.Deadline = &deadline
```

### Workload Patterns
```go
// Define workload characteristics for adaptation
workload := &mas.WorkloadPattern{
    TaskTypes:        []string{"analysis", "processing"},
    IntensityProfile: map[string]float64{"analysis": 0.9},
    TimePattern:      "peak", // "peak", "burst", "constant"
    Duration:         time.Hour,
}
```

## Advanced Features

### Real-time Event Monitoring
```go
// Subscribe to topology events
topology.Subscribe("node_added", func(event *mas.TopologyEvent) {
    fmt.Printf("New node added: %s\n", event.NodeID)
})

topology.Subscribe("topology_optimized", func(event *mas.TopologyEvent) {
    efficiency := event.Data["efficiency_score"]
    fmt.Printf("Topology optimized: efficiency %.2f\n", efficiency)
})
```

### Performance Metrics
```go
type TopologyMetrics struct {
    TotalNodes        int           // Total number of nodes
    TotalEdges        int           // Total number of connections
    AverageLoad       float64       // Average node workload
    NetworkEfficiency float64       // Overall network efficiency  
    LoadBalance       float64       // Load distribution balance
    MessageThroughput float64       // Communication throughput
    AverageLatency    time.Duration // Average message latency
    FaultTolerance    float64       // Network resilience score
}
```

### Bottleneck Prediction
```go
type BottleneckPrediction struct {
    NodeID           string        // Node identifier
    PredictedLoad    float64       // Predicted future load
    TimeToBottleneck time.Duration // Time until bottleneck
    Confidence       float64       // Prediction confidence
    Mitigation       []string      // Suggested mitigation actions
}
```

## Use Cases

1. **Large-scale Data Processing**: Distribute analysis tasks across specialized agents
2. **Content Creation Pipelines**: Research → Analysis → Writing → Review workflows  
3. **Customer Service Networks**: Route inquiries to appropriate specialist agents
4. **Scientific Computing**: Coordinate complex simulations across agent clusters
5. **Real-time Decision Systems**: Adaptive networks for dynamic environments

## Benefits

- **🕸️ Automatic Organization**: Agents self-organize into optimal collaboration patterns
- **⚡ Dynamic Adaptation**: Network structure adapts to changing workload patterns
- **📊 Intelligent Distribution**: Tasks assigned based on capabilities and current load
- **🔄 Self-Optimization**: Continuous performance monitoring and automatic improvements
- **📈 Scalability**: Seamlessly scales from small teams to large agent networks
- **🛡️ Fault Tolerance**: Robust network design with bottleneck prediction and mitigation

This demonstrates how MAS transforms static multi-agent systems into intelligent, self-organizing collaboration networks that automatically optimize for performance, scalability, and efficiency.
