package recipemd

import (
	"encoding/json"
	"testing"
)

func TestAmount_MarshalJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    Amount
		want string
	}{
		{"integer no unit", Amount{Factor: 3, Unit: nil}, `{"factor":"3","unit":null}`},
		{"decimal no unit", Amount{Factor: 1.5, Unit: nil}, `{"factor":"1.5","unit":null}`},
		{"with unit", Amount{Factor: 2, Unit: new("cups")}, `{"factor":"2","unit":"cups"}`},
		{"rounds to 3 decimals", Amount{Factor: 1.0 / 3.0, Unit: nil}, `{"factor":"0.333","unit":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.a)
			if err != nil {
				t.Fatalf("MarshalJSON error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAmount_FormatFactor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		factor   float64
		rounding int
		want     string
	}{
		{"integer", 3, 3, "3"},
		{"one decimal", 1.5, 3, "1.5"},
		{"trailing zeros trimmed", 2.10, 3, "2.1"},
		{"no rounding", 1.123456, -1, "1.123456"},
		{"round to 0", 1.7, 0, "2"},
		{"round to 2", 1.555, 2, "1.56"},
		{"zero", 0, 3, "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := Amount{Factor: tt.factor}
			got := a.FormatFactor(tt.rounding)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAmount_Serialize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		a        Amount
		rounding int
		want     string
	}{
		{"no unit", Amount{Factor: 2, Unit: nil}, 3, "2"},
		{"with unit", Amount{Factor: 1.5, Unit: new("cups")}, 3, "1.5 cups"},
		{"integer with unit", Amount{Factor: 3, Unit: new("tsp")}, 0, "3 tsp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.a.Serialize(tt.rounding)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIngredient_Serialize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		i    Ingredient
		want string
	}{
		{"name only", Ingredient{Name: "salt"}, "salt"},
		{"with amount", Ingredient{Name: "flour", Amount: &Amount{Factor: 2, Unit: new("cups")}}, "2 cups flour"},
		{"amount no unit", Ingredient{Name: "eggs", Amount: &Amount{Factor: 3, Unit: nil}}, "3 eggs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.i.Serialize(3)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAmount_Scale(t *testing.T) {
	t.Parallel()
	a := Amount{Factor: 2, Unit: new("cups")}
	a.Scale(3)
	if a.Factor != 6 {
		t.Errorf("Factor = %v, want 6", a.Factor)
	}
	if *a.Unit != "cups" {
		t.Errorf("Unit changed unexpectedly")
	}
}

func TestIngredient_Scale(t *testing.T) {
	t.Parallel()
	t.Run("with amount", func(t *testing.T) {
		t.Parallel()
		i := Ingredient{Name: "flour", Amount: &Amount{Factor: 2, Unit: new("cups")}}
		i.Scale(0.5)
		if i.Amount.Factor != 1 {
			t.Errorf("Factor = %v, want 1", i.Amount.Factor)
		}
	})
	t.Run("nil amount", func(t *testing.T) {
		t.Parallel()
		i := Ingredient{Name: "salt"}
		i.Scale(2) // should not panic
	})
}

func TestIngredientGroup_Scale(t *testing.T) {
	t.Parallel()
	g := IngredientGroup{
		Title: "Sauce",
		Ingredients: []Ingredient{
			{Name: "tomato", Amount: &Amount{Factor: 2, Unit: new("cups")}},
			{Name: "basil"},
		},
		IngredientGroups: []IngredientGroup{
			{
				Title:       "Spices",
				Ingredients: []Ingredient{{Name: "pepper", Amount: &Amount{Factor: 1, Unit: new("tsp")}}},
			},
		},
	}
	g.Scale(3)
	if g.Ingredients[0].Amount.Factor != 6 {
		t.Errorf("tomato factor = %v, want 6", g.Ingredients[0].Amount.Factor)
	}
	if g.IngredientGroups[0].Ingredients[0].Amount.Factor != 3 {
		t.Errorf("pepper factor = %v, want 3", g.IngredientGroups[0].Ingredients[0].Amount.Factor)
	}
}

func TestRecipe_Scale(t *testing.T) {
	t.Parallel()
	r := &Recipe{
		Yields: []Amount{
			{Factor: 4, Unit: new("servings")},
		},
		Ingredients: []Ingredient{
			{Name: "flour", Amount: &Amount{Factor: 2, Unit: new("cups")}},
		},
		IngredientGroups: []IngredientGroup{
			{
				Title:       "Sauce",
				Ingredients: []Ingredient{{Name: "tomato", Amount: &Amount{Factor: 1, Unit: nil}}},
			},
		},
	}
	r.Scale(2)
	if r.Yields[0].Factor != 8 {
		t.Errorf("yield = %v, want 8", r.Yields[0].Factor)
	}
	if r.Ingredients[0].Amount.Factor != 4 {
		t.Errorf("flour = %v, want 4", r.Ingredients[0].Amount.Factor)
	}
	if r.IngredientGroups[0].Ingredients[0].Amount.Factor != 2 {
		t.Errorf("tomato = %v, want 2", r.IngredientGroups[0].Ingredients[0].Amount.Factor)
	}
}

func TestRecipe_ScaleForYield(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		yields     []Amount
		desired    Amount
		wantErr    bool
		wantFactor float64
	}{
		{
			name:       "match unit",
			yields:     []Amount{{Factor: 4, Unit: new("servings")}},
			desired:    Amount{Factor: 8, Unit: new("servings")},
			wantFactor: 4, // 2*2
		},
		{
			name:       "match unitless",
			yields:     []Amount{{Factor: 2, Unit: nil}},
			desired:    Amount{Factor: 6, Unit: nil},
			wantFactor: 6, // 2*(6/2)
		},
		{
			name:       "unitless fallback to multiplier",
			yields:     []Amount{{Factor: 4, Unit: new("servings")}},
			desired:    Amount{Factor: 3, Unit: nil},
			wantFactor: 6, // 2*3
		},
		{
			name:    "no matching unit",
			yields:  []Amount{{Factor: 4, Unit: new("servings")}},
			desired: Amount{Factor: 2, Unit: new("liters")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &Recipe{
				Yields:      tt.yields,
				Ingredients: []Ingredient{{Name: "x", Amount: &Amount{Factor: 2, Unit: nil}}},
			}
			err := r.ScaleForYield(tt.desired)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.Ingredients[0].Amount.Factor != tt.wantFactor {
				t.Errorf("factor = %v, want %v", r.Ingredients[0].Amount.Factor, tt.wantFactor)
			}
		})
	}
}

func TestRecipe_LeafIngredients(t *testing.T) {
	t.Parallel()
	r := &Recipe{
		Ingredients: []Ingredient{{Name: "a"}, {Name: "b"}},
		IngredientGroups: []IngredientGroup{
			{
				Title:       "G1",
				Ingredients: []Ingredient{{Name: "c"}},
				IngredientGroups: []IngredientGroup{
					{Title: "G2", Ingredients: []Ingredient{{Name: "d"}}},
				},
			},
		},
	}
	leaves := r.LeafIngredients()
	names := make([]string, len(leaves))
	for i, l := range leaves {
		names[i] = l.Name
	}
	want := []string{"a", "b", "c", "d"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestIngredientGroup_LeafIngredients(t *testing.T) {
	t.Parallel()
	g := &IngredientGroup{
		Title:       "Top",
		Ingredients: []Ingredient{{Name: "x"}},
		IngredientGroups: []IngredientGroup{
			{Title: "Sub", Ingredients: []Ingredient{{Name: "y"}, {Name: "z"}}},
		},
	}
	leaves := g.LeafIngredients()
	if len(leaves) != 3 {
		t.Fatalf("got %d leaves, want 3", len(leaves))
	}
}
