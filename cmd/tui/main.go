package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ekrishgupta/HushApp/internal/chat"
	"github.com/ekrishgupta/HushApp/internal/network"
	"github.com/ekrishgupta/HushApp/internal/ui"
)

func main() {
	username := promptUsername()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to deliver the chat instance once network is ready
	readyCh := make(chan *chat.Chat, 1)
	errCh := make(chan error, 1)

	// Initialize network in background so the TUI renders instantly
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

	// Launch TUI immediately (shows "connecting..." until network is ready)
	model := ui.NewModel(username, nil, nil)
	model.SetNetworkChannels(readyCh, errCh, ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}

func promptUsername() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("👻 enter a username (or press enter for a random one): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if name == "" {
		name = fmt.Sprintf("Ghost-%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(900)+100)
	}

	fmt.Printf("   welcome, %s\n\n", name)
	return name
}
