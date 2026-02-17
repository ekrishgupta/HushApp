package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors — Adaptive for Light/Dark terminal backgrounds
	ghostPurple = lipgloss.AdaptiveColor{Light: "#6200EA", Dark: "#B388FF"}
	ghostPink   = lipgloss.AdaptiveColor{Light: "#C51162", Dark: "#FF80AB"}
	softGreen   = lipgloss.AdaptiveColor{Light: "#00C853", Dark: "#69F0AE"}
	warmWhite   = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#F5F5F5"}
	dimGray     = lipgloss.AdaptiveColor{Light: "#9E9E9E", Dark: "#666666"}
	warningRed  = lipgloss.AdaptiveColor{Light: "#D50000", Dark: "#FF5252"}

	// Header — the "Hush" banner
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ghostPurple).
			Padding(0, 1)

	// Status bar below header
	StatusStyle = lipgloss.NewStyle().
			Foreground(dimGray).
			Italic(true).
			Padding(0, 1)

	// Messages from other peers
	PeerMsgSender = lipgloss.NewStyle().
			Foreground(ghostPink).
			Bold(true)

	PeerMsgContent = lipgloss.NewStyle().
			Foreground(warmWhite)

	// Messages from self
	SelfMsgSender = lipgloss.NewStyle().
			Foreground(softGreen).
			Bold(true)

	SelfMsgContent = lipgloss.NewStyle().
			Foreground(warmWhite)

	// Timestamp
	TimestampStyle = lipgloss.NewStyle().
			Foreground(dimGray)

	// Warning text for anti-spam
	WarningStyle = lipgloss.NewStyle().
			Foreground(warningRed).
			Bold(true)

	// Input area border
	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ghostPurple).
				Padding(0, 1)

	InputBorderWarnStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(warningRed).
				Padding(0, 1)

	// Divider line
	DividerStyle = lipgloss.NewStyle().
			Foreground(dimGray)

	// Selected message highlight
	SelectedMsgStyle = lipgloss.NewStyle().
				Background(lipgloss.AdaptiveColor{Light: "#E0E0E0", Dark: "#333333"})
)

// Header renders the app title.
func Header() string {
	return HeaderStyle.Render("👻 Hush — Ghost Chat")
}

// Divider renders a horizontal line.
func Divider(width int) string {
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return DividerStyle.Render(line)
}
