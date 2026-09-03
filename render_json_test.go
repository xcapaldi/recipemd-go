package recipemd

import (
	"encoding/json"
	"testing"
)

func TestRenderJSON(t *testing.T) {
	t.Parallel()
	r := &Recipe{
		Title:            "Test",
		Yields:           []Amount{{Factor: 4, Unit: new("servings")}},
		Tags:             []string{"easy"},
		Ingredients:      []Ingredient{{Name: "salt"}},
		IngredientGroups: []IngredientGroup{},
	}
	got, err := r.RenderJSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["title"] != "Test" {
		t.Errorf("title = %v", parsed["title"])
	}
}
