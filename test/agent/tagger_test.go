package agent_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"miss-raspberry-agent/internal/agent/tagger"
)

func testTags() []tagger.Tag {
	return []tagger.Tag{
		{Name: "positive", Description: "praise", ApplyRule: "the text is positive"},
		{Name: "negative", Description: "complaint", ApplyRule: "the text is negative"},
	}
}

func newTaggerForTest(content string, err error) (*tagger.Tagger, *fakeChatModel) {
	fake := &fakeChatModel{content: content, err: err}
	return tagger.NewTagger(fake), fake
}

func TestTaggerRunReturnsBestMatch(t *testing.T) {
	tg, fake := newTaggerForTest(`{"name":"positive","reason":"expresses praise"}`, nil)

	result, err := tg.Run(context.Background(), testTags(), "I love this update")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "positive" || result.Reason != "expresses praise" {
		t.Fatalf("unexpected result: %+v", result)
	}

	if fake.calls != 1 {
		t.Fatalf("expected 1 model call, got %d", fake.calls)
	}
	if len(fake.messages) != 2 ||
		fake.messages[0].Role != schema.System ||
		fake.messages[1].Role != schema.User {
		t.Fatalf("unexpected prompt messages: %+v", fake.messages)
	}
	if !strings.Contains(fake.messages[1].Content, "the text is positive") ||
		!strings.Contains(fake.messages[1].Content, "I love this update") {
		t.Errorf("user prompt missing tag rules or text: %q", fake.messages[1].Content)
	}

	opts := model.GetCommonOptions(nil, fake.options...)
	if opts.Temperature == nil || *opts.Temperature != 0 {
		t.Errorf("expected temperature 0, got %+v", opts.Temperature)
	}
}

func TestTaggerRunNoMatch(t *testing.T) {
	tg, _ := newTaggerForTest(`{"name":"","reason":""}`, nil)

	result, err := tg.Run(context.Background(), testTags(), "no opinion here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "" {
		t.Fatalf("expected no match, got %+v", result)
	}
}

func TestTaggerRunIgnoresUnknownTag(t *testing.T) {
	tg, _ := newTaggerForTest(`{"name":"hallucinated","reason":"made up"}`, nil)

	result, err := tg.Run(context.Background(), testTags(), "some text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "" {
		t.Fatalf("expected unknown tag to be dropped, got %+v", result)
	}
}

func TestTaggerRunRequiresReasonWhenSelected(t *testing.T) {
	tg, _ := newTaggerForTest(`{"name":"positive","reason":""}`, nil)

	_, err := tg.Run(context.Background(), testTags(), "some text")
	if err == nil || !strings.Contains(err.Error(), "reason is required") {
		t.Fatalf("expected reason-required error, got %v", err)
	}
}

func TestTaggerRunParsesFencedOutput(t *testing.T) {
	fenced := "```json\n" + `{"name":"negative","reason":"complaint"}` + "\n```"
	tg, _ := newTaggerForTest(fenced, nil)

	result, err := tg.Run(context.Background(), testTags(), "this is terrible")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "negative" {
		t.Fatalf("expected negative match from fenced JSON, got %+v", result)
	}
}

func TestTaggerRunRejectsInvalidInput(t *testing.T) {
	tg, _ := newTaggerForTest(`{"name":"positive","reason":"praise"}`, nil)

	if _, err := tg.Run(context.Background(), nil, "text"); err == nil {
		t.Error("expected error for no tags")
	}
	if _, err := tg.Run(context.Background(), testTags(), "  "); err == nil {
		t.Error("expected error for blank text")
	}
}

func TestTaggerRunInvalidModelOutput(t *testing.T) {
	tests := []struct {
		name    string
		content string
		genErr  error
	}{
		{"not json", "I cannot decide.", nil},
		{"malformed json", `{"name":"positive",`, nil},
		{"model error", "", errors.New("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg, fake := newTaggerForTest(tt.content, tt.genErr)

			if _, err := tg.Run(context.Background(), testTags(), "some text"); err == nil {
				t.Fatal("expected error")
			}
			if fake.calls != 1 {
				t.Errorf("expected exactly 1 model call, got %d", fake.calls)
			}
		})
	}
}
