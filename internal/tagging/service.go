package tagging

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"miss-raspberry-agent/internal/agent/tagger"
)

// ErrInvalidTagSet is returned when a tag set or tag definition is malformed.
var ErrInvalidTagSet = errors.New("invalid tag set")

// ErrDuplicateTagName is returned when a tag set contains two tags with the same name.
var ErrDuplicateTagName = errors.New("duplicate tag name")

// ErrTagSetNotFound is returned when tagging references an unregistered set.
var ErrTagSetNotFound = errors.New("tag set not found")

// Tagger selects the best-matching tag for a text. It is satisfied by
// agent/tagger.Tagger and is defined here, next to its consumer.
type Tagger interface {
	Run(ctx context.Context, set tagger.TagSet, text string) (tagger.Result, error)
}

// Service registers tag sets and tags text against them.
type Service struct {
	store  *Store
	tagger Tagger
}

// NewService creates a tagging service backed by the given store and tagger.
func NewService(store *Store, tg Tagger) *Service {
	return &Service{store: store, tagger: tg}
}

// RegisterSet validates and stores a tag set. A set with the same name is
// replaced. Tags with duplicate names within the request are rejected.
func (s *Service) RegisterSet(ctx context.Context, set TagSet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.store == nil {
		return errors.New("tagging: no store configured")
	}

	set.Name = strings.TrimSpace(set.Name)
	if set.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidTagSet)
	}
	set.Prompt = strings.TrimSpace(set.Prompt)
	if set.Prompt == "" {
		return fmt.Errorf("%w: prompt is required", ErrInvalidTagSet)
	}
	if len(set.Tags) == 0 {
		return fmt.Errorf("%w: at least one tag is required", ErrInvalidTagSet)
	}

	seen := make(map[string]struct{}, len(set.Tags))
	cleaned := make([]Tag, 0, len(set.Tags))
	for i, tag := range set.Tags {
		tag.Name = strings.TrimSpace(tag.Name)
		tag.Description = strings.TrimSpace(tag.Description)
		tag.ApplyRule = strings.TrimSpace(tag.ApplyRule)
		if tag.Name == "" {
			return fmt.Errorf("%w: tag %d name is required", ErrInvalidTagSet, i)
		}
		if tag.ApplyRule == "" {
			return fmt.Errorf("%w: tag %q apply_rule is required", ErrInvalidTagSet, tag.Name)
		}
		if _, ok := seen[tag.Name]; ok {
			return fmt.Errorf("%w: tag %q appears more than once", ErrDuplicateTagName, tag.Name)
		}
		seen[tag.Name] = struct{}{}
		cleaned = append(cleaned, tag)
	}
	set.Tags = cleaned

	s.store.Replace(set)
	return nil
}

// Tag applies the named tag set to text and returns the single best-matching
// tag. It returns (nil, nil) when no tag applies.
func (s *Service) Tag(ctx context.Context, setName, text string) (*Match, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, errors.New("tagging: no store configured")
	}
	if s.tagger == nil {
		return nil, errors.New("tagging: no tagger configured")
	}

	setName = strings.TrimSpace(setName)
	if setName == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidTagSet)
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("%w: text is required", ErrInvalidTagSet)
	}
	set, ok := s.store.Get(setName)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrTagSetNotFound, setName)
	}

	result, err := s.tagger.Run(ctx, toAgentSet(set), text)
	if err != nil {
		return nil, fmt.Errorf("tagger run: %w", err)
	}
	if result.Name == "" {
		return nil, nil
	}

	for _, tag := range set.Tags {
		if tag.Name == result.Name {
			return &Match{Tag: tag, Reason: result.Reason}, nil
		}
	}
	return nil, nil
}

// toAgentSet converts a stored tag set into the tagger agent's input type.
func toAgentSet(set TagSet) tagger.TagSet {
	out := tagger.TagSet{
		Name:   set.Name,
		Prompt: set.Prompt,
		Tags:   make([]tagger.Tag, 0, len(set.Tags)),
	}
	for _, tag := range set.Tags {
		out.Tags = append(out.Tags, tagger.Tag{
			Name:        tag.Name,
			Description: tag.Description,
			ApplyRule:   tag.ApplyRule,
		})
	}
	return out
}
