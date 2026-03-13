package recipemd

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
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

func TestParseAmountString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		wantF    float64
		wantUnit *string
		wantErr  bool
	}{
		{"integer", "5", 5, nil, false},
		{"integer with unit", "5 cups", 5, new("cups"), false},
		{"decimal dot", "1.5 ml", 1.5, new("ml"), false},
		{"decimal comma", "1,5 ml", 1.5, new("ml"), false},
		{"leading decimal", ".5", 0.5, nil, false},
		{"fraction", "1/2", 0.5, nil, false},
		{"fraction with unit", "1/2 cup", 0.5, new("cup"), false},
		{"improper fraction", "1 1/2", 1.5, nil, false},
		{"improper fraction with unit", "2 1/4 cups", 2.25, new("cups"), false},
		{"vulgar half", "½", 0.5, nil, false},
		{"vulgar quarter", "¼ cup", 0.25, new("cup"), false},
		{"improper vulgar", "1 ½", 1.5, nil, false},
		{"negative", "-3 oz", -3, new("oz"), false},
		{"negative fraction", "-1/2", -0.5, nil, false},
		{"whitespace trimmed", "  5  cups  ", 5, new("cups"), false},
		{"unit only", "cups", 0, nil, true},
		{"empty string", "", 0, nil, false},
		{"fraction spaces", "1 / 2", 0.5, nil, false},
		{"zero denominator falls to integer", "1/0", 1, new("/0"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			amt, err := ParseAmountString(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(amt.Factor-tt.wantF) > 1e-9 {
				t.Errorf("Factor = %v, want %v", amt.Factor, tt.wantF)
			}
			if tt.wantUnit == nil && amt.Unit != nil {
				t.Errorf("Unit = %q, want nil", *amt.Unit)
			}
			if tt.wantUnit != nil {
				if amt.Unit == nil {
					t.Errorf("Unit = nil, want %q", *tt.wantUnit)
				} else if *amt.Unit != *tt.wantUnit {
					t.Errorf("Unit = %q, want %q", *amt.Unit, *tt.wantUnit)
				}
			}
		})
	}
}

func TestSplitList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "a, b, c", []string{"a", "b", "c"}},
		{"decimal comma preserved", "1,5 cups, 2,5 oz", []string{"1,5 cups", "2,5 oz"}},
		{"empty parts skipped", "a,, b", []string{"a", "b"}},
		{"single item", "hello", []string{"hello"}},
		{"empty string", "", nil},
		{"whitespace only", "  ,  ,  ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := splitList(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseTags(t *testing.T) {
	t.Parallel()
	got := parseTags("sauce, vegan, easy")
	want := []string{"sauce", "vegan", "easy"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseYields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{"single yield", "4 servings", 1, false},
		{"multiple yields", "4 servings, 200g", 2, false},
		{"unitless", "4", 1, false},
		{"invalid", "cups", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			yields, err := parseYields(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(yields) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(yields), tt.wantLen)
			}
		})
	}
}

func TestStripFrontmatter_Extended(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"yaml", "---\nfoo: bar\n---\ncontent", "content"},
		{"toml", "+++\nfoo = 1\n+++\ncontent", "content"},
		{"no frontmatter", "# Title\n\ncontent", "# Title\n\ncontent"},
		{"unclosed fence", "---\nfoo: bar\ncontent", "---\nfoo: bar\ncontent"},
		{"trailing space on fence", "---  \nfoo: bar\n---\ncontent", "content"},
		{"extra chars on opening", "--- extra\nfoo\n---\n", "--- extra\nfoo\n---\n"},
		{"short input", "ab", "ab"},
		{"empty closing", "---\n---\n", ""},
		{"closing at eof", "---\nfoo\n---", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(stripFrontmatter([]byte(tt.in)))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewParser(t *testing.T) {
	t.Parallel()
	t.Run("default", func(t *testing.T) {
		t.Parallel()
		p := NewParser()
		if p.Frontmatter {
			t.Error("Frontmatter should be false by default")
		}
		if p.hasTaskList {
			t.Error("hasTaskList should be false by default")
		}
	})
	t.Run("with frontmatter", func(t *testing.T) {
		t.Parallel()
		p := NewParser(WithFrontmatter())
		if !p.Frontmatter {
			t.Error("Frontmatter should be true")
		}
	})
	t.Run("with GFM", func(t *testing.T) {
		t.Parallel()
		p := NewParser(WithGithubFormattedMarkdown())
		if !p.hasTaskList {
			t.Error("hasTaskList should be true")
		}
	})
	t.Run("combined options", func(t *testing.T) {
		t.Parallel()
		p := NewParser(WithFrontmatter(), WithGithubFormattedMarkdown())
		if !p.Frontmatter || !p.hasTaskList {
			t.Error("both options should be set")
		}
	})
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("minimal recipe", func(t *testing.T) {
		t.Parallel()
		r, err := NewParser().Parse([]byte("# Title\n\n---\n\n- salt\n"))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if r.Title != "Title" {
			t.Errorf("Title = %q, want %q", r.Title, "Title")
		}
		if r.Description != nil {
			t.Errorf("Description = %q, want nil", *r.Description)
		}
		if len(r.Ingredients) != 1 || r.Ingredients[0].Name != "salt" {
			t.Errorf("Ingredients = %+v", r.Ingredients)
		}
		if r.Instructions != nil {
			t.Errorf("Instructions should be nil")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		_, err := NewParser().Parse([]byte(""))
		if err == nil {
			t.Fatal("expected error for empty input")
		}
	})

	t.Run("no heading", func(t *testing.T) {
		t.Parallel()
		_, err := NewParser().Parse([]byte("Not a heading\n\n---\n\n- x\n"))
		if err == nil {
			t.Fatal("expected error for missing heading")
		}
	})

	t.Run("wrong heading level", func(t *testing.T) {
		t.Parallel()
		_, err := NewParser().Parse([]byte("## Level 2\n\n---\n\n- x\n"))
		if err == nil {
			t.Fatal("expected error for level 2 heading")
		}
	})

	t.Run("missing thematic break", func(t *testing.T) {
		t.Parallel()
		_, err := NewParser().Parse([]byte("# Title\n\n- x\n"))
		if err == nil {
			t.Fatal("expected error for missing thematic break")
		}
	})

	t.Run("description", func(t *testing.T) {
		t.Parallel()
		r, err := NewParser().Parse([]byte("# Title\n\nA description.\n\n---\n\n- x\n"))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if r.Description == nil || *r.Description != "A description." {
			t.Errorf("Description = %v", r.Description)
		}
	})

	t.Run("tags", func(t *testing.T) {
		t.Parallel()
		r, err := NewParser().Parse([]byte("# Title\n\n*sauce, vegan*\n\n---\n\n- x\n"))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(r.Tags) != 2 || r.Tags[0] != "sauce" || r.Tags[1] != "vegan" {
			t.Errorf("Tags = %v", r.Tags)
		}
	})

	t.Run("yields", func(t *testing.T) {
		t.Parallel()
		r, err := NewParser().Parse([]byte("# Title\n\n**4 servings**\n\n---\n\n- x\n"))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(r.Yields) != 1 || r.Yields[0].Factor != 4 || *r.Yields[0].Unit != "servings" {
			t.Errorf("Yields = %+v", r.Yields)
		}
	})

	t.Run("tags and yields", func(t *testing.T) {
		t.Parallel()
		input := "# Title\n\n*sauce*\n\n**4 servings**\n\n---\n\n- x\n"
		r, err := NewParser().Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(r.Tags) != 1 {
			t.Errorf("Tags = %v", r.Tags)
		}
		if len(r.Yields) != 1 {
			t.Errorf("Yields = %+v", r.Yields)
		}
	})

	t.Run("duplicate tags error", func(t *testing.T) {
		t.Parallel()
		input := "# Title\n\n*a*\n\n*b*\n\n---\n\n- x\n"
		_, err := NewParser().Parse([]byte(input))
		if err == nil {
			t.Fatal("expected error for duplicate tags")
		}
	})

	t.Run("duplicate yields error", func(t *testing.T) {
		t.Parallel()
		input := "# Title\n\n**4 servings**\n\n**8 servings**\n\n---\n\n- x\n"
		_, err := NewParser().Parse([]byte(input))
		if err == nil {
			t.Fatal("expected error for duplicate yields")
		}
	})

	t.Run("ingredient with amount", func(t *testing.T) {
		t.Parallel()
		r, err := NewParser().Parse([]byte("# T\n\n---\n\n- *2 cups* flour\n"))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		ing := r.Ingredients[0]
		if ing.Amount == nil || ing.Amount.Factor != 2 || *ing.Amount.Unit != "cups" {
			t.Errorf("Amount = %+v", ing.Amount)
		}
		if ing.Name != "flour" {
			t.Errorf("Name = %q", ing.Name)
		}
	})

	t.Run("ingredient with link", func(t *testing.T) {
		t.Parallel()
		r, err := NewParser().Parse([]byte("# T\n\n---\n\n- [flour](flour.md)\n"))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		ing := r.Ingredients[0]
		if ing.Link == nil || *ing.Link != "flour.md" {
			t.Errorf("Link = %v", ing.Link)
		}
		if ing.Name != "flour" {
			t.Errorf("Name = %q", ing.Name)
		}
	})

	t.Run("ingredient with amount and link", func(t *testing.T) {
		t.Parallel()
		r, err := NewParser().Parse([]byte("# T\n\n---\n\n- *2 cups* [flour](flour.md)\n"))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		ing := r.Ingredients[0]
		if ing.Amount == nil || ing.Amount.Factor != 2 {
			t.Errorf("Amount = %+v", ing.Amount)
		}
		if ing.Link == nil || *ing.Link != "flour.md" {
			t.Errorf("Link = %v", ing.Link)
		}
	})

	t.Run("multiple ingredients", func(t *testing.T) {
		t.Parallel()
		input := "# T\n\n---\n\n- *1* a\n- *2* b\n- c\n"
		r, err := NewParser().Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(r.Ingredients) != 3 {
			t.Fatalf("len = %d, want 3", len(r.Ingredients))
		}
	})

	t.Run("ingredient groups", func(t *testing.T) {
		t.Parallel()
		input := "# T\n\n---\n\n- base\n\n## Sauce\n\n- tomato\n"
		r, err := NewParser().Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(r.Ingredients) != 1 || r.Ingredients[0].Name != "base" {
			t.Errorf("Ingredients = %+v", r.Ingredients)
		}
		if len(r.IngredientGroups) != 1 {
			t.Fatalf("IngredientGroups len = %d", len(r.IngredientGroups))
		}
		g := r.IngredientGroups[0]
		if g.Title != "Sauce" {
			t.Errorf("group title = %q", g.Title)
		}
		if len(g.Ingredients) != 1 || g.Ingredients[0].Name != "tomato" {
			t.Errorf("group ingredients = %+v", g.Ingredients)
		}
	})

	t.Run("nested ingredient groups", func(t *testing.T) {
		t.Parallel()
		input := "# T\n\n---\n\n## Dough\n\n- flour\n\n### Filling\n\n- cheese\n"
		r, err := NewParser().Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(r.IngredientGroups) != 1 {
			t.Fatalf("top groups = %d", len(r.IngredientGroups))
		}
		if len(r.IngredientGroups[0].IngredientGroups) != 1 {
			t.Fatalf("sub groups = %d", len(r.IngredientGroups[0].IngredientGroups))
		}
		sub := r.IngredientGroups[0].IngredientGroups[0]
		if sub.Title != "Filling" || sub.Ingredients[0].Name != "cheese" {
			t.Errorf("sub group = %+v", sub)
		}
	})

	t.Run("instructions", func(t *testing.T) {
		t.Parallel()
		input := "# T\n\n---\n\n- x\n\n---\n\nDo the thing.\n"
		r, err := NewParser().Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if r.Instructions == nil || *r.Instructions != "Do the thing." {
			t.Errorf("Instructions = %v", r.Instructions)
		}
	})

	t.Run("no instructions", func(t *testing.T) {
		t.Parallel()
		r, err := NewParser().Parse([]byte("# T\n\n---\n\n- x\n"))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if r.Instructions != nil {
			t.Errorf("Instructions should be nil, got %q", *r.Instructions)
		}
	})

	t.Run("description with tags and yields excluded", func(t *testing.T) {
		t.Parallel()
		input := "# Title\n\nHello world.\n\n*vegan*\n\n**4 servings**\n\n---\n\n- x\n"
		r, err := NewParser().Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if r.Description == nil || *r.Description != "Hello world." {
			t.Errorf("Description = %v", r.Description)
		}
		if len(r.Tags) != 1 || r.Tags[0] != "vegan" {
			t.Errorf("Tags = %v", r.Tags)
		}
	})

	t.Run("paragraph in ingredients errors", func(t *testing.T) {
		t.Parallel()
		input := "# T\n\n---\n\nNot a list\n"
		_, err := NewParser().Parse([]byte(input))
		if err == nil {
			t.Fatal("expected error for paragraph in ingredients")
		}
	})

	t.Run("setext heading title", func(t *testing.T) {
		t.Parallel()
		input := "Title\n=====\n\n---\n\n- x\n"
		r, err := NewParser().Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if r.Title != "Title" {
			t.Errorf("Title = %q", r.Title)
		}
	})

	t.Run("frontmatter stripped", func(t *testing.T) {
		t.Parallel()
		input := "---\ntitle: meta\n---\n# Real Title\n\n---\n\n- x\n"
		r, err := NewParser(WithFrontmatter()).Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if r.Title != "Real Title" {
			t.Errorf("Title = %q", r.Title)
		}
	})

	t.Run("GFM task list with amounts", func(t *testing.T) {
		t.Parallel()
		input := "# T\n\n---\n\n- [ ] *1 cup* flour\n- [x] *2 cups* sugar\n"
		r, err := NewParser(WithGithubFormattedMarkdown()).Parse([]byte(input))
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if len(r.Ingredients) != 2 {
			t.Fatalf("len = %d", len(r.Ingredients))
		}
		if r.Ingredients[0].Name != "flour" {
			t.Errorf("first = %q", r.Ingredients[0].Name)
		}
		if r.Ingredients[1].Name != "sugar" {
			t.Errorf("second = %q", r.Ingredients[1].Name)
		}
		if r.Ingredients[0].Amount == nil || r.Ingredients[0].Amount.Factor != 1 {
			t.Errorf("first amount = %+v", r.Ingredients[0].Amount)
		}
	})
}

func TestFlatten(t *testing.T) {
	t.Parallel()

	t.Run("no links unchanged", func(t *testing.T) {
		t.Parallel()
		p := NewParser()
		r := &Recipe{
			Ingredients:      []Ingredient{{Name: "salt"}},
			IngredientGroups: []IngredientGroup{},
		}
		if err := p.Flatten(r, "/fake/recipe.md"); err != nil {
			t.Fatal(err)
		}
		if len(r.Ingredients) != 1 || r.Ingredients[0].Name != "salt" {
			t.Errorf("unexpected change: %+v", r.Ingredients)
		}
	})

	t.Run("remote link preserved", func(t *testing.T) {
		t.Parallel()
		p := NewParser()
		r := &Recipe{
			Ingredients:      []Ingredient{{Name: "sauce", Link: new("https://example.com/sauce.md")}},
			IngredientGroups: []IngredientGroup{},
		}
		if err := p.Flatten(r, "/fake/recipe.md"); err != nil {
			t.Fatal(err)
		}
		if r.Ingredients[0].Link == nil {
			t.Error("remote link should be preserved")
		}
	})

	t.Run("local link resolved", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		linked := "# Sauce\n\n---\n\n- *1 cup* tomato\n- basil\n"
		if err := os.WriteFile(filepath.Join(dir, "sauce.md"), []byte(linked), 0644); err != nil {
			t.Fatal(err)
		}
		main := filepath.Join(dir, "main.md")

		p := NewParser()
		r := &Recipe{
			Ingredients:      []Ingredient{{Name: "sauce", Link: new("sauce.md"), Amount: &Amount{Factor: 2, Unit: new("cups")}}},
			IngredientGroups: []IngredientGroup{},
		}
		if err := p.Flatten(r, main); err != nil {
			t.Fatal(err)
		}
		if len(r.Ingredients) < 1 {
			t.Fatal("expected inlined ingredients")
		}
	})

	t.Run("missing file error", func(t *testing.T) {
		t.Parallel()
		p := NewParser()
		r := &Recipe{
			Ingredients:      []Ingredient{{Name: "x", Link: new("nonexistent.md")}},
			IngredientGroups: []IngredientGroup{},
		}
		if err := p.Flatten(r, "/fake/recipe.md"); err == nil {
			t.Fatal("expected error for missing linked file")
		}
	})
}

func TestEncodeURLPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain path", "recipe.md", "recipe.md"},
		{"spaces", "my recipe.md", "my%20recipe.md"},
		{"already encoded", "my%20recipe.md", "my%20recipe.md"},
		{"relative path", "../other/recipe.md", "../other/recipe.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := encodeURLPath(tt.in)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExcludeRangesFromSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		src    string
		ranges [][2]int
		offset int
		want   string
	}{
		{"no ranges", "hello", nil, 0, "hello"},
		{"exclude middle", "abcdef", [][2]int{{2, 4}}, 0, "abef"},
		{"with offset", "abcdef", [][2]int{{12, 14}}, 10, "abef"},
		{"exclude start", "abcdef", [][2]int{{0, 2}}, 0, "cdef"},
		{"exclude end", "abcdef", [][2]int{{4, 6}}, 0, "abcd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := excludeRangesFromSource([]byte(tt.src), tt.ranges, tt.offset)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindDashLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
		minPos int
		want   int
	}{
		{"at start", "---\ntext", 0, 0},
		{"after text", "text\n---\n", 0, 5},
		{"min skips first", "---\n---\n", 4, 4},
		{"no dashes", "text\nmore\n", 0, -1},
		{"requires 3 dashes", "--\n---\n", 0, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := findDashLine([]byte(tt.source), tt.minPos)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSkipLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
		pos    int
		want   int
	}{
		{"normal", "abc\ndef\n", 0, 4},
		{"negative pos", "abc\n", -1, 4},
		{"at newline", "abc\n", 3, 4},
		{"no newline", "abc", 0, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := skipLine([]byte(tt.source), tt.pos)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

