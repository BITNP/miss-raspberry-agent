package tagging_test

import (
	"context"
	"errors"
	"testing"

	"miss-raspberry-agent/internal/tagging"
)

func TestRegisterSetStoresSet(t *testing.T) {
	svc := tagging.NewService(tagging.NewStore())

	err := svc.RegisterSet(context.Background(), "sentiment", []tagging.Tag{
		{Name: "positive", Description: "praise", ApplyRule: "the text is positive"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.Tag(context.Background(), "sentiment", "hello")
	if !errors.Is(err, tagging.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for a stored set, got %v", err)
	}
}

func TestRegisterSetReplacesSameName(t *testing.T) {
	store := tagging.NewStore()
	svc := tagging.NewService(store)
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
	svc := tagging.NewService(tagging.NewStore())

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
			svc := tagging.NewService(tagging.NewStore())
			err := svc.RegisterSet(context.Background(), tc.set, tc.tags)
			if !errors.Is(err, tagging.ErrInvalidTagSet) {
				t.Fatalf("expected ErrInvalidTagSet, got %v", err)
			}
		})
	}
}

func TestTagRejectsInvalidInput(t *testing.T) {
	svc := tagging.NewService(tagging.NewStore())
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
}
