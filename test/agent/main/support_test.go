package mainagent_test

import (
	"context"

	"miss-raspberry-agent/internal/messaging"
)

// stubSender implements messaging.Sender without touching a real NapCat client.
type stubSender struct{}

func (stubSender) SendMessage(context.Context, messaging.Target, string) error { return nil }

// stubHistory implements messaging.HistoryProvider and always returns an empty history.
type stubHistory struct{}

func (stubHistory) GetMessageHistory(context.Context, messaging.Target, int64, int) ([]messaging.HistoryMessage, error) {
	return nil, nil
}
