// Package agent defines the project's agent roles and the registry that hands
// them out. Concrete agent implementations live in subpackages such as
// agent/moderator and agent/main.
package agent

import (
	"context"

	"miss-raspberry-agent/internal/agent/moderator"
)

// Role names an externally callable agent behavior.
type Role string

// RoleCommentModerator judges whether a comment may be posted.
const RoleCommentModerator Role = "comment_moderator"

// Agent is the minimal contract every registered role implements.
// The result type is shared by all current roles; evolve this interface
// when a role needs a different result shape.
type Agent interface {
	Run(ctx context.Context, input string) (moderator.Verdict, error)
}
