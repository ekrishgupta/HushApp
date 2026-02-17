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
	"github.com/charmbracelet/lipgloss"

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
			// If a message is selected, toggle expansion
			if m.selectedMsg != -1 {
				m.expanded[m.selectedMsg] = !m.expanded[m.selectedMsg]
				m.viewport.SetContent(m.renderMessages())
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

			// Calculate space occupied by static elements
			// Margin(2) + Sender + ": " (2) + "   " (3 min padding) + Timestamp
			prefixWidth := lipgloss.Width(margin) + lipgloss.Width(senderLabel) + 2
			suffixWidth := 3 + lipgloss.Width(tsRaw)
			availableWidth := m.width - prefixWidth - suffixWidth

			// Sanity check
			if availableWidth < 10 {
				availableWidth = 10
			}

			// Clean content processing
			contentRunes := []rune(rawContent)
			hasNewline := strings.Contains(rawContent, "\n")
			needsTruncation := len(contentRunes) > availableWidth || hasNewline

			var displayContent string
			var ellipsis string

			if needsTruncation {
				// We need to truncate
				// Determine ellipsis style
				if isSelected {
					// Highlight the (...)
					ellipsis = SelectedMsgStyle.Render(" (...)")
				} else {
					ellipsis = " (...)"
				}

				cutLen := availableWidth - 6 // Leave room for " (...)" which is 6 chars length
				if cutLen < 0 {
					cutLen = 0
				}

				// If newline comes before cutLen, cut there
				newlineIdx := strings.Index(rawContent, "\n")
				if newlineIdx != -1 && newlineIdx < cutLen {
					cutLen = newlineIdx
				}

				if cutLen < len(contentRunes) {
					displayContent = string(contentRunes[:cutLen])
				} else {
					displayContent = string(contentRunes[:cutLen])
				}
			} else {
				displayContent = rawContent
			}

			// Render content
			if msg.Sender == m.username {
				styledContent = SelfMsgContent.Render(displayContent)
			} else {
				styledContent = PeerMsgContent.Render(displayContent)
			}

			// Append highlighted ellipsis if needed
			if needsTruncation {
				styledContent += ellipsis
			}

			left := fmt.Sprintf("%s%s: %s", margin, styledSender, styledContent)

			// Calculate padding manually to push timestamp to right
			currentLen := lipgloss.Width(left)
			padding := m.width - currentLen - lipgloss.Width(ts)
			if padding < 2 {
				padding = 2
			}

			lines = fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", padding), ts)

		} else {
			// Expanded view: Multiline

			// 1. Setup Header Prefix
			// Format: "MarginSender: "
			colon := ": "
			headerPrefix := fmt.Sprintf("%s%s%s", margin, styledSender, colon)
			indentWidth := lipgloss.Width(headerPrefix)

			// 2. Calculate Available Width for Line 1 Content
			// Total - Indent - Timestamp - MinPadding(2)
			tsWidth := lipgloss.Width(ts)
			maxLine1Width := m.width - indentWidth - tsWidth - 2
			if maxLine1Width < 0 {
				maxLine1Width = 0
			}

			// 3. Split Content
			runes := []rune(rawContent)
			splitIdx := len(runes)
			if splitIdx > maxLine1Width {
				splitIdx = maxLine1Width
			}

			// Check for newline in the first chunk
			isNewlineSplit := false
			if idx := strings.Index(string(runes[:splitIdx]), "\n"); idx != -1 {
				splitIdx = idx
				isNewlineSplit = true
			}

			line1Text := string(runes[:splitIdx])

			remainingStart := splitIdx
			if isNewlineSplit && remainingStart < len(runes) {
				remainingStart++ // Skip the newline
			}
			remainingText := string(runes[remainingStart:])

			// 4. Render Line 1
			var styledLine1 string
			if msg.Sender == m.username {
				styledLine1 = SelfMsgContent.Render(line1Text)
			} else {
				styledLine1 = PeerMsgContent.Render(line1Text)
			}

			// Construct full line 1: Prefix + Content + Padding + Timestamp
			currentLen := lipgloss.Width(headerPrefix) + lipgloss.Width(styledLine1)
			paddingNeeded := m.width - currentLen - tsWidth
			if paddingNeeded < 2 {
				paddingNeeded = 2
			}

			lines = fmt.Sprintf("%s%s%s%s", headerPrefix, styledLine1, strings.Repeat(" ", paddingNeeded), ts)

			// 5. Render Remaining Lines
			if len(remainingText) > 0 {
				wrapW := maxLine1Width
				if wrapW < 10 {
					wrapW = 10
				}

				// Style and Wrap
				var wrappedBlock string
				if msg.Sender == m.username {
					wrappedBlock = SelfMsgContent.Width(wrapW).Render(remainingText)
				} else {
					wrappedBlock = PeerMsgContent.Width(wrapW).Render(remainingText)
				}

				// Indent
				indentStr := strings.Repeat(" ", indentWidth)
				rows := strings.Split(wrappedBlock, "\n")
				for _, r := range rows {
					lines += "\n" + indentStr + r
				}
			}
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
