package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

	"github.com/ekrishgupta/HushApp/internal/chat"
)

const (
	spamCooldown   = 1500 * time.Millisecond
	MaxMessageSize = 512
)

// ── Bubble Tea messages ─────────────────────────────

// IncomingMsg is a Bubble Tea message wrapping a chat message from the network.
type IncomingMsg chat.ChatMessage

type tickMsg time.Time

// ── ASCII banner ────────────────────────────────────

var hushASCII = `
 ██╗  ██╗██╗   ██╗███████╗██╗  ██╗
 ██║  ██║██║   ██║██╔════╝██║  ██║
 ███████║██║   ██║███████╗███████║
 ██╔══██║██║   ██║╚════██║██╔══██║
 ██║  ██║╚██████╔╝███████║██║  ██║
 ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝`

// ── Model ───────────────────────────────────────────

// Model is the Bubble Tea model for the chat TUI.
type Model struct {
	screen   string // "welcome" or "chat"
	username string
	chat     *chat.Chat
	msgChan  <-chan chat.ChatMessage

	messages []chat.ChatMessage
	viewport viewport.Model
	input    textinput.Model // For welcome screen
	textArea textarea.Model  // For chat screen

	lastSent    time.Time
	showWarning bool
	warningMsg  string

	width  int
	height int
	ready  bool

	renderer        *glamour.TermRenderer
	compactRenderer *glamour.TermRenderer

	peerCount int

	// Navigation & Truncation
	expanded    map[int]bool // map[messageIndex]bool
	selectedMsg int          // index of selected message, -1 if none (input focused)
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// NewModel creates a new chat TUI model.
func NewModel(username string, c *chat.Chat, msgChan <-chan chat.ChatMessage) Model {
	// Welcome Screen Input
	ti := textinput.New()
	ti.Placeholder = "enter your name..."
	ti.Focus()
	ti.CharLimit = 30
	ti.Width = 40

	// Chat Screen Input
	ta := textarea.New()
	ta.Placeholder = "type a message..."
	ta.CharLimit = MaxMessageSize
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.SetWidth(60) // Will be updated on resize
	ta.Prompt = "> "

	// Remove default cursor line highlight (dark background)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	screen := "welcome"
	if username != "" {
		screen = "chat"
		ti.Blur()
		ta.Focus()
	}

	return Model{
		screen:      screen,
		username:    username,
		chat:        c,
		msgChan:     msgChan,
		input:       ti,
		textArea:    ta,
		messages:    []chat.ChatMessage{},
		expanded:    make(map[int]bool),
		selectedMsg: -1,
		// internal/ui/model.go
		// We initialize renderer later on resize or here with default
		// Actually best to init here with safe default
	}
}

// waitForMsg returns a command that waits for the next network message.
func (m Model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		if m.msgChan == nil {
			return nil
		}
		msg, ok := <-m.msgChan
		if !ok {
			return nil
		}
		return IncomingMsg(msg)
	}
}

// Init starts listening for network messages.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}
	if m.msgChan != nil {
		cmds = append(cmds, m.waitForMsg(), tick())
	}
	return tea.Batch(cmds...)
}

// ── Update ──────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		if m.chat != nil {
			m.peerCount = m.chat.PeerCount()
		}
		cmds = append(cmds, tick())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update glamour renderer with new width
		// Subtract some padding for aesthetics
		wrapWidth := m.width - 10
		if wrapWidth < 20 {
			wrapWidth = 20
		}
		style := styles.DraculaStyleConfig
		var zero uint = 0
		style.Document.Margin = &zero
		// Keep padding/indent for structured content? Maybe set to 0 too for compact.
		// But expanded view needs indentation?
		// Since we use the same renderer, let's just make it compact.
		// Expanded view handles indentation manually anyway.
		m.renderer, _ = glamour.NewTermRenderer(
			glamour.WithStyles(style),
			glamour.WithWordWrap(wrapWidth),
		)

		// Compact renderer: No wrap, no margin
		styleCompact := styles.DraculaStyleConfig
		styleCompact.Document.Margin = &zero
		m.compactRenderer, _ = glamour.NewTermRenderer(
			glamour.WithStyles(styleCompact),
			glamour.WithWordWrap(0),
		)

		if m.screen == "chat" {
			headerH := 3
			warnH := 1

			inputLines := m.textArea.LineCount()
			if inputLines < 1 {
				inputLines = 1
			}
			if inputLines > 5 {
				inputLines = 5
			}
			inputH := inputLines + 2

			vpHeight := m.height - headerH - inputH - warnH - 1
			if vpHeight < 0 {
				vpHeight = 0
			}

			if !m.ready {
				m.viewport = viewport.New(m.width, vpHeight)
				m.viewport.SetContent(m.renderMessages())
				m.ready = true
			} else {
				m.viewport.Width = m.width
				m.viewport.Height = vpHeight
				// Re-render specifically to handle dynamic width changes like truncation points
				m.viewport.SetContent(m.renderMessages())
			}
			m.textArea.SetWidth(m.width - 6)
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.selectedMsg != -1 {
				m.selectedMsg = -1
				m.viewport.GotoBottom()
				return m, nil
			}
			return m, tea.Quit

		case tea.KeyUp, tea.KeyDown:
			if m.screen == "chat" && len(m.messages) > 0 {
				if msg.Type == tea.KeyUp {
					if m.selectedMsg == -1 {
						// Select last message
						m.selectedMsg = len(m.messages) - 1
					} else if m.selectedMsg > 0 {
						m.selectedMsg--
					}
				} else { // KeyDown
					if m.selectedMsg != -1 {
						if m.selectedMsg < len(m.messages)-1 {
							m.selectedMsg++
						} else {
							// Deselect, return to input
							m.selectedMsg = -1
							m.viewport.GotoBottom()
						}
					}
				}
				// Re-render to show selection
				m.viewport.SetContent(m.renderMessages())
				return m, nil
			}

		case tea.KeyEnter:
			if m.screen == "welcome" {
				return m.handleWelcomeEnter()
			}
			// If a message is selected, ignore Enter (disable expansion)
			if m.selectedMsg != -1 {
				return m, nil
			}
			return m.handleChatEnter()

		default:
			m.showWarning = false
		}

	case IncomingMsg:
		m.messages = append(m.messages, chat.ChatMessage(msg))
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		cmds = append(cmds, m.waitForMsg())
	}

	// Update sub-components
	var cmd tea.Cmd

	if m.screen == "welcome" {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		// Update TextArea
		var taCmd tea.Cmd
		m.textArea, taCmd = m.textArea.Update(msg)
		cmds = append(cmds, taCmd)

		// Remove placeholder permanently once user types
		if m.textArea.Value() != "" {
			m.textArea.Placeholder = ""
		}

		// Dynamic Resize Logic
		lines := m.textArea.LineCount()

		// Adjust prompt based on multiline state
		if lines > 1 {
			m.textArea.Prompt = "  "
		} else {
			m.textArea.Prompt = "> "
		}

		if lines < 1 {
			lines = 1
		}
		if lines > 5 {
			lines = 5
		} // Cap expansion

		if lines != m.textArea.Height() {
			m.textArea.SetHeight(lines)

			// Recalculate viewport height
			headerH := 3
			warnH := 1
			inputH := lines + 2
			vpHeight := m.height - headerH - inputH - warnH - 1
			if vpHeight < 0 {
				vpHeight = 0
			}

			m.viewport.Height = vpHeight
		}

		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleWelcomeEnter() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		name = fmt.Sprintf("Ghost-%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(900)+100)
	}

	m.username = name
	m.screen = "chat"
	m.ready = false

	// Switch to Chat TextArea
	m.input.Blur()
	m.input.Reset()

	m.textArea.SetValue("")
	m.textArea.Focus()
	m.textArea.SetHeight(1)
	m.textArea.SetWidth(m.width - 6)

	var cmds []tea.Cmd
	cmds = append(cmds, tick())

	// Start listening now
	if m.chat != nil {
		cmds = append(cmds, m.waitForMsg())
	}

	// Force a WindowSize re-calc by sending the current size
	if m.width > 0 {
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		})
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleChatEnter() (tea.Model, tea.Cmd) {
	if m.chat == nil {
		return m, nil
	}
	content := strings.TrimSpace(m.textArea.Value())
	if content == "" {
		return m, nil
	}

	if time.Since(m.lastSent) < spamCooldown {
		m.showWarning = true
		m.warningMsg = "⚡ Slow down!"
		return m, nil
	}

	m.showWarning = false
	if err := m.chat.Publish(m.username, content); err == nil {
		ownMsg := chat.NewChatMessage(m.username, content)
		m.messages = append(m.messages, ownMsg)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		m.lastSent = time.Now()
	}
	m.textArea.Reset()
	m.textArea.SetHeight(1)

	// Recalculate viewport height immediately
	headerH := 3
	warnH := 1
	inputH := 3 // 1 line + 2 border
	vpHeight := m.height - headerH - inputH - warnH - 1
	if vpHeight < 0 {
		vpHeight = 0
	}
	m.viewport.Height = vpHeight

	return m, nil
}

// ── Render: Messages ────────────────────────────────

func (m Model) renderMessages() string {
	if len(m.messages) == 0 {
		return StatusStyle.Render("\n  waiting for ghosts to appear... 👻\n")
	}

	var b strings.Builder
	for i, msg := range m.messages {
		tsRaw := msg.Time().Format("15:04:05")
		ts := TimestampStyle.Render(tsRaw)

		// Determine sender label
		var senderLabel string
		if msg.Sender == m.username {
			senderLabel = "you"
		} else {
			senderLabel = msg.Sender
		}

		isSelected := (i == m.selectedMsg)
		isExpanded := m.expanded[i]

		// Margin/Cursor
		// Default margin is 2 spaces. If selected, use "> ".
		var margin string
		if isSelected {
			margin = lipgloss.NewStyle().Foreground(ghostPink).Render("> ")
		} else {
			margin = "  "
		}

		// Prepare styles
		var (
			rawContent    = msg.Content
			styledSender  string
			styledContent string
		)

		if msg.Sender == m.username {
			styledSender = SelfMsgSender.Render(senderLabel)
		} else {
			styledSender = PeerMsgSender.Render(senderLabel)
		}

		var lines string

		if !isExpanded {
			// Compact view: Single line with truncation
			// Render Markdown first to get ANSI
			var rendered string
			if m.compactRenderer != nil {
				// Use compact renderer
				out, err := m.compactRenderer.Render(rawContent)
				if err == nil {
					rendered = strings.TrimSpace(out)
				} else {
					rendered = rawContent
				}
			} else {
				rendered = rawContent
			}

			// Extract first line
			if idx := strings.Index(rendered, "\n"); idx != -1 {
				rendered = rendered[:idx]
			}

			// Calculate space
			prefixWidth := lipgloss.Width(margin) + lipgloss.Width(senderLabel) + 2
			suffixWidth := 3 + lipgloss.Width(tsRaw)
			availableWidth := m.width - prefixWidth - suffixWidth
			if availableWidth < 10 {
				availableWidth = 10
			}

			// Prepare ellipsis
			var tail string
			if isSelected {
				tail = SelectedMsgStyle.Render(" (...)")
			} else {
				tail = " (...)"
			}

			// Truncate ANSI-aware
			var displayContent string
			// Check if we actually need truncation to avoid unnecessary ellipsis
			if lipgloss.Width(rendered) > availableWidth {
				displayContent = truncate.StringWithTail(rendered, uint(availableWidth), tail)
			} else {
				displayContent = rendered
			}

			// For style consistency (green/pink colors), we might want to apply them ONLY if
			// the content is plain text (no escape codes). But checking for escape codes is fragile.
			// Let's rely on glamour's dracula theme which is nice enough.
			styledContent = displayContent

			left := fmt.Sprintf("%s%s: %s", margin, styledSender, styledContent)
			currentLen := lipgloss.Width(left)
			padding := m.width - currentLen - lipgloss.Width(ts)
			if padding < 2 {
				padding = 2
			}

			lines = fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", padding), ts)

		} else {
			// Expanded view: Multiline Markdown (glamour)
			// We render the Header (Sender + Timestamp) followed by Content

			// Header
			// Format: "MarginSender:                               TIMESTAMP"
			colon := ": "
			headerLeft := fmt.Sprintf("%s%s%s", margin, styledSender, colon) // Includes color codes

			// Calculate padding for timestamp alignment
			// Ensure timestamp is pinned to right
			tsWidth := lipgloss.Width(ts)
			headerWidth := lipgloss.Width(headerLeft)
			padding := m.width - headerWidth - tsWidth - 2 // -2 margin right
			if padding < 2 {
				padding = 2
			}

			header := fmt.Sprintf("%s%s%s", headerLeft, strings.Repeat(" ", padding), ts)

			// Content (Markdown)
			// Use our glamour renderer
			var renderedContent string
			if m.renderer != nil {
				out, err := m.renderer.Render(rawContent)
				if err != nil {
					renderedContent = rawContent // Fallback
				} else {
					renderedContent = out
				}
			} else {
				renderedContent = rawContent
			}

			// Trim excessive newlines from glamour output
			renderedContent = strings.TrimSpace(renderedContent)

			// Indent the rendered block to align with content start (optional)
			// Or just indent a bit (2 spaces)
			// Glamour adds margins usually, so maybe just 2 spaces
			indent := strings.Repeat(" ", 2)
			rows := strings.Split(renderedContent, "\n")
			var indentedBlock string
			for i, r := range rows {
				if i == 0 {
					indentedBlock += indent + r
				} else {
					indentedBlock += "\n" + indent + r
				}
			}

			lines = header + "\n" + indentedBlock
		}

		b.WriteString(lines + "\n")
	}
	return b.String()
}

func (m Model) View() string {
	if m.screen == "welcome" {
		return m.viewWelcome()
	}
	return m.viewChat()
}

func (m Model) viewWelcome() string {
	var b strings.Builder

	// Calculate available height
	availableH := m.height
	contentH := 0

	// Determine what to show based on height
	// Banner is ~6 lines tall.
	showBanner := availableH >= 20

	var banner string
	if showBanner {
		banner = HeaderStyle.Render(hushASCII)
		contentH += lipgloss.Height(banner) + 1 // +1 for spacing
	}

	tagline := StatusStyle.Render("talk to anyone on your wifi — no servers, no trace")
	contentH += 2 // tagline + spacing (newline)

	// Input box
	// We'll wrap the input in a box style. No extra "> " prompt since we want a cleaner look.
	inputView := m.input.View()
	inputBox := InputBorderStyle.Width(40).Render(inputView)
	contentH += lipgloss.Height(inputBox) + 1

	hint := StatusStyle.Render("press enter to join")
	contentH += 1

	// Top padding to vertically center the block
	topPad := (availableH - contentH) / 2
	if topPad > 0 {
		b.WriteString(strings.Repeat("\n", topPad))
	}

	// Helper to center a line/block horizontally
	center := func(s string) string {
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(s)
	}

	if showBanner {
		// Render banner line by line to center it if it's multiline
		// But lipgloss.Place or Align(Center) on the block works if width is set
		b.WriteString(center(banner) + "\n")
	}

	b.WriteString(center(tagline) + "\n\n")
	b.WriteString(center(inputBox) + "\n")
	b.WriteString(center(hint))

	return lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(b.String())
}

func (m Model) viewChat() string {
	if !m.ready {
		return "  initializing...\n"
	}

	var b strings.Builder

	// Header
	b.WriteString(Header())
	b.WriteString("\n")

	// Status bar
	status := fmt.Sprintf("  online as %s  (%d active peers)", m.username, m.peerCount)
	b.WriteString(StatusStyle.Render(status))
	b.WriteString("\n")
	b.WriteString(Divider(m.width))
	b.WriteString("\n")

	// Message viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Divider
	b.WriteString(Divider(m.width))
	b.WriteString("\n")

	// Warning
	if m.showWarning {
		b.WriteString(WarningStyle.Render("  " + m.warningMsg))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// Input
	inputStyle := InputBorderStyle
	if m.showWarning {
		inputStyle = InputBorderWarnStyle
	}
	b.WriteString(inputStyle.Width(m.width - 4).Render(m.textArea.View()))

	return lipgloss.NewStyle().MaxWidth(m.width).Render(b.String())
}
