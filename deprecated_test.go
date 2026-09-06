package recipemd

import (
	"strings"
	"testing"
)

func deprecatedTestRecipe(instructions *string) *Recipe {
	return &Recipe{
		Title:            "Test",
		Yields:           []Amount{},
		Tags:             []string{},
		Ingredients:      []Ingredient{{Name: "salt"}},
		IngredientGroups: []IngredientGroup{},
		Instructions:     instructions,
	}
}

// The deprecated Parser wrappers must stay byte-identical to the Recipe
// methods they delegate to, so that v1 callers see no behaviour change.
func TestDeprecatedWrappersMatchRecipeMethods(t *testing.T) {
	t.Parallel()
	p := NewParser()
	r := deprecatedTestRecipe(nil)

	t.Run("json", func(t *testing.T) {
		t.Parallel()
		want, err := r.RenderJSON()
		if err != nil {
			t.Fatalf("Recipe.RenderJSON: %v", err)
		}
		got, err := p.RenderJSON(r)
		if err != nil {
			t.Fatalf("Parser.RenderJSON: %v", err)
		}
		if string(got) != string(want) {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("markdown", func(t *testing.T) {
		t.Parallel()
		if got, want := p.RenderMarkdown(r, 3), r.RenderMarkdown(3); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("html", func(t *testing.T) {
		t.Parallel()
		if got, want := p.RenderHTML(r, 3), r.RenderHTML(3); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// Parser.RenderHTML renders with the processor the parser was built with, so a
// GFM parser keeps producing GFM output without an explicit render option.
func TestDeprecatedRenderHTMLUsesParserProcessor(t *testing.T) {
	t.Parallel()
	instructions := "Strike ~~this~~ out.\n"
	r := deprecatedTestRecipe(&instructions)

	t.Run("gfm parser", func(t *testing.T) {
		t.Parallel()
		p := NewParser(WithGithubFormattedMarkdown())
		got := p.RenderHTML(r, 3)
		if !strings.Contains(got, "<del>") {
			t.Errorf("GFM parser should render strikethrough, got:\n%s", got)
		}
		if want := r.RenderHTML(3, WithGFMRendering()); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("commonmark parser", func(t *testing.T) {
		t.Parallel()
		p := NewParser()
		got := p.RenderHTML(r, 3)
		if strings.Contains(got, "<del>") {
			t.Errorf("CommonMark parser should not render strikethrough, got:\n%s", got)
		}
	})
}
