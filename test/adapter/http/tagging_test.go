package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"

	transporthttp "miss-raspberry-agent/internal/adapter/http"
	"miss-raspberry-agent/internal/adapter/http/handler"
	"miss-raspberry-agent/internal/agent/tagger"
	"miss-raspberry-agent/internal/tagging"
)

// fakeTagger lets tests force the tagger agent's answer without a model.
type fakeTagger struct {
	result tagger.Result
	err    error
}

func (f *fakeTagger) Run(context.Context, tagger.TagSet, string) (tagger.Result, error) {
	return f.result, f.err
}

func newTaggerTestRouter() (*gin.Engine, *fakeTagger) {
	fake := &fakeTagger{}
	router := transporthttp.NewRouter(
		handler.NewMessageHandler(fakeSubmitter{}),
		handler.NewTaggingHandler(tagging.NewService(tagging.NewStore(), fake)),
		testToken,
	)
	return router, fake
}

const tagSetBody = `{"name":"sentiment","prompt":"Tag the text by its sentiment.","tags":[{"name":"positive","description":"praise","apply_rule":"the text is positive"},{"name":"negative","description":"complaint","apply_rule":"the text is negative"}]}`

func TestRegisterTagSetAccepted(t *testing.T) {
	router, _ := newTestRouter()
	rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag-sets", "Bearer "+testToken, tagSetBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Name     string `json:"name"`
		TagCount int    `json:"tag_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Name != "sentiment" || resp.TagCount != 2 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRegisterTagSetRejectsBadRequests(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"malformed json", `{"name":`, http.StatusBadRequest},
		{"missing name", `{"prompt":"p","tags":[{"name":"a","apply_rule":"r"}]}`, http.StatusBadRequest},
		{"missing prompt", `{"name":"s","tags":[{"name":"a","apply_rule":"r"}]}`, http.StatusBadRequest},
		{"blank prompt", `{"name":"s","prompt":" ","tags":[{"name":"a","apply_rule":"r"}]}`, http.StatusBadRequest},
		{"empty tags", `{"name":"s","prompt":"p","tags":[]}`, http.StatusBadRequest},
		{"missing apply rule", `{"name":"s","prompt":"p","tags":[{"name":"a"}]}`, http.StatusBadRequest},
		{"duplicate tag names", `{"name":"s","prompt":"p","tags":[{"name":"a","apply_rule":"r"},{"name":"a","apply_rule":"r"}]}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, _ := newTestRouter()
			rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag-sets", "Bearer "+testToken, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tc.want, rec.Body.String())
			}
			if got := decodeError(t, rec); got == "" {
				t.Error("expected a non-empty error message")
			}
		})
	}
}

func TestTagUnknownSetReturnsNotFound(t *testing.T) {
	router, _ := newTestRouter()
	rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag", "Bearer "+testToken, `{"name":"missing","text":"hi"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestTagReturnsBestMatch(t *testing.T) {
	router, fake := newTaggerTestRouter()
	if rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag-sets", "Bearer "+testToken, tagSetBody); rec.Code != http.StatusOK {
		t.Fatalf("register status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	fake.result = tagger.Result{Name: "positive", Reason: "expresses praise"}

	rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag", "Bearer "+testToken, `{"name":"sentiment","text":"I love it"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Name string `json:"name"`
		Tag  *struct {
			Name   string `json:"name"`
			Reason string `json:"reason"`
		} `json:"tag"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Name != "sentiment" || resp.Tag == nil {
		t.Fatalf("unexpected response: %s", rec.Body.String())
	}
	if resp.Tag.Name != "positive" || resp.Tag.Reason != "expresses praise" {
		t.Fatalf("unexpected tag: %+v", resp.Tag)
	}
}

func TestTagNoMatchReturnsNull(t *testing.T) {
	router, _ := newTaggerTestRouter()
	if rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag-sets", "Bearer "+testToken, tagSetBody); rec.Code != http.StatusOK {
		t.Fatalf("register status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag", "Bearer "+testToken, `{"name":"sentiment","text":"nothing notable"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Name string          `json:"name"`
		Tag  json.RawMessage `json:"tag"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if string(resp.Tag) != "null" {
		t.Fatalf("expected null tag, got %s", rec.Body.String())
	}
}

func TestTaggerRoutesRequireAuth(t *testing.T) {
	router, _ := newTestRouter()
	rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag-sets", "", tagSetBody)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// stubChatModel is a minimal model.BaseChatModel returning a fixed answer, used
// to drive the real tagger agent end-to-end.
type stubChatModel struct {
	content string
}

func (m *stubChatModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage(m.content, nil), nil
}

func (m *stubChatModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("stream not implemented")
}

// TestTaggerAgentWiredThroughHandler registers a set and tags text using the real
// tagger agent (backed by a stub model), real service, real handler, and real
// router. It proves the agent and the handler are wired together in the server.
func TestTaggerAgentWiredThroughHandler(t *testing.T) {
	agent := tagger.NewTagger(&stubChatModel{content: `{"name":"positive","reason":"expresses praise"}`})
	service := tagging.NewService(tagging.NewStore(), agent)
	router := transporthttp.NewRouter(
		handler.NewMessageHandler(fakeSubmitter{}),
		handler.NewTaggingHandler(service),
		testToken,
	)

	if rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag-sets", "Bearer "+testToken, tagSetBody); rec.Code != http.StatusOK {
		t.Fatalf("register status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag", "Bearer "+testToken, `{"name":"sentiment","text":"I love it"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("tag status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Name string `json:"name"`
		Tag  *struct {
			Name   string `json:"name"`
			Reason string `json:"reason"`
		} `json:"tag"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Tag == nil || resp.Tag.Name != "positive" || resp.Tag.Reason != "expresses praise" {
		t.Fatalf("expected positive match from the real agent, got %s", rec.Body.String())
	}
}
