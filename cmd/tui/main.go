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

	// 1. Create libp2p host
	h, err := network.NewHost()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer h.Close()

	// 2. Start mDNS discovery
	if err := network.SetupDiscovery(h); err != nil {
		fmt.Fprintf(os.Stderr, "discovery error: %v\n", err)
		os.Exit(1)
	}

	// 3. Set up GossipSub
	topic, sub, err := network.SetupPubSub(ctx, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pubsub error: %v\n", err)
		os.Exit(1)
	}

	// 4. Create chat and start listening
	c := chat.NewChat(topic, sub, h.ID())
	msgChan := c.ListenForMessages(ctx)

	// 5. Launch TUI
	model := ui.NewModel(username, c, msgChan)
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
