package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ekrishgupta/Hush/internal/chat"
	"github.com/ekrishgupta/Hush/internal/network"
	"github.com/ekrishgupta/Hush/internal/ui"
)

func main() {
	// 1. Initialize network (blocking)
	// This will take a few seconds, but eliminates the need for complex async loading states in the UI.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := network.NewHost()
	if err != nil {
		fmt.Fprintf(os.Stderr, "host error: %v\n", err)
		os.Exit(1)
	}
	defer h.Close()

	// Start mDNS discovery
	if err := network.SetupDiscovery(h); err != nil {
		fmt.Fprintf(os.Stderr, "discovery error: %v\n", err)
		os.Exit(1)
	}

	// Set up GossipSub
	topic, sub, err := network.SetupPubSub(ctx, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pubsub error: %v\n", err)
		os.Exit(1)
	}

	// Create chat and start listening
	c := chat.NewChat(topic, sub, h.ID())
	msgChan := c.ListenForMessages(ctx)

	// 2. Launch TUI
	// The model is initialized with the ready chat instance.
	// We pass an empty username because the first screen is the "Welcome" prompt.
	model := ui.NewModel("", c, msgChan)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}
