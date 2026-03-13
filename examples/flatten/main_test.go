package main

import (
	"os"
	"path/filepath"
	"testing"

	recipemd "github.com/xcapaldi/recipemd-go"
)

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
	r, err := p.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Flatten(r, recipeFile); err != nil {
		t.Fatalf("Flatten: %v", err)
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

// TestFlattenHTTPLinksPreserved verifies that HTTP(S) links are left as-is.
func TestFlattenHTTPLinksPreserved(t *testing.T) {
	input := []byte("# Recipe\n\n---\n\n- [sauce](https://example.org/sauce.md)\n")

	dir := t.TempDir()
	recipeFile := filepath.Join(dir, "recipe.md")
	if err := os.WriteFile(recipeFile, input, 0644); err != nil {
		t.Fatal(err)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(input)
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Flatten(r, recipeFile); err != nil {
		t.Fatalf("Flatten: %v", err)
	}

	if len(r.Ingredients) != 1 {
		t.Fatalf("got %d ingredients, want 1", len(r.Ingredients))
	}
	if r.Ingredients[0].Link == nil {
		t.Error("link ingredient should be preserved with its link")
	}
}
