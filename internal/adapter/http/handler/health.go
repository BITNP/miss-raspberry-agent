package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"miss-raspberry-agent/internal/adapter/http/dto"
	"miss-raspberry-agent/internal/health"
)

// HealthChecker reports the service's readiness to serve work.
type HealthChecker interface {
	Check(ctx context.Context) (health.Report, bool)
}

// HealthHandler serves the readiness endpoint.
type HealthHandler struct {
	checker HealthChecker
}

// NewHealthHandler creates a HealthHandler backed by the given checker.
func NewHealthHandler(checker HealthChecker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

// Check handles GET /api/v1/health: it probes the configured dependencies and returns 200 when
// they are all healthy, or 503 when any dependency is down.
func (h *HealthHandler) Check(c *gin.Context) {
	report, healthy := h.checker.Check(c.Request.Context())

	resp := dto.ReadinessResponse{
		Status:       report.Status,
		QueueLength:  report.QueueLength,
		Dependencies: make([]dto.ReadinessDependencyStatus, 0, len(report.Dependencies)),
	}
	for _, dep := range report.Dependencies {
		resp.Dependencies = append(resp.Dependencies, dto.ReadinessDependencyStatus{
			Name:   dep.Name,
			Status: dep.Status,
			Error:  dep.Error,
		})
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, resp)
}
