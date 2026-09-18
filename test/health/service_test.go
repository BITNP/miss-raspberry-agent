package health_test

import (
	"context"
	"errors"
	"testing"

	"miss-raspberry-agent/internal/health"
)

// fakeQueue lets tests control the reported pending-work count.
type fakeQueue struct {
	length int
}

func (q fakeQueue) Len() int { return q.length }

func TestCheckReportsHealthy(t *testing.T) {
	svc := health.NewService(fakeQueue{length: 3},
		health.Check{Name: "napcat", Ready: func(context.Context) error { return nil }},
	)

	report, healthy := svc.Check(context.Background())
	if !healthy {
		t.Fatalf("healthy = false, want true (report=%+v)", report)
	}
	if report.Status != health.StatusOK {
		t.Errorf("status = %q, want %q", report.Status, health.StatusOK)
	}
	if report.QueueLength != 3 {
		t.Errorf("queue length = %d, want 3", report.QueueLength)
	}
	if len(report.Dependencies) != 1 || report.Dependencies[0].Status != health.StatusOK {
		t.Errorf("unexpected dependencies: %+v", report.Dependencies)
	}
}

func TestCheckReportsDegradedWhenDependencyFails(t *testing.T) {
	svc := health.NewService(nil,
		health.Check{Name: "napcat", Ready: func(context.Context) error { return nil }},
		health.Check{Name: "directory", Ready: func(context.Context) error { return errors.New("not loaded") }},
	)

	report, healthy := svc.Check(context.Background())
	if healthy {
		t.Fatal("healthy = true, want false")
	}
	if report.Status != health.StatusDegraded {
		t.Errorf("status = %q, want %q", report.Status, health.StatusDegraded)
	}

	var failed health.DependencyStatus
	for _, dep := range report.Dependencies {
		if dep.Name == "directory" {
			failed = dep
		}
	}
	if failed.Status != health.StatusDegraded || failed.Error != "not loaded" {
		t.Errorf("unexpected failed dependency: %+v", failed)
	}
}
