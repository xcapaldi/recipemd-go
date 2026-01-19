package extension

import (
	"encoding/json"
	"io"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/xcapaldi/recipemd-go/pkg/extension/ast"
)

// RecipeData represents the JSON structure for a recipe
type RecipeData struct {
	Title            string              `json:"title"`
	Description      *string             `json:"description"`
	Tags             []string            `json:"tags"`
	Yields           []YieldData         `json:"yields"`
	Ingredients      []IngredientData    `json:"ingredients"`
	IngredientGroups []IngredientGroup   `json:"ingredient_groups"`
	Instructions     *string             `json:"instructions"`
}

// YieldData represents a yield amount
type YieldData struct {
	Factor string  `json:"factor"`
	Unit   *string `json:"unit"`
}

// IngredientData represents an ingredient
type IngredientData struct {
	Name   string       `json:"name"`
	Amount *AmountData  `json:"amount"`
	Link   *string      `json:"link"`
}

// AmountData represents an ingredient amount
type AmountData struct {
	Factor string  `json:"factor"`
	Unit   *string `json:"unit"`
}

// IngredientGroup represents a group of ingredients
type IngredientGroup struct {
	Title            string            `json:"title"`
	Ingredients      []IngredientData  `json:"ingredients"`
	IngredientGroups []IngredientGroup `json:"ingredient_groups"`
}

// JSONRenderer renders RecipeMD to JSON
type JSONRenderer struct {
}

// NewJSONRenderer creates a new JSONRenderer
func NewJSONRenderer() renderer.Renderer {
	return &JSONRenderer{}
}

// Render renders the recipe to JSON
func (r *JSONRenderer) Render(w io.Writer, source []byte, node gast.Node) error {
	// Find the Recipe node
	var recipe *ast.Recipe
	if r, ok := node.(*gast.Document); ok {
		child := r.FirstChild()
		if rec, ok := child.(*ast.Recipe); ok {
			recipe = rec
		}
	} else if rec, ok := node.(*ast.Recipe); ok {
		recipe = rec
	}

	if recipe == nil {
		return nil
	}

	data := r.buildRecipeData(recipe)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// buildRecipeData builds the RecipeData structure from the AST
func (r *JSONRenderer) buildRecipeData(recipe *ast.Recipe) *RecipeData {
	data := &RecipeData{
		Tags:             []string{},
		Yields:           []YieldData{},
		Ingredients:      []IngredientData{},
		IngredientGroups: []IngredientGroup{},
	}

	child := recipe.FirstChild()
	for child != nil {
		switch n := child.(type) {
		case *ast.RecipeMetadata:
			data.Title = n.Title
			if n.Description != "" {
				data.Description = &n.Description
			}
			r.processMetadata(n, data)

		case *ast.IngredientList:
			r.processIngredientList(n, &data.Ingredients)

		case *ast.IngredientGroup:
			group := r.buildIngredientGroup(n)
			data.IngredientGroups = append(data.IngredientGroups, group)

		case *ast.Instructions:
			if n.Content != "" {
				data.Instructions = &n.Content
			}
		}

		child = child.NextSibling()
	}

	return data
}

// processMetadata processes the metadata node
func (r *JSONRenderer) processMetadata(metadata *ast.RecipeMetadata, data *RecipeData) {
	child := metadata.FirstChild()
	for child != nil {
		switch n := child.(type) {
		case *ast.RecipeTags:
			data.Tags = n.Tags

		case *ast.RecipeYields:
			yieldChild := n.FirstChild()
			for yieldChild != nil {
				if y, ok := yieldChild.(*ast.Yield); ok {
					data.Yields = append(data.Yields, YieldData{
						Factor: y.Factor,
						Unit:   y.Unit,
					})
				}
				yieldChild = yieldChild.NextSibling()
			}
		}

		child = child.NextSibling()
	}
}

// processIngredientList processes an ingredient list
func (r *JSONRenderer) processIngredientList(list *ast.IngredientList, ingredients *[]IngredientData) {
	child := list.FirstChild()
	for child != nil {
		if ing, ok := child.(*ast.Ingredient); ok {
			ingData := r.buildIngredientData(ing)
			*ingredients = append(*ingredients, ingData)
		}
		child = child.NextSibling()
	}
}

// buildIngredientData builds an IngredientData from an Ingredient node
func (r *JSONRenderer) buildIngredientData(ing *ast.Ingredient) IngredientData {
	data := IngredientData{
		Name: ing.Name,
		Link: ing.Link,
	}

	// Check for amount child
	child := ing.FirstChild()
	if amount, ok := child.(*ast.Amount); ok {
		data.Amount = &AmountData{
			Factor: amount.Factor,
			Unit:   amount.Unit,
		}
	}

	return data
}

// buildIngredientGroup builds an IngredientGroup from an IngredientGroup node
func (r *JSONRenderer) buildIngredientGroup(group *ast.IngredientGroup) IngredientGroup {
	data := IngredientGroup{
		Title:            group.Title,
		Ingredients:      []IngredientData{},
		IngredientGroups: []IngredientGroup{},
	}

	child := group.FirstChild()
	for child != nil {
		switch n := child.(type) {
		case *ast.Ingredient:
			ingData := r.buildIngredientData(n)
			data.Ingredients = append(data.Ingredients, ingData)

		case *ast.IngredientGroup:
			subGroup := r.buildIngredientGroup(n)
			data.IngredientGroups = append(data.IngredientGroups, subGroup)
		}

		child = child.NextSibling()
	}

	return data
}

// AddOptions adds options to the renderer
func (r *JSONRenderer) AddOptions(...renderer.Option) {
	// No options needed for now
}

// RegisterFuncs registers rendering functions
func (r *JSONRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Register rendering functions for each node type
	reg.Register(ast.KindRecipe, r.renderRecipe)
	reg.Register(ast.KindRecipeMetadata, r.renderDummy)
	reg.Register(ast.KindRecipeTags, r.renderDummy)
	reg.Register(ast.KindRecipeYields, r.renderDummy)
	reg.Register(ast.KindYield, r.renderDummy)
	reg.Register(ast.KindIngredientList, r.renderDummy)
	reg.Register(ast.KindIngredientGroup, r.renderDummy)
	reg.Register(ast.KindIngredient, r.renderDummy)
	reg.Register(ast.KindAmount, r.renderDummy)
	reg.Register(ast.KindInstructions, r.renderDummy)
}

// renderRecipe renders the recipe node
func (r *JSONRenderer) renderRecipe(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}

	recipe := node.(*ast.Recipe)
	data := r.buildRecipeData(recipe)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return gast.WalkStop, err
	}

	return gast.WalkSkipChildren, nil
}

// renderDummy is a dummy renderer for nodes that don't need individual rendering
func (r *JSONRenderer) renderDummy(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}
