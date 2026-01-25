package renderer

import (
	"encoding/json"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/xcapaldi/recipemd-go/pkg/goldmark-recipemd/ast"
)

// JSONRenderer renders RecipeMD AST nodes to JSON
type JSONRenderer struct {
	recipe jsonRecipe
	source []byte
}

type jsonRecipe struct {
	Title        string              `json:"title"`
	Description  string              `json:"description,omitempty"`
	Tags         []string            `json:"tags,omitempty"`
	Yields       []string            `json:"yields,omitempty"`
	Ingredients  []jsonIngredient    `json:"ingredients,omitempty"`
	Groups       []jsonGroup         `json:"ingredientGroups,omitempty"`
	Instructions string              `json:"instructions,omitempty"`
}

type jsonIngredient struct {
	Amount string `json:"amount,omitempty"`
	Name   string `json:"name"`
}

type jsonGroup struct {
	Name        string           `json:"name"`
	Ingredients []jsonIngredient `json:"ingredients"`
}

// NewJSONRenderer creates a new JSON renderer
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// RegisterFuncs registers node rendering functions
func (r *JSONRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
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

func (r *JSONRenderer) renderRecipe(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		r.source = source
		r.recipe = jsonRecipe{}
	} else {
		data, _ := json.MarshalIndent(r.recipe, "", "  ")
		_, _ = w.Write(data)
		_, _ = w.WriteString("\n")
	}
	return gast.WalkContinue, nil
}

func (r *JSONRenderer) renderTitle(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		r.recipe.Title = extractText(node, source)
	}
	return gast.WalkSkipChildren, nil
}

func (r *JSONRenderer) renderDescription(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		var parts []string
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			parts = append(parts, strings.TrimSpace(extractText(child, source)))
		}
		r.recipe.Description = strings.Join(parts, "\n\n")
	}
	return gast.WalkSkipChildren, nil
}

func (r *JSONRenderer) renderTags(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		text := extractText(node, source)
		for _, tag := range strings.Split(text, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				r.recipe.Tags = append(r.recipe.Tags, tag)
			}
		}
	}
	return gast.WalkSkipChildren, nil
}

func (r *JSONRenderer) renderYields(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		text := extractText(node, source)
		for _, y := range strings.Split(text, ",") {
			y = strings.TrimSpace(y)
			if y != "" {
				r.recipe.Yields = append(r.recipe.Yields, y)
			}
		}
	}
	return gast.WalkSkipChildren, nil
}

func (r *JSONRenderer) renderIngredients(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

func (r *JSONRenderer) renderIngredient(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		ing := node.(*ast.Ingredient)
		ji := jsonIngredient{Amount: ing.Amount, Name: ing.Name}

		// Check if we're inside a group
		if _, ok := node.Parent().(*ast.IngredientGroup); ok {
			// Will be handled by group
		} else {
			r.recipe.Ingredients = append(r.recipe.Ingredients, ji)
		}
	}
	return gast.WalkSkipChildren, nil
}

func (r *JSONRenderer) renderIngredientGroup(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		group := node.(*ast.IngredientGroup)
		jg := jsonGroup{Name: group.Name}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if ing, ok := child.(*ast.Ingredient); ok {
				jg.Ingredients = append(jg.Ingredients, jsonIngredient{
					Amount: ing.Amount,
					Name:   ing.Name,
				})
			}
		}
		r.recipe.Groups = append(r.recipe.Groups, jg)
	}
	return gast.WalkSkipChildren, nil
}

func (r *JSONRenderer) renderInstructions(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		var parts []string
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			parts = append(parts, strings.TrimSpace(extractText(child, source)))
		}
		r.recipe.Instructions = strings.Join(parts, "\n\n")
	}
	return gast.WalkSkipChildren, nil
}

var _ renderer.NodeRenderer = (*JSONRenderer)(nil)
