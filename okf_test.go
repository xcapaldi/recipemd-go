package recipemd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const okfRecipe = `---
type: Recipe
title: Guacamole
description: A simple avocado dip.
resource: https://example.com/recipes/guacamole
tags: [mexican, dip]
timestamp: 2026-05-28T14:30:00Z
author: Jane Doe
difficulty: easy
---
# Guacamole

---

- *2* avocados
`

func TestParser_WithOKF_FullFrontmatter(t *testing.T) {
	p := NewParser(WithOKF())
	recipe, err := p.Parse(strings.NewReader(okfRecipe))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if recipe.Title != "Guacamole" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Guacamole")
	}
	o := recipe.OKF
	if o == nil {
		t.Fatal("Recipe.OKF is nil, want parsed OKF metadata")
	}
	if o.Type != "Recipe" {
		t.Errorf("OKF.Type = %q, want %q", o.Type, "Recipe")
	}
	if o.Title == nil || *o.Title != "Guacamole" {
		t.Errorf("OKF.Title = %v, want %q", o.Title, "Guacamole")
	}
	if o.Description == nil || *o.Description != "A simple avocado dip." {
		t.Errorf("OKF.Description = %v, want %q", o.Description, "A simple avocado dip.")
	}
	if o.Resource == nil || *o.Resource != "https://example.com/recipes/guacamole" {
		t.Errorf("OKF.Resource = %v, want %q", o.Resource, "https://example.com/recipes/guacamole")
	}
	if len(o.Tags) != 2 || o.Tags[0] != "mexican" || o.Tags[1] != "dip" {
		t.Errorf("OKF.Tags = %v, want [mexican dip]", o.Tags)
	}
	want := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)
	if o.Timestamp == nil || !o.Timestamp.Equal(want) {
		t.Errorf("OKF.Timestamp = %v, want %v", o.Timestamp, want)
	}
	if len(o.Extensions) != 2 {
		t.Errorf("OKF.Extensions = %v, want 2 entries", o.Extensions)
	}
	if v, ok := o.Extensions["author"].(string); !ok || v != "Jane Doe" {
		t.Errorf("OKF.Extensions[author] = %v, want %q", o.Extensions["author"], "Jane Doe")
	}
	if v, ok := o.Extensions["difficulty"].(string); !ok || v != "easy" {
		t.Errorf("OKF.Extensions[difficulty] = %v, want %q", o.Extensions["difficulty"], "easy")
	}
}

func TestParser_WithOKF_TypeOnly(t *testing.T) {
	input := `---
type: Recipe
---
# Guacamole

---

- avocado
`
	recipe, err := NewParser(WithOKF()).Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	o := recipe.OKF
	if o == nil {
		t.Fatal("Recipe.OKF is nil, want parsed OKF metadata")
	}
	if o.Type != "Recipe" {
		t.Errorf("OKF.Type = %q, want %q", o.Type, "Recipe")
	}
	if o.Title != nil || o.Description != nil || o.Resource != nil || o.Timestamp != nil {
		t.Errorf("optional fields not nil: %+v", o)
	}
	if o.Tags == nil || len(o.Tags) != 0 {
		t.Errorf("OKF.Tags = %v, want empty non-nil slice", o.Tags)
	}
	if o.Extensions == nil || len(o.Extensions) != 0 {
		t.Errorf("OKF.Extensions = %v, want empty non-nil map", o.Extensions)
	}
}

func TestParser_WithOKF_NoFrontmatter(t *testing.T) {
	input := `# Guacamole

---

- avocado
`
	recipe, err := NewParser(WithOKF()).Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if recipe.OKF != nil {
		t.Errorf("Recipe.OKF = %+v, want nil without frontmatter", recipe.OKF)
	}
}

func TestParser_WithOKF_MissingType(t *testing.T) {
	input := `---
title: Guacamole
---
# Guacamole

---

- avocado
`
	_, err := NewParser(WithOKF()).Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for OKF frontmatter without type")
	}
	if !strings.Contains(err.Error(), `missing required "type" field`) {
		t.Errorf("error = %v, want missing type message", err)
	}
}

func TestParser_WithOKF_InvalidYAML(t *testing.T) {
	input := `---
type: [unclosed
---
# Guacamole

---

- avocado
`
	_, err := NewParser(WithOKF()).Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for invalid YAML frontmatter")
	}
	if !strings.Contains(err.Error(), "invalid OKF frontmatter") {
		t.Errorf("error = %v, want invalid frontmatter message", err)
	}
}

func TestParser_WithOKF_TOMLFrontmatterIgnored(t *testing.T) {
	input := `+++
title = "ignored"
+++
# Guacamole

---

- avocado
`
	// WithOKF only recognises YAML frontmatter; combined with
	// WithFrontmatter the TOML block is stripped and OKF stays nil.
	recipe, err := NewParser(WithOKF(), WithFrontmatter()).Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if recipe.OKF != nil {
		t.Errorf("Recipe.OKF = %+v, want nil for TOML frontmatter", recipe.OKF)
	}
	if recipe.Title != "Guacamole" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Guacamole")
	}
}

func TestParser_WithOKF_EmptyFrontmatter(t *testing.T) {
	input := `---
---
# Guacamole

---

- avocado
`
	recipe, err := NewParser(WithOKF()).Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if recipe.OKF != nil {
		t.Errorf("Recipe.OKF = %+v, want nil for empty frontmatter", recipe.OKF)
	}
}

func TestParser_WithOKF_MarkdownRoundTrip(t *testing.T) {
	p := NewParser(WithOKF())
	recipe, err := p.Parse(strings.NewReader(okfRecipe))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	rendered := p.RenderMarkdown(recipe, 2)
	if !strings.HasPrefix(rendered, "---\ntype: Recipe\n") {
		t.Errorf("rendered markdown does not start with OKF frontmatter:\n%s", rendered)
	}

	reparsed, err := p.Parse(strings.NewReader(rendered))
	if err != nil {
		t.Fatalf("Parse of rendered markdown error: %v\n%s", err, rendered)
	}
	got, want := reparsed.OKF, recipe.OKF
	if got == nil {
		t.Fatalf("reparsed OKF is nil; rendered:\n%s", rendered)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Errorf("round-tripped OKF = %s, want %s", gotJSON, wantJSON)
	}
	if reparsed.Title != recipe.Title {
		t.Errorf("round-tripped Title = %q, want %q", reparsed.Title, recipe.Title)
	}
}

func TestParser_WithOKF_RenderJSON(t *testing.T) {
	p := NewParser(WithOKF())
	recipe, err := p.Parse(strings.NewReader(okfRecipe))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	data, err := p.RenderJSON(recipe)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	okf, ok := decoded["okf"].(map[string]any)
	if !ok {
		t.Fatalf("JSON missing okf object: %s", data)
	}
	if okf["type"] != "Recipe" {
		t.Errorf("okf.type = %v, want %q", okf["type"], "Recipe")
	}
	ext, ok := okf["extensions"].(map[string]any)
	if !ok || ext["author"] != "Jane Doe" {
		t.Errorf("okf.extensions = %v, want author preserved", okf["extensions"])
	}
}

func TestParser_WithoutOKF_JSONOmitsOKF(t *testing.T) {
	input := `# Guacamole

---

- avocado
`
	p := NewParser()
	recipe, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	data, err := p.RenderJSON(recipe)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}
	if bytes.Contains(data, []byte(`"okf"`)) {
		t.Errorf("JSON output contains okf key for recipe without OKF: %s", data)
	}
}

func TestParseOKF_NestedExtensions(t *testing.T) {
	fm := []byte(`type: Recipe
nutrition:
  calories: 250
  fat: 22g
sources:
  - https://example.com/a
  - https://example.com/b
`)
	o, err := ParseOKF(fm)
	if err != nil {
		t.Fatalf("ParseOKF error: %v", err)
	}
	nutrition, ok := o.Extensions["nutrition"].(map[string]any)
	if !ok {
		t.Fatalf("Extensions[nutrition] = %T, want map", o.Extensions["nutrition"])
	}
	if nutrition["fat"] != "22g" {
		t.Errorf("nutrition.fat = %v, want %q", nutrition["fat"], "22g")
	}
	sources, ok := o.Extensions["sources"].([]any)
	if !ok || len(sources) != 2 {
		t.Fatalf("Extensions[sources] = %v, want 2-element list", o.Extensions["sources"])
	}
	// Nested extensions must survive JSON encoding (spec: consumers
	// preserve unknown keys when round-tripping).
	if _, err := json.Marshal(o); err != nil {
		t.Errorf("json.Marshal of OKF with nested extensions: %v", err)
	}
}
