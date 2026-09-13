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
// set with an existing name replaces the previous one.
type TagSet struct {
	Name string
	Tags []Tag
}
