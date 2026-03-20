package recipemd

import (
	"testing"
)

func TestInlineIngredients(t *testing.T) {
	t.Parallel()

	t.Run("nil instructions returns nil", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Instructions: nil,
			Ingredients:  []Ingredient{{Name: "salt", Amount: &Amount{Factor: 1}}},
		}
		if got := InlineIngredients(r, 3); got != nil {
			t.Errorf("expected nil, got %q", *got)
		}
	})

	t.Run("ingredient without amount is not injected", func(t *testing.T) {
		t.Parallel()
		instructions := "add salt and stir"
		r := &Recipe{
			Instructions:     &instructions,
			Ingredients:      []Ingredient{{Name: "salt"}},
			IngredientGroups: []IngredientGroup{},
		}
		got := InlineIngredients(r, 3)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if *got != instructions {
			t.Errorf("expected %q, got %q", instructions, *got)
		}
	})

	t.Run("single ingredient injected", func(t *testing.T) {
		t.Parallel()
		instructions := "then add cinnamon and mix"
		r := &Recipe{
			Instructions: &instructions,
			Ingredients: []Ingredient{
				{Name: "cinnamon", Amount: &Amount{Factor: 0.5, Unit: new("tsp")}},
			},
			IngredientGroups: []IngredientGroup{},
		}
		got := InlineIngredients(r, 3)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		want := "then add 0.5 tsp cinnamon and mix"
		if *got != want {
			t.Errorf("expected %q, got %q", want, *got)
		}
	})

	t.Run("case-insensitive match preserves original casing", func(t *testing.T) {
		t.Parallel()
		instructions := "Add Cinnamon to the bowl"
		r := &Recipe{
			Instructions: &instructions,
			Ingredients: []Ingredient{
				{Name: "cinnamon", Amount: &Amount{Factor: 0.5, Unit: new("tsp")}},
			},
			IngredientGroups: []IngredientGroup{},
		}
		got := InlineIngredients(r, 3)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		want := "Add 0.5 tsp Cinnamon to the bowl"
		if *got != want {
			t.Errorf("expected %q, got %q", want, *got)
		}
	})

	t.Run("multi-word ingredient matched before partial", func(t *testing.T) {
		t.Parallel()
		instructions := "stir in brown sugar then add more sugar"
		r := &Recipe{
			Instructions: &instructions,
			Ingredients: []Ingredient{
				{Name: "brown sugar", Amount: &Amount{Factor: 1, Unit: new("cup")}},
				{Name: "sugar", Amount: &Amount{Factor: 2, Unit: new("tsp")}},
			},
			IngredientGroups: []IngredientGroup{},
		}
		got := InlineIngredients(r, 3)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		want := "stir in 1 cup brown sugar then add more 2 tsp sugar"
		if *got != want {
			t.Errorf("expected %q, got %q", want, *got)
		}
	})

	t.Run("multiple occurrences are all replaced", func(t *testing.T) {
		t.Parallel()
		instructions := "add flour, then more flour"
		r := &Recipe{
			Instructions: &instructions,
			Ingredients: []Ingredient{
				{Name: "flour", Amount: &Amount{Factor: 2, Unit: new("cup")}},
			},
			IngredientGroups: []IngredientGroup{},
		}
		got := InlineIngredients(r, 3)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		want := "add 2 cup flour, then more 2 cup flour"
		if *got != want {
			t.Errorf("expected %q, got %q", want, *got)
		}
	})

	t.Run("ingredient in group is also considered", func(t *testing.T) {
		t.Parallel()
		instructions := "fold in vanilla"
		r := &Recipe{
			Instructions: &instructions,
			Ingredients:  []Ingredient{},
			IngredientGroups: []IngredientGroup{
				{
					Title: "Flavouring",
					Ingredients: []Ingredient{
						{Name: "vanilla", Amount: &Amount{Factor: 1, Unit: new("tsp")}},
					},
					IngredientGroups: []IngredientGroup{},
				},
			},
		}
		got := InlineIngredients(r, 3)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		want := "fold in 1 tsp vanilla"
		if *got != want {
			t.Errorf("expected %q, got %q", want, *got)
		}
	})

	t.Run("word boundary prevents partial match", func(t *testing.T) {
		t.Parallel()
		instructions := "add salted butter"
		r := &Recipe{
			Instructions: &instructions,
			Ingredients: []Ingredient{
				{Name: "salt", Amount: &Amount{Factor: 1, Unit: new("tsp")}},
			},
			IngredientGroups: []IngredientGroup{},
		}
		got := InlineIngredients(r, 3)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		// "salt" must NOT match inside "salted"
		if *got != instructions {
			t.Errorf("expected %q unchanged, got %q", instructions, *got)
		}
	})

	t.Run("unitless amount injected", func(t *testing.T) {
		t.Parallel()
		instructions := "crack eggs into bowl"
		r := &Recipe{
			Instructions: &instructions,
			Ingredients: []Ingredient{
				{Name: "eggs", Amount: &Amount{Factor: 3}},
			},
			IngredientGroups: []IngredientGroup{},
		}
		got := InlineIngredients(r, 3)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		want := "crack 3 eggs into bowl"
		if *got != want {
			t.Errorf("expected %q, got %q", want, *got)
		}
	})
}
