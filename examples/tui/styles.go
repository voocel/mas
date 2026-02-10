package main

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// Colors
var (
	colorPrimary   = lipgloss.Color("63")
	colorUser      = lipgloss.Color("39")
	colorAssistant = lipgloss.Color("213")
	colorTool      = lipgloss.Color("214")
	colorError     = lipgloss.Color("196")
	colorMuted = lipgloss.Color("241")
)

// Styles
var (
	statusBarStyle = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1).
			Bold(true)

	userPrefixStyle = lipgloss.NewStyle().
			Foreground(colorUser).
			Bold(true)

	assistantPrefixStyle = lipgloss.NewStyle().
				Foreground(colorAssistant).
				Bold(true)

	toolNameStyle = lipgloss.NewStyle().
			Foreground(colorTool).
			Bold(true)

	toolResultStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	streamingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)

// newGlamourRenderer creates a glamour markdown renderer with the given width.
func newGlamourRenderer(width int) *glamour.TermRenderer {
	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	return r
}
