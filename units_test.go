package recipemd

import "testing"

func TestLookupUnit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantKey string
		wantNil bool
	}{
		// Weight - metric
		{name: "gram lowercase", input: "g", wantKey: "g"},
		{name: "gram full", input: "gram", wantKey: "g"},
		{name: "grams plural", input: "grams", wantKey: "g"},
		{name: "kilogram", input: "kg", wantKey: "kg"},
		{name: "milligram", input: "mg", wantKey: "mg"},
		// Weight - imperial
		{name: "ounce abbrev", input: "oz", wantKey: "oz"},
		{name: "ounce full", input: "ounce", wantKey: "oz"},
		{name: "ounces plural", input: "ounces", wantKey: "oz"},
		{name: "pound", input: "lb", wantKey: "lb"},
		{name: "pounds plural", input: "lbs", wantKey: "lb"},
		{name: "pounds full", input: "pounds", wantKey: "lb"},
		// Volume - metric
		{name: "ml", input: "ml", wantKey: "ml"},
		{name: "millilitre british", input: "millilitre", wantKey: "ml"},
		{name: "liter", input: "L", wantKey: "L"},
		{name: "liter lowercase", input: "l", wantKey: "L"},
		{name: "litres", input: "litres", wantKey: "L"},
		// Volume - imperial
		{name: "tsp", input: "tsp", wantKey: "tsp"},
		{name: "teaspoon", input: "teaspoon", wantKey: "tsp"},
		{name: "tbsp", input: "tbsp", wantKey: "tbsp"},
		{name: "tablespoons", input: "tablespoons", wantKey: "tbsp"},
		{name: "tbs", input: "tbs", wantKey: "tbsp"},
		{name: "fl oz", input: "fl oz", wantKey: "fl oz"},
		{name: "cup", input: "cup", wantKey: "cup"},
		{name: "cups", input: "cups", wantKey: "cup"},
		{name: "pint", input: "pint", wantKey: "pint"},
		{name: "quart", input: "quart", wantKey: "quart"},
		{name: "gallon", input: "gallon", wantKey: "gallon"},
		{name: "gal", input: "gal", wantKey: "gallon"},
		// Case insensitive
		{name: "case upper G", input: "G", wantKey: "g"},
		{name: "case CUPS", input: "CUPS", wantKey: "cup"},
		{name: "case Tablespoon", input: "Tablespoon", wantKey: "tbsp"},
		// Whitespace
		{name: "leading space", input: " g", wantKey: "g"},
		{name: "trailing space", input: "g ", wantKey: "g"},
		// Unknown
		{name: "unknown unit", input: "pieces", wantNil: true},
		{name: "empty", input: "", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := lookupUnit(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("lookupUnit(%q) = %v, want nil", tt.input, got.key)
				}
				return
			}
			if got == nil {
				t.Fatalf("lookupUnit(%q) = nil, want key %q", tt.input, tt.wantKey)
			}
			if got.key != tt.wantKey {
				t.Errorf("lookupUnit(%q).key = %q, want %q", tt.input, got.key, tt.wantKey)
			}
		})
	}
}

func TestUnitHierarchy(t *testing.T) {
	t.Parallel()

	t.Run("imperial volume descending", func(t *testing.T) {
		t.Parallel()
		h := unitHierarchy(kindVolume, Imperial)
		if len(h) == 0 {
			t.Fatal("expected non-empty hierarchy")
		}
		for i := 1; i < len(h); i++ {
			if h[i].toBase > h[i-1].toBase {
				t.Errorf("hierarchy not descending: %s (%f) > %s (%f)",
					h[i].key, h[i].toBase, h[i-1].key, h[i-1].toBase)
			}
		}
	})

	t.Run("metric weight descending", func(t *testing.T) {
		t.Parallel()
		h := unitHierarchy(kindWeight, Metric)
		if len(h) == 0 {
			t.Fatal("expected non-empty hierarchy")
		}
		for i := 1; i < len(h); i++ {
			if h[i].toBase > h[i-1].toBase {
				t.Errorf("hierarchy not descending: %s (%f) > %s (%f)",
					h[i].key, h[i].toBase, h[i-1].key, h[i-1].toBase)
			}
		}
	})
}
