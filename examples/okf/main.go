// Okf reads a RecipeMD file and writes it as an Open Knowledge Format (OKF)
// concept document to stdout.
//
// Usage: okf <recipe.md>
//
// The output is a Markdown file with YAML frontmatter (type: Recipe, title,
// description, tags) followed by the recipe rendered as standard RecipeMD,
// suitable for inclusion in an OKF knowledge bundle.
//
//	okf recipe.md > recipe.okf.md
package main

import (
	"fmt"
	"os"

	recipemd "github.com/xcapaldi/recipemd-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: okf <recipe.md>")
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

	fmt.Print(p.RenderOKF(r, 2))
}
