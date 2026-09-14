// Package tagger chooses the single best-matching tag for a piece of text
// given a set of tag definitions.
package tagger

// TagSet is the tag set supplied to the tagger. Prompt is a general instruction
// describing the function of the set and the notice the tagger must follow when
// marking text within it. Tags are the candidate labels.
type TagSet struct {
	Name   string
	Prompt string
	Tags   []Tag
}

// Tag is one candidate label supplied to the tagger. ApplyRule is a
// natural-language rule describing when the tag applies.
type Tag struct {
	Name        string
	Description string
	ApplyRule   string
}

// Result is the tagger's answer. Name is empty when no tag matches.
type Result struct {
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}
