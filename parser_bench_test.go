package recipemd

import (
	"os"
	"testing"
)

var benchSource []byte

func init() {
	var err error
	benchSource, err = os.ReadFile("testdata/canonical/recipe.md")
	if err != nil {
		panic(err)
	}
}

func BenchmarkParseRecipe(b *testing.B) {
	for b.Loop() {
		_, err := ParseRecipe(benchSource)
		if err != nil {
			b.Fatal(err)
		}
	}
}
