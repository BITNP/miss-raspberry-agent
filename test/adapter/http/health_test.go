package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	transporthttp "miss-raspberry-agent/internal/adapter/http"
	"miss-raspberry-agent/internal/adapter/http/handler"
	"miss-raspberry-agent/internal/health"
	"miss-raspberry-agent/internal/tagging"
)

// fakeQueue is the minimal health.Queue used by the health endpoint tests.
type fakeQueue struct {
	length int
}

func (q fakeQueue) Len() int { return q.length }

func newHealthTestRouter(ready func(context.Context) error) *gin.Engine {
	svc := health.NewService(fakeQueue{length: 2},
		health.Check{Name: "napcat", Ready: ready},
	)
	router := transporthttp.NewRouter(
		handler.NewMessageHandler(fakeSubmitter{}),
		handler.NewTaggingHandler(tagging.NewService(tagging.NewStore(), &fakeTagger{})),
		handler.NewHealthHandler(svc),
		testToken,
	)
	return router
}

func TestHealthRequiresAuth(t *testing.T) {
	router := newHealthTestRouter(func(context.Context) error { return nil })
	rec := doRequest(router, http.MethodGet, "/api/v1/health", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHealthReportsOK(t *testing.T) {
	router := newHealthTestRouter(func(context.Context) error { return nil })
	rec := doRequest(router, http.MethodGet, "/api/v1/health", "Bearer "+testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Status       string `json:"status"`
		QueueLength  int    `json:"queue_length"`
		Dependencies []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "ok" || resp.QueueLength != 2 {
		t.Fatalf("unexpected response: %s", rec.Body.String())
	}
	if len(resp.Dependencies) != 1 || resp.Dependencies[0].Status != "ok" {
		t.Fatalf("unexpected dependencies: %s", rec.Body.String())
	}
}

func TestHealthReportsDegraded(t *testing.T) {
	router := newHealthTestRouter(func(context.Context) error { return errors.New("no bot connected") })
	rec := doRequest(router, http.MethodGet, "/api/v1/health", "Bearer "+testToken, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}

	var resp struct {
		Status       string `json:"status"`
		Dependencies []struct {
			Error string `json:"error"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "degraded" || len(resp.Dependencies) != 1 || resp.Dependencies[0].Error == "" {
		t.Fatalf("unexpected response: %s", rec.Body.String())
	}
}
