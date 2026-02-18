package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/ekrishgupta/Hush/internal/chat"
	"github.com/ekrishgupta/Hush/internal/network"
)

// App struct
type App struct {
	ctx      context.Context
	chat     *chat.Chat
	username string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		username: fmt.Sprintf("Ghost-%d", time.Now().Unix()%1000),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize libp2p host
	h, err := network.NewHost()
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to create host: %v", err)
		return
	}

	// Setup mDNS discovery
	if err := network.SetupDiscovery(h); err != nil {
		runtime.LogErrorf(ctx, "Failed to setup discovery: %v", err)
		return
	}

	// Setup GossipSub
	topic, sub, err := network.SetupPubSub(ctx, h)
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to setup pubsub: %v", err)
		return
	}

	a.chat = chat.NewChat(topic, sub, h.ID())

	// Start listening for messages and pipe them to frontend events
	go func() {
		for msg := range a.chat.ListenForMessages(ctx) {
			runtime.EventsEmit(ctx, "new_message", msg)
		}
	}()

	runtime.LogInfo(ctx, "App started successfully, listening for messages...")
}

// SendMessage publishes a message to the network
func (a *App) SendMessage(text string) {
	if a.chat == nil {
		return
	}

	if err := a.chat.Publish(a.username, text); err != nil {
		runtime.LogErrorf(a.ctx, "Failed to publish message: %v", err)
		return
	}

	// Emit back to UI immediately (as a "self" message)
	msg := chat.NewChatMessage(a.username, text)
	runtime.EventsEmit(a.ctx, "new_message", msg)
}

// SetUsername updates the current user's name
func (a *App) SetUsername(name string) {
	a.username = name
}

// GetUsername returns the current user's name
func (a *App) GetUsername() string {
	return a.username
}

// GetPeerCount returns the number of active peers
func (a *App) GetPeerCount() int {
	if a.chat == nil {
		return 0
	}
	return a.chat.PeerCount()
}
