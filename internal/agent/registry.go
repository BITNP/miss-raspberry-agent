package agent

import (
	"fmt"

	"github.com/cloudwego/eino/components/model"

	"miss-raspberry-agent/internal/agent/moderator"
)

var _ Agent = (*moderator.Moderator)(nil)

// Registry builds every supported agent once at startup and hands them out
// by role. The set of agents is fixed after construction, so concurrent Get
// calls are safe. Construction failures should abort startup.
type Registry struct {
	agents map[Role]Agent
}

// NewRegistry constructs all known agents with the given shared chat model.
// It must be called once at startup, before serving requests.
func NewRegistry(chat model.BaseChatModel) (*Registry, error) {
	return &Registry{
		agents: map[Role]Agent{
			RoleCommentModerator: moderator.NewModerator(chat),
		},
	}, nil
}

// Get returns the agent registered under the given role.
func (r *Registry) Get(role Role) (Agent, error) {
	a, ok := r.agents[role]
	if !ok {
		return nil, fmt.Errorf("unknown agent role %q", role)
	}
	return a, nil
}
