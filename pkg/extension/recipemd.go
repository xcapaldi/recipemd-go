package extension

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// RecipeMD is an extension that provides RecipeMD markdown functionalities.
type RecipeMD struct {
}

// NewRecipeMD creates a new RecipeMD extension
func NewRecipeMD() *RecipeMD {
	return &RecipeMD{}
}

// Extend extends the goldmark parser with RecipeMD support
func (e *RecipeMD) Extend(m goldmark.Markdown) {
	// Register the AST transformer
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(NewRecipeParser(), 100),
		),
	)

	// Register the renderer
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewJSONRenderer(), 100),
		),
	)
}
