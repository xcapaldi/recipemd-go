package recipemd

import (
	"encoding/json"
	"testing"
)

func TestParse_TitleAndDescription(t *testing.T) {
	input := []byte(`# Guacamole

Some people call it guac.

It's delicious with chips.

---

- avocado
`)
	recipe, err := ParseRecipe(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	b, _ := json.MarshalIndent(recipe, "", "  ")
	t.Logf("Parsed recipe:\n%s", b)

	if recipe.Title != "Guacamole" {
		t.Errorf("Title = %q, want %q", recipe.Title, "Guacamole")
	}

	wantDesc := "Some people call it guac.\n\nIt's delicious with chips."
	if recipe.Description == nil {
		t.Fatal("Description is nil")
	}
	if *recipe.Description != wantDesc {
		t.Errorf("Description = %q, want %q", *recipe.Description, wantDesc)
	}
}

var sampleRecipe = []byte(`# Guacamole

Some people call it guac.

*sauce, vegan*

**4 Servings, 200g**

---

- *1* avocado
- *.5 teaspoon* salt
- *1 1/2 pinches* red pepper flakes
- lemon juice

---

Remove flesh from avocado and roughly mash with fork. Season to taste
with salt, pepper and lemon juice.
`)

func TestParse_FullRecipe(t *testing.T) {
	recipe, err := ParseRecipe(sampleRecipe)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	b, _ := json.MarshalIndent(recipe, "", "  ")
	t.Logf("Parsed recipe:\n%s", b)
}

