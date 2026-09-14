package dto

// RegisterTagSetRequest is the body of POST /api/v1/agents/tagger/tag-sets.
type RegisterTagSetRequest struct {
	Name   string          `json:"name" binding:"required"`
	Prompt string          `json:"prompt" binding:"required"`
	Tags   []TagDefinition `json:"tags" binding:"required,min=1,dive"`
}

// TagDefinition is one tag inside a RegisterTagSetRequest.
type TagDefinition struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	ApplyRule   string `json:"apply_rule" binding:"required"`
}

// RegisterTagSetResponse is returned once a tag set has been stored.
type RegisterTagSetResponse struct {
	Name     string `json:"name"`
	TagCount int    `json:"tag_count"`
}

// TagRequest is the body of POST /api/v1/agents/tagger/tag.
type TagRequest struct {
	Name string `json:"name" binding:"required"`
	Text string `json:"text" binding:"required"`
}

// TagItem is the best-matching tag in a TagResponse.
type TagItem struct {
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

// TagResponse is returned by POST /api/v1/agents/tagger/tag. Tag is null when no
// registered tag applies to the text.
type TagResponse struct {
	Name string   `json:"name"`
	Tag  *TagItem `json:"tag"`
}
