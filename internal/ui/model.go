package ui

import (
	"context"
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

// NetworkReady signals that the network has finished initializing.
type NetworkReady struct {
	Chat *chat.Chat
}

// NetworkError signals a network initialization failure.
type NetworkError struct {
	Err error
}

type tickMsg time.Time
type spinTickMsg struct{}

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

	// Async network init
	connecting bool
	networkErr string
	readyCh    chan *chat.Chat
	errCh      chan error
	netCtx     context.Context
	spinFrame  int
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func spinTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return spinTickMsg{}
	})
}

// SetNetworkChannels sets the channels for async network initialization.
func (m *Model) SetNetworkChannels(readyCh chan *chat.Chat, errCh chan error, ctx context.Context) {
	m.connecting = true
	m.readyCh = readyCh
	m.errCh = errCh
	m.netCtx = ctx
}

// NewModel creates a new chat TUI model.
func NewModel(username string, c *chat.Chat, msgChan <-chan chat.ChatMessage) Model {
	ti := textinput.New()
	ti.Placeholder = "enter your name..."
	ti.Focus()
	ti.CharLimit = 30
	ti.Width = 40

	screen := "welcome"
	connecting := c == nil
	if username != "" {
		screen = "chat"
		ti.Placeholder = "type a message..."
		ti.CharLimit = 500
		ti.Width = 60
	}

	return Model{
		screen:     screen,
		username:   username,
		chat:       c,
		msgChan:    msgChan,
		input:      ti,
		messages:   []chat.ChatMessage{},
		connecting: connecting,
	}
}

// waitForNetwork returns a command that waits for the network to be ready.
func (m Model) waitForNetwork() tea.Cmd {
	return func() tea.Msg {
		select {
		case c := <-m.readyCh:
			return NetworkReady{Chat: c}
		case err := <-m.errCh:
			return NetworkError{Err: err}
		}
	}
}

// waitForMsg returns a command that waits for the next network message.
func (m Model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
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
	if m.connecting {
		cmds = append(cmds, m.waitForNetwork(), spinTick())
	} else if m.msgChan != nil {
		cmds = append(cmds, m.waitForMsg(), tick())
	}
	return tea.Batch(cmds...)
}

// ── Update ──────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinTickMsg:
		if m.connecting {
			m.spinFrame++
			cmds = append(cmds, spinTick())
		}

	case NetworkReady:
		m.connecting = false
		m.chat = msg.Chat
		if m.screen == "chat" {
			// Already on chat screen, start listening
			m.msgChan = msg.Chat.ListenForMessages(m.netCtx)
			m.peerCount = m.chat.PeerCount()
			if m.ready {
				m.viewport.SetContent(m.renderMessages())
			}
			cmds = append(cmds, m.waitForMsg(), tick())
		}

	case NetworkError:
		m.connecting = false
		m.networkErr = msg.Err.Error()
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
		}

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

	// If network is already ready, start listening now
	if m.chat != nil {
		m.msgChan = m.chat.ListenForMessages(m.netCtx)
		m.peerCount = m.chat.PeerCount()
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
	if m.connecting || m.chat == nil {
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
	if m.connecting {
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinChars[m.spinFrame%len(spinChars)]
		return StatusStyle.Render(fmt.Sprintf("\n  %s connecting to network...\n", frame))
	}

	if m.networkErr != "" {
		return WarningStyle.Render(fmt.Sprintf("\n  ✗ network error: %s\n", m.networkErr))
	}

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

	// Center vertically: calculate padding
	contentHeight := 12 // approximate height of welcome content
	topPad := (m.height - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}
	b.WriteString(strings.Repeat("\n", topPad))

	// ASCII banner
	banner := HeaderStyle.Render(hushASCII)
	// Center horizontally
	for _, line := range strings.Split(banner, "\n") {
		pad := (m.width - lipgloss.Width(line)) / 2
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad) + line + "\n")
	}

	// Tagline
	tagline := StatusStyle.Render("talk to anyone on your wifi — no servers, no trace")
	tagPad := (m.width - lipgloss.Width(tagline)) / 2
	if tagPad < 0 {
		tagPad = 0
	}
	b.WriteString(strings.Repeat(" ", tagPad) + tagline + "\n\n")

	// Network status indicator
	var netStatus string
	if m.connecting {
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinChars[m.spinFrame%len(spinChars)]
		netStatus = StatusStyle.Render(fmt.Sprintf("%s connecting...", frame))
	} else if m.networkErr != "" {
		netStatus = WarningStyle.Render("✗ network error")
	} else {
		netStatus = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#00C853", Dark: "#69F0AE"}).Render("● connected")
	}
	netPad := (m.width - lipgloss.Width(netStatus)) / 2
	if netPad < 0 {
		netPad = 0
	}
	b.WriteString(strings.Repeat(" ", netPad) + netStatus + "\n\n")

	// Input with prompt
	inputBox := InputBorderStyle.Width(40).Render("> " + m.input.View())
	inputPad := (m.width - lipgloss.Width(inputBox)) / 2
	if inputPad < 0 {
		inputPad = 0
	}
	b.WriteString(strings.Repeat(" ", inputPad) + inputBox + "\n")

	// Hint
	hint := StatusStyle.Render("press enter to join")
	hintPad := (m.width - lipgloss.Width(hint)) / 2
	if hintPad < 0 {
		hintPad = 0
	}
	b.WriteString(strings.Repeat(" ", hintPad) + hint)

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
	if m.connecting {
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinChars[m.spinFrame%len(spinChars)]
		b.WriteString(StatusStyle.Render(fmt.Sprintf("  %s connecting...", frame)))
	} else if m.networkErr != "" {
		b.WriteString(WarningStyle.Render("  ✗ failed to connect"))
	} else {
		status := fmt.Sprintf("  online as %s  (%d active peers)", m.username, m.peerCount)
		b.WriteString(StatusStyle.Render(status))
	}
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
