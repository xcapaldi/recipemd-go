package recipemd

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	t.Parallel()
	p := NewParser()

	t.Run("minimal", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Test",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "salt"}},
			IngredientGroups: []IngredientGroup{},
		}
		got := p.RenderMarkdown(r, 3)
		if got == "" {
			t.Fatal("empty output")
		}
		if !strings.Contains(got, "# Test") {
			t.Error("missing title")
		}
		if !strings.Contains(got, "- salt") {
			t.Error("missing ingredient")
		}
	})

	t.Run("full recipe", func(t *testing.T) {
		t.Parallel()
		desc := "A great recipe."
		instructions := "Mix well."
		r := &Recipe{
			Title:       "Guac",
			Description: &desc,
			Tags:        []string{"sauce", "vegan"},
			Yields:      []Amount{{Factor: 4, Unit: new("servings")}},
			Ingredients: []Ingredient{
				{Name: "avocado", Amount: &Amount{Factor: 1, Unit: nil}},
				{Name: "salt"},
			},
			IngredientGroups: []IngredientGroup{
				{
					Title:            "Topping",
					Ingredients:      []Ingredient{{Name: "cilantro"}},
					IngredientGroups: []IngredientGroup{},
				},
			},
			Instructions: &instructions,
		}
		got := p.RenderMarkdown(r, 3)
		if !strings.Contains(got, "# Guac") {
			t.Error("missing title")
		}
		if !strings.Contains(got, "A great recipe.") {
			t.Error("missing description")
		}
		if !strings.Contains(got, "*sauce, vegan*") {
			t.Error("missing tags")
		}
		if !strings.Contains(got, "**4 servings**") {
			t.Error("missing yields")
		}
		if !strings.Contains(got, "- *1* avocado") {
			t.Error("missing amount ingredient")
		}
		if !strings.Contains(got, "- salt") {
			t.Error("missing plain ingredient")
		}
		if !strings.Contains(got, "## Topping") {
			t.Error("missing ingredient group heading")
		}
		if !strings.Contains(got, "Mix well.") {
			t.Error("missing instructions")
		}
	})

	t.Run("ingredient with link", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "T",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "sauce", Link: new("sauce.md")}},
			IngredientGroups: []IngredientGroup{},
		}
		got := p.RenderMarkdown(r, 3)
		if !strings.Contains(got, "[sauce](sauce.md)") {
			t.Errorf("missing link rendering in: %s", got)
		}
	})
}
