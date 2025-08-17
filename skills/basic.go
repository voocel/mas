package skills

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/voocel/mas"
)

// MathSkill provides mathematical calculation capabilities
func MathSkill() mas.Skill {
	return mas.NewSkill(
		"math_calculation",
		"Perform mathematical calculations and analysis",
		mas.CerebellumLayer,
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			expression, ok := params["expression"].(string)
			if !ok {
				return nil, fmt.Errorf("expression parameter required")
			}

			// Simple math evaluation
			result, err := evaluateBasicMath(expression)
			if err != nil {
				return nil, fmt.Errorf("math evaluation failed: %w", err)
			}

			return map[string]interface{}{
				"expression": expression,
				"result":     result,
				"type":       "math_calculation",
			}, nil
		},
	)
}

// TextAnalysisSkill provides text analysis capabilities
func TextAnalysisSkill() mas.Skill {
	return mas.NewSkill(
		"text_analysis",
		"Analyze text for sentiment, keywords, and structure",
		mas.CortexLayer,
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			text, ok := params["text"].(string)
			if !ok {
				return nil, fmt.Errorf("text parameter required")
			}

			analysis := analyzeText(text)
			return analysis, nil
		},
	)
}

// QuickResponseSkill provides immediate reactive responses
func QuickResponseSkill() mas.Skill {
	return mas.NewSkill(
		"quick_response",
		"Provide immediate responses to urgent situations",
		mas.ReflexLayer,
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			trigger, ok := params["trigger"].(string)
			if !ok {
				return nil, fmt.Errorf("trigger parameter required")
			}

			response := generateQuickResponse(trigger)
			return map[string]interface{}{
				"trigger":  trigger,
				"response": response,
				"type":     "quick_response",
			}, nil
		},
	)
}

// PlanningSkill provides high-level planning capabilities
func PlanningSkill() mas.Skill {
	return mas.NewSkill(
		"task_planning",
		"Break down complex tasks into manageable steps",
		mas.MetaLayer,
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			task, ok := params["task"].(string)
			if !ok {
				return nil, fmt.Errorf("task parameter required")
			}

			plan := createTaskPlan(task)
			return plan, nil
		},
	)
}

// Helper functions

func evaluateBasicMath(expression string) (float64, error) {
	// Remove spaces
	expression = strings.ReplaceAll(expression, " ", "")

	// Handle basic operations
	if strings.Contains(expression, "+") {
		parts := strings.Split(expression, "+")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(parts[0], 64)
			b, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil {
				return a + b, nil
			}
		}
	}

	if strings.Contains(expression, "-") {
		parts := strings.Split(expression, "-")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(parts[0], 64)
			b, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil {
				return a - b, nil
			}
		}
	}

	if strings.Contains(expression, "*") {
		parts := strings.Split(expression, "*")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(parts[0], 64)
			b, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil {
				return a * b, nil
			}
		}
	}

	if strings.Contains(expression, "/") {
		parts := strings.Split(expression, "/")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(parts[0], 64)
			b, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil && b != 0 {
				return a / b, nil
			}
		}
	}

	// Handle functions
	if strings.HasPrefix(expression, "sqrt(") && strings.HasSuffix(expression, ")") {
		numStr := expression[5 : len(expression)-1]
		num, err := strconv.ParseFloat(numStr, 64)
		if err == nil && num >= 0 {
			return math.Sqrt(num), nil
		}
	}

	// Try to parse as simple number
	return strconv.ParseFloat(expression, 64)
}

func analyzeText(text string) map[string]interface{} {
	words := strings.Fields(text)
	sentences := strings.Split(text, ".")

	// Simple sentiment analysis
	sentiment := "neutral"
	positiveWords := []string{"good", "great", "excellent", "amazing", "wonderful", "happy", "positive"}
	negativeWords := []string{"bad", "terrible", "awful", "horrible", "sad", "negative", "hate"}

	textLower := strings.ToLower(text)
	positiveCount := 0
	negativeCount := 0

	for _, word := range positiveWords {
		if strings.Contains(textLower, word) {
			positiveCount++
		}
	}

	for _, word := range negativeWords {
		if strings.Contains(textLower, word) {
			negativeCount++
		}
	}

	if positiveCount > negativeCount {
		sentiment = "positive"
	} else if negativeCount > positiveCount {
		sentiment = "negative"
	}

	// Extract keywords (simple version)
	keywords := make([]string, 0)
	for _, word := range words {
		if len(word) > 4 && !isCommonWord(strings.ToLower(word)) {
			keywords = append(keywords, word)
		}
	}

	// Limit keywords
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}

	return map[string]interface{}{
		"word_count":     len(words),
		"sentence_count": len(sentences) - 1, // -1 because split adds empty string at end
		"sentiment":      sentiment,
		"keywords":       keywords,
		"readability":    calculateReadability(words, sentences),
	}
}

func generateQuickResponse(trigger string) string {
	triggerLower := strings.ToLower(trigger)

	if strings.Contains(triggerLower, "emergency") || strings.Contains(triggerLower, "urgent") {
		return "Immediate attention required. Escalating to priority handling."
	} else if strings.Contains(triggerLower, "error") || strings.Contains(triggerLower, "fail") {
		return "Error detected. Initiating diagnostic procedures."
	} else if strings.Contains(triggerLower, "help") || strings.Contains(triggerLower, "assist") {
		return "Assistance requested. Activating support protocols."
	} else {
		return "Acknowledged. Processing request with standard priority."
	}
}

func createTaskPlan(task string) map[string]interface{} {
	steps := []string{}
	taskLower := strings.ToLower(task)

	if strings.Contains(taskLower, "analysis") || strings.Contains(taskLower, "analyze") {
		steps = []string{
			"Gather relevant data and information",
			"Apply analytical frameworks and methods",
			"Identify patterns and insights",
			"Draw conclusions and recommendations",
			"Prepare comprehensive report",
		}
	} else if strings.Contains(taskLower, "research") {
		steps = []string{
			"Define research scope and objectives",
			"Identify reliable sources and methods",
			"Collect and organize data",
			"Analyze findings and trends",
			"Synthesize results into actionable insights",
		}
	} else if strings.Contains(taskLower, "create") || strings.Contains(taskLower, "build") {
		steps = []string{
			"Define requirements and specifications",
			"Design architecture and structure",
			"Implement core functionality",
			"Test and validate results",
			"Refine and optimize final output",
		}
	} else {
		steps = []string{
			"Break down task into smaller components",
			"Prioritize components by importance",
			"Execute each component systematically",
			"Review and integrate results",
			"Validate completion against objectives",
		}
	}

	return map[string]interface{}{
		"task":           task,
		"steps":          steps,
		"estimated_time": len(steps) * 15, // 15 minutes per step
		"complexity":     determineComplexity(task),
	}
}

func isCommonWord(word string) bool {
	commonWords := []string{"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by", "from", "up", "about", "into", "through", "during", "before", "after", "above", "below", "between", "among", "within", "without", "under", "over"}
	for _, common := range commonWords {
		if word == common {
			return true
		}
	}
	return false
}

func calculateReadability(words, sentences []string) string {
	if len(sentences) == 0 {
		return "unknown"
	}

	avgWordsPerSentence := float64(len(words)) / float64(len(sentences)-1)

	if avgWordsPerSentence < 10 {
		return "easy"
	} else if avgWordsPerSentence < 20 {
		return "medium"
	} else {
		return "complex"
	}
}

func determineComplexity(task string) string {
	taskLower := strings.ToLower(task)

	complexKeywords := []string{"comprehensive", "detailed", "complex", "advanced", "sophisticated", "intricate"}
	simpleKeywords := []string{"simple", "basic", "quick", "easy", "straightforward"}

	for _, keyword := range complexKeywords {
		if strings.Contains(taskLower, keyword) {
			return "high"
		}
	}

	for _, keyword := range simpleKeywords {
		if strings.Contains(taskLower, keyword) {
			return "low"
		}
	}

	return "medium"
}
