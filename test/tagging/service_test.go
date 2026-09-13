package tagging_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"miss-raspberry-agent/internal/agent/tagger"
	"miss-raspberry-agent/internal/tagging"
)

// fakeTagger lets tests control the tagger agent's answer without a model.
type fakeTagger struct {
	result tagger.Result
	err    error

	calls int
	tags  []tagger.Tag
	text  string
}

func (f *fakeTagger) Run(_ context.Context, tags []tagger.Tag, text string) (tagger.Result, error) {
	f.calls++
	f.tags = tags
	f.text = text
	return f.result, f.err
}

func sentimentSet() []tagging.Tag {
	return []tagging.Tag{
		{Name: "positive", Description: "praise", ApplyRule: "the text is positive"},
		{Name: "negative", Description: "complaint", ApplyRule: "the text is negative"},
	}
}

func TestRegisterSetStoresSet(t *testing.T) {
	fake := &fakeTagger{result: tagger.Result{Name: "positive", Reason: "praise"}}
	svc := tagging.NewService(tagging.NewStore(), fake)
	ctx := context.Background()

	if err := svc.RegisterSet(ctx, "sentiment", sentimentSet()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	match, err := svc.Tag(ctx, "sentiment", "I love it")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match == nil || match.Tag.Name != "positive" || match.Reason != "praise" {
		t.Fatalf("unexpected match: %+v", match)
	}
	if match.Tag.Description != "praise" {
		t.Errorf("match should carry the stored description, got %q", match.Tag.Description)
	}
	if fake.calls != 1 || fake.text != "I love it" || len(fake.tags) != 2 {
		t.Errorf("unexpected tagger call: calls=%d text=%q tags=%+v", fake.calls, fake.text, fake.tags)
	}
}

func TestRegisterSetReplacesSameName(t *testing.T) {
	store := tagging.NewStore()
	svc := tagging.NewService(store, &fakeTagger{})
	ctx := context.Background()

	if err := svc.RegisterSet(ctx, "sentiment", []tagging.Tag{{Name: "a", ApplyRule: "r"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := svc.RegisterSet(ctx, "sentiment", []tagging.Tag{
		{Name: "b", ApplyRule: "r"},
		{Name: "c", ApplyRule: "r"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	set, ok := store.Get("sentiment")
	if !ok {
		t.Fatal("expected set to exist")
	}
	if len(set.Tags) != 2 || set.Tags[0].Name != "b" || set.Tags[1].Name != "c" {
		t.Fatalf("expected replacement with b,c, got %+v", set.Tags)
	}
}

func TestRegisterSetRejectsDuplicateTagNames(t *testing.T) {
	svc := tagging.NewService(tagging.NewStore(), &fakeTagger{})

	err := svc.RegisterSet(context.Background(), "s", []tagging.Tag{
		{Name: "a", ApplyRule: "r"},
		{Name: "a", ApplyRule: "r"},
	})
	if !errors.Is(err, tagging.ErrDuplicateTagName) {
		t.Fatalf("expected ErrDuplicateTagName, got %v", err)
	}
}

func TestRegisterSetRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		set  string
		tags []tagging.Tag
	}{
		{"blank set name", "  ", []tagging.Tag{{Name: "a", ApplyRule: "r"}}},
		{"no tags", "s", nil},
		{"blank tag name", "s", []tagging.Tag{{Name: " ", ApplyRule: "r"}}},
		{"blank apply rule", "s", []tagging.Tag{{Name: "a", ApplyRule: " "}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := tagging.NewService(tagging.NewStore(), &fakeTagger{})
			err := svc.RegisterSet(context.Background(), tc.set, tc.tags)
			if !errors.Is(err, tagging.ErrInvalidTagSet) {
				t.Fatalf("expected ErrInvalidTagSet, got %v", err)
			}
		})
	}
}

func TestTagRejectsInvalidInput(t *testing.T) {
	fake := &fakeTagger{}
	svc := tagging.NewService(tagging.NewStore(), fake)
	ctx := context.Background()

	if err := svc.RegisterSet(ctx, "s", []tagging.Tag{{Name: "a", ApplyRule: "r"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct {
		name    string
		setName string
		text    string
		want    error
	}{
		{"blank set name", " ", "hi", tagging.ErrInvalidTagSet},
		{"blank text", "s", " ", tagging.ErrInvalidTagSet},
		{"unknown set", "missing", "hi", tagging.ErrTagSetNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Tag(ctx, tc.setName, tc.text)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
	if fake.calls != 0 {
		t.Errorf("tagger should not run on invalid input, ran %d times", fake.calls)
	}
}

func TestTagNoMatch(t *testing.T) {
	svc := tagging.NewService(tagging.NewStore(), &fakeTagger{result: tagger.Result{}})
	ctx := context.Background()

	if err := svc.RegisterSet(ctx, "sentiment", sentimentSet()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	match, err := svc.Tag(ctx, "sentiment", "nothing notable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match != nil {
		t.Fatalf("expected no match, got %+v", match)
	}
}

func TestTagUnknownModelTagIsNoMatch(t *testing.T) {
	svc := tagging.NewService(tagging.NewStore(), &fakeTagger{result: tagger.Result{Name: "invented", Reason: "x"}})
	ctx := context.Background()

	if err := svc.RegisterSet(ctx, "sentiment", sentimentSet()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	match, err := svc.Tag(ctx, "sentiment", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match != nil {
		t.Fatalf("expected unknown tag to be dropped, got %+v", match)
	}
}

func TestTagTaggerErrorIsWrapped(t *testing.T) {
	svc := tagging.NewService(tagging.NewStore(), &fakeTagger{err: errors.New("boom")})
	ctx := context.Background()

	if err := svc.RegisterSet(ctx, "sentiment", sentimentSet()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.Tag(ctx, "sentiment", "text")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped tagger error, got %v", err)
	}
}

func TestTagMissingTagger(t *testing.T) {
	svc := tagging.NewService(tagging.NewStore(), nil)

	err := svc.RegisterSet(context.Background(), "s", []tagging.Tag{{Name: "a", ApplyRule: "r"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.Tag(context.Background(), "s", "text"); err == nil {
		t.Fatal("expected error when no tagger is configured")
	}
}
