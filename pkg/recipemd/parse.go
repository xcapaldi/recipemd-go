package recipemd

import (
	"fmt"
	"os"
)

// Parse parses a RecipeMD source into a Recipe.
// This is the main entry point for the library.
func Parse(source []byte) (*Recipe, error) {
	// Phase 1: Extract markdown structure
	mr, err := ParseMarkdownStructure(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown structure: %w", err)
	}

	// Phase 2: Convert to domain model
	recipe, err := mr.ToRecipe()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to recipe: %w", err)
	}

	return recipe, nil
}

// ParseFile parses a RecipeMD file from the given path.
func ParseFile(path string) (*Recipe, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return Parse(source)
}

// MustParse parses a RecipeMD source and panics on error.
// This is useful for testing and static recipe definitions.
func MustParse(source []byte) *Recipe {
	recipe, err := Parse(source)
	if err != nil {
		panic(err)
	}
	return recipe
}
