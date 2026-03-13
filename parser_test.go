package recipemd

import (
	"encoding/json"
	"testing"
)

func TestParser_WithFrontmatter_YAML(t *testing.T) {
	input := []byte(`---
title: ignored
---
# Guacamole

---

- avocado
`)
	p := NewParser(WithFrontmatter())
	recipe, err := p.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if recipe.Title != "Guacamole" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Guacamole")
	}
}

func TestParser_WithFrontmatter_TOML(t *testing.T) {
	input := []byte(`+++
title = "ignored"
+++
# Guacamole

---

- avocado
`)
	p := NewParser(WithFrontmatter())
	recipe, err := p.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if recipe.Title != "Guacamole" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Guacamole")
	}
}

func TestParser_FrontmatterWithoutOption_Fails(t *testing.T) {
	input := []byte(`---
title: ignored
---
# Guacamole

---

- avocado
`)
	_, err := NewParser().Parse(input)
	if err == nil {
		t.Fatal("expected error parsing frontmatter without WithFrontmatter")
	}
}

func TestParser_WithGFM(t *testing.T) {
	input := []byte(`# Guacamole

Check out https://example.com for more info.

---

- avocado
`)
	p := NewParser(WithGithubFormattedMarkdown())
	recipe, err := p.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if recipe.Title != "Guacamole" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Guacamole")
	}
}

func TestParser_GFM_LinkifyIngredient(t *testing.T) {
	input := []byte("# Test\n\n---\n\n- *1 cup* https://example.com/flour\n")
	p := NewParser(WithGithubFormattedMarkdown())
	recipe, err := p.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	ing := recipe.Ingredients[0]
	if ing.Link == nil {
		t.Fatal("expected link from autolinked URL, got nil")
	}
	if *ing.Link != "https://example.com/flour" {
		t.Errorf("link = %q, want %q", *ing.Link, "https://example.com/flour")
	}
	if ing.Amount == nil || ing.Amount.Factor != 1 {
		t.Errorf("amount factor = %v, want 1", ing.Amount)
	}
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"yaml", "---\nfoo: bar\n---\ncontent", "content"},
		{"toml", "+++\nfoo = 1\n+++\ncontent", "content"},
		{"no frontmatter", "# Title\n\ncontent", "# Title\n\ncontent"},
		{"unclosed fence", "---\nfoo: bar\ncontent", "---\nfoo: bar\ncontent"},
		{"fence with trailing space", "---  \nfoo: bar\n---\ncontent", "content"},
		{"extra chars on opening", "--- extra\nfoo\n---\n", "--- extra\nfoo\n---\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(stripFrontmatter([]byte(tt.in)))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParse_TitleAndDescription(t *testing.T) {
	input := []byte(`# Guacamole

Some people call it guac.

It's delicious with chips.

---

- avocado
`)
	recipe, err := NewParser().Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	b, _ := json.MarshalIndent(recipe, "", "  ")
	t.Logf("Parsed recipe:\n%s", b)

	if recipe.Title != "Guacamole" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Guacamole")
	}

	wantDesc := "Some people call it guac.\n\nIt's delicious with chips."
	if recipe.Description == nil {
		t.Fatal("Description is nil")
	}
	if *recipe.Description != wantDesc {
		t.Errorf("Description = %q, want %q", *recipe.Description, wantDesc)
	}
}

var sampleRecipe = []byte(`# Guacamole

Some people call it guac.

*sauce, vegan*

**4 Servings, 200g**

---

- *1* avocado
- *.5 teaspoon* salt
- *1 1/2 pinches* red pepper flakes
- lemon juice

---

Remove flesh from avocado and roughly mash with fork. Season to taste
with salt, pepper and lemon juice.
`)

func TestParse_FullRecipe(t *testing.T) {
	recipe, err := NewParser().Parse(sampleRecipe)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	b, _ := json.MarshalIndent(recipe, "", "  ")
	t.Logf("Parsed recipe:\n%s", b)
}

