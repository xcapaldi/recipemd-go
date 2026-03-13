// Parse reads a RecipeMD file and writes compact JSON to stdout.
//
// Usage: parse <recipe.md>
//
// The output can be piped to jq for further processing:
//
//	parse recipe.md | jq .title
//	parse recipe.md | jq -r '.tags[]'
//	parse recipe.md | jq -r '[.. | .ingredients[]? | .name] | unique[]'
package main

import (
	"fmt"
	"os"

	recipemd "github.com/xcapaldi/recipemd-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: parse <recipe.md>")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out, err := p.RenderJSON(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(append(out, '\n'))
}
