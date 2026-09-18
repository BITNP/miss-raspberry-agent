// Package health aggregates the readiness of the service's dependencies into a single report
// for callers (for example, another service checking whether this one can do useful work).
package health

import "context"

// Status values reported by a dependency check and by the aggregate report.
const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
)

// Queue reports how many work items are waiting for the main agent.
type Queue interface {
	Len() int
}

// Check is one dependency probe. Name labels the dependency in the report; Ready returns nil
// when the dependency is healthy, or an error explaining why it is not.
type Check struct {
	Name  string
	Ready func(ctx context.Context) error
}

// DependencyStatus is the outcome of one check.
type DependencyStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Report is the aggregate readiness of the service.
type Report struct {
	Status       string             `json:"status"`
	QueueLength  int                `json:"queue_length"`
	Dependencies []DependencyStatus `json:"dependencies"`
}

// Service runs the configured dependency checks.
type Service struct {
	queue  Queue
	checks []Check
}

// NewService creates a readiness service. The queue (normally the main agent's todo queue) may
// be nil; the checks are probed in the order given.
func NewService(queue Queue, checks ...Check) *Service {
	return &Service{queue: queue, checks: checks}
}

// Check probes every dependency and returns the aggregate report. Healthy is false when any
// dependency is down, in which case the service should not be considered ready to serve work.
func (s *Service) Check(ctx context.Context) (Report, bool) {
	report := Report{
		Status:       StatusOK,
		Dependencies: make([]DependencyStatus, 0, len(s.checks)),
	}
	if s.queue != nil {
		report.QueueLength = s.queue.Len()
	}

	healthy := true
	for _, check := range s.checks {
		status := DependencyStatus{Name: check.Name, Status: StatusOK}
		if err := check.Ready(ctx); err != nil {
			status.Status = StatusDegraded
			status.Error = err.Error()
			healthy = false
		}
		report.Dependencies = append(report.Dependencies, status)
	}
	if !healthy {
		report.Status = StatusDegraded
	}
	return report, healthy
}
