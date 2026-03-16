package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	recipemd "github.com/xcapaldi/recipemd-go"
	"github.com/xcapaldi/recipemd-go/examples/helper"
)

func ptr[T any](v T) *T { return &v }

// TestFlattenInlinesLinksRecursively verifies that linked ingredients are
// resolved relative to each file, not the working directory, and that chains
// of links (main -> sauce -> subdir/stock) are fully inlined.
func TestFlattenInlinesLinksRecursively(t *testing.T) {
	recipeFile, err := filepath.Abs("../../testdata/flatten/main.md")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(recipeFile)
	if err != nil {
		t.Fatal(err)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if err := helper.Flatten(p, r, recipeFile); err != nil {
		t.Fatalf("flatten: %v", err)
	}

	want := []string{"pasta", "olive oil", "water", "bouillon cube"}
	got := make([]string, len(r.Ingredients))
	for i, ing := range r.Ingredients {
		got[i] = ing.Name
	}
	if len(got) != len(want) {
		t.Fatalf("ingredients = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ingredient[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFlatten(t *testing.T) {
	t.Parallel()

	t.Run("no links unchanged", func(t *testing.T) {
		t.Parallel()
		p := recipemd.NewParser()
		r := &recipemd.Recipe{
			Ingredients:      []recipemd.Ingredient{{Name: "salt"}},
			IngredientGroups: []recipemd.IngredientGroup{},
		}
		if err := helper.Flatten(p, r, "/fake/recipe.md"); err != nil {
			t.Fatal(err)
		}
		if len(r.Ingredients) != 1 || r.Ingredients[0].Name != "salt" {
			t.Errorf("unexpected change: %+v", r.Ingredients)
		}
	})

	t.Run("remote link preserved", func(t *testing.T) {
		t.Parallel()
		p := recipemd.NewParser()
		r := &recipemd.Recipe{
			Ingredients:      []recipemd.Ingredient{{Name: "sauce", Link: ptr("https://example.com/sauce.md")}},
			IngredientGroups: []recipemd.IngredientGroup{},
		}
		if err := helper.Flatten(p, r, "/fake/recipe.md"); err != nil {
			t.Fatal(err)
		}
		if r.Ingredients[0].Link == nil {
			t.Error("remote link should be preserved")
		}
	})

	t.Run("local link resolved", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		linked := "# Sauce\n\n---\n\n- *1 cup* tomato\n- basil\n"
		if err := os.WriteFile(filepath.Join(dir, "sauce.md"), []byte(linked), 0644); err != nil {
			t.Fatal(err)
		}
		main := filepath.Join(dir, "main.md")

		p := recipemd.NewParser()
		r := &recipemd.Recipe{
			Ingredients:      []recipemd.Ingredient{{Name: "sauce", Link: ptr("sauce.md"), Amount: &recipemd.Amount{Factor: 2, Unit: ptr("cups")}}},
			IngredientGroups: []recipemd.IngredientGroup{},
		}
		if err := helper.Flatten(p, r, main); err != nil {
			t.Fatal(err)
		}
		if len(r.Ingredients) < 1 {
			t.Fatal("expected inlined ingredients")
		}
	})

	t.Run("missing file error", func(t *testing.T) {
		t.Parallel()
		p := recipemd.NewParser()
		r := &recipemd.Recipe{
			Ingredients:      []recipemd.Ingredient{{Name: "x", Link: ptr("nonexistent.md")}},
			IngredientGroups: []recipemd.IngredientGroup{},
		}
		if err := helper.Flatten(p, r, "/fake/recipe.md"); err == nil {
			t.Fatal("expected error for missing linked file")
		}
	})
}

// TestFlattenHTTPLinksPreserved verifies that HTTP(S) links are left as-is.
func TestFlattenHTTPLinksPreserved(t *testing.T) {
	input := []byte("# Recipe\n\n---\n\n- [sauce](https://example.org/sauce.md)\n")

	dir := t.TempDir()
	recipeFile := filepath.Join(dir, "recipe.md")
	if err := os.WriteFile(recipeFile, input, 0644); err != nil {
		t.Fatal(err)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	if err := helper.Flatten(p, r, recipeFile); err != nil {
		t.Fatalf("flatten: %v", err)
	}

	if len(r.Ingredients) != 1 {
		t.Fatalf("got %d ingredients, want 1", len(r.Ingredients))
	}
	if r.Ingredients[0].Link == nil {
		t.Error("link ingredient should be preserved with its link")
	}
}
