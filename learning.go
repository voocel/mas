package mas

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// ExperienceType represents the type of experience
type ExperienceType int

const (
	ChatExperience     ExperienceType = iota // Chat interaction experience
	ToolExperience                           // Tool execution experience
	SkillExperience                          // Skill execution experience
	GoalExperience                           // Goal pursuit experience
	DecisionExperience                       // Decision making experience
	PlanExperience                           // Planning experience
)

// Experience represents a learning experience record
type Experience struct {
	ID          string                 `json:"id"`
	Type        ExperienceType         `json:"type"`
	Context     map[string]interface{} `json:"context"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters"`
	Result      interface{}            `json:"result"`
	Success     bool                   `json:"success"`
	Performance float64                `json:"performance"` // 0.0 to 1.0
	Duration    time.Duration          `json:"duration"`
	Feedback    string                 `json:"feedback"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// LearningPattern represents a discovered behavioral pattern
type LearningPattern struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Context         map[string]interface{} `json:"context"`
	Actions         []string               `json:"actions"`
	SuccessRate     float64                `json:"success_rate"`
	Confidence      float64                `json:"confidence"`
	Frequency       int                    `json:"frequency"`
	LastSeen        time.Time              `json:"last_seen"`
	PerformanceGain float64                `json:"performance_gain"`
}

// LearningStrategy defines how the agent learns
type LearningStrategy int

const (
	ReinforcementLearning LearningStrategy = iota // Learn from rewards/penalties
	ImitationLearning                             // Learn from successful patterns
	ExplorationLearning                           // Learn through exploration
	ReflectionLearning                            // Learn through self-reflection
	HybridLearning                                // Combination of strategies
)

// AdaptationMode defines how the agent adapts behavior
type AdaptationMode int

const (
	ConservativeAdaptation AdaptationMode = iota // Slow, careful changes
	AggressiveAdaptation                         // Fast, bold changes
	BalancedAdaptation                           // Moderate changes
	ContextualAdaptation                         // Context-dependent changes
)

// LearningEngine manages the agent's learning and adaptation
type LearningEngine interface {
	// Experience management
	RecordExperience(ctx context.Context, experience *Experience) error
	GetExperiences(filter ExperienceFilter) ([]*Experience, error)
	AnalyzeExperiences(ctx context.Context) (*LearningAnalysis, error)

	// Pattern discovery
	DiscoverPatterns(ctx context.Context) ([]*LearningPattern, error)
	GetPatterns(filter PatternFilter) ([]*LearningPattern, error)
	ApplyPattern(ctx context.Context, patternID string, context map[string]interface{}) (*PatternApplication, error)

	// Strategy optimization
	OptimizeStrategy(ctx context.Context, domain string) (*StrategyOptimization, error)
	PredictPerformance(ctx context.Context, action string, context map[string]interface{}) (float64, error)

	// Self-reflection and adaptation
	SelfReflect(ctx context.Context) (*SelfReflection, error)
	AdaptBehavior(ctx context.Context, insights *LearningAnalysis) error

	// Configuration
	SetLearningStrategy(strategy LearningStrategy) error
	SetAdaptationMode(mode AdaptationMode) error
	GetLearningMetrics() *LearningMetrics
}

// ExperienceFilter defines filtering criteria for experiences
type ExperienceFilter struct {
	Types       []ExperienceType `json:"types,omitempty"`
	MinSuccess  *float64         `json:"min_success,omitempty"`
	TimeRange   *TimeRange       `json:"time_range,omitempty"`
	ContextKeys []string         `json:"context_keys,omitempty"`
	Limit       int              `json:"limit,omitempty"`
}

// PatternFilter defines filtering criteria for patterns
type PatternFilter struct {
	MinSuccessRate *float64   `json:"min_success_rate,omitempty"`
	MinConfidence  *float64   `json:"min_confidence,omitempty"`
	MinFrequency   *int       `json:"min_frequency,omitempty"`
	TimeRange      *TimeRange `json:"time_range,omitempty"`
}

// TimeRange represents a time range for filtering
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// LearningAnalysis represents the result of experience analysis
type LearningAnalysis struct {
	TotalExperiences   int                    `json:"total_experiences"`
	SuccessRate        float64                `json:"success_rate"`
	AveragePerformance float64                `json:"average_performance"`
	PerformanceTrend   string                 `json:"performance_trend"` // "improving", "declining", "stable"
	TopPatterns        []*LearningPattern     `json:"top_patterns"`
	WeakAreas          []string               `json:"weak_areas"`
	Recommendations    []string               `json:"recommendations"`
	ConfidenceScore    float64                `json:"confidence_score"`
	LastAnalyzed       time.Time              `json:"last_analyzed"`
	Insights           map[string]interface{} `json:"insights"`
}

// PatternApplication represents the result of applying a pattern
type PatternApplication struct {
	PatternID         string                 `json:"pattern_id"`
	Confidence        float64                `json:"confidence"`
	RecommendedAction string                 `json:"recommended_action"`
	ExpectedOutcome   string                 `json:"expected_outcome"`
	RiskAssessment    string                 `json:"risk_assessment"`
	Parameters        map[string]interface{} `json:"parameters"`
}

// StrategyOptimization represents optimization recommendations
type StrategyOptimization struct {
	Domain              string                 `json:"domain"`
	CurrentPerformance  float64                `json:"current_performance"`
	OptimizedStrategy   string                 `json:"optimized_strategy"`
	ExpectedImprovement float64                `json:"expected_improvement"`
	Confidence          float64                `json:"confidence"`
	Recommendations     []string               `json:"recommendations"`
	Parameters          map[string]interface{} `json:"parameters"`
}

// SelfReflection represents the agent's self-analysis
type SelfReflection struct {
	OverallAssessment  string                 `json:"overall_assessment"`
	Strengths          []string               `json:"strengths"`
	Weaknesses         []string               `json:"weaknesses"`
	LearningProgress   float64                `json:"learning_progress"`
	AdaptationNeeded   bool                   `json:"adaptation_needed"`
	SelfConfidence     float64                `json:"self_confidence"`
	GoalsAlignment     float64                `json:"goals_alignment"`
	RecommendedActions []string               `json:"recommended_actions"`
	ReflectionDepth    string                 `json:"reflection_depth"` // "surface", "deep", "profound"
	Insights           map[string]interface{} `json:"insights"`
	ReflectedAt        time.Time              `json:"reflected_at"`
}

// LearningMetrics tracks learning performance
type LearningMetrics struct {
	TotalExperiences       int           `json:"total_experiences"`
	LearningRate           float64       `json:"learning_rate"`
	AdaptationRate         float64       `json:"adaptation_rate"`
	PatternDiscoveryRate   float64       `json:"pattern_discovery_rate"`
	PerformanceImprovement float64       `json:"performance_improvement"`
	KnowledgeRetention     float64       `json:"knowledge_retention"`
	ExplorationRatio       float64       `json:"exploration_ratio"`
	LastLearningSession    time.Time     `json:"last_learning_session"`
	LearningStreak         int           `json:"learning_streak"`
	TotalLearningTime      time.Duration `json:"total_learning_time"`
}

// basicLearningEngine implements LearningEngine
type basicLearningEngine struct {
	experiences      []*Experience
	patterns         []*LearningPattern
	learningStrategy LearningStrategy
	adaptationMode   AdaptationMode
	agent            Agent
	mu               sync.RWMutex
	metrics          *LearningMetrics
}

// NewLearningEngine creates a new learning engine for an agent
func NewLearningEngine(agent Agent) LearningEngine {
	return &basicLearningEngine{
		experiences:      make([]*Experience, 0),
		patterns:         make([]*LearningPattern, 0),
		learningStrategy: HybridLearning,
		adaptationMode:   BalancedAdaptation,
		agent:            agent,
		metrics: &LearningMetrics{
			TotalExperiences:       0,
			LearningRate:           0.1,
			AdaptationRate:         0.05,
			PatternDiscoveryRate:   0.02,
			PerformanceImprovement: 0.0,
			KnowledgeRetention:     0.8,
			ExplorationRatio:       0.2,
			LastLearningSession:    time.Now(),
			LearningStreak:         0,
			TotalLearningTime:      0,
		},
	}
}

// RecordExperience implements LearningEngine.RecordExperience
func (le *basicLearningEngine) RecordExperience(ctx context.Context, experience *Experience) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	if experience.ID == "" {
		experience.ID = generateExperienceID()
	}

	experience.Timestamp = time.Now()
	le.experiences = append(le.experiences, experience)

	// Update metrics
	le.metrics.TotalExperiences++
	le.metrics.LastLearningSession = time.Now()

	// Trigger pattern discovery if we have enough experiences
	if len(le.experiences)%10 == 0 {
		go func() {
			patterns, err := le.DiscoverPatterns(ctx)
			if err == nil && len(patterns) > 0 {
				le.mu.Lock()
				le.patterns = append(le.patterns, patterns...)
				le.mu.Unlock()
			}
		}()
	}

	// Emit learning event
	if le.agent.GetEventBus() != nil {
		le.agent.PublishEvent(ctx, EventType("learning.experience.recorded"), EventData(
			"experience_id", experience.ID,
			"type", experience.Type,
			"success", experience.Success,
			"performance", experience.Performance,
		))
	}

	return nil
}

// GetExperiences implements LearningEngine.GetExperiences
func (le *basicLearningEngine) GetExperiences(filter ExperienceFilter) ([]*Experience, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	var filtered []*Experience

	for _, exp := range le.experiences {
		if le.matchesExperienceFilter(exp, filter) {
			filtered = append(filtered, exp)
		}
	}

	// Apply limit
	if filter.Limit > 0 && len(filtered) > filter.Limit {
		filtered = filtered[:filter.Limit]
	}

	return filtered, nil
}

// matchesExperienceFilter checks if experience matches filter
func (le *basicLearningEngine) matchesExperienceFilter(exp *Experience, filter ExperienceFilter) bool {
	// Check type filter
	if len(filter.Types) > 0 {
		matched := false
		for _, t := range filter.Types {
			if exp.Type == t {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check success rate filter
	if filter.MinSuccess != nil {
		var successRate float64
		if exp.Success {
			successRate = 1.0
		} else {
			successRate = 0.0
		}
		if successRate < *filter.MinSuccess {
			return false
		}
	}

	// Check time range filter
	if filter.TimeRange != nil {
		if exp.Timestamp.Before(filter.TimeRange.Start) || exp.Timestamp.After(filter.TimeRange.End) {
			return false
		}
	}

	return true
}

// AnalyzeExperiences implements LearningEngine.AnalyzeExperiences
func (le *basicLearningEngine) AnalyzeExperiences(ctx context.Context) (*LearningAnalysis, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	if len(le.experiences) == 0 {
		return &LearningAnalysis{
			TotalExperiences: 0,
			SuccessRate:      0.0,
			LastAnalyzed:     time.Now(),
		}, nil
	}

	// Calculate basic metrics
	successCount := 0
	totalPerformance := 0.0

	for _, exp := range le.experiences {
		if exp.Success {
			successCount++
		}
		totalPerformance += exp.Performance
	}

	successRate := float64(successCount) / float64(len(le.experiences))
	avgPerformance := totalPerformance / float64(len(le.experiences))

	// Analyze performance trend
	trend := le.analyzePerformanceTrend()

	// Get top patterns
	topPatterns := le.getTopPatterns(5)

	// Identify weak areas
	weakAreas := le.identifyWeakAreas()

	// Generate recommendations
	recommendations := le.generateRecommendations(successRate, avgPerformance, trend)

	analysis := &LearningAnalysis{
		TotalExperiences:   len(le.experiences),
		SuccessRate:        successRate,
		AveragePerformance: avgPerformance,
		PerformanceTrend:   trend,
		TopPatterns:        topPatterns,
		WeakAreas:          weakAreas,
		Recommendations:    recommendations,
		ConfidenceScore:    le.calculateConfidenceScore(),
		LastAnalyzed:       time.Now(),
		Insights:           make(map[string]interface{}),
	}

	// Emit analysis event
	if le.agent.GetEventBus() != nil {
		le.agent.PublishEvent(ctx, EventType("learning.analysis.completed"), EventData(
			"total_experiences", analysis.TotalExperiences,
			"success_rate", analysis.SuccessRate,
			"performance_trend", analysis.PerformanceTrend,
		))
	}

	return analysis, nil
}

// analyzePerformanceTrend analyzes recent performance trend
func (le *basicLearningEngine) analyzePerformanceTrend() string {
	if len(le.experiences) < 6 {
		return "insufficient_data"
	}

	// Take last 10 experiences and compare with previous 10
	recent := le.experiences[len(le.experiences)-5:]
	previous := le.experiences[len(le.experiences)-10 : len(le.experiences)-5]

	recentAvg := 0.0
	for _, exp := range recent {
		recentAvg += exp.Performance
	}
	recentAvg /= float64(len(recent))

	previousAvg := 0.0
	for _, exp := range previous {
		previousAvg += exp.Performance
	}
	previousAvg /= float64(len(previous))

	diff := recentAvg - previousAvg
	threshold := 0.05

	if diff > threshold {
		return "improving"
	} else if diff < -threshold {
		return "declining"
	} else {
		return "stable"
	}
}

// getTopPatterns returns the top performing patterns
func (le *basicLearningEngine) getTopPatterns(limit int) []*LearningPattern {
	if len(le.patterns) == 0 {
		return []*LearningPattern{}
	}

	// Sort patterns by success rate * confidence
	sorted := make([]*LearningPattern, len(le.patterns))
	copy(sorted, le.patterns)

	sort.Slice(sorted, func(i, j int) bool {
		scoreI := sorted[i].SuccessRate * sorted[i].Confidence
		scoreJ := sorted[j].SuccessRate * sorted[j].Confidence
		return scoreI > scoreJ
	})

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	return sorted
}

// identifyWeakAreas identifies areas needing improvement
func (le *basicLearningEngine) identifyWeakAreas() []string {
	weakAreas := []string{}

	// Analyze by experience type
	typePerformance := make(map[ExperienceType][]float64)

	for _, exp := range le.experiences {
		typePerformance[exp.Type] = append(typePerformance[exp.Type], exp.Performance)
	}

	for expType, performances := range typePerformance {
		if len(performances) > 0 {
			avg := 0.0
			for _, p := range performances {
				avg += p
			}
			avg /= float64(len(performances))

			if avg < 0.6 { // Below 60% performance
				weakAreas = append(weakAreas, expType.String())
			}
		}
	}

	return weakAreas
}

// generateRecommendations creates improvement recommendations
func (le *basicLearningEngine) generateRecommendations(successRate, avgPerformance float64, trend string) []string {
	recommendations := []string{}

	if successRate < 0.7 {
		recommendations = append(recommendations, "Focus on improving success rate through skill practice")
	}

	if avgPerformance < 0.6 {
		recommendations = append(recommendations, "Enhance performance by analyzing successful patterns")
	}

	if trend == "declining" {
		recommendations = append(recommendations, "Review recent failures and adjust strategy")
	}

	if len(le.patterns) < 3 {
		recommendations = append(recommendations, "Gather more experiences to discover behavioral patterns")
	}

	return recommendations
}

// calculateConfidenceScore calculates overall confidence in learning
func (le *basicLearningEngine) calculateConfidenceScore() float64 {
	if len(le.experiences) < 10 {
		return 0.3 // Low confidence with insufficient data
	}

	// Base confidence on experience count, success rate, and pattern quality
	expFactor := math.Min(float64(len(le.experiences))/100.0, 1.0)

	successCount := 0
	for _, exp := range le.experiences {
		if exp.Success {
			successCount++
		}
	}
	successFactor := float64(successCount) / float64(len(le.experiences))

	patternFactor := math.Min(float64(len(le.patterns))/10.0, 1.0)

	confidence := (expFactor*0.3 + successFactor*0.5 + patternFactor*0.2)
	return math.Min(confidence, 1.0)
}

// Helper functions

// generateExperienceID generates a unique ID for an experience
func generateExperienceID() string {
	return fmt.Sprintf("exp_%d", time.Now().UnixNano())
}

// String returns string representation of ExperienceType
func (et ExperienceType) String() string {
	switch et {
	case ChatExperience:
		return "chat"
	case ToolExperience:
		return "tool"
	case SkillExperience:
		return "skill"
	case GoalExperience:
		return "goal"
	case DecisionExperience:
		return "decision"
	case PlanExperience:
		return "plan"
	default:
		return "unknown"
	}
}

// DiscoverPatterns implements LearningEngine.DiscoverPatterns
func (le *basicLearningEngine) DiscoverPatterns(ctx context.Context) ([]*LearningPattern, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	if len(le.experiences) < 5 {
		return []*LearningPattern{}, nil // Need minimum experiences for pattern discovery
	}

	var newPatterns []*LearningPattern

	// Group experiences by success and analyze patterns
	successfulExps := []*Experience{}
	failedExps := []*Experience{}

	for _, exp := range le.experiences {
		if exp.Success && exp.Performance > 0.7 {
			successfulExps = append(successfulExps, exp)
		} else if !exp.Success || exp.Performance < 0.3 {
			failedExps = append(failedExps, exp)
		}
	}

	// Discover successful patterns
	if len(successfulExps) >= 3 {
		pattern := le.analyzeSuccessfulPattern(successfulExps)
		if pattern != nil {
			newPatterns = append(newPatterns, pattern)
		}
	}

	// Discover failure patterns (to avoid)
	if len(failedExps) >= 3 {
		pattern := le.analyzeFailurePattern(failedExps)
		if pattern != nil {
			newPatterns = append(newPatterns, pattern)
		}
	}

	// Emit pattern discovery event
	if len(newPatterns) > 0 && le.agent.GetEventBus() != nil {
		le.agent.PublishEvent(ctx, EventType("learning.patterns.discovered"), EventData(
			"pattern_count", len(newPatterns),
		))
	}

	return newPatterns, nil
}

// analyzeSuccessfulPattern analyzes successful experiences for patterns
func (le *basicLearningEngine) analyzeSuccessfulPattern(experiences []*Experience) *LearningPattern {
	if len(experiences) < 3 {
		return nil
	}

	// Find common actions
	actionFreq := make(map[string]int)
	for _, exp := range experiences {
		actionFreq[exp.Action]++
	}

	// Find most common action
	var commonAction string
	maxFreq := 0
	for action, freq := range actionFreq {
		if freq > maxFreq {
			maxFreq = freq
			commonAction = action
		}
	}

	if maxFreq < 2 {
		return nil // Not enough frequency to be a pattern
	}

	// Calculate success rate for this action
	successRate := float64(maxFreq) / float64(len(experiences))

	// Calculate average performance
	totalPerf := 0.0
	for _, exp := range experiences {
		if exp.Action == commonAction {
			totalPerf += exp.Performance
		}
	}
	avgPerf := totalPerf / float64(maxFreq)

	pattern := &LearningPattern{
		ID:              generatePatternID(),
		Name:            fmt.Sprintf("Successful %s Pattern", commonAction),
		Description:     fmt.Sprintf("Pattern for successful execution of %s", commonAction),
		Actions:         []string{commonAction},
		SuccessRate:     successRate,
		Confidence:      le.calculatePatternConfidence(experiences),
		Frequency:       maxFreq,
		LastSeen:        time.Now(),
		PerformanceGain: avgPerf,
		Context:         le.extractCommonContext(experiences),
	}

	return pattern
}

// analyzeFailurePattern analyzes failed experiences for patterns to avoid
func (le *basicLearningEngine) analyzeFailurePattern(experiences []*Experience) *LearningPattern {
	if len(experiences) < 3 {
		return nil
	}

	// Find common failure actions
	actionFreq := make(map[string]int)
	for _, exp := range experiences {
		actionFreq[exp.Action]++
	}

	var commonAction string
	maxFreq := 0
	for action, freq := range actionFreq {
		if freq > maxFreq {
			maxFreq = freq
			commonAction = action
		}
	}

	if maxFreq < 2 {
		return nil
	}

	failureRate := float64(maxFreq) / float64(len(experiences))

	pattern := &LearningPattern{
		ID:              generatePatternID(),
		Name:            fmt.Sprintf("Failure %s Pattern", commonAction),
		Description:     fmt.Sprintf("Pattern indicating likely failure when executing %s", commonAction),
		Actions:         []string{commonAction},
		SuccessRate:     1.0 - failureRate, // Inverted for failure pattern
		Confidence:      le.calculatePatternConfidence(experiences),
		Frequency:       maxFreq,
		LastSeen:        time.Now(),
		PerformanceGain: -0.5, // Negative gain for failure pattern
		Context:         le.extractCommonContext(experiences),
	}

	return pattern
}

// calculatePatternConfidence calculates confidence in a discovered pattern
func (le *basicLearningEngine) calculatePatternConfidence(experiences []*Experience) float64 {
	if len(experiences) < 3 {
		return 0.1
	}

	// Base confidence on frequency and consistency
	freqFactor := math.Min(float64(len(experiences))/10.0, 1.0)

	// Calculate consistency (variance in performance)
	performances := []float64{}
	for _, exp := range experiences {
		performances = append(performances, exp.Performance)
	}

	variance := le.calculateVariance(performances)
	consistencyFactor := 1.0 - math.Min(variance, 1.0)

	confidence := (freqFactor*0.6 + consistencyFactor*0.4)
	return math.Max(0.1, math.Min(confidence, 1.0))
}

// calculateVariance calculates variance of performance values
func (le *basicLearningEngine) calculateVariance(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))

	return variance
}

// extractCommonContext extracts common context elements from experiences
func (le *basicLearningEngine) extractCommonContext(experiences []*Experience) map[string]interface{} {
	commonContext := make(map[string]interface{})

	if len(experiences) == 0 {
		return commonContext
	}

	// Find keys that appear in most experiences
	keyCount := make(map[string]int)
	for _, exp := range experiences {
		for key := range exp.Context {
			keyCount[key]++
		}
	}

	threshold := len(experiences) / 2 // Key must appear in at least half of experiences

	for key, count := range keyCount {
		if count >= threshold {
			// Get most common value for this key
			valueCount := make(map[interface{}]int)
			for _, exp := range experiences {
				if value, exists := exp.Context[key]; exists {
					valueCount[value]++
				}
			}

			var mostCommonValue interface{}
			maxCount := 0
			for value, count := range valueCount {
				if count > maxCount {
					maxCount = count
					mostCommonValue = value
				}
			}

			commonContext[key] = mostCommonValue
		}
	}

	return commonContext
}

// GetPatterns implements LearningEngine.GetPatterns
func (le *basicLearningEngine) GetPatterns(filter PatternFilter) ([]*LearningPattern, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	var filtered []*LearningPattern

	for _, pattern := range le.patterns {
		if le.matchesPatternFilter(pattern, filter) {
			filtered = append(filtered, pattern)
		}
	}

	return filtered, nil
}

// matchesPatternFilter checks if pattern matches filter criteria
func (le *basicLearningEngine) matchesPatternFilter(pattern *LearningPattern, filter PatternFilter) bool {
	if filter.MinSuccessRate != nil && pattern.SuccessRate < *filter.MinSuccessRate {
		return false
	}

	if filter.MinConfidence != nil && pattern.Confidence < *filter.MinConfidence {
		return false
	}

	if filter.MinFrequency != nil && pattern.Frequency < *filter.MinFrequency {
		return false
	}

	if filter.TimeRange != nil {
		if pattern.LastSeen.Before(filter.TimeRange.Start) || pattern.LastSeen.After(filter.TimeRange.End) {
			return false
		}
	}

	return true
}

// ApplyPattern implements LearningEngine.ApplyPattern
func (le *basicLearningEngine) ApplyPattern(ctx context.Context, patternID string, context map[string]interface{}) (*PatternApplication, error) {
	le.mu.RLock()
	defer le.mu.RUnlock()

	var pattern *LearningPattern
	for _, p := range le.patterns {
		if p.ID == patternID {
			pattern = p
			break
		}
	}

	if pattern == nil {
		return nil, fmt.Errorf("pattern not found: %s", patternID)
	}

	// Calculate context match confidence
	contextMatch := le.calculateContextMatch(pattern.Context, context)

	// Determine recommended action
	recommendedAction := ""
	if len(pattern.Actions) > 0 {
		recommendedAction = pattern.Actions[0]
	}

	// Assess risk
	riskAssessment := "low"
	if pattern.SuccessRate < 0.7 || pattern.Confidence < 0.6 {
		riskAssessment = "medium"
	}
	if pattern.SuccessRate < 0.5 || pattern.Confidence < 0.4 {
		riskAssessment = "high"
	}

	application := &PatternApplication{
		PatternID:         patternID,
		Confidence:        pattern.Confidence * contextMatch,
		RecommendedAction: recommendedAction,
		ExpectedOutcome:   fmt.Sprintf("Success probability: %.2f", pattern.SuccessRate),
		RiskAssessment:    riskAssessment,
		Parameters:        context,
	}

	// Emit pattern application event
	if le.agent.GetEventBus() != nil {
		le.agent.PublishEvent(ctx, EventType("learning.pattern.applied"), EventData(
			"pattern_id", patternID,
			"confidence", application.Confidence,
			"risk", riskAssessment,
		))
	}

	return application, nil
}

// calculateContextMatch calculates how well current context matches pattern context
func (le *basicLearningEngine) calculateContextMatch(patternContext, currentContext map[string]interface{}) float64 {
	if len(patternContext) == 0 {
		return 1.0 // No context constraints
	}

	matches := 0
	for key, expectedValue := range patternContext {
		if currentValue, exists := currentContext[key]; exists {
			if le.valuesMatch(expectedValue, currentValue) {
				matches++
			}
		}
	}

	return float64(matches) / float64(len(patternContext))
}

// valuesMatch checks if two interface{} values match
func (le *basicLearningEngine) valuesMatch(a, b interface{}) bool {
	// Simple equality check - in production you might want more sophisticated comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// SelfReflect implements LearningEngine.SelfReflect
func (le *basicLearningEngine) SelfReflect(ctx context.Context) (*SelfReflection, error) {
	analysis, err := le.AnalyzeExperiences(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze experiences: %w", err)
	}

	// Use agent's reasoning capability for self-reflection
	situation := NewSituation(
		map[string]interface{}{
			"total_experiences":   analysis.TotalExperiences,
			"success_rate":        analysis.SuccessRate,
			"average_performance": analysis.AveragePerformance,
			"performance_trend":   analysis.PerformanceTrend,
			"weak_areas":          analysis.WeakAreas,
		},
		[]string{
			"reflect on learning progress",
			"assess strengths and weaknesses",
			"identify areas for improvement",
		},
	)

	decision, err := le.agent.Reason(ctx, situation)
	if err != nil {
		return nil, fmt.Errorf("failed to perform self-reflection: %w", err)
	}

	// Parse reasoning result into structured reflection
	reflection := &SelfReflection{
		OverallAssessment: decision.Reasoning,
		LearningProgress:  analysis.SuccessRate,
		AdaptationNeeded:  analysis.PerformanceTrend == "declining",
		SelfConfidence:    analysis.ConfidenceScore,
		GoalsAlignment:    le.calculateGoalsAlignment(),
		ReflectionDepth:   "deep",
		ReflectedAt:       time.Now(),
		Insights:          make(map[string]interface{}),
	}

	// Extract strengths and weaknesses
	reflection.Strengths = le.identifyStrengths(analysis)
	reflection.Weaknesses = analysis.WeakAreas

	// Generate recommended actions
	reflection.RecommendedActions = le.generateSelfImprovementActions(analysis)

	// Emit self-reflection event
	if le.agent.GetEventBus() != nil {
		le.agent.PublishEvent(ctx, EventType("learning.self_reflection.completed"), EventData(
			"learning_progress", reflection.LearningProgress,
			"self_confidence", reflection.SelfConfidence,
			"adaptation_needed", reflection.AdaptationNeeded,
		))
	}

	return reflection, nil
}

// identifyStrengths identifies agent's strengths based on analysis
func (le *basicLearningEngine) identifyStrengths(analysis *LearningAnalysis) []string {
	strengths := []string{}

	if analysis.SuccessRate > 0.8 {
		strengths = append(strengths, "High success rate in task execution")
	}

	if analysis.AveragePerformance > 0.7 {
		strengths = append(strengths, "Consistently high performance")
	}

	if analysis.PerformanceTrend == "improving" {
		strengths = append(strengths, "Continuous learning and improvement")
	}

	if len(analysis.TopPatterns) > 3 {
		strengths = append(strengths, "Strong pattern recognition and application")
	}

	return strengths
}

// generateSelfImprovementActions generates action recommendations
func (le *basicLearningEngine) generateSelfImprovementActions(analysis *LearningAnalysis) []string {
	actions := []string{}

	if analysis.SuccessRate < 0.7 {
		actions = append(actions, "Focus on improving task execution accuracy")
	}

	if analysis.PerformanceTrend == "declining" {
		actions = append(actions, "Analyze recent failures and adjust approach")
	}

	if len(analysis.WeakAreas) > 0 {
		actions = append(actions, fmt.Sprintf("Strengthen capabilities in: %v", analysis.WeakAreas))
	}

	if len(analysis.TopPatterns) < 3 {
		actions = append(actions, "Gain more diverse experiences to discover patterns")
	}

	return actions
}

// calculateGoalsAlignment calculates alignment with goals
func (le *basicLearningEngine) calculateGoalsAlignment() float64 {
	// Simple implementation - could be enhanced with actual goal analysis
	return 0.75 // Default reasonable alignment
}

// OptimizeStrategy implements LearningEngine.OptimizeStrategy
func (le *basicLearningEngine) OptimizeStrategy(ctx context.Context, domain string) (*StrategyOptimization, error) {
	// Analyze performance in the specific domain
	domainExperiences := []*Experience{}
	for _, exp := range le.experiences {
		// Simple domain matching - could be more sophisticated
		if exp.Action == domain || fmt.Sprintf("%v", exp.Context["domain"]) == domain {
			domainExperiences = append(domainExperiences, exp)
		}
	}

	if len(domainExperiences) < 3 {
		return nil, fmt.Errorf("insufficient experience in domain: %s", domain)
	}

	// Calculate current performance
	currentPerf := 0.0
	for _, exp := range domainExperiences {
		currentPerf += exp.Performance
	}
	currentPerf /= float64(len(domainExperiences))

	// Find best performing strategy
	bestStrategy := "adaptive"
	expectedImprovement := 0.1

	optimization := &StrategyOptimization{
		Domain:              domain,
		CurrentPerformance:  currentPerf,
		OptimizedStrategy:   bestStrategy,
		ExpectedImprovement: expectedImprovement,
		Confidence:          0.7,
		Recommendations:     []string{"Apply best practices from successful patterns"},
		Parameters:          make(map[string]interface{}),
	}

	return optimization, nil
}

// PredictPerformance implements LearningEngine.PredictPerformance
func (le *basicLearningEngine) PredictPerformance(ctx context.Context, action string, context map[string]interface{}) (float64, error) {
	// Find similar past experiences
	similarExps := []*Experience{}
	for _, exp := range le.experiences {
		if exp.Action == action {
			// Could add context similarity calculation here
			similarExps = append(similarExps, exp)
		}
	}

	if len(similarExps) == 0 {
		return 0.5, nil // Default neutral prediction
	}

	// Calculate average performance for similar experiences
	totalPerf := 0.0
	for _, exp := range similarExps {
		totalPerf += exp.Performance
	}

	return totalPerf / float64(len(similarExps)), nil
}

// AdaptBehavior implements LearningEngine.AdaptBehavior
func (le *basicLearningEngine) AdaptBehavior(ctx context.Context, insights *LearningAnalysis) error {
	// Emit adaptation start event
	if le.agent.GetEventBus() != nil {
		le.agent.PublishEvent(ctx, EventType("learning.adaptation.started"), EventData(
			"success_rate", insights.SuccessRate,
			"performance_trend", insights.PerformanceTrend,
		))
	}

	// Adapt based on performance trend
	switch insights.PerformanceTrend {
	case "declining":
		le.adaptationMode = ConservativeAdaptation
	case "improving":
		le.adaptationMode = AggressiveAdaptation
	default:
		le.adaptationMode = BalancedAdaptation
	}

	// Update learning rate based on success rate
	if insights.SuccessRate > 0.8 {
		le.metrics.LearningRate = math.Min(le.metrics.LearningRate*1.1, 0.3)
	} else if insights.SuccessRate < 0.6 {
		le.metrics.LearningRate = math.Max(le.metrics.LearningRate*0.9, 0.05)
	}

	// Emit adaptation complete event
	if le.agent.GetEventBus() != nil {
		le.agent.PublishEvent(ctx, EventType("learning.adaptation.completed"), EventData(
			"adaptation_mode", le.adaptationMode,
			"learning_rate", le.metrics.LearningRate,
		))
	}

	return nil
}

// SetLearningStrategy implements LearningEngine.SetLearningStrategy
func (le *basicLearningEngine) SetLearningStrategy(strategy LearningStrategy) error {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.learningStrategy = strategy
	return nil
}

// SetAdaptationMode implements LearningEngine.SetAdaptationMode
func (le *basicLearningEngine) SetAdaptationMode(mode AdaptationMode) error {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.adaptationMode = mode
	return nil
}

// GetLearningMetrics implements LearningEngine.GetLearningMetrics
func (le *basicLearningEngine) GetLearningMetrics() *LearningMetrics {
	le.mu.RLock()
	defer le.mu.RUnlock()

	// Return a copy to prevent external modification
	metrics := *le.metrics
	return &metrics
}

// Helper functions

// generatePatternID generates a unique ID for a pattern
func generatePatternID() string {
	return fmt.Sprintf("pattern_%d", time.Now().UnixNano())
}

// NewExperience creates a new experience record
func NewExperience(expType ExperienceType, action string, success bool, performance float64) *Experience {
	return &Experience{
		ID:          generateExperienceID(),
		Type:        expType,
		Action:      action,
		Success:     success,
		Performance: performance,
		Context:     make(map[string]interface{}),
		Parameters:  make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
		Timestamp:   time.Now(),
	}
}
