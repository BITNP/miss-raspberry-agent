package qq

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"miss-raspberry-agent/internal/messaging"
)

type QQMessageSenderInput struct {
	TargetId   int64  `json:"target_id" jsonschema:"required,description=目标用户QQ号（target_type=private）或目标群聊QQ群号（target_type=group）"`
	TargetType string `json:"target_type" jsonschema:"enum=private,enum=group,default=private,description=目标类型：private=私聊，group=群聊"`
	Content    string `json:"content" jsonschema:"required,description=要发送的文本内容"`
}

type QQMessageSenderOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SendMessage puts the message into the sender's send queue; the underlying client performs
// the actual sending.
func SendMessage(ctx context.Context, sender messaging.Sender, in *QQMessageSenderInput) (*QQMessageSenderOutput, error) {
	target := messaging.Target{ID: in.TargetId}
	switch in.TargetType {
	case "", "private":
		target.Type = messaging.TargetTypePrivate
	case "group":
		target.Type = messaging.TargetTypeGroup
	default:
		return nil, fmt.Errorf("qq_message_sender: unknown target_type %q (expected private or group)", in.TargetType)
	}

	if err := sender.SendMessage(ctx, target, in.Content); err != nil {
		if errors.Is(err, messaging.ErrQueueFull) {
			return &QQMessageSenderOutput{Success: false, Message: "send queue is full"}, nil
		}
		return nil, err
	}
	return &QQMessageSenderOutput{
		Success: true,
		Message: "ok",
	}, nil
}

// NewQQMessageSender constructs the "send QQ message" tool.
func NewQQMessageSender(sender messaging.Sender) tool.BaseTool {
	fn := func(ctx context.Context, in *QQMessageSenderInput) (*QQMessageSenderOutput, error) {
		return SendMessage(ctx, sender, in)
	}
	t, err := utils.InferTool(
		"qq_message_sender", // tool name, used by the LLM to invoke it
		"向指定QQ用户或QQ群发送一条文本消息", // tool desc, written for the LLM
		fn,
	)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
