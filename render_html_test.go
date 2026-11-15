package recipemd

import (
	"strings"
	"testing"
)

func TestRenderHTML(t *testing.T) {
	t.Parallel()
	t.Run("minimal", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Test",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "salt"}},
			IngredientGroups: []IngredientGroup{},
		}
		got := r.RenderHTML(3)
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
			Yields:      []Amount{{Factor: 4, Unit: new("servings")}},
			Ingredients: []Ingredient{
				{Name: "avocado", Amount: &Amount{Factor: 2, Unit: new("cups")}},
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
		got := r.RenderHTML(3)
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
		if !strings.Contains(got, `<li class="recipemd-tag">sauce</li>`) ||
			!strings.Contains(got, `<li class="recipemd-tag">vegan</li>`) {
			t.Errorf("missing per-tag elements in: %s", got)
		}
		if !strings.Contains(got, `class="recipemd-yields"`) {
			t.Error("missing yields class")
		}
		if !strings.Contains(got, `<strong><span class="recipemd-yield" data-factor="4" data-unit="servings">`) {
			t.Errorf("missing yield span in strong: %s", got)
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
			Ingredients:      []Ingredient{{Name: "sauce", Link: new("sauce.md")}},
			IngredientGroups: []IngredientGroup{},
		}
		got := r.RenderHTML(3)
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
		got := r.RenderHTML(3)
		if !strings.Contains(got, `<h2 class="recipemd-group-title">Outer</h2>`) {
			t.Errorf("missing h2 group title in: %s", got)
		}
		if !strings.Contains(got, `<h3 class="recipemd-group-title">Inner</h3>`) {
			t.Errorf("missing h3 nested group title in: %s", got)
		}
		if !strings.Contains(got, `<div class="recipemd-ingredient-group" data-level="2">`+"\n"+`<h2 class="recipemd-group-title">Outer</h2>`) {
			t.Errorf("missing data-level=2 on outer group in: %s", got)
		}
		if !strings.Contains(got, `<div class="recipemd-ingredient-group" data-level="3">`+"\n"+`<h3 class="recipemd-group-title">Inner</h3>`) {
			t.Errorf("missing data-level=3 on inner group in: %s", got)
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
		got := r.RenderHTML(3)
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
		got := r.RenderHTML(3)
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
		got := r.RenderHTML(3)
		if strings.Contains(got, `class="recipemd-preamble"`) {
			t.Error("preamble should not be present when description/tags/yields are empty")
		}
	})
}

func TestRenderHTMLDataAttributes(t *testing.T) {
	t.Parallel()
	r := &Recipe{
		Title:  "Scalable",
		Tags:   []string{"a & b"},
		Yields: []Amount{{Factor: 12, Unit: new("cookies")}, {Factor: 1.0 / 3}},
		Ingredients: []Ingredient{
			{Name: "sugar", Amount: &Amount{Factor: 1.0 / 3, Unit: new(`"cup"`)}},
			{Name: "eggs", Amount: &Amount{Factor: 2}},
		},
		IngredientGroups: []IngredientGroup{},
	}
	got := r.RenderHTML(2)

	for _, want := range []string{
		// tags escaped, one element each
		`<li class="recipemd-tag">a &amp; b</li>`,
		// yields: one span each, comma-separated, unrounded data-factor
		`<span class="recipemd-yield" data-factor="12" data-unit="cookies"><span class="recipemd-amount-factor">12</span> <span class="recipemd-amount-unit">cookies</span></span>, <span class="recipemd-yield" data-factor="0.3333333333333333"><span class="recipemd-amount-factor">0.33</span></span>`,
		// unit is attribute-escaped
		`<em class="recipemd-amount" data-factor="0.3333333333333333" data-unit="&#34;cup&#34;"><span class="recipemd-amount-factor">0.33</span> <span class="recipemd-amount-unit">&#34;cup&#34;</span></em>`,
		// unitless amount has no data-unit or unit span
		`<em class="recipemd-amount" data-factor="2"><span class="recipemd-amount-factor">2</span></em> <span class="recipemd-ingredient-name">eggs</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s\nin:\n%s", want, got)
		}
	}
}

func TestRenderHTMLGFM(t *testing.T) {
	t.Parallel()
	instructions := "Strike ~~this~~ out.\n"

	t.Run("commonmark by default", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Plain",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "water"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got := r.RenderHTML(3)
		if strings.Contains(got, "<del>") {
			t.Error("strikethrough should not be rendered without WithGFMRendering")
		}
	})

	t.Run("gfm when requested", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Plain",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "water"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got := r.RenderHTML(3, WithGFMRendering())
		if !strings.Contains(got, "<del>") {
			t.Errorf("strikethrough should be rendered with WithGFMRendering, got:\n%s", got)
		}
	})
}
