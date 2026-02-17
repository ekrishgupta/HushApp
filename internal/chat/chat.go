package chat

import (
	"context"
	"encoding/json"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Chat manages publishing and subscribing to the local-gc topic.
type Chat struct {
	topic *pubsub.Topic
	sub   *pubsub.Subscription
	self  peer.ID
}

// NewChat wraps the topic and subscription.
func NewChat(topic *pubsub.Topic, sub *pubsub.Subscription, selfID peer.ID) *Chat {
	return &Chat{
		topic: topic,
		sub:   sub,
		self:  selfID,
	}
}

// Publish serializes and sends a ChatMessage to the topic.
func (c *Chat) Publish(sender, content string) error {
	msg := NewChatMessage(sender, content)
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}
	return c.topic.Publish(context.Background(), data)
}

// ListenForMessages reads from the subscription in a goroutine and
// sends decoded messages to the returned channel, filtering out
// messages from self.
func (c *Chat) ListenForMessages(ctx context.Context) <-chan ChatMessage {
	ch := make(chan ChatMessage, 32)

	go func() {
		defer close(ch)
		for {
			msg, err := c.sub.Next(ctx)
			if err != nil {
				return // context cancelled or subscription closed
			}

			// skip messages from ourselves
			if msg.ReceivedFrom == c.self {
				continue
			}

			var cm ChatMessage
			if err := json.Unmarshal(msg.Data, &cm); err != nil {
				continue // skip malformed messages
			}

			select {
			case ch <- cm:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// PeerCount returns the number of peers currently in the topic.
func (c *Chat) PeerCount() int {
	return len(c.topic.ListPeers())
}
