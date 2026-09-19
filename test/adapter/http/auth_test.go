package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	transporthttp "miss-raspberry-agent/internal/adapter/http"
	"miss-raspberry-agent/internal/adapter/http/handler"
	"miss-raspberry-agent/internal/health"
	"miss-raspberry-agent/internal/messaging"
	"miss-raspberry-agent/internal/tagging"
	"miss-raspberry-agent/internal/tool/todo"
)

const testToken = "test-token"

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	os.Exit(m.Run())
}

// fakeSubmitter lets tests force application-service outcomes without a real queue.
type fakeSubmitter struct {
	id  string
	err error
}

func (f fakeSubmitter) Submit(context.Context, messaging.Message) (string, error) {
	return f.id, f.err
}

func newTestRouter() (*gin.Engine, *todo.Store) {
	queue := todo.NewStore()
	router := transporthttp.NewRouter(
		handler.NewMessageHandler(messaging.NewService(queue)),
		handler.NewTaggingHandler(tagging.NewService(tagging.NewStore(), &fakeTagger{})),
		handler.NewHealthHandler(health.NewService(nil)),
		testToken,
	)
	return router, queue
}

func doRequest(router *gin.Engine, method, path, authorization, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v (body=%s)", err, rec.Body.String())
	}
	return body.Error
}

func validBody() string {
	return `{"platform":"qq","target_id":"10001","content":"你好","context":"初次打招呼"}`
}

func TestHealthzIsPublic(t *testing.T) {
	router, _ := newTestRouter()
	rec := doRequest(router, http.MethodGet, "/healthz", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestTokenAuthRejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name          string
		authorization string
	}{
		{"missing header", ""},
		{"wrong token", "Bearer nope"},
		{"wrong scheme", "Token " + testToken},
		{"raw token", testToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, _ := newTestRouter()
			rec := doRequest(router, http.MethodPost, "/api/v1/agents/main/messages", tc.authorization, validBody())
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if got := decodeError(t, rec); got != "unauthorized" {
				t.Errorf("error = %q, want %q", got, "unauthorized")
			}
		})
	}
}

func TestTokenAuthAcceptsValidToken(t *testing.T) {
	router, _ := newTestRouter()
	rec := doRequest(router, http.MethodPost, "/api/v1/agents/main/messages", "Bearer "+testToken, validBody())
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
}
