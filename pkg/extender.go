package recipemd

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	rparser "github.com/xcapaldi/recipemd-go/pkg/goldmark-recipemd/parser"
	rrenderer "github.com/xcapaldi/recipemd-go/pkg/goldmark-recipemd/renderer"
)

// RecipeMD is a goldmark extension for parsing RecipeMD documents
type RecipeMD struct {
	schemaOrg bool
}

// Option configures the RecipeMD extension
type Option func(*RecipeMD)

// WithSchemaOrg enables schema.org microdata in HTML output
func WithSchemaOrg() Option {
	return func(r *RecipeMD) {
		r.schemaOrg = true
	}
}

// New creates a new RecipeMD extension
func New(opts ...Option) *RecipeMD {
	r := &RecipeMD{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Extend implements goldmark.Extender
func (r *RecipeMD) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(rparser.NewRecipeTransformer(), 100),
		),
	)

	var htmlOpts []rrenderer.HTMLRendererOption
	if r.schemaOrg {
		htmlOpts = append(htmlOpts, rrenderer.WithSchemaOrg())
	}

	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(rrenderer.NewHTMLRenderer(htmlOpts...), 100),
		),
	)
}

var _ goldmark.Extender = (*RecipeMD)(nil)
