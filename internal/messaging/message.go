// Package messaging is the application-level messaging abstraction. It owns the
// project's message and target types plus the small capability interfaces that
// concrete transports (e.g. the NapCat adapter) implement, so the agent and tools
// never depend on a concrete transport.
package messaging

import (
	"context"
	"errors"
)

// TargetType discriminates a private chat from a group chat.
type TargetType string

const (
	// TargetTypePrivate is a one-to-one private chat.
	TargetTypePrivate TargetType = "private"
	// TargetTypeGroup is a group chat.
	TargetTypeGroup TargetType = "group"
)

// Target identifies a chat target: a private user or a group.
type Target struct {
	Type TargetType
	ID   int64
}

// ErrQueueFull is returned by a Sender when its outgoing queue cannot accept a message.
var ErrQueueFull = errors.New("message send queue is full")

// Sender sends a text message to a chat target. Implementations translate the
// application-level target into their concrete transport (e.g. NapCat).
type Sender interface {
	SendMessage(ctx context.Context, target Target, content string) error
}

// HistoryMessage is a single historical text message.
type HistoryMessage struct {
	MessageID  int64
	MessageSeq int64
	Time       int64
	UserID     int64
	NickName   string
	Content    string
}

// HistoryProvider fetches recent message history for a chat target, paging
// backwards when beforeSeq is greater than zero.
type HistoryProvider interface {
	GetMessageHistory(ctx context.Context, target Target, beforeSeq int64, count int) ([]HistoryMessage, error)
}
