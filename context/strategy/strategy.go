package strategy

import (
	"context"
	"fmt"
	"sort"

	contextpkg "github.com/voocel/mas/context"
)

// strategyResult represents the result of applying a strategy
type strategyResult struct {
	strategy ContextStrategy
	state    *contextpkg.ContextState
	err      error
}

// ContextStrategy defines the interface for context engineering strategies
type ContextStrategy interface {
	Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error)
	Name() string
	Priority() int
	Description() string
}

// BaseStrategy provides common functionality for strategies
type BaseStrategy struct {
	name        string
	priority    int
	description string
}

// Name returns the strategy name
func (bs *BaseStrategy) Name() string {
	return bs.name
}

// Priority returns the strategy priority
func (bs *BaseStrategy) Priority() int {
	return bs.priority
}

// Description returns the strategy description
func (bs *BaseStrategy) Description() string {
	return bs.description
}

// CompositeStrategy combines multiple strategies
type CompositeStrategy struct {
	BaseStrategy
	strategies []ContextStrategy
	sequential bool // If true, apply strategies sequentially; if false, apply in parallel
}

// NewCompositeStrategy creates a new composite strategy
func NewCompositeStrategy(name string, sequential bool, strategies ...ContextStrategy) *CompositeStrategy {
	return &CompositeStrategy{
		BaseStrategy: BaseStrategy{
			name:        name,
			priority:    calculateCompositePriority(strategies),
			description: fmt.Sprintf("Composite strategy combining %d strategies", len(strategies)),
		},
		strategies: strategies,
		sequential: sequential,
	}
}

// Apply applies all strategies in the composite
func (cs *CompositeStrategy) Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	if len(cs.strategies) == 0 {
		return state, nil
	}

	if cs.sequential {
		return cs.applySequential(ctx, state)
	}
	return cs.applyParallel(ctx, state)
}

// applySequential applies strategies one after another
func (cs *CompositeStrategy) applySequential(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	currentState := state.Copy()

	// Sort strategies by priority (higher priority first)
	sortedStrategies := make([]ContextStrategy, len(cs.strategies))
	copy(sortedStrategies, cs.strategies)
	sort.Slice(sortedStrategies, func(i, j int) bool {
		return sortedStrategies[i].Priority() > sortedStrategies[j].Priority()
	})

	for _, strategy := range sortedStrategies {
		newState, err := strategy.Apply(ctx, currentState)
		if err != nil {
			return nil, fmt.Errorf("strategy %s failed: %w", strategy.Name(), err)
		}
		currentState = newState
	}

	return currentState, nil
}

// applyParallel applies strategies in parallel and merges results
func (cs *CompositeStrategy) applyParallel(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {

	results := make(chan strategyResult, len(cs.strategies))

	// Apply strategies in parallel
	for _, strategy := range cs.strategies {
		go func(s ContextStrategy) {
			newState, err := s.Apply(ctx, state.Copy())
			results <- strategyResult{
				strategy: s,
				state:    newState,
				err:      err,
			}
		}(strategy)
	}

	// Collect results
	strategyResults := make([]strategyResult, 0, len(cs.strategies))
	for i := 0; i < len(cs.strategies); i++ {
		result := <-results
		if result.err != nil {
			return nil, fmt.Errorf("strategy %s failed: %w", result.strategy.Name(), result.err)
		}
		strategyResults = append(strategyResults, result)
	}

	// Merge results
	return cs.mergeResults(state, strategyResults)
}

// mergeResults merges the results from parallel strategy execution
func (cs *CompositeStrategy) mergeResults(originalState *contextpkg.ContextState, results []strategyResult) (*contextpkg.ContextState, error) {
	merged := originalState.Copy()

	// Merge messages (avoid duplicates)
	messageSet := make(map[string]bool)
	for _, msg := range merged.Messages {
		key := fmt.Sprintf("%s_%s_%d", msg.Role, msg.Content, msg.Timestamp.Unix())
		messageSet[key] = true
	}

	for _, result := range results {
		for _, msg := range result.state.Messages {
			key := fmt.Sprintf("%s_%s_%d", msg.Role, msg.Content, msg.Timestamp.Unix())
			if !messageSet[key] {
				merged.Messages = append(merged.Messages, msg)
				messageSet[key] = true
			}
		}
	}

	// Merge scratchpad data
	for _, result := range results {
		for k, v := range result.state.Scratchpad {
			merged.Scratchpad[k] = v
		}
	}

	// Merge selected data
	for _, result := range results {
		for k, v := range result.state.SelectedData {
			merged.SelectedData[k] = v
		}
	}

	// Merge isolated context
	for _, result := range results {
		for k, v := range result.state.IsolatedCtx {
			merged.IsolatedCtx[k] = v
		}
	}

	// Use the compressed context from the highest priority strategy that produced one
	for _, result := range results {
		if result.state.CompressedCtx != nil {
			if merged.CompressedCtx == nil || result.strategy.Priority() > cs.findStrategyPriority(merged.CompressedCtx) {
				merged.CompressedCtx = result.state.CompressedCtx
			}
		}
	}

	return merged, nil
}

// findStrategyPriority finds the priority of the strategy that created the compressed context
func (cs *CompositeStrategy) findStrategyPriority(compressedCtx *contextpkg.CompressedContext) int {
	// This is a simplified implementation
	// In a real implementation, you might store strategy metadata in the compressed context
	return 0
}

// calculateCompositePriority calculates the priority for a composite strategy
func calculateCompositePriority(strategies []ContextStrategy) int {
	if len(strategies) == 0 {
		return 0
	}

	totalPriority := 0
	for _, strategy := range strategies {
		totalPriority += strategy.Priority()
	}

	return totalPriority / len(strategies)
}

// ConditionalStrategy applies a strategy only if a condition is met
type ConditionalStrategy struct {
	BaseStrategy
	condition func(*contextpkg.ContextState) bool
	strategy  ContextStrategy
}

// NewConditionalStrategy creates a new conditional strategy
func NewConditionalStrategy(
	name string,
	condition func(*contextpkg.ContextState) bool,
	strategy ContextStrategy,
) *ConditionalStrategy {
	return &ConditionalStrategy{
		BaseStrategy: BaseStrategy{
			name:        name,
			priority:    strategy.Priority(),
			description: fmt.Sprintf("Conditional wrapper for %s", strategy.Name()),
		},
		condition: condition,
		strategy:  strategy,
	}
}

// Apply applies the strategy only if the condition is met
func (cs *ConditionalStrategy) Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	if !cs.condition(state) {
		return state, nil // Return unchanged state if condition not met
	}

	return cs.strategy.Apply(ctx, state)
}

// StrategyChain represents a chain of strategies to be applied in order
type StrategyChain struct {
	BaseStrategy
	strategies []ContextStrategy
}

// NewStrategyChain creates a new strategy chain
func NewStrategyChain(name string, strategies ...ContextStrategy) *StrategyChain {
	return &StrategyChain{
		BaseStrategy: BaseStrategy{
			name:        name,
			priority:    calculateCompositePriority(strategies),
			description: fmt.Sprintf("Strategy chain with %d strategies", len(strategies)),
		},
		strategies: strategies,
	}
}

// Apply applies all strategies in the chain sequentially
func (sc *StrategyChain) Apply(ctx context.Context, state *contextpkg.ContextState) (*contextpkg.ContextState, error) {
	currentState := state.Copy()

	for i, strategy := range sc.strategies {
		newState, err := strategy.Apply(ctx, currentState)
		if err != nil {
			return nil, fmt.Errorf("strategy %d (%s) in chain failed: %w", i, strategy.Name(), err)
		}
		currentState = newState
	}

	return currentState, nil
}

// AddStrategy adds a strategy to the chain
func (sc *StrategyChain) AddStrategy(strategy ContextStrategy) {
	sc.strategies = append(sc.strategies, strategy)
	sc.priority = calculateCompositePriority(sc.strategies)
}

// RemoveStrategy removes a strategy from the chain by name
func (sc *StrategyChain) RemoveStrategy(name string) bool {
	for i, strategy := range sc.strategies {
		if strategy.Name() == name {
			sc.strategies = append(sc.strategies[:i], sc.strategies[i+1:]...)
			sc.priority = calculateCompositePriority(sc.strategies)
			return true
		}
	}
	return false
}

// GetStrategies returns a copy of the strategies in the chain
func (sc *StrategyChain) GetStrategies() []ContextStrategy {
	strategies := make([]ContextStrategy, len(sc.strategies))
	copy(strategies, sc.strategies)
	return strategies
}
