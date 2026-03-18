//go:build js && wasm

// Package main is a WebAssembly entry point that exposes the recipemd-go
// library to JavaScript running in a browser.
//
// Build with:
//
//	GOOS=js GOARCH=wasm go build -o recipemd.wasm ./examples/wasm
//
// Copy the Go runtime support file alongside the .wasm:
//
//	cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" examples/wasm/
//
// Then load both files in an HTML page (see index.html in this directory).
//
// Exposed JavaScript API (available as window.recipemd after the WASM loads):
//
//	recipemd.renderHTML(markdown, rounding?)
//	  → {result: string, error: string|null}
//
//	recipemd.scaleByFactor(markdown, factor, rounding?)
//	  → {result: string, error: string|null}
//
//	recipemd.scaleForYield(markdown, yieldString, rounding?)
//	  → {result: string, error: string|null}
//	  yieldString examples: "6 servings", "2", "12 cookies"
//
//	recipemd.parseToJSON(markdown)
//	  → {result: string, error: string|null}
package main

import (
	"encoding/json"
	"strings"
	"syscall/js"

	recipemd "github.com/xcapaldi/recipemd-go"
)

func main() {
	js.Global().Set("recipemd", js.ValueOf(map[string]any{
		"renderHTML":    js.FuncOf(renderHTML),
		"scaleByFactor": js.FuncOf(scaleByFactor),
		"scaleForYield": js.FuncOf(scaleForYield),
		"parseToJSON":   js.FuncOf(parseToJSON),
	}))
	// Block forever — the WASM module must stay alive to serve JS calls.
	select {}
}

// ok wraps a successful result.
func ok(val string) map[string]any {
	return map[string]any{"result": val, "error": nil}
}

// fail wraps an error result.
func fail(err string) map[string]any {
	return map[string]any{"result": "", "error": err}
}

// rounding extracts an optional integer rounding argument at position idx,
// defaulting to 2 when absent or undefined.
func roundingArg(args []js.Value, idx int) int {
	if idx < len(args) && !args[idx].IsUndefined() && !args[idx].IsNull() {
		return args[idx].Int()
	}
	return 2
}

// renderHTML parses a RecipeMD markdown string and returns an HTML <article>.
//
// JS: recipemd.renderHTML(markdown, rounding?)
func renderHTML(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return fail("renderHTML requires a markdown string argument")
	}
	md := args[0].String()
	rounding := roundingArg(args, 1)

	p := recipemd.NewParser()
	recipe, err := p.Parse(strings.NewReader(md))
	if err != nil {
		return fail(err.Error())
	}
	return ok(p.RenderHTML(recipe, rounding))
}

// scaleByFactor parses a RecipeMD markdown string, scales it by factor, and
// returns the scaled recipe as a RecipeMD markdown string.
//
// JS: recipemd.scaleByFactor(markdown, factor, rounding?)
func scaleByFactor(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return fail("scaleByFactor requires markdown and factor arguments")
	}
	md := args[0].String()
	factor := args[1].Float()
	rounding := roundingArg(args, 2)

	p := recipemd.NewParser()
	recipe, err := p.Parse(strings.NewReader(md))
	if err != nil {
		return fail(err.Error())
	}
	recipe.Scale(factor)
	return ok(p.RenderMarkdown(recipe, rounding))
}

// scaleForYield parses a RecipeMD markdown string, scales it to the requested
// yield, and returns the scaled recipe as a RecipeMD markdown string.
//
// yieldString is a human-readable amount like "6 servings" or "24 cookies".
// A bare number like "2" doubles the recipe (treated as a raw multiplier when
// no matching yield unit exists).
//
// JS: recipemd.scaleForYield(markdown, yieldString, rounding?)
func scaleForYield(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return fail("scaleForYield requires markdown and yieldString arguments")
	}
	md := args[0].String()
	yieldStr := args[1].String()
	rounding := roundingArg(args, 2)

	amount, err := recipemd.ParseAmountString(yieldStr)
	if err != nil || amount.Factor == 0 {
		return fail("invalid yield: must be a number or a quantity with a unit (e.g. \"6 servings\")")
	}

	p := recipemd.NewParser()
	recipe, err := p.Parse(strings.NewReader(md))
	if err != nil {
		return fail(err.Error())
	}

	if amount.Unit != nil {
		if err := recipe.ScaleForYield(amount); err != nil {
			return fail(err.Error())
		}
	} else {
		recipe.Scale(amount.Factor)
	}

	return ok(p.RenderMarkdown(recipe, rounding))
}

// parseToJSON parses a RecipeMD markdown string and returns the structured
// recipe as a JSON string.
//
// JS: recipemd.parseToJSON(markdown)
func parseToJSON(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return fail("parseToJSON requires a markdown string argument")
	}
	md := args[0].String()

	p := recipemd.NewParser()
	recipe, err := p.Parse(strings.NewReader(md))
	if err != nil {
		return fail(err.Error())
	}

	b, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return fail(err.Error())
	}
	return ok(string(b))
}
