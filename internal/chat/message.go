package chat

import "time"

// ChatMessage is the JSON structure sent over the wire.
type ChatMessage struct {
	Sender    string `json:"sender"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// NewChatMessage creates a new message with the current timestamp.
func NewChatMessage(sender, content string) ChatMessage {
	return ChatMessage{
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
}

// Time returns the message timestamp as a time.Time.
func (m ChatMessage) Time() time.Time {
	return time.Unix(m.Timestamp, 0)
}
