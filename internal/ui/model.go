package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ekrishgupta/HushApp/internal/chat"
)

const spamCooldown = 1500 * time.Millisecond

// IncomingMsg is a Bubble Tea message wrapping a chat message from the network.
type IncomingMsg chat.ChatMessage

// Model is the Bubble Tea model for the chat TUI.
type Model struct {
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

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// NewModel creates a new chat TUI model.
func NewModel(username string, c *chat.Chat, msgChan <-chan chat.ChatMessage) Model {
	ti := textinput.New()
	ti.Placeholder = "type a message..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	return Model{
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
		msg, ok := <-m.msgChan
		if !ok {
			return nil
		}
		return IncomingMsg(msg)
	}
}

// Init starts listening for network messages.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.waitForMsg(),
		tick(),
	)
}

// Update handles events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		m.peerCount = m.chat.PeerCount()
		cmds = append(cmds, tick())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerH := 3 // header + status + divider
		inputH := 3  // input box area
		warnH := 1   // warning line
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

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			content := strings.TrimSpace(m.input.Value())
			if content == "" {
				break
			}

			// Anti-spam cooldown
			if time.Since(m.lastSent) < spamCooldown {
				m.showWarning = true
				m.warningMsg = "⚡ Slow down!"
				break
			}

			m.showWarning = false
			if err := m.chat.Publish(m.username, content); err == nil {
				// Add own message to local history
				ownMsg := chat.NewChatMessage(m.username, content)
				m.messages = append(m.messages, ownMsg)
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				m.lastSent = time.Now()
			}
			m.input.Reset()

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

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// renderMessages formats all messages for the viewport.
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
		// Pad the timestamp to the right
		padding := m.width - lipgloss.Width(left) - lipgloss.Width(ts) - 2
		if padding < 2 {
			padding = 2
		}
		b.WriteString(fmt.Sprintf("%s%s%s\n", left, strings.Repeat(" ", padding), ts))
	}
	return b.String()
}

// View renders the full TUI.
func (m Model) View() string {
	if !m.ready {
		return "  initializing...\n"
	}

	var b strings.Builder

	// Header
	b.WriteString(Header())
	b.WriteString("\n")
	// Status bar below header
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
