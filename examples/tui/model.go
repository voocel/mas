package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/voocel/mas"
)

// blockKind identifies the type of a rendered content block.
type blockKind int

const (
	blockUser blockKind = iota
	blockAssistant
	blockToolStart
	blockToolEnd
	blockError
)

// block is an immutable rendered content entry in the conversation history.
type block struct {
	kind    blockKind
	content string // pre-rendered string
}

// model is the bubbletea Model for the TUI.
type model struct {
	// Agent
	agent     *mas.Agent
	modelName string

	// Bubbles components
	viewport viewport.Model
	input    textarea.Model
	spinner  spinner.Model

	// Conversation content
	blocks    []block          // completed rendered blocks
	streaming strings.Builder  // current streaming assistant content
	isStream  bool             // true between MessageStart and MessageEnd

	// Agent state tracking
	running      bool
	turnCount    int
	pendingTools map[string]string // toolID -> toolName

	// Layout
	width, height int
	ready         bool // set after first WindowSizeMsg
	autoScroll    bool

	// Rendering
	glamour *glamour.TermRenderer
}

func newModel(agent *mas.Agent, modelName string) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorAssistant)

	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to send, Esc to abort)"
	ta.Focus()
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.CharLimit = 0

	return model{
		agent:        agent,
		modelName:    modelName,
		spinner:      sp,
		input:        ta,
		pendingTools: make(map[string]string),
		autoScroll:   true,
	}
}

// --- tea.Model interface ---

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textarea.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case agentEventMsg:
		return m.handleAgentEvent(msg.event)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Forward to viewport for mouse wheel scrolling
	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		// Detect manual scroll: disable auto-scroll when user scrolls up
		if !m.viewport.AtBottom() {
			m.autoScroll = false
		}
	}

	// Forward to textarea for cursor blink etc.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	statusBar := m.renderStatusBar()
	return fmt.Sprintf("%s\n%s\n%s", statusBar, m.viewport.View(), m.input.View())
}

// --- Key handling ---

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		if m.running {
			m.agent.Abort()
		}
		return m, nil

	case "enter":
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}
		m.input.Reset()

		if m.running {
			// Steer: inject message while agent is running
			m.agent.Steer(mas.UserMsg(text))
		} else {
			// New prompt
			m.blocks = append(m.blocks, block{
				kind:    blockUser,
				content: userPrefixStyle.Render("> You") + "\n" + text,
			})
			m.autoScroll = true
			m.rebuildViewport()
			_ = m.agent.Prompt(text)
		}
		return m, nil

	case "pgup", "pgdown":
		// Forward to viewport for page scrolling
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		if m.viewport.AtBottom() {
			m.autoScroll = true
		} else {
			m.autoScroll = false
		}
		return m, cmd
	}

	// Forward other keys to textarea
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// --- Window resize ---

func (m model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	statusHeight := 1
	inputHeight := 5 // 3 lines + border
	vpHeight := m.height - statusHeight - inputHeight
	if vpHeight < 1 {
		vpHeight = 1
	}

	if !m.ready {
		m.viewport = viewport.New(m.width, vpHeight)
		m.viewport.MouseWheelEnabled = true
		m.viewport.MouseWheelDelta = 3
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = vpHeight
	}

	m.input.SetWidth(m.width - 2) // account for border
	m.glamour = newGlamourRenderer(m.width - 4)
	m.rebuildViewport()

	return m, nil
}

// --- Agent event handling ---

func (m model) handleAgentEvent(ev mas.Event) (tea.Model, tea.Cmd) {
	switch ev.Type {

	case mas.EventAgentStart:
		m.running = true

	case mas.EventAgentEnd:
		m.running = false

	case mas.EventTurnStart:
		m.turnCount++

	case mas.EventMessageStart:
		if role := msgRole(ev.Message); role == mas.RoleAssistant {
			m.isStream = true
			m.streaming.Reset()
		} else if role == mas.RoleUser {
			// Steer message echoed back
			if msg, ok := ev.Message.(mas.Message); ok && !msg.IsEmpty() {
				m.blocks = append(m.blocks, block{
					kind:    blockUser,
					content: userPrefixStyle.Render("> You") + "\n" + msg.TextContent(),
				})
			}
		}

	case mas.EventMessageUpdate:
		if m.isStream {
			m.streaming.WriteString(ev.Delta)
			m.rebuildViewport()
		}

	case mas.EventMessageEnd:
		if role := msgRole(ev.Message); role == mas.RoleAssistant {
			m.isStream = false
			content := m.streaming.String()
			m.streaming.Reset()

			rendered := m.renderMarkdown(content)
			m.blocks = append(m.blocks, block{
				kind:    blockAssistant,
				content: assistantPrefixStyle.Render("> Assistant") + "\n" + rendered,
			})
			m.rebuildViewport()
		}

	case mas.EventToolExecStart:
		name := ev.Tool
		m.pendingTools[ev.ToolID] = name

		argsStr := formatToolArgs(ev.Args)
		m.blocks = append(m.blocks, block{
			kind:    blockToolStart,
			content: toolNameStyle.Render("  ⚙ "+name) + " " + mutedStyle.Render(argsStr),
		})
		m.rebuildViewport()

	case mas.EventToolExecEnd:
		delete(m.pendingTools, ev.ToolID)

		resultStr := formatToolResult(ev.Result, ev.IsError)
		m.blocks = append(m.blocks, block{
			kind:    blockToolEnd,
			content: toolResultStyle.Render("  → " + resultStr),
		})
		m.rebuildViewport()

	case mas.EventError:
		errMsg := "unknown error"
		if ev.Err != nil {
			errMsg = ev.Err.Error()
		}
		m.blocks = append(m.blocks, block{
			kind:    blockError,
			content: errorStyle.Render("  ✗ Error: " + errMsg),
		})
		m.rebuildViewport()
	}

	return m, nil
}

// --- Rendering helpers ---

func (m *model) rebuildViewport() {
	var sb strings.Builder

	for _, b := range m.blocks {
		sb.WriteString(b.content)
		sb.WriteString("\n\n")
	}

	// Streaming content (raw text + spinner)
	if m.isStream {
		sb.WriteString(assistantPrefixStyle.Render("> Assistant"))
		sb.WriteString("\n")
		sb.WriteString(streamingStyle.Render(m.streaming.String()))
		sb.WriteString(m.spinner.View())
		sb.WriteString("\n\n")
	}

	// Pending tools spinner
	if len(m.pendingTools) > 0 && !m.isStream {
		for _, name := range m.pendingTools {
			sb.WriteString(m.spinner.View())
			sb.WriteString(" ")
			sb.WriteString(toolNameStyle.Render(name))
			sb.WriteString(" running...\n")
		}
		sb.WriteString("\n")
	}

	m.viewport.SetContent(sb.String())

	if m.autoScroll {
		m.viewport.GotoBottom()
	}
}

func (m *model) renderStatusBar() string {
	var status string
	if m.running {
		status = m.spinner.View() + " Thinking..."
	} else {
		status = "● Ready"
	}

	right := fmt.Sprintf("%s  Turn %d", m.modelName, m.turnCount)
	gap := m.width - lipgloss.Width(status) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	bar := status + strings.Repeat(" ", gap) + right
	return statusBarStyle.Width(m.width).Render(bar)
}

func (m *model) renderMarkdown(content string) string {
	if m.glamour == nil || content == "" {
		return content
	}
	rendered, err := m.glamour.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(rendered)
}

// --- Utility functions ---

func msgRole(msg mas.AgentMessage) mas.Role {
	if msg == nil {
		return ""
	}
	return msg.GetRole()
}

func formatToolArgs(args any) string {
	if args == nil {
		return ""
	}
	switch v := args.(type) {
	case []byte:
		s := string(v)
		if len(s) > 80 {
			s = s[:77] + "..."
		}
		return s
	case string:
		if len(v) > 80 {
			v = v[:77] + "..."
		}
		return v
	default:
		s := fmt.Sprintf("%v", v)
		if len(s) > 80 {
			s = s[:77] + "..."
		}
		return s
	}
}

func formatToolResult(result any, isError bool) string {
	prefix := ""
	if isError {
		prefix = "error: "
	}

	if result == nil {
		return prefix + "(no output)"
	}

	var s string
	switch v := result.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		s = fmt.Sprintf("%v", v)
	}

	s = strings.TrimSpace(s)
	// Truncate for display
	lines := strings.SplitN(s, "\n", 6)
	if len(lines) > 5 {
		lines = lines[:5]
		s = strings.Join(lines, "\n") + "\n..."
	}
	if len(s) > 200 {
		s = s[:197] + "..."
	}

	return prefix + s
}
