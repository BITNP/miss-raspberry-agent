package http_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

const tagSetBody = `{"name":"sentiment","tags":[{"name":"positive","description":"praise","apply_rule":"the text is positive"}]}`

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
	if resp.Name != "sentiment" || resp.TagCount != 1 {
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
		{"missing name", `{"tags":[{"name":"a","apply_rule":"r"}]}`, http.StatusBadRequest},
		{"empty tags", `{"name":"s","tags":[]}`, http.StatusBadRequest},
		{"missing apply rule", `{"name":"s","tags":[{"name":"a"}]}`, http.StatusBadRequest},
		{"duplicate tag names", `{"name":"s","tags":[{"name":"a","apply_rule":"r"},{"name":"a","apply_rule":"r"}]}`, http.StatusBadRequest},
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

func TestTagReturnsNotImplemented(t *testing.T) {
	router, _ := newTestRouter()
	if rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag-sets", "Bearer "+testToken, tagSetBody); rec.Code != http.StatusOK {
		t.Fatalf("register status = %d, want %d (body=%s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag", "Bearer "+testToken, `{"name":"sentiment","text":"I love it"}`)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusNotImplemented, rec.Body.String())
	}
}

func TestTaggerRoutesRequireAuth(t *testing.T) {
	router, _ := newTestRouter()
	rec := doRequest(router, http.MethodPost, "/api/v1/agents/tagger/tag-sets", "", tagSetBody)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
