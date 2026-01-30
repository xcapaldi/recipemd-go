package recipemd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParse_BasicRecipe(t *testing.T) {
	source := []byte(`# Guacamole

Some people call it guac.

*sauce, vegan*

**4 Servings, 200g**

---

- *1* avocado
- *.5 teaspoon* salt
- *1 1/2 pinches* red pepper flakes
- lemon juice

---

Remove flesh from avocado and roughly mash with fork.
Season to taste with salt, pepper and lemon juice.
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check title
	if recipe.Title != "Guacamole" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Guacamole")
	}

	// Check description
	if recipe.Description == nil {
		t.Error("Description is nil")
	} else if !strings.Contains(*recipe.Description, "Some people call it guac.") {
		t.Errorf("Description = %q, want to contain %q", *recipe.Description, "Some people call it guac.")
	}

	// Check tags
	expectedTags := []string{"sauce", "vegan"}
	if len(recipe.Tags) != len(expectedTags) {
		t.Errorf("Tags = %v, want %v", recipe.Tags, expectedTags)
	}
	for i, tag := range expectedTags {
		if i < len(recipe.Tags) && recipe.Tags[i] != tag {
			t.Errorf("Tags[%d] = %q, want %q", i, recipe.Tags[i], tag)
		}
	}

	// Check yields
	if len(recipe.Yields) != 2 {
		t.Errorf("len(Yields) = %d, want 2", len(recipe.Yields))
	} else {
		if recipe.Yields[0].Factor != "4" {
			t.Errorf("Yields[0].Factor = %q, want %q", recipe.Yields[0].Factor, "4")
		}
		if recipe.Yields[1].Factor != "200" {
			t.Errorf("Yields[1].Factor = %q, want %q", recipe.Yields[1].Factor, "200")
		}
	}

	// Check ingredients
	if len(recipe.Ingredients) != 4 {
		t.Errorf("len(Ingredients) = %d, want 4", len(recipe.Ingredients))
	}

	// Check first ingredient (avocado)
	if len(recipe.Ingredients) > 0 {
		ing := recipe.Ingredients[0]
		if ing.Name != "avocado" {
			t.Errorf("Ingredients[0].Name = %q, want %q", ing.Name, "avocado")
		}
		if ing.Amount == nil || ing.Amount.Factor != "1" {
			t.Errorf("Ingredients[0].Amount = %v, want Factor=1", ing.Amount)
		}
	}

	// Check second ingredient (salt with unit)
	if len(recipe.Ingredients) > 1 {
		ing := recipe.Ingredients[1]
		if ing.Name != "salt" {
			t.Errorf("Ingredients[1].Name = %q, want %q", ing.Name, "salt")
		}
		if ing.Amount == nil || ing.Amount.Factor != "0.5" {
			t.Errorf("Ingredients[1].Amount.Factor = %v, want 0.5", ing.Amount)
		}
		if ing.Amount == nil || ing.Amount.Unit == nil || *ing.Amount.Unit != "teaspoon" {
			t.Errorf("Ingredients[1].Amount.Unit = %v, want teaspoon", ing.Amount)
		}
	}

	// Check third ingredient (mixed number)
	if len(recipe.Ingredients) > 2 {
		ing := recipe.Ingredients[2]
		if ing.Amount == nil || ing.Amount.Factor != "1.5" {
			t.Errorf("Ingredients[2].Amount.Factor = %v, want 1.5", ing.Amount)
		}
	}

	// Check fourth ingredient (no amount)
	if len(recipe.Ingredients) > 3 {
		ing := recipe.Ingredients[3]
		if ing.Name != "lemon juice" {
			t.Errorf("Ingredients[3].Name = %q, want %q", ing.Name, "lemon juice")
		}
		if ing.Amount != nil {
			t.Errorf("Ingredients[3].Amount = %v, want nil", ing.Amount)
		}
	}

	// Check instructions
	if recipe.Instructions == nil {
		t.Error("Instructions is nil")
	} else if !strings.Contains(*recipe.Instructions, "Remove flesh from avocado") {
		t.Errorf("Instructions = %q, want to contain instructions text", *recipe.Instructions)
	}
}

func TestParse_IngredientsWithFractions(t *testing.T) {
	source := []byte(`# Recipe with ingredients

---

- *20 ml* water
- *1 cup* earl grey, hot
- *1 1/2 cup* coffee
- *¼ kg* cheese
- salt
- ingredients may contain *markdown*
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	tests := []struct {
		index      int
		name       string
		factor     string
		unit       *string
		hasAmount  bool
	}{
		{0, "water", "20", strPtr("ml"), true},
		{1, "earl grey, hot", "1", strPtr("cup"), true},
		{2, "coffee", "1.5", strPtr("cup"), true},
		{3, "cheese", "0.25", strPtr("kg"), true},
		{4, "salt", "", nil, false},
		{5, "ingredients may contain *markdown*", "", nil, false},
	}

	if len(recipe.Ingredients) != len(tests) {
		t.Fatalf("len(Ingredients) = %d, want %d", len(recipe.Ingredients), len(tests))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ing := recipe.Ingredients[tt.index]
			if ing.Name != tt.name {
				t.Errorf("Ingredients[%d].Name = %q, want %q", tt.index, ing.Name, tt.name)
			}
			if tt.hasAmount {
				if ing.Amount == nil {
					t.Errorf("Ingredients[%d].Amount is nil, want Factor=%s", tt.index, tt.factor)
				} else {
					if ing.Amount.Factor != tt.factor {
						t.Errorf("Ingredients[%d].Amount.Factor = %q, want %q", tt.index, ing.Amount.Factor, tt.factor)
					}
					if (ing.Amount.Unit == nil) != (tt.unit == nil) {
						t.Errorf("Ingredients[%d].Amount.Unit = %v, want %v", tt.index, ing.Amount.Unit, tt.unit)
					}
					if tt.unit != nil && ing.Amount.Unit != nil && *ing.Amount.Unit != *tt.unit {
						t.Errorf("Ingredients[%d].Amount.Unit = %q, want %q", tt.index, *ing.Amount.Unit, *tt.unit)
					}
				}
			} else if ing.Amount != nil {
				t.Errorf("Ingredients[%d].Amount = %v, want nil", tt.index, ing.Amount)
			}
		})
	}
}

func TestParse_Tags(t *testing.T) {
	source := []byte(`# Tags

*tag1, tag2, tag3, tag4, tag with special! char, tag5*
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	expected := []string{"tag1", "tag2", "tag3", "tag4", "tag with special! char", "tag5"}
	if len(recipe.Tags) != len(expected) {
		t.Fatalf("len(Tags) = %d, want %d", len(recipe.Tags), len(expected))
	}
	for i, tag := range expected {
		if recipe.Tags[i] != tag {
			t.Errorf("Tags[%d] = %q, want %q", i, recipe.Tags[i], tag)
		}
	}
}

func TestParse_Yields(t *testing.T) {
	source := []byte(`# Yields

**1.2 cups, 1,5 Tassen, 1 1/4 servings, 5 servings, 5**
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	tests := []struct {
		factor string
		unit   *string
	}{
		{"1.2", strPtr("cups")},
		{"1.5", strPtr("Tassen")},
		{"1.25", strPtr("servings")},
		{"5", strPtr("servings")},
		{"5", nil},
	}

	if len(recipe.Yields) != len(tests) {
		t.Fatalf("len(Yields) = %d, want %d", len(recipe.Yields), len(tests))
	}

	for i, tt := range tests {
		y := recipe.Yields[i]
		if y.Factor != tt.factor {
			t.Errorf("Yields[%d].Factor = %q, want %q", i, y.Factor, tt.factor)
		}
		if (y.Unit == nil) != (tt.unit == nil) {
			t.Errorf("Yields[%d].Unit = %v, want %v", i, y.Unit, tt.unit)
		}
		if tt.unit != nil && y.Unit != nil && *y.Unit != *tt.unit {
			t.Errorf("Yields[%d].Unit = %q, want %q", i, *y.Unit, *tt.unit)
		}
	}
}

func TestParse_IngredientGroups(t *testing.T) {
	source := []byte(`# Recipe with ingredient groups

---

- ingredient 0

## Group 1

- ingredient 1
- ingredient 2

### Subgroup 1.1

- ingredient 3
- ingredient 4

## Group 2

- ingredient 5
- ingredient 6
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check top-level ingredient
	if len(recipe.Ingredients) != 1 {
		t.Errorf("len(Ingredients) = %d, want 1", len(recipe.Ingredients))
	}
	if len(recipe.Ingredients) > 0 && recipe.Ingredients[0].Name != "ingredient 0" {
		t.Errorf("Ingredients[0].Name = %q, want %q", recipe.Ingredients[0].Name, "ingredient 0")
	}

	// Check groups
	if len(recipe.IngredientGroups) != 2 {
		t.Errorf("len(IngredientGroups) = %d, want 2", len(recipe.IngredientGroups))
	}

	// Check Group 1
	if len(recipe.IngredientGroups) > 0 {
		g1 := recipe.IngredientGroups[0]
		if g1.Title != "Group 1" {
			t.Errorf("IngredientGroups[0].Title = %q, want %q", g1.Title, "Group 1")
		}
		if len(g1.Ingredients) != 2 {
			t.Errorf("IngredientGroups[0].Ingredients = %d, want 2", len(g1.Ingredients))
		}
		// Check subgroup
		if len(g1.IngredientGroups) != 1 {
			t.Errorf("IngredientGroups[0].IngredientGroups = %d, want 1", len(g1.IngredientGroups))
		} else if g1.IngredientGroups[0].Title != "Subgroup 1.1" {
			t.Errorf("Subgroup title = %q, want %q", g1.IngredientGroups[0].Title, "Subgroup 1.1")
		}
	}

	// Check Group 2
	if len(recipe.IngredientGroups) > 1 {
		g2 := recipe.IngredientGroups[1]
		if g2.Title != "Group 2" {
			t.Errorf("IngredientGroups[1].Title = %q, want %q", g2.Title, "Group 2")
		}
		if len(g2.Ingredients) != 2 {
			t.Errorf("IngredientGroups[1].Ingredients = %d, want 2", len(g2.Ingredients))
		}
	}
}

func TestParse_LinkedIngredients(t *testing.T) {
	source := []byte(`# Recipe with links

---

- *1* [flour](./flour-recipe.md)
- [yeast](https://example.com/yeast)
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(recipe.Ingredients) != 2 {
		t.Fatalf("len(Ingredients) = %d, want 2", len(recipe.Ingredients))
	}

	// Check first linked ingredient
	ing1 := recipe.Ingredients[0]
	if ing1.Name != "flour" {
		t.Errorf("Ingredients[0].Name = %q, want %q", ing1.Name, "flour")
	}
	if ing1.Link == nil || *ing1.Link != "./flour-recipe.md" {
		t.Errorf("Ingredients[0].Link = %v, want %q", ing1.Link, "./flour-recipe.md")
	}
	if ing1.Amount == nil || ing1.Amount.Factor != "1" {
		t.Errorf("Ingredients[0].Amount = %v, want Factor=1", ing1.Amount)
	}

	// Check second linked ingredient (no amount)
	ing2 := recipe.Ingredients[1]
	if ing2.Name != "yeast" {
		t.Errorf("Ingredients[1].Name = %q, want %q", ing2.Name, "yeast")
	}
	if ing2.Link == nil || *ing2.Link != "https://example.com/yeast" {
		t.Errorf("Ingredients[1].Link = %v, want %q", ing2.Link, "https://example.com/yeast")
	}
	if ing2.Amount != nil {
		t.Errorf("Ingredients[1].Amount = %v, want nil", ing2.Amount)
	}
}

func TestParse_MinimalRecipe(t *testing.T) {
	source := []byte(`# Minimal Recipe

---

- water
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if recipe.Title != "Minimal Recipe" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Minimal Recipe")
	}
	if recipe.Description != nil {
		t.Errorf("Description = %v, want nil", recipe.Description)
	}
	if len(recipe.Tags) != 0 {
		t.Errorf("Tags = %v, want []", recipe.Tags)
	}
	if len(recipe.Yields) != 0 {
		t.Errorf("Yields = %v, want []", recipe.Yields)
	}
	if len(recipe.Ingredients) != 1 {
		t.Errorf("len(Ingredients) = %d, want 1", len(recipe.Ingredients))
	}
	if recipe.Instructions != nil {
		t.Errorf("Instructions = %v, want nil", recipe.Instructions)
	}
}

func TestParse_InvalidRecipes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "empty document",
			source:  "",
			wantErr: true,
		},
		{
			name:    "no title",
			source:  "Just some text without a heading",
			wantErr: true,
		},
		{
			name:    "second level heading as title",
			source:  "## This is not a valid title",
			wantErr: true,
		},
		{
			name: "paragraph in ingredients section",
			source: `# Title

---

A paragraph is not valid here.
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.source))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecipe_ToJSON(t *testing.T) {
	source := []byte(`# Test Recipe

*tag1, tag2*

**4 servings**

---

- *1* ingredient

---

Instructions here.
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	jsonBytes, err := recipe.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if result["title"] != "Test Recipe" {
		t.Errorf("JSON title = %v, want %q", result["title"], "Test Recipe")
	}
}

func TestRecipe_Scale(t *testing.T) {
	source := []byte(`# Scalable Recipe

**4 servings**

---

- *1* apple
- *2.5 cups* flour
- salt

---

Mix.
`)

	recipe, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Scale by 2
	recipe.Scale(2.0)

	// Check yields
	if len(recipe.Yields) > 0 && recipe.Yields[0].Factor != "8" {
		t.Errorf("After scale, Yields[0].Factor = %q, want %q", recipe.Yields[0].Factor, "8")
	}

	// Check ingredients
	if len(recipe.Ingredients) > 0 && recipe.Ingredients[0].Amount.Factor != "2" {
		t.Errorf("After scale, Ingredients[0].Amount.Factor = %q, want %q", recipe.Ingredients[0].Amount.Factor, "2")
	}
	if len(recipe.Ingredients) > 1 && recipe.Ingredients[1].Amount.Factor != "5" {
		t.Errorf("After scale, Ingredients[1].Amount.Factor = %q, want %q", recipe.Ingredients[1].Amount.Factor, "5")
	}
	// Salt has no amount, should be unchanged
	if len(recipe.Ingredients) > 2 && recipe.Ingredients[2].Amount != nil {
		t.Errorf("After scale, salt should still have no amount")
	}
}

func TestRecipe_Validate(t *testing.T) {
	tests := []struct {
		name    string
		recipe  Recipe
		wantErr bool
	}{
		{
			name: "valid recipe",
			recipe: Recipe{
				Title: "Test",
				Ingredients: []Ingredient{
					{Name: "water"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing title",
			recipe: Recipe{
				Title: "",
			},
			wantErr: true,
		},
		{
			name: "ingredient without name",
			recipe: Recipe{
				Title: "Test",
				Ingredients: []Ingredient{
					{Name: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "ingredient with invalid amount",
			recipe: Recipe{
				Title: "Test",
				Ingredients: []Ingredient{
					{Name: "water", Amount: &Amount{Factor: "abc"}},
				},
			},
			wantErr: true,
		},
		{
			name: "group without title",
			recipe: Recipe{
				Title: "Test",
				IngredientGroups: []IngredientGroup{
					{Title: "", Ingredients: []Ingredient{{Name: "water"}}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.recipe.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecipe_Clone(t *testing.T) {
	source := []byte(`# Original

Some description.

*tag1, tag2*

**4 servings**

---

- *1* ingredient

---

Instructions.
`)

	original, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	clone := original.Clone()

	// Modify clone
	clone.Title = "Modified"
	clone.Tags[0] = "modified-tag"
	clone.Scale(2.0)

	// Original should be unchanged
	if original.Title != "Original" {
		t.Errorf("Original title was modified")
	}
	if original.Tags[0] != "tag1" {
		t.Errorf("Original tags were modified")
	}
	if original.Ingredients[0].Amount.Factor != "1" {
		t.Errorf("Original ingredients were modified")
	}
}
