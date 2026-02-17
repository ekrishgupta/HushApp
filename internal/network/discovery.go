package network

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const ServiceTag = "_ghost-chat-wifi"

// discoveryNotifee is notified when a new peer is found via mDNS.
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to a newly discovered peer.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.h.ID() {
		return // ignore self
	}
	if err := n.h.Connect(context.Background(), pi); err != nil {
		fmt.Printf("⚠ failed to connect to peer %s: %v\n", pi.ID.String()[:8], err)
	} else {

	}
}

// SetupDiscovery starts mDNS peer discovery on the local network.
func SetupDiscovery(h host.Host) error {
	n := &discoveryNotifee{h: h}
	svc := mdns.NewMdnsService(h, ServiceTag, n)
	return svc.Start()
}
