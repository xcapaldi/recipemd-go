// Scale reads a RecipeMD file, scales it by the given amount, and writes
// markdown to stdout.
//
// Usage: scale <recipe.md> <amount>
//
// <amount> is either a bare number to multiply all ingredients by that factor,
// or a quantity with a unit to scale for a specific yield:
//
//	scale recipe.md 2          # double the recipe
//	scale recipe.md 0.5        # halve the recipe
//	scale recipe.md "6 servings"  # scale to 6 servings
package main

import (
	"fmt"
	"os"
	"strings"

	recipemd "github.com/xcapaldi/recipemd-go"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: scale <recipe.md> <amount>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	amount, err := recipemd.ParseAmountString(strings.Join(os.Args[2:], " "))
	if err != nil || amount.Factor == 0 {
		fmt.Fprintln(os.Stderr, "invalid amount: must be a number or a quantity with a unit (e.g. \"6 servings\")")
		os.Exit(1)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if amount.Unit != nil {
		if err := r.ScaleForYield(amount); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		r.Scale(amount.Factor)
	}

	fmt.Print(p.RenderMarkdown(r, 2))
}
