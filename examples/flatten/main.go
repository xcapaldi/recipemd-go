// Flatten reads a RecipeMD file, inlines all locally-linked recipes, and
// writes the combined recipe as markdown to stdout.
//
// Usage: flatten <recipe.md>
//
// Only local file links are resolved. HTTP(S) links are left as-is.
package main

import (
	"fmt"
	"os"

	recipemd "github.com/xcapaldi/recipemd-go"
	"github.com/xcapaldi/recipemd-go/examples/helper"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: flatten <recipe.md>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	p := recipemd.NewParser()
	r, err := p.Parse(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := helper.Flatten(p, r, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Print(r.RenderMarkdown(2))
}
