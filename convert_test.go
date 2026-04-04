package recipemd

import (
	"math"
	"testing"
)

func sp(s string) *string { return &s }

func TestConvertAmount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		amount     Amount
		system     UnitSystem
		wantFactor float64
		wantUnit   string
		tolerance  float64
	}{
		{
			name:       "grams to ounces",
			amount:     Amount{Factor: 100, Unit: sp("g")},
			system:     Imperial,
			wantFactor: 3.527,
			wantUnit:   "oz",
			tolerance:  0.01,
		},
		{
			name:       "kg to pounds",
			amount:     Amount{Factor: 1, Unit: sp("kg")},
			system:     Imperial,
			wantFactor: 2.205,
			wantUnit:   "lb",
			tolerance:  0.01,
		},
		{
			name:       "ounces to grams",
			amount:     Amount{Factor: 4, Unit: sp("oz")},
			system:     Metric,
			wantFactor: 113.398,
			wantUnit:   "g",
			tolerance:  0.01,
		},
		{
			name:       "pounds to grams",
			amount:     Amount{Factor: 0.5, Unit: sp("lb")},
			system:     Metric,
			wantFactor: 226.796,
			wantUnit:   "g",
			tolerance:  0.01,
		},
		{
			name:       "ml to tsp",
			amount:     Amount{Factor: 5, Unit: sp("ml")},
			system:     Imperial,
			wantFactor: 1.014,
			wantUnit:   "tsp",
			tolerance:  0.01,
		},
		{
			name:       "liters to quarts",
			amount:     Amount{Factor: 2, Unit: sp("L")},
			system:     Imperial,
			wantFactor: 2.113,
			wantUnit:   "quart",
			tolerance:  0.01,
		},
		{
			name:       "cups to ml",
			amount:     Amount{Factor: 1, Unit: sp("cup")},
			system:     Metric,
			wantFactor: 236.588,
			wantUnit:   "ml",
			tolerance:  0.1,
		},
		{
			name:       "already in target system",
			amount:     Amount{Factor: 100, Unit: sp("g")},
			system:     Metric,
			wantFactor: 100,
			wantUnit:   "g",
			tolerance:  0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertAmount(tt.amount, tt.system)
			if got.Unit == nil {
				t.Fatal("got nil unit")
			}
			if *got.Unit != tt.wantUnit {
				t.Errorf("unit = %q, want %q", *got.Unit, tt.wantUnit)
			}
			if math.Abs(got.Factor-tt.wantFactor) > tt.tolerance {
				t.Errorf("factor = %f, want %f (±%f)", got.Factor, tt.wantFactor, tt.tolerance)
			}
		})
	}
}

func TestConvertAmountUnitless(t *testing.T) {
	t.Parallel()
	a := Amount{Factor: 3}
	got := convertAmount(a, Imperial)
	if got.Factor != 3 || got.Unit != nil {
		t.Errorf("unitless amount should be unchanged, got factor=%f unit=%v", got.Factor, got.Unit)
	}
}

func TestConvertAmountUnknownUnit(t *testing.T) {
	t.Parallel()
	a := Amount{Factor: 2, Unit: sp("pieces")}
	got := convertAmount(a, Metric)
	if got.Factor != 2 || *got.Unit != "pieces" {
		t.Errorf("unknown unit should be unchanged, got factor=%f unit=%q", got.Factor, *got.Unit)
	}
}

func TestDecomposeAmount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		amount     Amount
		system     UnitSystem
		wantParts  int
		wantFirst  string // expected first unit display
		wantSecond string // expected second unit display (if > 1 part)
		desc       string // human-readable expected output
	}{
		{
			name:       "17 tbsp to cups and tbsp",
			amount:     Amount{Factor: 17, Unit: sp("tbsp")},
			system:     Imperial,
			wantParts:  2,
			wantFirst:  "cup",
			wantSecond: "tbsp",
			desc:       "1 cup and 1 tbsp",
		},
		{
			name:      "1 tsp stays as tsp",
			amount:    Amount{Factor: 1, Unit: sp("tsp")},
			system:    Imperial,
			wantParts: 1,
			wantFirst: "tsp",
			desc:      "1 tsp",
		},
		{
			name:       "500g to imperial",
			amount:     Amount{Factor: 500, Unit: sp("g")},
			system:     Imperial,
			wantParts:  2,
			wantFirst:  "lb",
			wantSecond: "oz",
			desc:       "1 lb and ~1.6 oz",
		},
		{
			name:      "small ml to tsp",
			amount:    Amount{Factor: 5, Unit: sp("ml")},
			system:    Imperial,
			wantParts: 1,
			wantFirst: "tsp",
			desc:      "~1 tsp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := decomposeAmount(tt.amount, tt.system)
			if len(got) != tt.wantParts {
				t.Fatalf("got %d parts, want %d. parts: %v", len(got), tt.wantParts, got)
			}
			if got[0].Unit == nil || *got[0].Unit != tt.wantFirst {
				unitStr := "<nil>"
				if got[0].Unit != nil {
					unitStr = *got[0].Unit
				}
				t.Errorf("first unit = %q, want %q", unitStr, tt.wantFirst)
			}
			if tt.wantParts > 1 {
				if got[1].Unit == nil || *got[1].Unit != tt.wantSecond {
					unitStr := "<nil>"
					if got[1].Unit != nil {
						unitStr = *got[1].Unit
					}
					t.Errorf("second unit = %q, want %q", unitStr, tt.wantSecond)
				}
			}
		})
	}
}

func TestDecompose3Point3Cups(t *testing.T) {
	t.Parallel()
	// 3.3126 cups should become approximately 3¼ cup and 1 tbsp
	got := decomposeAmount(Amount{Factor: 3.3126, Unit: sp("cups")}, Imperial)
	if len(got) < 1 {
		t.Fatal("expected at least 1 component")
	}
	if got[0].Unit == nil || *got[0].Unit != "cup" {
		t.Errorf("first component unit = %v, want cup", got[0].Unit)
	}
	// The first component should be around 3.25 (3¼)
	if math.Abs(got[0].Factor-3.25) > 0.1 {
		t.Errorf("first component factor = %f, want ~3.25", got[0].Factor)
	}
	if len(got) == 2 {
		if got[1].Unit == nil || *got[1].Unit != "tbsp" {
			t.Errorf("second component unit = %v, want tbsp", got[1].Unit)
		}
		if math.Abs(got[1].Factor-1) > 0.5 {
			t.Errorf("second component factor = %f, want ~1", got[1].Factor)
		}
	}
}

func TestRecipeConvertUnits(t *testing.T) {
	t.Parallel()

	g := "g"
	ml := "ml"
	recipe := &Recipe{
		Title: "Test",
		Yields: []Amount{
			{Factor: 500, Unit: &g},
		},
		Ingredients: []Ingredient{
			{Name: "flour", Amount: &Amount{Factor: 200, Unit: &g}},
			{Name: "water", Amount: &Amount{Factor: 300, Unit: &ml}},
			{Name: "salt"},
		},
	}

	recipe.ConvertUnits(Imperial)

	// Yields should be converted
	if recipe.Yields[0].Unit == nil || *recipe.Yields[0].Unit != "lb" {
		t.Errorf("yield unit = %v, want lb", recipe.Yields[0].Unit)
	}

	// Flour should be converted to oz
	if recipe.Ingredients[0].Amount == nil || recipe.Ingredients[0].Amount.Unit == nil {
		t.Fatal("flour amount should not be nil after conversion")
	}

	// Water should be converted to imperial volume
	if recipe.Ingredients[1].Amount == nil || recipe.Ingredients[1].Amount.Unit == nil {
		t.Fatal("water amount should not be nil after conversion")
	}

	// Salt (no amount) should be unchanged
	if recipe.Ingredients[2].Amount != nil {
		t.Error("salt should have no amount")
	}
}

func TestBestUnit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		base    float64 // in base units (g or ml)
		kind    unitKind
		system  UnitSystem
		wantKey string
	}{
		{name: "50g metric", base: 50, kind: kindWeight, system: Metric, wantKey: "g"},
		{name: "1500g metric", base: 1500, kind: kindWeight, system: Metric, wantKey: "kg"},
		{name: "100ml metric", base: 100, kind: kindVolume, system: Metric, wantKey: "ml"},
		{name: "2000ml metric", base: 2000, kind: kindVolume, system: Metric, wantKey: "L"},
		{name: "30g imperial", base: 30, kind: kindWeight, system: Imperial, wantKey: "oz"},
		{name: "500g imperial", base: 500, kind: kindWeight, system: Imperial, wantKey: "lb"},
		{name: "15ml imperial", base: 15, kind: kindVolume, system: Imperial, wantKey: "tbsp"},
		{name: "250ml imperial", base: 250, kind: kindVolume, system: Imperial, wantKey: "cup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := bestUnit(tt.base, tt.kind, tt.system)
			if got == nil {
				t.Fatal("bestUnit returned nil")
			}
			if got.key != tt.wantKey {
				t.Errorf("bestUnit(%f, %v, %v) = %q, want %q",
					tt.base, tt.kind, tt.system, got.key, tt.wantKey)
			}
		})
	}
}
