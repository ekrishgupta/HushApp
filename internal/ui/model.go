package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ekrishgupta/HushApp/internal/chat"
)

const spamCooldown = 1500 * time.Millisecond

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
	input    textinput.Model

	lastSent    time.Time
	showWarning bool
	warningMsg  string

	width  int
	height int
	ready  bool

	peerCount int
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// NewModel creates a new chat TUI model.
func NewModel(username string, c *chat.Chat, msgChan <-chan chat.ChatMessage) Model {
	ti := textinput.New()
	ti.Placeholder = "enter your name..."
	ti.Focus()
	ti.CharLimit = 30
	ti.Width = 40

	screen := "welcome"
	if username != "" {
		screen = "chat"
		ti.Placeholder = "type a message..."
		ti.CharLimit = 500
		ti.Width = 60
	}

	return Model{
		screen:   screen,
		username: username,
		chat:     c,
		msgChan:  msgChan,
		input:    ti,
		messages: []chat.ChatMessage{},
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
			inputH := 3
			warnH := 1
			vpHeight := m.height - headerH - inputH - warnH - 1

			if !m.ready {
				m.viewport = viewport.New(m.width, vpHeight)
				m.viewport.SetContent(m.renderMessages())
				m.ready = true
			} else {
				m.viewport.Width = m.width
				m.viewport.Height = vpHeight
			}
			m.input.Width = m.width - 6
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.screen == "welcome" {
				return m.handleWelcomeEnter()
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
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	if m.screen == "chat" {
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

	// Reset input for chat
	m.input.Reset()
	m.input.Placeholder = "type a message..."
	m.input.CharLimit = 500
	m.input.Width = 60

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
	content := strings.TrimSpace(m.input.Value())
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
	m.input.Reset()
	return m, nil
}

// ── Render: Messages ────────────────────────────────

func (m Model) renderMessages() string {
	if len(m.messages) == 0 {
		return StatusStyle.Render("\n  waiting for ghosts to appear... 👻\n")
	}

	var b strings.Builder
	for _, msg := range m.messages {
		ts := TimestampStyle.Render(msg.Time().Format("15:04:05"))

		var sender, content string
		if msg.Sender == m.username {
			sender = SelfMsgSender.Render("you")
			content = SelfMsgContent.Render(msg.Content)
		} else {
			sender = PeerMsgSender.Render(msg.Sender)
			content = PeerMsgContent.Render(msg.Content)
		}

		left := fmt.Sprintf("  %s: %s", sender, content)
		padding := m.width - lipgloss.Width(left) - lipgloss.Width(ts) - 2
		if padding < 2 {
			padding = 2
		}
		b.WriteString(fmt.Sprintf("%s%s%s\n", left, strings.Repeat(" ", padding), ts))
	}
	return b.String()
}

// ── View ────────────────────────────────────────────

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
	b.WriteString(inputStyle.Width(m.width - 4).Render(m.input.View()))

	return lipgloss.NewStyle().MaxWidth(m.width).Render(b.String())
}
