package renderer

import (
	"fmt"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/xcapaldi/recipemd-go/pkg/goldmark-recipemd/ast"
)

// MarkdownRenderer renders RecipeMD AST nodes back to markdown
type MarkdownRenderer struct{}

// NewMarkdownRenderer creates a new markdown renderer
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// RegisterFuncs registers node rendering functions
func (r *MarkdownRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
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

func (r *MarkdownRenderer) renderRecipe(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

func (r *MarkdownRenderer) renderTitle(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = fmt.Fprintf(w, "# %s\n\n", extractText(node, source))
	}
	return gast.WalkSkipChildren, nil
}

func (r *MarkdownRenderer) renderDescription(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			text := strings.TrimSpace(extractText(child, source))
			if text != "" {
				_, _ = fmt.Fprintf(w, "%s\n\n", text)
			}
		}
	}
	return gast.WalkSkipChildren, nil
}

func (r *MarkdownRenderer) renderTags(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		text := extractText(node, source)
		_, _ = fmt.Fprintf(w, "*%s*\n\n", text)
	}
	return gast.WalkSkipChildren, nil
}

func (r *MarkdownRenderer) renderYields(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		text := extractText(node, source)
		_, _ = fmt.Fprintf(w, "**%s**\n\n", text)
	}
	return gast.WalkSkipChildren, nil
}

func (r *MarkdownRenderer) renderIngredients(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("---\n\n")
	} else {
		_, _ = w.WriteString("\n")
	}
	return gast.WalkContinue, nil
}

func (r *MarkdownRenderer) renderIngredient(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		ing := node.(*ast.Ingredient)
		_, _ = w.WriteString("- ")
		if ing.Amount != "" {
			_, _ = fmt.Fprintf(w, "*%s* ", ing.Amount)
		}
		_, _ = fmt.Fprintf(w, "%s\n", ing.Name)
	}
	return gast.WalkSkipChildren, nil
}

func (r *MarkdownRenderer) renderIngredientGroup(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		group := node.(*ast.IngredientGroup)
		_, _ = fmt.Fprintf(w, "\n## %s\n\n", group.Name)
	}
	return gast.WalkContinue, nil
}

func (r *MarkdownRenderer) renderInstructions(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("---\n\n")
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			text := strings.TrimSpace(extractText(child, source))
			if text != "" {
				_, _ = fmt.Fprintf(w, "%s\n\n", text)
			}
		}
	}
	return gast.WalkSkipChildren, nil
}

var _ renderer.NodeRenderer = (*MarkdownRenderer)(nil)
