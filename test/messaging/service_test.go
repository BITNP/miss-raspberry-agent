package messaging_test

import (
	"context"
	"errors"
	"testing"

	"miss-raspberry-agent/internal/messaging"
	"miss-raspberry-agent/internal/tool/todo"
)

func TestSubmitEnqueuesMessage(t *testing.T) {
	queue := todo.NewStore()
	svc := messaging.NewService(queue)

	id, err := svc.Submit(context.Background(), messaging.Message{
		Platform: messaging.PlatformQQ,
		TargetID: "10001",
		Content:  "明天下午的会别忘了",
		Context:  "用户刚出差回来，可能没看到群公告",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if id == "" {
		t.Fatal("expected a non-empty todo id")
	}

	items := queue.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 queued item, got %d", len(items))
	}
	got := items[0]
	if got.ID != id {
		t.Errorf("id = %q, want %q", got.ID, id)
	}
	if got.Content != "明天下午的会别忘了" {
		t.Errorf("content = %q", got.Content)
	}
	if got.Context != "用户刚出差回来，可能没看到群公告" {
		t.Errorf("context = %q", got.Context)
	}
	if got.TargetType != "private" || got.TargetID != 10001 || got.UserID != 10001 {
		t.Errorf("unexpected target: type=%q id=%d user=%d", got.TargetType, got.TargetID, got.UserID)
	}
	if got.Source != "API(平台=qq,目标=10001)" {
		t.Errorf("source = %q", got.Source)
	}
}

func TestSubmitTrimsWhitespace(t *testing.T) {
	queue := todo.NewStore()
	svc := messaging.NewService(queue)

	if _, err := svc.Submit(context.Background(), messaging.Message{
		Platform: messaging.PlatformQQ,
		TargetID: " 7 ",
		Content:  "  hi  ",
		Context:  "  ctx  ",
	}); err != nil {
		t.Fatalf("submit: %v", err)
	}

	got := queue.List()[0]
	if got.Content != "hi" {
		t.Errorf("content = %q, want %q", got.Content, "hi")
	}
	if got.Context != "ctx" {
		t.Errorf("context = %q, want %q", got.Context, "ctx")
	}
	if got.TargetID != 7 {
		t.Errorf("target id = %d, want 7", got.TargetID)
	}
}

func TestSubmitValidation(t *testing.T) {
	cases := []struct {
		name string
		msg  messaging.Message
		want error
	}{
		{"unsupported platform", messaging.Message{Platform: "discord", TargetID: "1", Content: "hi"}, messaging.ErrUnsupportedPlatform},
		{"empty platform", messaging.Message{TargetID: "1", Content: "hi"}, messaging.ErrUnsupportedPlatform},
		{"empty content", messaging.Message{Platform: messaging.PlatformQQ, TargetID: "1"}, messaging.ErrInvalidMessage},
		{"blank content", messaging.Message{Platform: messaging.PlatformQQ, TargetID: "1", Content: "   "}, messaging.ErrInvalidMessage},
		{"empty target id", messaging.Message{Platform: messaging.PlatformQQ, Content: "hi"}, messaging.ErrInvalidMessage},
		{"non-numeric target id", messaging.Message{Platform: messaging.PlatformQQ, TargetID: "abc", Content: "hi"}, messaging.ErrInvalidMessage},
		{"zero target id", messaging.Message{Platform: messaging.PlatformQQ, TargetID: "0", Content: "hi"}, messaging.ErrInvalidMessage},
		{"negative target id", messaging.Message{Platform: messaging.PlatformQQ, TargetID: "-5", Content: "hi"}, messaging.ErrInvalidMessage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			queue := todo.NewStore()
			svc := messaging.NewService(queue)
			if _, err := svc.Submit(context.Background(), tc.msg); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if !queue.IsEmpty() {
				t.Fatal("invalid message must not be enqueued")
			}
		})
	}
}

func TestSubmitRespectsCanceledContext(t *testing.T) {
	queue := todo.NewStore()
	svc := messaging.NewService(queue)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.Submit(ctx, messaging.Message{Platform: messaging.PlatformQQ, TargetID: "1", Content: "hi"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if !queue.IsEmpty() {
		t.Fatal("canceled submit must not be enqueued")
	}
}
