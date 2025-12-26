package guardrail

import (
	"context"
	"regexp"
	"strings"

	"github.com/voocel/mas/schema"
)

// --- Reusable validation logic ---

func keywordCheck(keywords []string, reason string) ValidateFunc {
	lower := make([]string, len(keywords))
	for i, k := range keywords {
		lower[i] = strings.ToLower(k)
	}
	return func(_ context.Context, msg *schema.Message) Result {
		content := strings.ToLower(msg.Content)
		for _, kw := range lower {
			if strings.Contains(content, kw) {
				return Block(reason)
			}
		}
		return Pass()
	}
}

func patternCheck(patterns []string, reason string) (ValidateFunc, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, re)
	}
	return func(_ context.Context, msg *schema.Message) Result {
		for _, re := range compiled {
			if re.MatchString(msg.Content) {
				return Block(reason)
			}
		}
		return Pass()
	}, nil
}

func lengthCheck(maxLength int) ValidateFunc {
	return func(_ context.Context, msg *schema.Message) Result {
		if len(msg.Content) > maxLength {
			return BlockWithDetails("content exceeds maximum length", map[string]interface{}{
				"max_length":    maxLength,
				"actual_length": len(msg.Content),
			})
		}
		return Pass()
	}
}

// --- Input Guardrails ---

// KeywordBlocker creates an input guardrail that blocks content containing any keyword.
func KeywordBlocker(name string, keywords []string, reason string) InputGuardrail {
	return NewInputGuardrail(name, keywordCheck(keywords, reason))
}

// ContentFilter creates an input guardrail that blocks content matching any pattern.
func ContentFilter(name string, patterns []string, reason string) (InputGuardrail, error) {
	fn, err := patternCheck(patterns, reason)
	if err != nil {
		return nil, err
	}
	return NewInputGuardrail(name, fn), nil
}

// LengthLimit creates an input guardrail that limits content length.
func LengthLimit(name string, maxLength int) InputGuardrail {
	return NewInputGuardrail(name, lengthCheck(maxLength))
}

// --- Output Guardrails ---

// OutputKeywordBlocker creates an output guardrail that blocks content containing any keyword.
func OutputKeywordBlocker(name string, keywords []string, reason string) OutputGuardrail {
	return NewOutputGuardrail(name, keywordCheck(keywords, reason))
}

// OutputContentFilter creates an output guardrail that blocks content matching any pattern.
func OutputContentFilter(name string, patterns []string, reason string) (OutputGuardrail, error) {
	fn, err := patternCheck(patterns, reason)
	if err != nil {
		return nil, err
	}
	return NewOutputGuardrail(name, fn), nil
}

// OutputLengthLimit creates an output guardrail that limits content length.
func OutputLengthLimit(name string, maxLength int) OutputGuardrail {
	return NewOutputGuardrail(name, lengthCheck(maxLength))
}
