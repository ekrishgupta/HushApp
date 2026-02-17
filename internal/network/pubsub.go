package network

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const TopicName = "local-gc"

// SetupPubSub creates a GossipSub router and joins the global topic.
func SetupPubSub(ctx context.Context, h host.Host) (*pubsub.Topic, *pubsub.Subscription, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, nil, fmt.Errorf("creating gossipsub: %w", err)
	}

	topic, err := ps.Join(TopicName)
	if err != nil {
		return nil, nil, fmt.Errorf("joining topic %q: %w", TopicName, err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, nil, fmt.Errorf("subscribing to topic %q: %w", TopicName, err)
	}

	return topic, sub, nil
}
