package recipemd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

// OKF holds the metadata fields defined by the Google Open Knowledge Format
// (OKF) specification, parsed from a document's YAML frontmatter block.
//
// Type is always present; it is the only field the OKF specification
// requires. The recommended fields Title, Description, Resource, and
// Timestamp are optional and represented as pointers; a nil pointer means
// the field was absent in the frontmatter. Tags and Extensions are
// initialised to empty (non-nil) values by [ParseOKF].
//
// See https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
type OKF struct {
	// Type is the required short string identifying the kind of concept
	// (e.g. "Recipe", "Playbook"). Type values are not registered centrally.
	Type string `json:"type"`
	// Title is the optional human-readable display name. Nil when absent.
	Title *string `json:"title"`
	// Description is the optional single-sentence summary of the concept.
	// Nil when absent.
	Description *string `json:"description"`
	// Resource is the optional URI uniquely identifying the underlying
	// asset the concept describes. Nil when absent.
	Resource *string `json:"resource"`
	// Tags is the optional list of short strings for cross-cutting
	// categorization.
	Tags []string `json:"tags"`
	// Timestamp is the optional ISO 8601 datetime of the last meaningful
	// change. Nil when absent.
	Timestamp *time.Time `json:"timestamp"`
	// Extensions holds all producer-defined frontmatter keys beyond the
	// fields named by the specification. The OKF spec requires consumers to
	// preserve unknown keys, so they are retained here verbatim.
	Extensions map[string]any `json:"extensions"`
}

// okfFieldNames are the frontmatter keys named by the OKF specification, in
// the spec's priority order. Any other key is a producer extension.
var okfFieldNames = []string{"type", "title", "description", "resource", "tags", "timestamp"}

// ParseOKF parses a YAML frontmatter block (without its --- fences) into an
// [OKF] value.
//
// The block must be valid YAML and, per the OKF specification, must contain
// the required "type" field. All keys other than the six spec-defined fields
// are preserved verbatim in [OKF.Extensions]. An empty or all-comment block
// yields (nil, nil), meaning no OKF metadata is present.
//
// Most callers will not use ParseOKF directly: create a parser with the
// [WithOKF] option and read [Recipe.OKF] instead.
func ParseOKF(frontmatter []byte) (*OKF, error) {
	var known struct {
		Type        *string    `yaml:"type"`
		Title       *string    `yaml:"title"`
		Description *string    `yaml:"description"`
		Resource    *string    `yaml:"resource"`
		Tags        []string   `yaml:"tags"`
		Timestamp   *time.Time `yaml:"timestamp"`
	}
	if err := yaml.Unmarshal(frontmatter, &known); err != nil {
		return nil, fmt.Errorf("invalid OKF frontmatter: %w", err)
	}
	var all map[string]any
	if err := yaml.Unmarshal(frontmatter, &all); err != nil {
		return nil, fmt.Errorf("invalid OKF frontmatter: %w", err)
	}
	if len(all) == 0 {
		return nil, nil
	}
	if known.Type == nil || strings.TrimSpace(*known.Type) == "" {
		return nil, fmt.Errorf("OKF frontmatter missing required \"type\" field")
	}

	o := &OKF{
		Type:        *known.Type,
		Title:       known.Title,
		Description: known.Description,
		Resource:    known.Resource,
		Tags:        known.Tags,
		Timestamp:   known.Timestamp,
		Extensions:  map[string]any{},
	}
	if o.Tags == nil {
		o.Tags = []string{}
	}
	for key, value := range all {
		known := false
		for _, name := range okfFieldNames {
			if key == name {
				known = true
				break
			}
		}
		if !known {
			o.Extensions[key] = value
		}
	}
	return o, nil
}

// renderFrontmatter serialises the OKF metadata back to a fenced YAML
// frontmatter block. Spec-defined fields appear first in the spec's priority
// order, followed by extension keys sorted alphabetically for deterministic
// output.
func (o *OKF) renderFrontmatter() string {
	ms := yaml.MapSlice{{Key: "type", Value: o.Type}}
	if o.Title != nil {
		ms = append(ms, yaml.MapItem{Key: "title", Value: *o.Title})
	}
	if o.Description != nil {
		ms = append(ms, yaml.MapItem{Key: "description", Value: *o.Description})
	}
	if o.Resource != nil {
		ms = append(ms, yaml.MapItem{Key: "resource", Value: *o.Resource})
	}
	if len(o.Tags) > 0 {
		ms = append(ms, yaml.MapItem{Key: "tags", Value: o.Tags})
	}
	if o.Timestamp != nil {
		ms = append(ms, yaml.MapItem{Key: "timestamp", Value: o.Timestamp.Format(time.RFC3339)})
	}
	keys := make([]string, 0, len(o.Extensions))
	for key := range o.Extensions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		ms = append(ms, yaml.MapItem{Key: key, Value: o.Extensions[key]})
	}

	data, err := yaml.Marshal(ms)
	if err != nil {
		return ""
	}
	return "---\n" + string(data) + "---\n"
}
