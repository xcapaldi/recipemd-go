package recipemd

import (
	"strings"
	"testing"
)

func TestRenderHTML(t *testing.T) {
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
		got := p.RenderHTML(r, 3)
		if got == "" {
			t.Fatal("empty output")
		}
		if !strings.Contains(got, `class="recipemd-recipe"`) {
			t.Error("missing recipe class")
		}
		if !strings.Contains(got, `class="recipemd-title"`) {
			t.Error("missing title class")
		}
		if !strings.Contains(got, "Test") {
			t.Error("missing title text")
		}
		if !strings.Contains(got, "salt") {
			t.Error("missing ingredient")
		}
		if !strings.Contains(got, `class="recipemd-ingredient"`) {
			t.Error("missing ingredient class")
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
			Yields:      []Amount{{Factor: 4, Unit: ptr("servings")}},
			Ingredients: []Ingredient{
				{Name: "avocado", Amount: &Amount{Factor: 2, Unit: ptr("cups")}},
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
		got := p.RenderHTML(r, 3)
		if !strings.Contains(got, "Guac") {
			t.Error("missing title")
		}
		if !strings.Contains(got, `class="recipemd-description"`) {
			t.Error("missing description class")
		}
		if !strings.Contains(got, "A great recipe.") {
			t.Error("missing description text")
		}
		if !strings.Contains(got, `class="recipemd-tags"`) {
			t.Error("missing tags class")
		}
		if !strings.Contains(got, "<em>sauce, vegan</em>") {
			t.Error("missing tags in em")
		}
		if !strings.Contains(got, `class="recipemd-yields"`) {
			t.Error("missing yields class")
		}
		if !strings.Contains(got, "<strong>4 servings</strong>") {
			t.Error("missing yields in strong")
		}
		if !strings.Contains(got, `class="recipemd-amount"`) {
			t.Error("missing amount class")
		}
		if !strings.Contains(got, "<em") && !strings.Contains(got, "2 cups") {
			t.Error("missing amount in em")
		}
		if !strings.Contains(got, `class="recipemd-ingredient-group"`) {
			t.Error("missing ingredient group class")
		}
		if !strings.Contains(got, `class="recipemd-group-title"`) {
			t.Error("missing group title class")
		}
		if !strings.Contains(got, "Topping") {
			t.Error("missing group title text")
		}
		if !strings.Contains(got, `class="recipemd-instructions"`) {
			t.Error("missing instructions class")
		}
		if !strings.Contains(got, "Mix well.") {
			t.Error("missing instructions text")
		}
	})

	t.Run("ingredient with link", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "T",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "sauce", Link: ptr("sauce.md")}},
			IngredientGroups: []IngredientGroup{},
		}
		got := p.RenderHTML(r, 3)
		if !strings.Contains(got, `href="sauce.md"`) {
			t.Errorf("missing link href in: %s", got)
		}
		if !strings.Contains(got, `class="recipemd-ingredient-link"`) {
			t.Error("missing ingredient link class")
		}
	})

	t.Run("nested ingredient groups", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:       "Layered",
			Yields:      []Amount{},
			Tags:        []string{},
			Ingredients: []Ingredient{},
			IngredientGroups: []IngredientGroup{
				{
					Title:       "Outer",
					Ingredients: []Ingredient{{Name: "butter"}},
					IngredientGroups: []IngredientGroup{
						{
							Title:            "Inner",
							Ingredients:      []Ingredient{{Name: "sugar"}},
							IngredientGroups: []IngredientGroup{},
						},
					},
				},
			},
		}
		got := p.RenderHTML(r, 3)
		if !strings.Contains(got, `<h2 class="recipemd-group-title">Outer</h2>`) {
			t.Errorf("missing h2 group title in: %s", got)
		}
		if !strings.Contains(got, `<h3 class="recipemd-group-title">Inner</h3>`) {
			t.Errorf("missing h3 nested group title in: %s", got)
		}
	})

	t.Run("markdown in description and instructions", func(t *testing.T) {
		t.Parallel()
		desc := "A recipe with **bold** and *italic* text."
		instructions := "## Step 1\n\nMix ingredients.\n\n## Step 2\n\nBake at 180°C."
		r := &Recipe{
			Title:            "Bake",
			Description:      &desc,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got := p.RenderHTML(r, 3)
		if !strings.Contains(got, "<strong>bold</strong>") {
			t.Error("bold not rendered in description")
		}
		if !strings.Contains(got, "<em>italic</em>") {
			t.Error("italic not rendered in description")
		}
		if !strings.Contains(got, "<h2>Step 1</h2>") {
			t.Error("heading not rendered in instructions")
		}
	})

	t.Run("html escaping in text fields", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Tom & Jerry's <Cake>",
			Yields:           []Amount{},
			Tags:             []string{"sweet & sour"},
			Ingredients:      []Ingredient{{Name: "sugar & spice"}},
			IngredientGroups: []IngredientGroup{},
		}
		got := p.RenderHTML(r, 3)
		if strings.Contains(got, "<Cake>") {
			t.Error("title should have < and > escaped")
		}
		if !strings.Contains(got, "Tom &amp; Jerry") {
			t.Error("& in title should be escaped")
		}
	})

	t.Run("no preamble section when empty", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Plain",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "water"}},
			IngredientGroups: []IngredientGroup{},
		}
		got := p.RenderHTML(r, 3)
		if strings.Contains(got, `class="recipemd-preamble"`) {
			t.Error("preamble should not be present when description/tags/yields are empty")
		}
	})
}
