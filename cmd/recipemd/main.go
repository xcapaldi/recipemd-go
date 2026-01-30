// Command recipemd is a CLI tool for parsing, scaling, and converting RecipeMD files.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xcapaldi/recipemd-go/pkg/recipemd"
)

var (
	outputFormat = flag.String("format", "json", "Output format: json, markdown, html")
	scaleFactor  = flag.Float64("scale", 1.0, "Scale factor for ingredient amounts and yields")
	validate     = flag.Bool("validate", false, "Validate the recipe and exit")
	outputFile   = flag.String("output", "", "Output file (default: stdout)")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	// Read input file
	source, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse recipe
	recipe, err := recipemd.Parse(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing recipe: %v\n", err)
		os.Exit(1)
	}

	// Validate if requested
	if *validate {
		if err := recipe.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Recipe is valid.")
		return
	}

	// Scale if requested
	if *scaleFactor != 1.0 {
		recipe.Scale(*scaleFactor)
	}

	// Render output
	var output []byte
	switch *outputFormat {
	case "json":
		output, err = recipe.ToJSON()
	case "markdown", "md":
		output, err = recipe.ToMarkdown()
	case "html":
		output, err = recipe.ToHTML()
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *outputFormat)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, output, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Output written to %s\n", *outputFile)
	} else {
		fmt.Print(string(output))
	}
}

func usage() {
	progName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, `Usage: %s [options] <recipe.md>

Parse, scale, and convert RecipeMD files.

Options:
`, progName)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Examples:
  # Parse and output JSON
  %s recipe.md

  # Scale recipe by 2x and output markdown
  %s -scale 2.0 -format markdown recipe.md

  # Validate recipe
  %s -validate recipe.md

  # Convert to HTML and save to file
  %s -format html -output recipe.html recipe.md

`, progName, progName, progName, progName)
}
