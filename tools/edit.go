package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EditTool performs exact string replacement in a file.
// Supports line ending normalization, fuzzy matching, and returns unified diff.
type EditTool struct{}

func NewEdit() *EditTool { return &EditTool{} }

func (t *EditTool) Name() string  { return "edit" }
func (t *EditTool) Label() string { return "Edit File" }
func (t *EditTool) Description() string {
	return "Edit a file by replacing exact text. The oldText must match exactly (including whitespace). Use this for precise, surgical edits."
}
func (t *EditTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to edit (relative or absolute)",
			},
			"old_text": map[string]any{
				"type":        "string",
				"description": "Exact text to find and replace (must be unique in the file)",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "New text to replace the old text with",
			},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}

type editArgs struct {
	Path    string `json:"path"`
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

func (t *EditTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a editArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}

	data, err := os.ReadFile(a.Path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", a.Path)
	}

	raw := string(data)

	// Strip BOM (LLM won't include invisible BOM in oldText)
	bom := ""
	if strings.HasPrefix(raw, "\uFEFF") {
		bom = "\uFEFF"
		raw = raw[len("\uFEFF"):]
	}

	// Detect and normalize line endings
	originalEnding := detectLineEnding(raw)
	content := normalizeToLF(raw)
	oldText := normalizeToLF(a.OldText)
	newText := normalizeToLF(a.NewText)

	// Try exact match first, then fuzzy match
	matchIdx, matchLen, baseContent := fuzzyFind(content, oldText)
	if matchIdx < 0 {
		return nil, fmt.Errorf("could not find the exact text in %s. The old text must match exactly including all whitespace and newlines", a.Path)
	}

	// Check uniqueness using fuzzy-normalized content
	fuzzyContent := normalizeForFuzzy(content)
	fuzzyOld := normalizeForFuzzy(oldText)
	if count := strings.Count(fuzzyContent, fuzzyOld); count > 1 {
		return nil, fmt.Errorf("found %d occurrences of the text in %s. The text must be unique. Provide more context", count, a.Path)
	}

	// Perform replacement
	newContent := baseContent[:matchIdx] + newText + baseContent[matchIdx+matchLen:]
	if baseContent == newContent {
		return nil, fmt.Errorf("no changes made to %s. The replacement produced identical content", a.Path)
	}

	// Restore original line endings and BOM
	finalContent := bom + restoreLineEndings(newContent, originalEnding)
	if err := os.WriteFile(a.Path, []byte(finalContent), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", a.Path, err)
	}

	// Generate unified diff
	diff := generateDiff(baseContent, newContent)

	return json.Marshal(map[string]any{
		"message": fmt.Sprintf("Successfully replaced text in %s.", a.Path),
		"diff":    diff,
	})
}

// --- Line ending utilities ---

func detectLineEnding(content string) string {
	crlfIdx := strings.Index(content, "\r\n")
	lfIdx := strings.Index(content, "\n")
	if lfIdx == -1 || crlfIdx == -1 {
		return "\n"
	}
	if crlfIdx < lfIdx {
		return "\r\n"
	}
	return "\n"
}

func normalizeToLF(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func restoreLineEndings(text, ending string) string {
	if ending == "\r\n" {
		return strings.ReplaceAll(text, "\n", "\r\n")
	}
	return text
}

// --- Fuzzy matching (matching pi's edit-diff.ts) ---

// normalizeForFuzzy strips trailing whitespace per line, normalizes smart quotes,
// Unicode dashes, and special spaces to ASCII equivalents.
func normalizeForFuzzy(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	text = strings.Join(lines, "\n")

	// Smart quotes → ASCII
	for _, r := range []rune{'\u2018', '\u2019', '\u201A', '\u201B'} {
		text = strings.ReplaceAll(text, string(r), "'")
	}
	for _, r := range []rune{'\u201C', '\u201D', '\u201E', '\u201F'} {
		text = strings.ReplaceAll(text, string(r), "\"")
	}

	// Unicode dashes → ASCII hyphen
	for _, r := range []rune{'\u2010', '\u2011', '\u2012', '\u2013', '\u2014', '\u2015', '\u2212'} {
		text = strings.ReplaceAll(text, string(r), "-")
	}

	// Special spaces → regular space
	for _, r := range []rune{'\u00A0', '\u2002', '\u2003', '\u2004', '\u2005', '\u2006', '\u2007', '\u2008', '\u2009', '\u200A', '\u202F', '\u205F', '\u3000'} {
		text = strings.ReplaceAll(text, string(r), " ")
	}

	return text
}

// fuzzyFind tries exact match first, then fuzzy match.
// Returns match index, match length, and the base content to use for replacement.
func fuzzyFind(content, oldText string) (index, matchLen int, baseContent string) {
	// Try exact match
	idx := strings.Index(content, oldText)
	if idx >= 0 {
		return idx, len(oldText), content
	}

	// Try fuzzy match
	fuzzyContent := normalizeForFuzzy(content)
	fuzzyOld := normalizeForFuzzy(oldText)
	idx = strings.Index(fuzzyContent, fuzzyOld)
	if idx >= 0 {
		return idx, len(fuzzyOld), fuzzyContent
	}

	return -1, 0, content
}

// --- Diff generation (matching pi's generateDiffString) ---

// generateDiff produces a unified diff with line numbers and context.
func generateDiff(oldContent, newContent string) string {
	const contextLines = 4

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Find the first and last differing lines
	maxOld := len(oldLines)
	maxNew := len(newLines)

	// Find common prefix
	prefix := 0
	for prefix < maxOld && prefix < maxNew && oldLines[prefix] == newLines[prefix] {
		prefix++
	}

	// Find common suffix (from the end, not overlapping prefix)
	suffixOld := maxOld - 1
	suffixNew := maxNew - 1
	for suffixOld > prefix && suffixNew > prefix && oldLines[suffixOld] == newLines[suffixNew] {
		suffixOld--
		suffixNew--
	}

	if prefix > suffixOld+1 && prefix > suffixNew+1 {
		return "(no changes)"
	}

	// Build diff output with context
	maxLineNum := max(maxOld, maxNew)
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	var sb strings.Builder

	// Leading context
	ctxStart := max(prefix-contextLines, 0)
	if ctxStart < prefix {
		if ctxStart > 0 {
			fmt.Fprintf(&sb, " %*s ...\n", lineNumWidth, "")
		}
		for i := ctxStart; i < prefix; i++ {
			fmt.Fprintf(&sb, " %*d %s\n", lineNumWidth, i+1, oldLines[i])
		}
	}

	// Removed lines
	for i := prefix; i <= suffixOld; i++ {
		fmt.Fprintf(&sb, "-%*d %s\n", lineNumWidth, i+1, oldLines[i])
	}

	// Added lines
	for i := prefix; i <= suffixNew; i++ {
		fmt.Fprintf(&sb, "+%*d %s\n", lineNumWidth, i+1, newLines[i])
	}

	// Trailing context
	trailStart := suffixOld + 1
	trailEnd := min(trailStart+contextLines, maxOld)
	for i := trailStart; i < trailEnd; i++ {
		fmt.Fprintf(&sb, " %*d %s\n", lineNumWidth, i+1, oldLines[i])
	}
	if trailEnd < maxOld {
		fmt.Fprintf(&sb, " %*s ...\n", lineNumWidth, "")
	}

	return sb.String()
}
