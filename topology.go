package mas

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// TopologyType represents different collaboration topology patterns
type TopologyType int

const (
	StarTopology      TopologyType = iota // Central coordinator with spoke agents
	ChainTopology                         // Sequential pipeline processing
	MeshTopology                          // Full mesh interconnected network
	HierarchyTopology                     // Tree-like hierarchical structure
	HubTopology                           // Multiple hubs with local clusters
	RingTopology                          // Circular ring communication
	AdaptiveTopology                      // Dynamic self-organizing topology
)

// CollaborationMode defines how agents collaborate
type CollaborationMode int

const (
	CompetitiveMode    CollaborationMode = iota // Agents compete for tasks
	CooperativeMode                             // Agents cooperate on shared goals
	DelegationMode                              // Hierarchical task delegation
	ConsensusMode                               // Consensus-based decision making
	SpecializationMode                          // Role-based specialization
	SwarmMode                                   // Swarm intelligence behavior
)

// AgentRole defines the role of an agent in the topology
type AgentRole int

const (
	CoordinatorRole AgentRole = iota // Central coordinator
	WorkerRole                       // Task worker
	SpecialistRole                   // Domain specialist
	BrokerRole                       // Message broker/router
	MonitorRole                      // Performance monitor
	DeciderRole                      // Decision maker
)

// TopologyNode represents a node in the collaboration topology
type TopologyNode struct {
	ID           string                 `json:"id"`
	Agent        Agent                  `json:"-"`
	Role         AgentRole              `json:"role"`
	Capabilities []string               `json:"capabilities"`
	Load         float64                `json:"load"`        // Current workload (0.0-1.0)
	Performance  float64                `json:"performance"` // Performance score (0.0-1.0)
	Availability bool                   `json:"availability"`
	LastActive   time.Time              `json:"last_active"`
	Connections  []string               `json:"connections"`        // Connected node IDs
	Position     *Position              `json:"position,omitempty"` // For visualization
	Metadata     map[string]interface{} `json:"metadata"`
}

// Position represents a 2D position for topology visualization
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// TopologyEdge represents a connection between two nodes
type TopologyEdge struct {
	From        string                 `json:"from"`
	To          string                 `json:"to"`
	Weight      float64                `json:"weight"`       // Connection strength
	Latency     time.Duration          `json:"latency"`      // Communication latency
	Bandwidth   float64                `json:"bandwidth"`    // Communication capacity
	MessageFlow int                    `json:"message_flow"` // Message count
	LastUsed    time.Time              `json:"last_used"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TopologyMetrics tracks topology performance
type TopologyMetrics struct {
	TotalNodes         int           `json:"total_nodes"`
	TotalEdges         int           `json:"total_edges"`
	AverageLoad        float64       `json:"average_load"`
	AveragePerformance float64       `json:"average_performance"`
	NetworkEfficiency  float64       `json:"network_efficiency"`
	MessageThroughput  float64       `json:"message_throughput"`
	AverageLatency     time.Duration `json:"average_latency"`
	TopologyDensity    float64       `json:"topology_density"`
	LoadBalance        float64       `json:"load_balance"`
	FaultTolerance     float64       `json:"fault_tolerance"`
	LastUpdated        time.Time     `json:"last_updated"`
}

// TopologyEvent represents topology change events
type TopologyEvent struct {
	Type      string                 `json:"type"`
	NodeID    string                 `json:"node_id,omitempty"`
	EdgeID    string                 `json:"edge_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// DynamicTopology manages the dynamic collaboration topology
type DynamicTopology interface {
	// Node management
	AddNode(node *TopologyNode) error
	RemoveNode(nodeID string) error
	GetNode(nodeID string) (*TopologyNode, error)
	ListNodes() []*TopologyNode
	UpdateNodeStatus(nodeID string, load float64, performance float64) error

	// Edge management
	AddEdge(edge *TopologyEdge) error
	RemoveEdge(from, to string) error
	GetEdge(from, to string) (*TopologyEdge, error)
	ListEdges() []*TopologyEdge
	UpdateEdgeMetrics(from, to string, latency time.Duration, messageCount int) error

	// Topology operations
	SetTopologyType(topologyType TopologyType) error
	GetTopologyType() TopologyType
	SetCollaborationMode(mode CollaborationMode) error
	GetCollaborationMode() CollaborationMode

	// Dynamic adaptation
	OptimizeTopology(ctx context.Context) error
	ReorganizeTopology(ctx context.Context, criteria OptimizationCriteria) error
	AdaptToWorkload(ctx context.Context, workload *WorkloadPattern) error

	// Task distribution
	DistributeTask(ctx context.Context, task *CollaborationTask) (*TaskAssignment, error)
	RouteMessage(ctx context.Context, message *TopologyMessage) error
	FindOptimalPath(from, to string) ([]string, error)

	// Analytics and monitoring
	GetMetrics() *TopologyMetrics
	AnalyzePerformance(ctx context.Context) (*TopologyAnalysis, error)
	PredictBottlenecks(ctx context.Context) ([]*BottleneckPrediction, error)

	// Events
	Subscribe(eventType string, handler func(*TopologyEvent)) error
	GetTopologySnapshot() *TopologySnapshot
}

// OptimizationCriteria defines criteria for topology optimization
type OptimizationCriteria struct {
	MinimizeLatency     bool    `json:"minimize_latency"`
	MaximizeThroughput  bool    `json:"maximize_throughput"`
	BalanceLoad         bool    `json:"balance_load"`
	MinimizeCost        bool    `json:"minimize_cost"`
	MaximizeReliability bool    `json:"maximize_reliability"`
	WeightLatency       float64 `json:"weight_latency"`
	WeightThroughput    float64 `json:"weight_throughput"`
	WeightBalance       float64 `json:"weight_balance"`
}

// WorkloadPattern represents workload characteristics
type WorkloadPattern struct {
	TaskTypes        []string               `json:"task_types"`
	IntensityProfile map[string]float64     `json:"intensity_profile"` // Task type -> intensity
	TimePattern      string                 `json:"time_pattern"`      // "constant", "peak", "burst"
	Duration         time.Duration          `json:"duration"`
	Requirements     map[string]interface{} `json:"requirements"`
}

// CollaborationTask represents a task for distributed execution
type CollaborationTask struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Priority     int                    `json:"priority"`
	Requirements []string               `json:"requirements"` // Required capabilities
	Data         map[string]interface{} `json:"data"`
	Deadline     *time.Time             `json:"deadline,omitempty"`
	Dependencies []string               `json:"dependencies"` // Task IDs this depends on
	Subtasks     []*CollaborationTask   `json:"subtasks,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// TaskAssignment represents task assignment to nodes
type TaskAssignment struct {
	TaskID        string                 `json:"task_id"`
	AssignedTo    []string               `json:"assigned_to"` // Node IDs
	Coordinator   string                 `json:"coordinator"` // Primary coordinator node
	Strategy      string                 `json:"strategy"`    // Assignment strategy used
	Confidence    float64                `json:"confidence"`  // Confidence in assignment
	EstimatedTime time.Duration          `json:"estimated_time"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// TopologyMessage represents messages passed between nodes
type TopologyMessage struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	To        []string               `json:"to"` // Can be multicast
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Priority  int                    `json:"priority"`
	TTL       int                    `json:"ttl"` // Time to live (hops)
	Timestamp time.Time              `json:"timestamp"`
	Route     []string               `json:"route"` // Path taken
}

// TopologyAnalysis represents analysis results
type TopologyAnalysis struct {
	OverallHealth      float64                `json:"overall_health"`
	BottleneckNodes    []string               `json:"bottleneck_nodes"`
	UnderutilizedNodes []string               `json:"underutilized_nodes"`
	CriticalPaths      [][]string             `json:"critical_paths"`
	Recommendations    []string               `json:"recommendations"`
	EfficiencyScore    float64                `json:"efficiency_score"`
	ScalabilityScore   float64                `json:"scalability_score"`
	ResilienceScore    float64                `json:"resilience_score"`
	OptimalTopology    TopologyType           `json:"optimal_topology"`
	Insights           map[string]interface{} `json:"insights"`
	AnalyzedAt         time.Time              `json:"analyzed_at"`
}

// BottleneckPrediction represents predicted performance bottlenecks
type BottleneckPrediction struct {
	NodeID           string        `json:"node_id"`
	PredictedLoad    float64       `json:"predicted_load"`
	TimeToBottleneck time.Duration `json:"time_to_bottleneck"`
	Confidence       float64       `json:"confidence"`
	Mitigation       []string      `json:"mitigation"` // Suggested actions
}

// TopologySnapshot represents a point-in-time topology state
type TopologySnapshot struct {
	Timestamp time.Time         `json:"timestamp"`
	Type      TopologyType      `json:"type"`
	Mode      CollaborationMode `json:"mode"`
	Nodes     []*TopologyNode   `json:"nodes"`
	Edges     []*TopologyEdge   `json:"edges"`
	Metrics   *TopologyMetrics  `json:"metrics"`
}

// basicDynamicTopology implements DynamicTopology
type basicDynamicTopology struct {
	nodes             map[string]*TopologyNode
	edges             map[string]*TopologyEdge // key: "from_to"
	topologyType      TopologyType
	collaborationMode CollaborationMode
	eventHandlers     map[string][]func(*TopologyEvent)
	metrics           *TopologyMetrics
	mu                sync.RWMutex
}

// NewDynamicTopology creates a new dynamic topology manager
func NewDynamicTopology(topologyType TopologyType, mode CollaborationMode) DynamicTopology {
	return &basicDynamicTopology{
		nodes:             make(map[string]*TopologyNode),
		edges:             make(map[string]*TopologyEdge),
		topologyType:      topologyType,
		collaborationMode: mode,
		eventHandlers:     make(map[string][]func(*TopologyEvent)),
		metrics: &TopologyMetrics{
			LastUpdated: time.Now(),
		},
	}
}

// AddNode implements DynamicTopology.AddNode
func (dt *basicDynamicTopology) AddNode(node *TopologyNode) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if node.ID == "" {
		node.ID = generateNodeID()
	}

	if node.Metadata == nil {
		node.Metadata = make(map[string]interface{})
	}

	node.LastActive = time.Now()
	node.Availability = true

	dt.nodes[node.ID] = node

	// Auto-connect based on topology type
	dt.autoConnectNode(node)

	// Update metrics
	dt.updateMetrics()

	// Emit event
	dt.emitEvent(&TopologyEvent{
		Type:      "node_added",
		NodeID:    node.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"role":         node.Role,
			"capabilities": node.Capabilities,
		},
	})

	return nil
}

// autoConnectNode automatically connects a new node based on topology type
func (dt *basicDynamicTopology) autoConnectNode(newNode *TopologyNode) {
	switch dt.topologyType {
	case StarTopology:
		dt.connectStar(newNode)
	case ChainTopology:
		dt.connectChain(newNode)
	case MeshTopology:
		dt.connectMesh(newNode)
	case HierarchyTopology:
		dt.connectHierarchy(newNode)
	case HubTopology:
		dt.connectHub(newNode)
	case RingTopology:
		dt.connectRing(newNode)
	case AdaptiveTopology:
		dt.connectAdaptive(newNode)
	}
}

// connectStar connects node in star topology pattern
func (dt *basicDynamicTopology) connectStar(newNode *TopologyNode) {
	// Find coordinator or make this node coordinator if none exists
	coordinator := dt.findCoordinator()

	if coordinator == nil && newNode.Role == CoordinatorRole {
		// This becomes the coordinator
		return
	} else if coordinator == nil {
		// No coordinator exists, find one or designate one
		for _, node := range dt.nodes {
			if node.Role == CoordinatorRole || node.ID != newNode.ID {
				coordinator = node
				coordinator.Role = CoordinatorRole
				break
			}
		}
	}

	if coordinator != nil && coordinator.ID != newNode.ID {
		// Connect to coordinator
		edge1 := &TopologyEdge{
			From:      newNode.ID,
			To:        coordinator.ID,
			Weight:    1.0,
			Bandwidth: 100.0,
			LastUsed:  time.Now(),
			Metadata:  make(map[string]interface{}),
		}
		edge2 := &TopologyEdge{
			From:      coordinator.ID,
			To:        newNode.ID,
			Weight:    1.0,
			Bandwidth: 100.0,
			LastUsed:  time.Now(),
			Metadata:  make(map[string]interface{}),
		}

		dt.edges[fmt.Sprintf("%s_%s", newNode.ID, coordinator.ID)] = edge1
		dt.edges[fmt.Sprintf("%s_%s", coordinator.ID, newNode.ID)] = edge2

		newNode.Connections = append(newNode.Connections, coordinator.ID)
		coordinator.Connections = append(coordinator.Connections, newNode.ID)
	}
}

// connectChain connects node in chain topology pattern
func (dt *basicDynamicTopology) connectChain(newNode *TopologyNode) {
	// Find the last node in chain and connect to it
	lastNode := dt.findLastInChain()

	if lastNode != nil && lastNode.ID != newNode.ID {
		edge := &TopologyEdge{
			From:      lastNode.ID,
			To:        newNode.ID,
			Weight:    1.0,
			Bandwidth: 100.0,
			LastUsed:  time.Now(),
			Metadata:  make(map[string]interface{}),
		}

		dt.edges[fmt.Sprintf("%s_%s", lastNode.ID, newNode.ID)] = edge
		lastNode.Connections = append(lastNode.Connections, newNode.ID)
		newNode.Connections = append(newNode.Connections, lastNode.ID)
	}
}

// connectMesh connects node in mesh topology pattern
func (dt *basicDynamicTopology) connectMesh(newNode *TopologyNode) {
	// Connect to all existing nodes
	for _, node := range dt.nodes {
		if node.ID != newNode.ID {
			edge1 := &TopologyEdge{
				From:      newNode.ID,
				To:        node.ID,
				Weight:    1.0,
				Bandwidth: 100.0,
				LastUsed:  time.Now(),
				Metadata:  make(map[string]interface{}),
			}
			edge2 := &TopologyEdge{
				From:      node.ID,
				To:        newNode.ID,
				Weight:    1.0,
				Bandwidth: 100.0,
				LastUsed:  time.Now(),
				Metadata:  make(map[string]interface{}),
			}

			dt.edges[fmt.Sprintf("%s_%s", newNode.ID, node.ID)] = edge1
			dt.edges[fmt.Sprintf("%s_%s", node.ID, newNode.ID)] = edge2

			newNode.Connections = append(newNode.Connections, node.ID)
			node.Connections = append(node.Connections, newNode.ID)
		}
	}
}

// connectHierarchy connects node in hierarchy topology pattern
func (dt *basicDynamicTopology) connectHierarchy(newNode *TopologyNode) {
	// Connect based on role hierarchy
	switch newNode.Role {
	case CoordinatorRole:
		// Top level - no parent
	case SpecialistRole, DeciderRole:
		// Connect to coordinator
		coordinator := dt.findCoordinator()
		if coordinator != nil {
			dt.createBidirectionalEdge(coordinator.ID, newNode.ID)
		}
	case WorkerRole:
		// Connect to specialists or coordinator
		parent := dt.findLeastLoadedParent()
		if parent != nil {
			dt.createBidirectionalEdge(parent.ID, newNode.ID)
		}
	}
}

// connectHub connects node in hub topology pattern
func (dt *basicDynamicTopology) connectHub(newNode *TopologyNode) {
	// Create local clusters around hub nodes
	if newNode.Role == CoordinatorRole || newNode.Role == BrokerRole {
		// This is a hub - connect to other hubs
		for _, node := range dt.nodes {
			if (node.Role == CoordinatorRole || node.Role == BrokerRole) && node.ID != newNode.ID {
				dt.createBidirectionalEdge(newNode.ID, node.ID)
			}
		}
	} else {
		// Connect to nearest hub
		hub := dt.findNearestHub(newNode)
		if hub != nil {
			dt.createBidirectionalEdge(newNode.ID, hub.ID)
		}
	}
}

// connectRing connects node in ring topology pattern
func (dt *basicDynamicTopology) connectRing(newNode *TopologyNode) {
	nodes := make([]*TopologyNode, 0, len(dt.nodes))
	for _, node := range dt.nodes {
		if node.ID != newNode.ID {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		return
	}

	if len(nodes) == 1 {
		// Connect to the only node
		dt.createBidirectionalEdge(newNode.ID, nodes[0].ID)
	} else {
		// Insert into ring - break one connection and insert new node
		// For simplicity, connect to first and last node
		firstNode := nodes[0]
		lastNode := nodes[len(nodes)-1]

		// Remove connection between first and last
		dt.removeEdgeDirect(firstNode.ID, lastNode.ID)

		// Connect new node between them
		dt.createBidirectionalEdge(lastNode.ID, newNode.ID)
		dt.createBidirectionalEdge(newNode.ID, firstNode.ID)
	}
}

// connectAdaptive connects node using adaptive strategy
func (dt *basicDynamicTopology) connectAdaptive(newNode *TopologyNode) {
	// Analyze current topology and connect optimally
	if len(dt.nodes) <= 1 {
		return
	}

	// Connect to nodes that would benefit most from this connection
	candidates := dt.findOptimalConnections(newNode)
	maxConnections := 3 // Limit connections for performance

	for i, candidate := range candidates {
		if i >= maxConnections {
			break
		}
		dt.createBidirectionalEdge(newNode.ID, candidate.ID)
	}
}

// Helper functions

func (dt *basicDynamicTopology) findCoordinator() *TopologyNode {
	for _, node := range dt.nodes {
		if node.Role == CoordinatorRole {
			return node
		}
	}
	return nil
}

func (dt *basicDynamicTopology) findLastInChain() *TopologyNode {
	// Find node with only one connection (tail of chain)
	for _, node := range dt.nodes {
		if len(node.Connections) <= 1 {
			return node
		}
	}
	// If no tail found, return any node
	for _, node := range dt.nodes {
		return node
	}
	return nil
}

func (dt *basicDynamicTopology) findLeastLoadedParent() *TopologyNode {
	var bestParent *TopologyNode
	lowestLoad := 1.0

	for _, node := range dt.nodes {
		if (node.Role == CoordinatorRole || node.Role == SpecialistRole || node.Role == DeciderRole) && node.Load < lowestLoad {
			bestParent = node
			lowestLoad = node.Load
		}
	}

	return bestParent
}

func (dt *basicDynamicTopology) findNearestHub(newNode *TopologyNode) *TopologyNode {
	// For simplicity, return least loaded hub
	var bestHub *TopologyNode
	lowestLoad := 1.0

	for _, node := range dt.nodes {
		if (node.Role == CoordinatorRole || node.Role == BrokerRole) && node.Load < lowestLoad {
			bestHub = node
			lowestLoad = node.Load
		}
	}

	return bestHub
}

func (dt *basicDynamicTopology) findOptimalConnections(newNode *TopologyNode) []*TopologyNode {
	type nodeScore struct {
		node  *TopologyNode
		score float64
	}

	scores := make([]nodeScore, 0)

	for _, node := range dt.nodes {
		if node.ID == newNode.ID {
			continue
		}

		score := dt.calculateConnectionScore(newNode, node)
		scores = append(scores, nodeScore{node: node, score: score})
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	result := make([]*TopologyNode, len(scores))
	for i, s := range scores {
		result[i] = s.node
	}

	return result
}

func (dt *basicDynamicTopology) calculateConnectionScore(node1, node2 *TopologyNode) float64 {
	score := 0.0

	// Capability complementarity
	capabilityScore := dt.calculateCapabilityComplementarity(node1.Capabilities, node2.Capabilities)
	score += capabilityScore * 0.4

	// Load balance
	avgLoad := (node1.Load + node2.Load) / 2
	loadScore := 1.0 - avgLoad // Prefer connecting less loaded nodes
	score += loadScore * 0.3

	// Performance
	avgPerformance := (node1.Performance + node2.Performance) / 2
	score += avgPerformance * 0.3

	return score
}

func (dt *basicDynamicTopology) calculateCapabilityComplementarity(caps1, caps2 []string) float64 {
	set1 := make(map[string]bool)
	for _, cap := range caps1 {
		set1[cap] = true
	}

	complementary := 0
	for _, cap := range caps2 {
		if !set1[cap] {
			complementary++
		}
	}

	if len(caps2) == 0 {
		return 0
	}

	return float64(complementary) / float64(len(caps2))
}

func (dt *basicDynamicTopology) createBidirectionalEdge(from, to string) {
	edge1 := &TopologyEdge{
		From:      from,
		To:        to,
		Weight:    1.0,
		Bandwidth: 100.0,
		LastUsed:  time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	edge2 := &TopologyEdge{
		From:      to,
		To:        from,
		Weight:    1.0,
		Bandwidth: 100.0,
		LastUsed:  time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	dt.edges[fmt.Sprintf("%s_%s", from, to)] = edge1
	dt.edges[fmt.Sprintf("%s_%s", to, from)] = edge2

	if fromNode, exists := dt.nodes[from]; exists {
		fromNode.Connections = append(fromNode.Connections, to)
	}
	if toNode, exists := dt.nodes[to]; exists {
		toNode.Connections = append(toNode.Connections, from)
	}
}

func (dt *basicDynamicTopology) removeEdgeDirect(from, to string) {
	delete(dt.edges, fmt.Sprintf("%s_%s", from, to))
	delete(dt.edges, fmt.Sprintf("%s_%s", to, from))

	if fromNode, exists := dt.nodes[from]; exists {
		fromNode.Connections = removeFromSlice(fromNode.Connections, to)
	}
	if toNode, exists := dt.nodes[to]; exists {
		toNode.Connections = removeFromSlice(toNode.Connections, from)
	}
}

func (dt *basicDynamicTopology) updateMetrics() {
	dt.metrics.TotalNodes = len(dt.nodes)
	dt.metrics.TotalEdges = len(dt.edges)
	dt.metrics.LastUpdated = time.Now()

	// Calculate other metrics
	totalLoad := 0.0
	totalPerformance := 0.0

	for _, node := range dt.nodes {
		totalLoad += node.Load
		totalPerformance += node.Performance
	}

	if len(dt.nodes) > 0 {
		dt.metrics.AverageLoad = totalLoad / float64(len(dt.nodes))
		dt.metrics.AveragePerformance = totalPerformance / float64(len(dt.nodes))
	}

	// Calculate topology density
	maxPossibleEdges := len(dt.nodes) * (len(dt.nodes) - 1)
	if maxPossibleEdges > 0 {
		dt.metrics.TopologyDensity = float64(len(dt.edges)) / float64(maxPossibleEdges)
	}

	// Calculate load balance (coefficient of variation)
	if len(dt.nodes) > 1 {
		variance := 0.0
		for _, node := range dt.nodes {
			diff := node.Load - dt.metrics.AverageLoad
			variance += diff * diff
		}
		variance /= float64(len(dt.nodes))
		stddev := math.Sqrt(variance)
		if dt.metrics.AverageLoad > 0 {
			cv := stddev / dt.metrics.AverageLoad
			dt.metrics.LoadBalance = 1.0 - math.Min(cv, 1.0) // Higher is better
		}
	}
}

func (dt *basicDynamicTopology) emitEvent(event *TopologyEvent) {
	if handlers, exists := dt.eventHandlers[event.Type]; exists {
		for _, handler := range handlers {
			go handler(event)
		}
	}
}

// Helper utility functions

func generateNodeID() string {
	return fmt.Sprintf("node_%d", time.Now().UnixNano())
}

func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// String representations

func (tt TopologyType) String() string {
	switch tt {
	case StarTopology:
		return "star"
	case ChainTopology:
		return "chain"
	case MeshTopology:
		return "mesh"
	case HierarchyTopology:
		return "hierarchy"
	case HubTopology:
		return "hub"
	case RingTopology:
		return "ring"
	case AdaptiveTopology:
		return "adaptive"
	default:
		return "unknown"
	}
}

func (cm CollaborationMode) String() string {
	switch cm {
	case CompetitiveMode:
		return "competitive"
	case CooperativeMode:
		return "cooperative"
	case DelegationMode:
		return "delegation"
	case ConsensusMode:
		return "consensus"
	case SpecializationMode:
		return "specialization"
	case SwarmMode:
		return "swarm"
	default:
		return "unknown"
	}
}

func (ar AgentRole) String() string {
	switch ar {
	case CoordinatorRole:
		return "coordinator"
	case WorkerRole:
		return "worker"
	case SpecialistRole:
		return "specialist"
	case BrokerRole:
		return "broker"
	case MonitorRole:
		return "monitor"
	case DeciderRole:
		return "decider"
	default:
		return "unknown"
	}
}

// NewTopologyNode creates a new topology node
func NewTopologyNode(agent Agent, role AgentRole, capabilities []string) *TopologyNode {
	return &TopologyNode{
		ID:           generateNodeID(),
		Agent:        agent,
		Role:         role,
		Capabilities: capabilities,
		Load:         0.0,
		Performance:  1.0,
		Availability: true,
		LastActive:   time.Now(),
		Connections:  make([]string, 0),
		Metadata:     make(map[string]interface{}),
	}
}

// Remaining DynamicTopology implementation

// RemoveNode implements DynamicTopology.RemoveNode
func (dt *basicDynamicTopology) RemoveNode(nodeID string) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	node, exists := dt.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Remove all edges connected to this node
	for _, connectedID := range node.Connections {
		dt.removeEdgeDirect(nodeID, connectedID)
	}

	// Remove the node
	delete(dt.nodes, nodeID)

	// Update metrics
	dt.updateMetrics()

	// Emit event
	dt.emitEvent(&TopologyEvent{
		Type:      "node_removed",
		NodeID:    nodeID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"role": node.Role,
		},
	})

	return nil
}

// GetNode implements DynamicTopology.GetNode
func (dt *basicDynamicTopology) GetNode(nodeID string) (*TopologyNode, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	node, exists := dt.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	// Return a copy to prevent external modification
	nodeCopy := *node
	return &nodeCopy, nil
}

// ListNodes implements DynamicTopology.ListNodes
func (dt *basicDynamicTopology) ListNodes() []*TopologyNode {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	nodes := make([]*TopologyNode, 0, len(dt.nodes))
	for _, node := range dt.nodes {
		nodeCopy := *node
		nodes = append(nodes, &nodeCopy)
	}

	return nodes
}

// UpdateNodeStatus implements DynamicTopology.UpdateNodeStatus
func (dt *basicDynamicTopology) UpdateNodeStatus(nodeID string, load float64, performance float64) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	node, exists := dt.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.Load = load
	node.Performance = performance
	node.LastActive = time.Now()

	// Update metrics
	dt.updateMetrics()

	// Emit event
	dt.emitEvent(&TopologyEvent{
		Type:      "node_updated",
		NodeID:    nodeID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"load":        load,
			"performance": performance,
		},
	})

	return nil
}

// AddEdge implements DynamicTopology.AddEdge
func (dt *basicDynamicTopology) AddEdge(edge *TopologyEdge) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	edgeKey := fmt.Sprintf("%s_%s", edge.From, edge.To)
	dt.edges[edgeKey] = edge

	// Update node connections
	if fromNode, exists := dt.nodes[edge.From]; exists {
		if !contains(fromNode.Connections, edge.To) {
			fromNode.Connections = append(fromNode.Connections, edge.To)
		}
	}

	// Update metrics
	dt.updateMetrics()

	// Emit event
	dt.emitEvent(&TopologyEvent{
		Type:      "edge_added",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"from":   edge.From,
			"to":     edge.To,
			"weight": edge.Weight,
		},
	})

	return nil
}

// RemoveEdge implements DynamicTopology.RemoveEdge
func (dt *basicDynamicTopology) RemoveEdge(from, to string) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	edgeKey := fmt.Sprintf("%s_%s", from, to)
	if _, exists := dt.edges[edgeKey]; !exists {
		return fmt.Errorf("edge not found: %s -> %s", from, to)
	}

	delete(dt.edges, edgeKey)

	// Update node connections
	if fromNode, exists := dt.nodes[from]; exists {
		fromNode.Connections = removeFromSlice(fromNode.Connections, to)
	}

	// Update metrics
	dt.updateMetrics()

	// Emit event
	dt.emitEvent(&TopologyEvent{
		Type:      "edge_removed",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"from": from,
			"to":   to,
		},
	})

	return nil
}

// GetEdge implements DynamicTopology.GetEdge
func (dt *basicDynamicTopology) GetEdge(from, to string) (*TopologyEdge, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	edgeKey := fmt.Sprintf("%s_%s", from, to)
	edge, exists := dt.edges[edgeKey]
	if !exists {
		return nil, fmt.Errorf("edge not found: %s -> %s", from, to)
	}

	// Return a copy
	edgeCopy := *edge
	return &edgeCopy, nil
}

// ListEdges implements DynamicTopology.ListEdges
func (dt *basicDynamicTopology) ListEdges() []*TopologyEdge {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	edges := make([]*TopologyEdge, 0, len(dt.edges))
	for _, edge := range dt.edges {
		edgeCopy := *edge
		edges = append(edges, &edgeCopy)
	}

	return edges
}

// UpdateEdgeMetrics implements DynamicTopology.UpdateEdgeMetrics
func (dt *basicDynamicTopology) UpdateEdgeMetrics(from, to string, latency time.Duration, messageCount int) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	edgeKey := fmt.Sprintf("%s_%s", from, to)
	edge, exists := dt.edges[edgeKey]
	if !exists {
		return fmt.Errorf("edge not found: %s -> %s", from, to)
	}

	edge.Latency = latency
	edge.MessageFlow += messageCount
	edge.LastUsed = time.Now()

	return nil
}

// SetTopologyType implements DynamicTopology.SetTopologyType
func (dt *basicDynamicTopology) SetTopologyType(topologyType TopologyType) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.topologyType = topologyType

	// Emit event
	dt.emitEvent(&TopologyEvent{
		Type:      "topology_type_changed",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"new_type": topologyType.String(),
		},
	})

	return nil
}

// GetTopologyType implements DynamicTopology.GetTopologyType
func (dt *basicDynamicTopology) GetTopologyType() TopologyType {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.topologyType
}

// SetCollaborationMode implements DynamicTopology.SetCollaborationMode
func (dt *basicDynamicTopology) SetCollaborationMode(mode CollaborationMode) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.collaborationMode = mode

	// Emit event
	dt.emitEvent(&TopologyEvent{
		Type:      "collaboration_mode_changed",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"new_mode": mode.String(),
		},
	})

	return nil
}

// GetCollaborationMode implements DynamicTopology.GetCollaborationMode
func (dt *basicDynamicTopology) GetCollaborationMode() CollaborationMode {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.collaborationMode
}

// OptimizeTopology implements DynamicTopology.OptimizeTopology
func (dt *basicDynamicTopology) OptimizeTopology(ctx context.Context) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Analyze current performance
	analysis := dt.analyzeTopologyPerformance()

	// Apply optimizations based on analysis
	if analysis.EfficiencyScore < 0.7 {
		// Low efficiency - consider restructuring
		if len(dt.nodes) > 10 && dt.topologyType == StarTopology {
			// Switch to hub topology for better scalability
			dt.topologyType = HubTopology
			dt.reorganizeToHubTopology()
		}
	}

	// Balance load by redistributing connections
	dt.balanceLoad()

	// Emit optimization event
	dt.emitEvent(&TopologyEvent{
		Type:      "topology_optimized",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"efficiency_score": analysis.EfficiencyScore,
		},
	})

	return nil
}

// ReorganizeTopology implements DynamicTopology.ReorganizeTopology
func (dt *basicDynamicTopology) ReorganizeTopology(ctx context.Context, criteria OptimizationCriteria) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Clear existing edges
	dt.edges = make(map[string]*TopologyEdge)
	for _, node := range dt.nodes {
		node.Connections = make([]string, 0)
	}

	// Reorganize based on criteria
	if criteria.BalanceLoad {
		dt.topologyType = HubTopology
		dt.reorganizeToHubTopology()
	} else if criteria.MinimizeLatency {
		dt.topologyType = MeshTopology
		dt.reorganizeToMeshTopology()
	} else if criteria.MaximizeThroughput {
		dt.topologyType = StarTopology
		dt.reorganizeToStarTopology()
	}

	// Update metrics
	dt.updateMetrics()

	return nil
}

// AdaptToWorkload implements DynamicTopology.AdaptToWorkload
func (dt *basicDynamicTopology) AdaptToWorkload(ctx context.Context, workload *WorkloadPattern) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Adapt topology based on workload characteristics
	switch workload.TimePattern {
	case "peak":
		// High load - use star topology with load balancing
		dt.topologyType = StarTopology
	case "burst":
		// Bursty load - use mesh topology for redundancy
		dt.topologyType = MeshTopology
	default:
		// Constant load - use adaptive topology
		dt.topologyType = AdaptiveTopology
	}

	// Adjust node roles based on task types
	for taskType, intensity := range workload.IntensityProfile {
		if intensity > 0.8 {
			// High intensity task - assign more specialists
			dt.promoteWorkersToSpecialists(taskType)
		}
	}

	return nil
}

// DistributeTask implements DynamicTopology.DistributeTask
func (dt *basicDynamicTopology) DistributeTask(ctx context.Context, task *CollaborationTask) (*TaskAssignment, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	// Find suitable nodes for the task
	candidates := dt.findTaskCandidates(task)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable nodes found for task %s", task.ID)
	}

	// Select best nodes based on load and capabilities
	selectedNodes := dt.selectOptimalNodes(candidates, task)

	// Determine coordinator
	coordinator := selectedNodes[0].ID
	if len(selectedNodes) > 1 {
		// Choose least loaded node as coordinator
		for _, node := range selectedNodes {
			if node.Load < dt.nodes[coordinator].Load {
				coordinator = node.ID
			}
		}
	}

	assignment := &TaskAssignment{
		TaskID:        task.ID,
		AssignedTo:    make([]string, len(selectedNodes)),
		Coordinator:   coordinator,
		Strategy:      "load_balanced",
		Confidence:    dt.calculateAssignmentConfidence(selectedNodes, task),
		EstimatedTime: dt.estimateTaskTime(task, selectedNodes),
		Metadata:      make(map[string]interface{}),
	}

	for i, node := range selectedNodes {
		assignment.AssignedTo[i] = node.ID
	}

	return assignment, nil
}

// RouteMessage implements DynamicTopology.RouteMessage
func (dt *basicDynamicTopology) RouteMessage(ctx context.Context, message *TopologyMessage) error {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	// Simple flooding for multicast
	if len(message.To) > 1 {
		for _, targetID := range message.To {
			singleMessage := *message
			singleMessage.To = []string{targetID}
			dt.routeSingleMessage(&singleMessage)
		}
		return nil
	}

	// Route single message
	if len(message.To) == 1 {
		return dt.routeSingleMessage(message)
	}

	return fmt.Errorf("no target specified for message")
}

// FindOptimalPath implements DynamicTopology.FindOptimalPath
func (dt *basicDynamicTopology) FindOptimalPath(from, to string) ([]string, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	// Simple breadth-first search for shortest path
	if from == to {
		return []string{from}, nil
	}

	queue := [][]string{{from}}
	visited := make(map[string]bool)
	visited[from] = true

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		current := path[len(path)-1]

		if currentNode, exists := dt.nodes[current]; exists {
			for _, neighbor := range currentNode.Connections {
				if neighbor == to {
					return append(path, neighbor), nil
				}

				if !visited[neighbor] {
					visited[neighbor] = true
					newPath := make([]string, len(path)+1)
					copy(newPath, path)
					newPath[len(path)] = neighbor
					queue = append(queue, newPath)
				}
			}
		}
	}

	return nil, fmt.Errorf("no path found from %s to %s", from, to)
}

// GetMetrics implements DynamicTopology.GetMetrics
func (dt *basicDynamicTopology) GetMetrics() *TopologyMetrics {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	// Return a copy
	metrics := *dt.metrics
	return &metrics
}

// AnalyzePerformance implements DynamicTopology.AnalyzePerformance
func (dt *basicDynamicTopology) AnalyzePerformance(ctx context.Context) (*TopologyAnalysis, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	return dt.analyzeTopologyPerformance(), nil
}

// PredictBottlenecks implements DynamicTopology.PredictBottlenecks
func (dt *basicDynamicTopology) PredictBottlenecks(ctx context.Context) ([]*BottleneckPrediction, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	predictions := make([]*BottleneckPrediction, 0)

	for _, node := range dt.nodes {
		if node.Load > 0.8 {
			// High load - potential bottleneck
			timeToBottleneck := time.Duration((1.0-node.Load)*60) * time.Minute

			prediction := &BottleneckPrediction{
				NodeID:           node.ID,
				PredictedLoad:    math.Min(node.Load*1.2, 1.0),
				TimeToBottleneck: timeToBottleneck,
				Confidence:       0.8,
				Mitigation:       []string{"redistribute_load", "add_parallel_node"},
			}
			predictions = append(predictions, prediction)
		}
	}

	return predictions, nil
}

// Subscribe implements DynamicTopology.Subscribe
func (dt *basicDynamicTopology) Subscribe(eventType string, handler func(*TopologyEvent)) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.eventHandlers[eventType] == nil {
		dt.eventHandlers[eventType] = make([]func(*TopologyEvent), 0)
	}

	dt.eventHandlers[eventType] = append(dt.eventHandlers[eventType], handler)
	return nil
}

// GetTopologySnapshot implements DynamicTopology.GetTopologySnapshot
func (dt *basicDynamicTopology) GetTopologySnapshot() *TopologySnapshot {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	return &TopologySnapshot{
		Timestamp: time.Now(),
		Type:      dt.topologyType,
		Mode:      dt.collaborationMode,
		Nodes:     dt.ListNodes(),
		Edges:     dt.ListEdges(),
		Metrics:   dt.GetMetrics(),
	}
}

// Helper methods for topology reorganization

func (dt *basicDynamicTopology) reorganizeToHubTopology() {
	// Select hub nodes based on capabilities and performance
	hubs := dt.selectHubNodes()

	// Connect hubs to each other
	for i, hub1 := range hubs {
		for j, hub2 := range hubs {
			if i != j {
				dt.createBidirectionalEdge(hub1.ID, hub2.ID)
			}
		}
	}

	// Connect remaining nodes to nearest hub
	for _, node := range dt.nodes {
		if !dt.isHub(node, hubs) {
			nearestHub := dt.findNearestHub(node)
			if nearestHub != nil {
				dt.createBidirectionalEdge(node.ID, nearestHub.ID)
			}
		}
	}
}

func (dt *basicDynamicTopology) reorganizeToMeshTopology() {
	// Connect all nodes to all other nodes
	nodes := make([]*TopologyNode, 0, len(dt.nodes))
	for _, node := range dt.nodes {
		nodes = append(nodes, node)
	}

	for i, node1 := range nodes {
		for j, node2 := range nodes {
			if i != j {
				dt.createBidirectionalEdge(node1.ID, node2.ID)
			}
		}
	}
}

func (dt *basicDynamicTopology) reorganizeToStarTopology() {
	// Select coordinator
	coordinator := dt.findCoordinator()
	if coordinator == nil {
		// Select best node as coordinator
		coordinator = dt.selectBestCoordinator()
		if coordinator != nil {
			coordinator.Role = CoordinatorRole
		}
	}

	if coordinator != nil {
		// Connect all other nodes to coordinator
		for _, node := range dt.nodes {
			if node.ID != coordinator.ID {
				dt.createBidirectionalEdge(coordinator.ID, node.ID)
			}
		}
	}
}

func (dt *basicDynamicTopology) selectHubNodes() []*TopologyNode {
	nodes := make([]*TopologyNode, 0, len(dt.nodes))
	for _, node := range dt.nodes {
		nodes = append(nodes, node)
	}

	// Sort by performance and select top nodes as hubs
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Performance > nodes[j].Performance
	})

	// Select top 20% or minimum 2 nodes as hubs
	hubCount := max(2, len(nodes)/5)
	if hubCount > len(nodes) {
		hubCount = len(nodes)
	}

	return nodes[:hubCount]
}

func (dt *basicDynamicTopology) isHub(node *TopologyNode, hubs []*TopologyNode) bool {
	for _, hub := range hubs {
		if hub.ID == node.ID {
			return true
		}
	}
	return false
}

func (dt *basicDynamicTopology) selectBestCoordinator() *TopologyNode {
	var best *TopologyNode
	bestScore := -1.0

	for _, node := range dt.nodes {
		score := node.Performance * (1.0 - node.Load)
		if score > bestScore {
			bestScore = score
			best = node
		}
	}

	return best
}

func (dt *basicDynamicTopology) analyzeTopologyPerformance() *TopologyAnalysis {
	totalPerf := 0.0
	bottlenecks := make([]string, 0)
	underutilized := make([]string, 0)

	for _, node := range dt.nodes {
		totalPerf += node.Performance

		if node.Load > 0.8 {
			bottlenecks = append(bottlenecks, node.ID)
		} else if node.Load < 0.2 && node.Availability {
			underutilized = append(underutilized, node.ID)
		}
	}

	avgPerf := totalPerf / float64(len(dt.nodes))

	analysis := &TopologyAnalysis{
		OverallHealth:      avgPerf * dt.metrics.LoadBalance,
		BottleneckNodes:    bottlenecks,
		UnderutilizedNodes: underutilized,
		EfficiencyScore:    dt.calculateEfficiencyScore(),
		ScalabilityScore:   dt.calculateScalabilityScore(),
		ResilienceScore:    dt.calculateResilienceScore(),
		OptimalTopology:    dt.recommendOptimalTopology(),
		Insights:           make(map[string]interface{}),
		AnalyzedAt:         time.Now(),
	}

	// Generate recommendations
	analysis.Recommendations = dt.generateTopologyRecommendations(analysis)

	return analysis
}

func (dt *basicDynamicTopology) calculateEfficiencyScore() float64 {
	if len(dt.nodes) == 0 {
		return 0.0
	}

	// Calculate based on load balance and connection efficiency
	return dt.metrics.LoadBalance * dt.metrics.AveragePerformance
}

func (dt *basicDynamicTopology) calculateScalabilityScore() float64 {
	// Based on topology type and current size
	switch dt.topologyType {
	case StarTopology:
		return math.Max(0.0, 1.0-float64(len(dt.nodes))/100.0) // Decreases with size
	case MeshTopology:
		return math.Max(0.0, 1.0-float64(len(dt.nodes))/50.0) // Poor scalability
	case HubTopology:
		return 0.8 // Good scalability
	case HierarchyTopology:
		return 0.9 // Excellent scalability
	default:
		return 0.6
	}
}

func (dt *basicDynamicTopology) calculateResilienceScore() float64 {
	// Based on redundancy and connectivity
	if len(dt.nodes) <= 1 {
		return 0.0
	}

	// Calculate average node degree
	totalConnections := 0
	for _, node := range dt.nodes {
		totalConnections += len(node.Connections)
	}

	avgDegree := float64(totalConnections) / float64(len(dt.nodes))
	maxDegree := float64(len(dt.nodes) - 1)

	return math.Min(avgDegree/maxDegree, 1.0)
}

func (dt *basicDynamicTopology) recommendOptimalTopology() TopologyType {
	nodeCount := len(dt.nodes)

	if nodeCount <= 5 {
		return MeshTopology
	} else if nodeCount <= 20 {
		return StarTopology
	} else if nodeCount <= 100 {
		return HubTopology
	} else {
		return HierarchyTopology
	}
}

func (dt *basicDynamicTopology) generateTopologyRecommendations(analysis *TopologyAnalysis) []string {
	recommendations := make([]string, 0)

	if analysis.EfficiencyScore < 0.6 {
		recommendations = append(recommendations, "Consider restructuring topology for better efficiency")
	}

	if len(analysis.BottleneckNodes) > 0 {
		recommendations = append(recommendations, "Redistribute load from bottleneck nodes")
	}

	if len(analysis.UnderutilizedNodes) > 0 {
		recommendations = append(recommendations, "Increase task assignment to underutilized nodes")
	}

	if analysis.ScalabilityScore < 0.5 {
		recommendations = append(recommendations, "Consider switching to more scalable topology")
	}

	if analysis.ResilienceScore < 0.4 {
		recommendations = append(recommendations, "Add more connections for better fault tolerance")
	}

	return recommendations
}

// Helper functions for task distribution

func (dt *basicDynamicTopology) findTaskCandidates(task *CollaborationTask) []*TopologyNode {
	candidates := make([]*TopologyNode, 0)

	for _, node := range dt.nodes {
		if dt.nodeCanHandleTask(node, task) {
			candidates = append(candidates, node)
		}
	}

	return candidates
}

func (dt *basicDynamicTopology) nodeCanHandleTask(node *TopologyNode, task *CollaborationTask) bool {
	if !node.Availability || node.Load > 0.9 {
		return false
	}

	// Check if node has required capabilities
	for _, requirement := range task.Requirements {
		if !contains(node.Capabilities, requirement) {
			return false
		}
	}

	return true
}

func (dt *basicDynamicTopology) selectOptimalNodes(candidates []*TopologyNode, task *CollaborationTask) []*TopologyNode {
	// Sort candidates by score (performance / load)
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := candidates[i].Performance / (candidates[i].Load + 0.1)
		scoreJ := candidates[j].Performance / (candidates[j].Load + 0.1)
		return scoreI > scoreJ
	})

	// Select top candidates (at least 1, at most 3)
	count := min(3, max(1, len(candidates)/2))
	return candidates[:count]
}

func (dt *basicDynamicTopology) calculateAssignmentConfidence(nodes []*TopologyNode, task *CollaborationTask) float64 {
	if len(nodes) == 0 {
		return 0.0
	}

	totalPerf := 0.0
	totalLoad := 0.0

	for _, node := range nodes {
		totalPerf += node.Performance
		totalLoad += node.Load
	}

	avgPerf := totalPerf / float64(len(nodes))
	avgLoad := totalLoad / float64(len(nodes))

	return avgPerf * (1.0 - avgLoad)
}

func (dt *basicDynamicTopology) estimateTaskTime(task *CollaborationTask, nodes []*TopologyNode) time.Duration {
	// Simple estimation based on task priority and node performance
	baseTime := time.Duration(60/task.Priority) * time.Minute

	if len(nodes) > 0 {
		avgPerf := 0.0
		for _, node := range nodes {
			avgPerf += node.Performance
		}
		avgPerf /= float64(len(nodes))

		// Adjust time based on performance
		adjustedTime := time.Duration(float64(baseTime) / avgPerf)
		return adjustedTime
	}

	return baseTime
}

func (dt *basicDynamicTopology) routeSingleMessage(message *TopologyMessage) error {
	if len(message.To) != 1 {
		return fmt.Errorf("single message must have exactly one target")
	}

	target := message.To[0]
	path, err := dt.FindOptimalPath(message.From, target)
	if err != nil {
		return fmt.Errorf("failed to find path: %w", err)
	}

	message.Route = path
	message.TTL = len(path)

	// Update edge metrics along the path
	for i := 0; i < len(path)-1; i++ {
		dt.UpdateEdgeMetrics(path[i], path[i+1], time.Millisecond*10, 1)
	}

	return nil
}

func (dt *basicDynamicTopology) balanceLoad() {
	// Simple load balancing by redistributing connections
	overloadedNodes := make([]*TopologyNode, 0)
	underloadedNodes := make([]*TopologyNode, 0)

	for _, node := range dt.nodes {
		if node.Load > 0.8 {
			overloadedNodes = append(overloadedNodes, node)
		} else if node.Load < 0.3 {
			underloadedNodes = append(underloadedNodes, node)
		}
	}

	// Redistribute connections from overloaded to underloaded nodes
	for _, overloaded := range overloadedNodes {
		for _, underloaded := range underloadedNodes {
			if len(overloaded.Connections) > 2 {
				// Create connection to underloaded node
				dt.createBidirectionalEdge(overloaded.ID, underloaded.ID)
				break
			}
		}
	}
}

func (dt *basicDynamicTopology) promoteWorkersToSpecialists(taskType string) {
	count := 0
	maxPromotions := 3

	for _, node := range dt.nodes {
		if node.Role == WorkerRole && count < maxPromotions {
			node.Role = SpecialistRole
			node.Capabilities = append(node.Capabilities, taskType)
			count++
		}
	}
}

// Utility functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// NewCollaborationTask creates a new collaboration task
func NewCollaborationTask(taskType string, priority int, requirements []string) *CollaborationTask {
	return &CollaborationTask{
		ID:           fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Type:         taskType,
		Priority:     priority,
		Requirements: requirements,
		Data:         make(map[string]interface{}),
		Dependencies: make([]string, 0),
		CreatedAt:    time.Now(),
	}
}
