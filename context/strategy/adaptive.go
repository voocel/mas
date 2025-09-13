package strategy

import (
	"context"
	"fmt"
	"sort"

	contextpkg "github.com/voocel/mas/context"
)

// AdaptiveStrategy implements an intelligent strategy that automatically selects
// and combines the best strategies based on context analysis
type AdaptiveStrategy struct {
	BaseStrategy
	strategies map[string]ContextStrategy
	rules      []AdaptiveRule
	config     AdaptiveConfig
	analyzer   *ContextAnalyzer
}

// AdaptiveConfig configures the adaptive strategy
type AdaptiveConfig struct {
	MaxTokens            int     `json:"max_tokens"`
	CompressionThreshold float64 `json:"compression_threshold"`
	IsolationThreshold   int     `json:"isolation_threshold"`
	SelectionThreshold   float64 `json:"selection_threshold"`
	EnableLearning       bool    `json:"enable_learning"`
	MaxStrategies        int     `json:"max_strategies"`
}

// AdaptiveRule defines a rule for strategy selection
type AdaptiveRule struct {
	Name        string                               `json:"name"`
	Condition   func(*contextpkg.StateAnalysis) bool `json:"-"`
	Strategy    string                               `json:"strategy"`
	Priority    int                                  `json:"priority"`
	Description string                               `json:"description"`
}

// ContextAnalyzer analyzes context state to make strategy decisions
type ContextAnalyzer struct {
	config AdaptiveConfig
}

// NewAdaptiveStrategy creates a new adaptive strategy
func NewAdaptiveStrategy(strategies map[string]ContextStrategy, config ...AdaptiveConfig) *AdaptiveStrategy {
	cfg := DefaultAdaptiveConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	as := &AdaptiveStrategy{
		BaseStrategy: BaseStrategy{
			name:        "adaptive",
			priority:    10, // Highest priority as it coordinates others
			description: "Intelligently selects and combines strategies based on context analysis",
		},
		strategies: strategies,
		config:     cfg,
		analyzer:   NewContextAnalyzer(cfg),
	}

	// Initialize default rules
	as.initializeDefaultRules()

	return as
}

// DefaultAdaptiveConfig returns the default adaptive configuration
func DefaultAdaptiveConfig() AdaptiveConfig {
	return AdaptiveConfig{
		MaxTokens:            4000,
		CompressionThreshold: 0.8,
		IsolationThreshold:   3,
		SelectionThreshold:   0.6,
		EnableLearning:       true,
		MaxStrategies:        3,
	}
}

// NewContextAnalyzer creates a new context analyzer
func NewContextAnalyzer(config AdaptiveConfig) *ContextAnalyzer {
	return &ContextAnalyzer{
		config: config,
	}
}

// Apply applies the adaptive strategy to the context state
func (as *AdaptiveStrategy) Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	// Analyze the current context state
	analysis := as.analyzer.AnalyzeContext(state)

	// Select appropriate strategies based on analysis
	selectedStrategies := as.selectStrategies(analysis)

	// Apply selected strategies in order of priority
	return as.applySelectedStrategies(ctx, state, selectedStrategies)
}

// initializeDefaultRules sets up the default adaptive rules
func (as *AdaptiveStrategy) initializeDefaultRules() {
	as.rules = []AdaptiveRule{
		{
			Name: "high_token_compression",
			Condition: func(analysis *contextpkg.StateAnalysis) bool {
				pressure := float64(analysis.TokenCount) / float64(as.config.MaxTokens)
				return pressure >= as.config.CompressionThreshold
			},
			Strategy:    "compress",
			Priority:    9,
			Description: "Apply compression when token count is high",
		},
		{
			Name: "multi_agent_isolation",
			Condition: func(analysis *contextpkg.StateAnalysis) bool {
				return analysis.AgentCount >= as.config.IsolationThreshold
			},
			Strategy:    "isolate",
			Priority:    8,
			Description: "Apply isolation when multiple agents are active",
		},
		{
			Name: "high_complexity_selection",
			Condition: func(analysis *contextpkg.StateAnalysis) bool {
				return analysis.Complexity >= as.config.SelectionThreshold
			},
			Strategy:    "select",
			Priority:    7,
			Description: "Apply selection when context complexity is high",
		},
		{
			Name: "always_write",
			Condition: func(analysis *contextpkg.StateAnalysis) bool {
				return true // Always apply write strategy for persistence
			},
			Strategy:    "write",
			Priority:    6,
			Description: "Always apply write strategy for information persistence",
		},
		{
			Name: "recent_activity_selection",
			Condition: func(analysis *contextpkg.StateAnalysis) bool {
				return analysis.RecentActivity > 5
			},
			Strategy:    "select",
			Priority:    5,
			Description: "Apply selection when there's high recent activity",
		},
	}
}

// selectStrategies selects appropriate strategies based on context analysis
func (as *AdaptiveStrategy) selectStrategies(analysis *contextpkg.StateAnalysis) []string {
	var selectedStrategies []string
	var applicableRules []AdaptiveRule

	// Find applicable rules
	for _, rule := range as.rules {
		if rule.Condition(analysis) {
			applicableRules = append(applicableRules, rule)
		}
	}

	// Sort rules by priority (descending)
	sort.Slice(applicableRules, func(i, j int) bool {
		return applicableRules[i].Priority > applicableRules[j].Priority
	})

	// Select strategies up to the maximum limit
	strategySet := make(map[string]bool)
	for _, rule := range applicableRules {
		if len(selectedStrategies) >= as.config.MaxStrategies {
			break
		}
		if !strategySet[rule.Strategy] {
			selectedStrategies = append(selectedStrategies, rule.Strategy)
			strategySet[rule.Strategy] = true
		}
	}

	return selectedStrategies
}

// applySelectedStrategies applies the selected strategies in sequence
func (as *AdaptiveStrategy) applySelectedStrategies(
	ctx context.Context,
	state *contextpkg.ContextState,
	strategyNames []string,
) (*contextpkg.ContextState, error) {
	currentState := state.Copy()

	for _, strategyName := range strategyNames {
		strategy, exists := as.strategies[strategyName]
		if !exists {
			continue // Skip unknown strategies
		}

		newState, err := strategy.Apply(ctx, currentState)
		if err != nil {
			return nil, fmt.Errorf("strategy %s failed: %w", strategyName, err)
		}
		currentState = newState
	}

	return currentState, nil
}

// AnalyzeContext analyzes the context state and returns analysis results
func (ca *ContextAnalyzer) AnalyzeContext(state *contextpkg.ContextState) *contextpkg.StateAnalysis {
	analysis := &contextpkg.StateAnalysis{
		TokenCount:   ca.estimateTokenCount(state),
		MessageCount: len(state.Messages),
		AgentCount:   ca.countActiveAgents(state),
		Complexity:   ca.calculateComplexity(state),
	}

	// Calculate memory pressure
	analysis.MemoryPressure = float64(analysis.TokenCount) / float64(ca.config.MaxTokens)
	if analysis.MemoryPressure > 1.0 {
		analysis.MemoryPressure = 1.0
	}

	// Count recent activity
	analysis.RecentActivity = ca.countRecentActivity(state)

	return analysis
}

// estimateTokenCount estimates the total token count for the context state
func (ca *ContextAnalyzer) estimateTokenCount(state *contextpkg.ContextState) int {
	tokens := 0

	// Count message tokens
	for _, msg := range state.Messages {
		tokens += len(msg.Content) / 4 // Rough estimation
	}

	// Count scratchpad tokens
	for _, value := range state.Scratchpad {
		if str, ok := value.(string); ok {
			tokens += len(str) / 4
		}
	}

	// Count selected data tokens (rough estimation)
	tokens += len(state.SelectedData) * 50

	// Count compressed context tokens
	if state.CompressedCtx != nil {
		tokens += len(state.CompressedCtx.Summary) / 4
		for _, point := range state.CompressedCtx.KeyPoints {
			tokens += len(point) / 4
		}
	}

	return tokens
}

// countActiveAgents counts the number of active agents
func (ca *ContextAnalyzer) countActiveAgents(state *contextpkg.ContextState) int {
	agents := make(map[string]bool)

	// Count agents from messages
	for _, msg := range state.Messages {
		if msg.Name != "" {
			agents[msg.Name] = true
		}
	}

	// Count current agent
	if state.AgentID != "" {
		agents[state.AgentID] = true
	}

	return len(agents)
}

// calculateComplexity calculates the complexity score of the context
func (ca *ContextAnalyzer) calculateComplexity(state *contextpkg.ContextState) float64 {
	complexity := 0.0

	// Factor 1: Number of messages
	complexity += float64(len(state.Messages)) * 0.1

	// Factor 2: Average message length
	if len(state.Messages) > 0 {
		totalLength := 0
		for _, msg := range state.Messages {
			totalLength += len(msg.Content)
		}
		avgLength := float64(totalLength) / float64(len(state.Messages))
		complexity += avgLength / 1000 // Normalize
	}

	// Factor 3: Scratchpad complexity
	complexity += float64(len(state.Scratchpad)) * 0.2

	// Factor 4: Selected data complexity
	complexity += float64(len(state.SelectedData)) * 0.15

	// Factor 5: Isolated context complexity
	complexity += float64(len(state.IsolatedCtx)) * 0.1

	// Normalize to 0-1 range
	if complexity > 10 {
		complexity = 10
	}
	return complexity / 10
}

// countRecentActivity counts recent activity in the context
func (ca *ContextAnalyzer) countRecentActivity(state *contextpkg.ContextState) int {
	// Count messages in the last few positions as recent activity
	recentCount := 5
	if len(state.Messages) < recentCount {
		return len(state.Messages)
	}
	return recentCount
}

// AddRule adds a new adaptive rule
func (as *AdaptiveStrategy) AddRule(rule AdaptiveRule) {
	as.rules = append(as.rules, rule)
}

// RemoveRule removes an adaptive rule by name
func (as *AdaptiveStrategy) RemoveRule(name string) bool {
	for i, rule := range as.rules {
		if rule.Name == name {
			as.rules = append(as.rules[:i], as.rules[i+1:]...)
			return true
		}
	}
	return false
}

// GetRules returns a copy of all adaptive rules
func (as *AdaptiveStrategy) GetRules() []AdaptiveRule {
	rules := make([]AdaptiveRule, len(as.rules))
	copy(rules, as.rules)
	return rules
}

// GetApplicableRules returns rules that would apply to the given analysis
func (as *AdaptiveStrategy) GetApplicableRules(analysis *contextpkg.StateAnalysis) []AdaptiveRule {
	var applicable []AdaptiveRule
	for _, rule := range as.rules {
		if rule.Condition(analysis) {
			applicable = append(applicable, rule)
		}
	}
	return applicable
}

// UpdateConfig updates the adaptive configuration
func (as *AdaptiveStrategy) UpdateConfig(config AdaptiveConfig) {
	as.config = config
	as.analyzer.config = config
}
