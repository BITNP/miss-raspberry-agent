package tagging

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidTagSet is returned when a tag set or tag definition is malformed.
var ErrInvalidTagSet = errors.New("invalid tag set")

// ErrDuplicateTagName is returned when a tag set contains two tags with the same name.
var ErrDuplicateTagName = errors.New("duplicate tag name")

// ErrTagSetNotFound is returned when tagging references an unregistered set.
var ErrTagSetNotFound = errors.New("tag set not found")

// ErrNotImplemented is returned while the tagger agent is not built yet.
var ErrNotImplemented = errors.New("tagger agent not implemented yet")

// Service registers tag sets and (eventually) tags text against them.
type Service struct {
	store *Store
}

// NewService creates a tagging service backed by the given store.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// RegisterSet validates and stores a tag set. A set with the same name is
// replaced. Tags with duplicate names within the request are rejected.
func (s *Service) RegisterSet(ctx context.Context, name string, tags []Tag) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.store == nil {
		return errors.New("tagging: no store configured")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidTagSet)
	}
	if len(tags) == 0 {
		return fmt.Errorf("%w: at least one tag is required", ErrInvalidTagSet)
	}

	seen := make(map[string]struct{}, len(tags))
	cleaned := make([]Tag, 0, len(tags))
	for i, tag := range tags {
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

	s.store.Replace(TagSet{Name: name, Tags: cleaned})
	return nil
}

// Tag applies the named tag set to text. The tagger agent does not exist yet,
// so after validating the request and confirming the set exists this returns
// ErrNotImplemented.
//
// TODO: invoke the tagger agent with the set's tags and text, and return the
// tags that apply.
func (s *Service) Tag(ctx context.Context, setName, text string) ([]Tag, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, errors.New("tagging: no store configured")
	}

	setName = strings.TrimSpace(setName)
	if setName == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidTagSet)
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("%w: text is required", ErrInvalidTagSet)
	}
	if _, ok := s.store.Get(setName); !ok {
		return nil, fmt.Errorf("%w: %q", ErrTagSetNotFound, setName)
	}

	return nil, ErrNotImplemented
}
