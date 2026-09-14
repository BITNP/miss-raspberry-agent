// Package tagging owns tag-set registration and the tagger application service.
// It stores tag sets independently of the agent that will eventually apply them.
package tagging

// Tag is one label within a tag set.
type Tag struct {
	Name        string
	Description string
	ApplyRule   string
}

// TagSet is a named collection of tags. A set is keyed by Name: registering a
// set with an existing name replaces the previous one. Prompt is a general
// instruction describing the function of the set and the notice the tagger must
// follow when marking text within it.
type TagSet struct {
	Name   string
	Prompt string
	Tags   []Tag
}

// Match is the best-matching tag for a piece of text together with the tagger's
// reason for choosing it.
type Match struct {
	Tag    Tag
	Reason string
}
