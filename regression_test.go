package recipemd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegression_EmptyIngredient(t *testing.T) {
	files, err := filepath.Glob("testdata/regression/empty_ingredient*.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test files found")
	}

	p := NewParser()
	for _, f := range files {
		name := filepath.Base(f)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			_, err = p.Parse(data)
			if err == nil {
				t.Error("expected parse error for empty ingredient")
			}
		})
	}
}
