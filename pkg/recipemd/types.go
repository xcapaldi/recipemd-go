// Package recipemd provides parsing, manipulation, and rendering for RecipeMD files.
// RecipeMD is a markdown format for recipe storage that follows the CommonMark specification.
package recipemd

import (
	"github.com/yuin/goldmark/ast"
)

// Recipe represents a parsed RecipeMD recipe with all its components.
type Recipe struct {
	Title            string            `json:"title"`
	Description      *string           `json:"description"`
	Tags             []string          `json:"tags"`
	Yields           []Yield           `json:"yields"`
	Ingredients      []Ingredient      `json:"ingredients"`
	IngredientGroups []IngredientGroup `json:"ingredient_groups"`
	Instructions     *string           `json:"instructions"`
}

// Yield represents a single yield specification (e.g., "4 Servings").
type Yield struct {
	Factor string  `json:"factor"` // The numeric factor as string to preserve original format
	Unit   *string `json:"unit"`   // The unit (e.g., "cups", "servings"), nil if not specified
}

// Amount represents the amount of an ingredient (e.g., "1.5 cups").
type Amount struct {
	Factor string  `json:"factor"` // The numeric factor as string
	Unit   *string `json:"unit"`   // The unit, nil if not specified
}

// Ingredient represents a single ingredient in the recipe.
type Ingredient struct {
	Name   string  `json:"name"`
	Amount *Amount `json:"amount"` // nil if no amount specified
	Link   *string `json:"link"`   // URL if ingredient is linked to another recipe
}

// IngredientGroup represents a group of related ingredients with a title.
type IngredientGroup struct {
	Title            string            `json:"title"`
	Ingredients      []Ingredient      `json:"ingredients"`
	IngredientGroups []IngredientGroup `json:"ingredient_groups"`
}

// MarkdownRecipe holds the intermediate AST representation of a RecipeMD document.
// This is the result of Phase 1 parsing before conversion to the domain model.
type MarkdownRecipe struct {
	TitleNode        ast.Node   // H1 heading
	DescriptionNodes []ast.Node // Paragraphs after title (before tags/yield/divider)
	TagsNode         ast.Node   // Single emphasis paragraph (optional)
	YieldNode        ast.Node   // Single strong emphasis paragraph (optional)
	IngredientNodes  []ast.Node // List items and headings between first --- and second ---
	InstructionNodes []ast.Node // Everything after second ---
	RawSource        []byte     // Original markdown source
}
