package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ekrishgupta/HushApp/internal/chat"
	"github.com/ekrishgupta/HushApp/internal/network"
	"github.com/ekrishgupta/HushApp/internal/ui"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start network init immediately in background
	readyCh := make(chan *chat.Chat, 1)
	errCh := make(chan error, 1)

	go func() {
		h, err := network.NewHost()
		if err != nil {
			errCh <- fmt.Errorf("host: %w", err)
			return
		}

		if err := network.SetupDiscovery(h); err != nil {
			errCh <- fmt.Errorf("discovery: %w", err)
			return
		}

		topic, sub, err := network.SetupPubSub(ctx, h)
		if err != nil {
			errCh <- fmt.Errorf("pubsub: %w", err)
			return
		}

		c := chat.NewChat(topic, sub, h.ID())
		readyCh <- c
	}()

	// Launch TUI immediately — username prompt is the first screen,
	// network connects in the background while the user types
	model := ui.NewModel("", nil, nil)
	model.SetNetworkChannels(readyCh, errCh, ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}
