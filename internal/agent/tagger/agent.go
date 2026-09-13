package tagger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// taggerSystemPrompt instructs the LLM to select the single best-matching tag.
const taggerSystemPrompt = `You are a precise text tagger.
You are given a set of tags. Each tag has a name, an optional description, and an
"apply_rule" written in natural language. You are also given a text to tag.

Choose the ONE tag whose apply_rule best matches the text. If no tag applies, choose none.
Judge the text fairly regardless of the language it is written in, and write the "reason"
in the same language as the text.

Reply with ONLY this JSON object and nothing else:
{
  "name": "<name of the best-matching tag, or empty string when none applies>",
  "reason": "<one short sentence explaining why that tag applies; empty when none>"
}`

// Tagger selects the best-matching tag for a text with a chat model.
type Tagger struct {
	chat model.BaseChatModel
}

// NewTagger creates a Tagger backed by the given chat model.
func NewTagger(chat model.BaseChatModel) *Tagger {
	return &Tagger{chat: chat}
}

// Run picks the single best-matching tag for text from tags. A Result with an
// empty Name means no tag applies.
func (t *Tagger) Run(ctx context.Context, tags []Tag, text string) (Result, error) {
	if len(tags) == 0 {
		return Result{}, errors.New("tagger: at least one tag is required")
	}
	if strings.TrimSpace(text) == "" {
		return Result{}, errors.New("tagger: text is required")
	}

	resp, err := t.chat.Generate(
		ctx,
		[]*schema.Message{
			schema.SystemMessage(taggerSystemPrompt),
			schema.UserMessage(buildTagInput(tags, text)),
		},
		model.WithTemperature(0),
	)
	if err != nil {
		return Result{}, fmt.Errorf("model generate: %w", err)
	}

	result, err := parseResult(resp.Content)
	if err != nil {
		return Result{}, fmt.Errorf("parse model output: %w", err)
	}
	return normalizeResult(result, tags)
}

// buildTagInput renders the tag definitions and the target text for the model.
func buildTagInput(tags []Tag, text string) string {
	var sb strings.Builder
	sb.WriteString("Tags:\n")
	for i, tag := range tags {
		fmt.Fprintf(&sb, "%d. name: %s\n", i+1, tag.Name)
		if tag.Description != "" {
			fmt.Fprintf(&sb, "   description: %s\n", tag.Description)
		}
		fmt.Fprintf(&sb, "   apply_rule: %s\n", tag.ApplyRule)
	}
	sb.WriteString("\nText to tag:\n")
	sb.WriteString(text)
	return sb.String()
}

// normalizeResult rejects hallucinated tag names and requires a reason when a
// tag was selected. A name that is not in the candidate set is treated as no match.
func normalizeResult(r Result, tags []Tag) (Result, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.Reason = strings.TrimSpace(r.Reason)
	if r.Name == "" {
		return Result{}, nil
	}

	for _, tag := range tags {
		if tag.Name == r.Name {
			if r.Reason == "" {
				return Result{}, fmt.Errorf("reason is required when tag %q is selected", r.Name)
			}
			return r, nil
		}
	}
	// Unknown tag name: the model invented a tag, so treat it as no match.
	return Result{}, nil
}

// parseResult decodes the JSON object produced by the model, tolerating
// surrounding whitespace or markdown fences.
func parseResult(content string) (Result, error) {
	s := strings.TrimSpace(content)
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end <= start {
		return Result{}, fmt.Errorf("no JSON object found in: %.100q", content)
	}
	var r Result
	if err := json.Unmarshal([]byte(s[start:end+1]), &r); err != nil {
		return Result{}, fmt.Errorf("decode tagger JSON: %w", err)
	}
	return r, nil
}
