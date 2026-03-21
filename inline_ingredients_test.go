package recipemd

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// wordForms (pluralization)
// ---------------------------------------------------------------------------

func TestWordForms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		word  string
		forms []string // expected lowercased forms
	}{
		// Regular +s
		{"apple", []string{"apple", "apples"}},
		{"egg", []string{"egg", "eggs"}},
		{"clove", []string{"clove", "cloves"}},
		// consonant+y → ies
		{"berry", []string{"berry", "berries"}},
		{"cherry", []string{"cherry", "cherries"}},
		// -o irregulars
		{"potato", []string{"potato", "potatoes"}},
		{"tomato", []string{"tomato", "tomatoes"}},
		// -f → ves
		{"leaf", []string{"leaf", "leaves"}},
		{"loaf", []string{"loaf", "loaves"}},
		// -fe → ves
		{"knife", []string{"knife", "knives"}},
		// -ch → es
		{"peach", []string{"peach", "peaches"}},
		// Invariant uncountables
		{"flour", []string{"flour"}},
		{"garlic", []string{"garlic"}},
		{"rice", []string{"rice"}},
		{"sugar", []string{"sugar"}},
		// Multi-word: only last word is inflected
		{"bay leaf", []string{"bay leaf", "bay leaves"}},
		{"olive oil", []string{"olive oil"}}, // oil is invariant
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.word, func(t *testing.T) {
			t.Parallel()
			got := wordForms(tc.word)
			if len(got) != len(tc.forms) {
				t.Fatalf("wordForms(%q) = %v, want %v", tc.word, got, tc.forms)
			}
			for i, f := range tc.forms {
				if got[i] != f {
					t.Errorf("wordForms(%q)[%d] = %q, want %q", tc.word, i, got[i], f)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WithInlineIngredients — parser option smoke test
// ---------------------------------------------------------------------------

func TestWithInlineIngredients_DefaultNoop(t *testing.T) {
	t.Parallel()
	// Without the option, instructions are unchanged.
	p := NewParser()
	instructions := "add eggs and flour"
	r := &Recipe{
		Title:            "T",
		Instructions:     &instructions,
		Yields:           []Amount{},
		Tags:             []string{},
		Ingredients:      []Ingredient{{Name: "egg", Amount: &Amount{Factor: 3}}},
		IngredientGroups: []IngredientGroup{},
	}
	md := p.RenderMarkdown(r, 3)
	// Plain parser must not inject amounts.
	if strings.Contains(md, "3 eggs") || strings.Contains(md, "3 egg") {
		t.Errorf("plain parser injected amount unexpectedly: %q", md)
	}
}

// ---------------------------------------------------------------------------
// RenderMarkdown — inline before/after
// ---------------------------------------------------------------------------

func TestRenderMarkdown_InlineBefore(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients(
		WithInlineFormat(InlineIngredientsBefore),
	))
	instructions := "crack eggs into the bowl then add flour"
	r := &Recipe{
		Title:        "T",
		Instructions: &instructions,
		Yields:       []Amount{},
		Tags:         []string{},
		Ingredients: []Ingredient{
			{Name: "egg", Amount: &Amount{Factor: 3}},
			{Name: "flour", Amount: &Amount{Factor: 2, Unit: new("cups")}},
		},
		IngredientGroups: []IngredientGroup{},
	}
	md := p.RenderMarkdown(r, 3)
	if !strings.Contains(md, "3 eggs") {
		t.Errorf("expected plural match injected as '3 eggs' in: %q", md)
	}
	if !strings.Contains(md, "2 cups flour") {
		t.Errorf("expected '2 cups flour' in: %q", md)
	}
}

func TestRenderMarkdown_InlineAfter(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients(
		WithInlineFormat(InlineIngredientsAfter),
	))
	instructions := "add eggs and stir"
	r := &Recipe{
		Title:        "T",
		Instructions: &instructions,
		Yields:       []Amount{},
		Tags:         []string{},
		Ingredients: []Ingredient{
			{Name: "egg", Amount: &Amount{Factor: 2}},
		},
		IngredientGroups: []IngredientGroup{},
	}
	md := p.RenderMarkdown(r, 3)
	if !strings.Contains(md, "eggs (2)") {
		t.Errorf("expected 'eggs (2)' in: %q", md)
	}
}

// ---------------------------------------------------------------------------
// Pluralization matching
// ---------------------------------------------------------------------------

func TestRenderMarkdown_Pluralization(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients())
	t.Run("singular ingredient matches plural in instructions", func(t *testing.T) {
		t.Parallel()
		instructions := "fold in the berries"
		r := &Recipe{
			Title:            "T",
			Instructions:     &instructions,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "berry", Amount: &Amount{Factor: 200, Unit: new("g")}}},
			IngredientGroups: []IngredientGroup{},
		}
		md := p.RenderMarkdown(r, 3)
		if !strings.Contains(md, "200 g berries") {
			t.Errorf("expected plural match in: %q", md)
		}
	})

	t.Run("word boundary not partial", func(t *testing.T) {
		t.Parallel()
		instructions := "use salted butter"
		r := &Recipe{
			Title:            "T",
			Instructions:     &instructions,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "salt", Amount: &Amount{Factor: 1, Unit: new("tsp")}}},
			IngredientGroups: []IngredientGroup{},
		}
		md := p.RenderMarkdown(r, 3)
		if strings.Contains(md, "1 tsp") {
			t.Errorf("'salt' should not match inside 'salted': %q", md)
		}
	})

	t.Run("case-insensitive preserves original case", func(t *testing.T) {
		t.Parallel()
		instructions := "Add Eggs to the bowl"
		r := &Recipe{
			Title:            "T",
			Instructions:     &instructions,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "egg", Amount: &Amount{Factor: 3}}},
			IngredientGroups: []IngredientGroup{},
		}
		md := p.RenderMarkdown(r, 3)
		if !strings.Contains(md, "3 Eggs") {
			t.Errorf("expected original casing preserved: %q", md)
		}
	})
}

// ---------------------------------------------------------------------------
// Preparation separators
// ---------------------------------------------------------------------------

func TestRenderMarkdown_PrepSeparators(t *testing.T) {
	t.Parallel()

	t.Run("comma separator before format", func(t *testing.T) {
		t.Parallel()
		p := NewParser(WithInlineIngredients(
			WithInlineFormat(InlineIngredientsBefore),
			WithInlinePrepSeparators(","),
		))
		instructions := "mince garlic and add to pan"
		r := &Recipe{
			Title:            "T",
			Instructions:     &instructions,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "garlic, minced", Amount: &Amount{Factor: 3, Unit: new("cloves")}}},
			IngredientGroups: []IngredientGroup{},
		}
		md := p.RenderMarkdown(r, 3)
		// base="garlic", prep="minced" → "3 cloves minced garlic"
		if !strings.Contains(md, "3 cloves minced garlic") {
			t.Errorf("expected '3 cloves minced garlic' in: %q", md)
		}
	})

	t.Run("paren separator after format", func(t *testing.T) {
		t.Parallel()
		p := NewParser(WithInlineIngredients(
			WithInlineFormat(InlineIngredientsAfter),
			WithInlinePrepSeparators("("),
		))
		instructions := "slice garlic thin and cook"
		r := &Recipe{
			Title:            "T",
			Instructions:     &instructions,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "garlic (sliced)", Amount: &Amount{Factor: 4, Unit: new("cloves")}}},
			IngredientGroups: []IngredientGroup{},
		}
		md := p.RenderMarkdown(r, 3)
		// base="garlic", prep="sliced" → "garlic (4 cloves sliced)"
		if !strings.Contains(md, "garlic (4 cloves sliced)") {
			t.Errorf("expected 'garlic (4 cloves sliced)' in: %q", md)
		}
	})

	t.Run("multi-word base with prep", func(t *testing.T) {
		t.Parallel()
		p := NewParser(WithInlineIngredients(
			WithInlineFormat(InlineIngredientsBefore),
			WithInlinePrepSeparators(","),
		))
		instructions := "add brown sugar and mix"
		r := &Recipe{
			Title:            "T",
			Instructions:     &instructions,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "brown sugar, packed", Amount: &Amount{Factor: 1, Unit: new("cup")}}},
			IngredientGroups: []IngredientGroup{},
		}
		md := p.RenderMarkdown(r, 3)
		// base="brown sugar", prep="packed" → "1 cup packed brown sugar"
		if !strings.Contains(md, "1 cup packed brown sugar") {
			t.Errorf("expected '1 cup packed brown sugar' in: %q", md)
		}
	})
}

// ---------------------------------------------------------------------------
// Multi-word priority
// ---------------------------------------------------------------------------

func TestRenderMarkdown_MultiWordPriority(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients(
		WithInlineFormat(InlineIngredientsBefore),
	))
	instructions := "mix brown sugar then add plain sugar"
	r := &Recipe{
		Title:        "T",
		Instructions: &instructions,
		Yields:       []Amount{},
		Tags:         []string{},
		Ingredients: []Ingredient{
			{Name: "brown sugar", Amount: &Amount{Factor: 1, Unit: new("cup")}},
			{Name: "sugar", Amount: &Amount{Factor: 2, Unit: new("tsp")}},
		},
		IngredientGroups: []IngredientGroup{},
	}
	md := p.RenderMarkdown(r, 3)
	if !strings.Contains(md, "1 cup brown sugar") {
		t.Errorf("expected '1 cup brown sugar' in: %q", md)
	}
	if !strings.Contains(md, "2 tsp sugar") {
		t.Errorf("expected '2 tsp sugar' in: %q", md)
	}
	// "sugar" inside "brown sugar" must not be double-injected.
	if strings.Contains(md, "2 tsp brown sugar") || strings.Contains(md, "1 cup brown 2 tsp sugar") {
		t.Errorf("double injection detected: %q", md)
	}
}

// ---------------------------------------------------------------------------
// Ingredient groups
// ---------------------------------------------------------------------------

func TestRenderMarkdown_IngredientsInGroups(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients())
	instructions := "fold in vanilla and top with blueberries"
	r := &Recipe{
		Title:        "T",
		Instructions: &instructions,
		Yields:       []Amount{},
		Tags:         []string{},
		Ingredients:  []Ingredient{},
		IngredientGroups: []IngredientGroup{
			{
				Title: "Batter",
				Ingredients: []Ingredient{
					{Name: "vanilla", Amount: &Amount{Factor: 1, Unit: new("tsp")}},
				},
				IngredientGroups: []IngredientGroup{},
			},
			{
				Title: "Topping",
				Ingredients: []Ingredient{
					{Name: "blueberry", Amount: &Amount{Factor: 100, Unit: new("g")}},
				},
				IngredientGroups: []IngredientGroup{},
			},
		},
	}
	md := p.RenderMarkdown(r, 3)
	if !strings.Contains(md, "1 tsp vanilla") {
		t.Errorf("expected '1 tsp vanilla' in: %q", md)
	}
	if !strings.Contains(md, "100 g blueberries") {
		t.Errorf("expected plural '100 g blueberries' in: %q", md)
	}
}

// ---------------------------------------------------------------------------
// RenderHTML — span wrapping and hover
// ---------------------------------------------------------------------------

func TestRenderHTML_InlineSpan(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients(
		WithInlineFormat(InlineIngredientsBefore),
	))
	instructions := "add eggs to the bowl"
	r := &Recipe{
		Title:        "T",
		Instructions: &instructions,
		Yields:       []Amount{},
		Tags:         []string{},
		Ingredients: []Ingredient{
			{Name: "egg", Amount: &Amount{Factor: 3}},
		},
		IngredientGroups: []IngredientGroup{},
	}
	got := p.RenderHTML(r, 3)
	if !strings.Contains(got, `class="recipemd-inline-ingredient"`) {
		t.Errorf("expected inline-ingredient span in: %q", got)
	}
	if !strings.Contains(got, "3 eggs") {
		t.Errorf("expected '3 eggs' in HTML output: %q", got)
	}
}

func TestRenderHTML_InlineAfterSpan(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients(
		WithInlineFormat(InlineIngredientsAfter),
	))
	instructions := "beat eggs until frothy"
	r := &Recipe{
		Title:        "T",
		Instructions: &instructions,
		Yields:       []Amount{},
		Tags:         []string{},
		Ingredients: []Ingredient{
			{Name: "egg", Amount: &Amount{Factor: 2}},
		},
		IngredientGroups: []IngredientGroup{},
	}
	got := p.RenderHTML(r, 3)
	if !strings.Contains(got, "eggs (2)") {
		t.Errorf("expected 'eggs (2)' in HTML output: %q", got)
	}
}

func TestRenderHTML_Hover(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients(
		WithInlineHTMLHover(),
	))
	instructions := "season with pepper to taste"
	r := &Recipe{
		Title:        "T",
		Instructions: &instructions,
		Yields:       []Amount{},
		Tags:         []string{},
		Ingredients: []Ingredient{
			{Name: "pepper", Amount: &Amount{Factor: 0.5, Unit: new("tsp")}},
		},
		IngredientGroups: []IngredientGroup{},
	}
	got := p.RenderHTML(r, 3)
	if !strings.Contains(got, `title="0.5 tsp"`) {
		t.Errorf("expected title attribute in: %q", got)
	}
	if !strings.Contains(got, `>pepper<`) {
		t.Errorf("expected ingredient name as visible text in: %q", got)
	}
}

func TestRenderHTML_HoverWithPrep(t *testing.T) {
	t.Parallel()
	p := NewParser(WithInlineIngredients(
		WithInlineHTMLHover(),
		WithInlinePrepSeparators(","),
	))
	instructions := "sauté onion until translucent"
	r := &Recipe{
		Title:        "T",
		Instructions: &instructions,
		Yields:       []Amount{},
		Tags:         []string{},
		Ingredients: []Ingredient{
			{Name: "onion, diced", Amount: &Amount{Factor: 1}},
		},
		IngredientGroups: []IngredientGroup{},
	}
	got := p.RenderHTML(r, 3)
	// title should include amount + prep
	if !strings.Contains(got, `title="1 diced"`) {
		t.Errorf("expected title='1 diced' in: %q", got)
	}
	if !strings.Contains(got, `>onion<`) {
		t.Errorf("expected 'onion' as visible text in: %q", got)
	}
}

func TestRenderHTML_NoInjectionWithoutOption(t *testing.T) {
	t.Parallel()
	p := NewParser()
	instructions := "add eggs"
	r := &Recipe{
		Title:            "T",
		Instructions:     &instructions,
		Yields:           []Amount{},
		Tags:             []string{},
		Ingredients:      []Ingredient{{Name: "egg", Amount: &Amount{Factor: 3}}},
		IngredientGroups: []IngredientGroup{},
	}
	got := p.RenderHTML(r, 3)
	if strings.Contains(got, `class="recipemd-inline-ingredient"`) {
		t.Errorf("unexpected inline span without option: %q", got)
	}
}
