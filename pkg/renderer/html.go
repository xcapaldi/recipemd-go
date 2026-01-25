package renderer

import (
	"fmt"
	"html"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/xcapaldi/recipemd-go/pkg/goldmark-recipemd/ast"
)

// HTMLRendererConfig configures the HTML renderer
type HTMLRendererConfig struct {
	SchemaOrg bool
}

// HTMLRenderer renders RecipeMD AST nodes to HTML
type HTMLRenderer struct {
	config HTMLRendererConfig
}

// NewHTMLRenderer creates a new HTML renderer
func NewHTMLRenderer(opts ...HTMLRendererOption) *HTMLRenderer {
	r := &HTMLRenderer{}
	for _, opt := range opts {
		opt(&r.config)
	}
	return r
}

type HTMLRendererOption func(*HTMLRendererConfig)

// WithSchemaOrg enables schema.org microdata in HTML output
func WithSchemaOrg() HTMLRendererOption {
	return func(c *HTMLRendererConfig) {
		c.SchemaOrg = true
	}
}

// RegisterFuncs registers node rendering functions
func (r *HTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindRecipe, r.renderRecipe)
	reg.Register(ast.KindTitle, r.renderTitle)
	reg.Register(ast.KindDescription, r.renderDescription)
	reg.Register(ast.KindTags, r.renderTags)
	reg.Register(ast.KindYields, r.renderYields)
	reg.Register(ast.KindIngredients, r.renderIngredients)
	reg.Register(ast.KindIngredient, r.renderIngredient)
	reg.Register(ast.KindIngredientGroup, r.renderIngredientGroup)
	reg.Register(ast.KindInstructions, r.renderInstructions)
}

func (r *HTMLRenderer) renderRecipe(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		if r.config.SchemaOrg {
			_, _ = w.WriteString(`<article itemscope itemtype="https://schema.org/Recipe">`)
		} else {
			_, _ = w.WriteString(`<article class="recipe">`)
		}
		_, _ = w.WriteString("\n")
	} else {
		_, _ = w.WriteString("</article>\n")
	}
	return gast.WalkContinue, nil
}

func (r *HTMLRenderer) renderTitle(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		if r.config.SchemaOrg {
			_, _ = w.WriteString(`<h1 itemprop="name">`)
		} else {
			_, _ = w.WriteString("<h1>")
		}
		_, _ = w.WriteString(html.EscapeString(extractText(node, source)))
		_, _ = w.WriteString("</h1>\n")
	}
	return gast.WalkSkipChildren, nil
}

func (r *HTMLRenderer) renderDescription(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		if r.config.SchemaOrg {
			_, _ = w.WriteString(`<div class="description" itemprop="description">`)
		} else {
			_, _ = w.WriteString(`<div class="description">`)
		}
		_, _ = w.WriteString("\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return gast.WalkContinue, nil
}

func (r *HTMLRenderer) renderTags(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<div class="tags">`)
		text := extractText(node, source)
		tags := strings.Split(text, ",")
		for i, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if i > 0 {
				_, _ = w.WriteString(" ")
			}
			if r.config.SchemaOrg {
				_, _ = fmt.Fprintf(w, `<span class="tag" itemprop="keywords">%s</span>`, html.EscapeString(tag))
			} else {
				_, _ = fmt.Fprintf(w, `<span class="tag">%s</span>`, html.EscapeString(tag))
			}
		}
		_, _ = w.WriteString("</div>\n")
	}
	return gast.WalkSkipChildren, nil
}

func (r *HTMLRenderer) renderYields(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<div class="yields">`)
		text := extractText(node, source)
		yields := strings.Split(text, ",")
		for i, yield := range yields {
			yield = strings.TrimSpace(yield)
			if yield == "" {
				continue
			}
			if i > 0 {
				_, _ = w.WriteString(", ")
			}
			if r.config.SchemaOrg {
				_, _ = fmt.Fprintf(w, `<span itemprop="recipeYield">%s</span>`, html.EscapeString(yield))
			} else {
				_, _ = fmt.Fprintf(w, `<span>%s</span>`, html.EscapeString(yield))
			}
		}
		_, _ = w.WriteString("</div>\n")
	}
	return gast.WalkSkipChildren, nil
}

func (r *HTMLRenderer) renderIngredients(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<section class="ingredients">`)
		_, _ = w.WriteString("\n<ul>\n")
	} else {
		_, _ = w.WriteString("</ul>\n</section>\n")
	}
	return gast.WalkContinue, nil
}

func (r *HTMLRenderer) renderIngredient(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		ing := node.(*ast.Ingredient)
		if r.config.SchemaOrg {
			_, _ = w.WriteString(`<li itemprop="recipeIngredient">`)
		} else {
			_, _ = w.WriteString("<li>")
		}
		if ing.Amount != "" {
			_, _ = fmt.Fprintf(w, "<em>%s</em> ", html.EscapeString(ing.Amount))
		}
		_, _ = w.WriteString(html.EscapeString(ing.Name))
		_, _ = w.WriteString("</li>\n")
	}
	return gast.WalkSkipChildren, nil
}

func (r *HTMLRenderer) renderIngredientGroup(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	group := node.(*ast.IngredientGroup)
	if entering {
		_, _ = w.WriteString("</ul>\n")
		_, _ = fmt.Fprintf(w, "<h3>%s</h3>\n<ul>\n", html.EscapeString(group.Name))
	}
	return gast.WalkContinue, nil
}

func (r *HTMLRenderer) renderInstructions(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		if r.config.SchemaOrg {
			_, _ = w.WriteString(`<section class="instructions" itemprop="recipeInstructions">`)
		} else {
			_, _ = w.WriteString(`<section class="instructions">`)
		}
		_, _ = w.WriteString("\n")
	} else {
		_, _ = w.WriteString("</section>\n")
	}
	return gast.WalkContinue, nil
}

func extractText(node gast.Node, source []byte) string {
	var sb strings.Builder
	extractTextInto(&sb, node, source)
	return sb.String()
}

func extractTextInto(sb *strings.Builder, node gast.Node, source []byte) {
	switch n := node.(type) {
	case *gast.Text:
		sb.Write(n.Segment.Value(source))
	case *gast.String:
		sb.Write(n.Value)
	default:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			extractTextInto(sb, child, source)
		}
	}
}

var _ renderer.NodeRenderer = (*HTMLRenderer)(nil)
