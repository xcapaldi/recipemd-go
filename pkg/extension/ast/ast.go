package ast

import (
	gast "github.com/yuin/goldmark/ast"
)

// NodeKind values for RecipeMD AST nodes
var (
	KindRecipe           = gast.NewNodeKind("Recipe")
	KindRecipeMetadata   = gast.NewNodeKind("RecipeMetadata")
	KindRecipeTags       = gast.NewNodeKind("RecipeTags")
	KindRecipeYields     = gast.NewNodeKind("RecipeYields")
	KindYield            = gast.NewNodeKind("Yield")
	KindIngredientList   = gast.NewNodeKind("IngredientList")
	KindIngredientGroup  = gast.NewNodeKind("IngredientGroup")
	KindIngredient       = gast.NewNodeKind("Ingredient")
	KindAmount           = gast.NewNodeKind("Amount")
	KindInstructions     = gast.NewNodeKind("Instructions")
)

// Recipe is the root node for a RecipeMD document
type Recipe struct {
	gast.BaseBlock
}

// Dump implements Node.Dump
func (n *Recipe) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Kind implements Node.Kind
func (n *Recipe) Kind() gast.NodeKind {
	return KindRecipe
}

// NewRecipe returns a new Recipe node
func NewRecipe() *Recipe {
	return &Recipe{}
}

// RecipeMetadata contains title, description, tags, and yields
type RecipeMetadata struct {
	gast.BaseBlock
	Title       string
	Description string
}

// Dump implements Node.Dump
func (n *RecipeMetadata) Dump(source []byte, level int) {
	m := map[string]string{
		"Title":       n.Title,
		"Description": n.Description,
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// Kind implements Node.Kind
func (n *RecipeMetadata) Kind() gast.NodeKind {
	return KindRecipeMetadata
}

// NewRecipeMetadata returns a new RecipeMetadata node
func NewRecipeMetadata(title, description string) *RecipeMetadata {
	return &RecipeMetadata{
		Title:       title,
		Description: description,
	}
}

// RecipeTags contains the list of tags
type RecipeTags struct {
	gast.BaseBlock
	Tags []string
}

// Dump implements Node.Dump
func (n *RecipeTags) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Kind implements Node.Kind
func (n *RecipeTags) Kind() gast.NodeKind {
	return KindRecipeTags
}

// NewRecipeTags returns a new RecipeTags node
func NewRecipeTags(tags []string) *RecipeTags {
	return &RecipeTags{Tags: tags}
}

// RecipeYields contains the list of yields
type RecipeYields struct {
	gast.BaseBlock
}

// Dump implements Node.Dump
func (n *RecipeYields) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Kind implements Node.Kind
func (n *RecipeYields) Kind() gast.NodeKind {
	return KindRecipeYields
}

// NewRecipeYields returns a new RecipeYields node
func NewRecipeYields() *RecipeYields {
	return &RecipeYields{}
}

// Yield represents a single yield value
type Yield struct {
	gast.BaseBlock
	Factor string
	Unit   *string // pointer because it can be null
}

// Dump implements Node.Dump
func (n *Yield) Dump(source []byte, level int) {
	m := map[string]string{
		"Factor": n.Factor,
	}
	if n.Unit != nil {
		m["Unit"] = *n.Unit
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// Kind implements Node.Kind
func (n *Yield) Kind() gast.NodeKind {
	return KindYield
}

// NewYield returns a new Yield node
func NewYield(factor string, unit *string) *Yield {
	return &Yield{
		Factor: factor,
		Unit:   unit,
	}
}

// IngredientList contains ungrouped ingredients
type IngredientList struct {
	gast.BaseBlock
}

// Dump implements Node.Dump
func (n *IngredientList) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Kind implements Node.Kind
func (n *IngredientList) Kind() gast.NodeKind {
	return KindIngredientList
}

// NewIngredientList returns a new IngredientList node
func NewIngredientList() *IngredientList {
	return &IngredientList{}
}

// IngredientGroup represents a group of ingredients with a title
type IngredientGroup struct {
	gast.BaseBlock
	Title string
	Level int // heading level
}

// Dump implements Node.Dump
func (n *IngredientGroup) Dump(source []byte, level int) {
	m := map[string]string{
		"Title": n.Title,
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// Kind implements Node.Kind
func (n *IngredientGroup) Kind() gast.NodeKind {
	return KindIngredientGroup
}

// NewIngredientGroup returns a new IngredientGroup node
func NewIngredientGroup(title string, headingLevel int) *IngredientGroup {
	return &IngredientGroup{
		Title: title,
		Level: headingLevel,
	}
}

// Ingredient represents a single ingredient
type Ingredient struct {
	gast.BaseBlock
	Name string
	Link *string // pointer because it can be null
}

// Dump implements Node.Dump
func (n *Ingredient) Dump(source []byte, level int) {
	m := map[string]string{
		"Name": n.Name,
	}
	if n.Link != nil {
		m["Link"] = *n.Link
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// Kind implements Node.Kind
func (n *Ingredient) Kind() gast.NodeKind {
	return KindIngredient
}

// NewIngredient returns a new Ingredient node
func NewIngredient(name string, link *string) *Ingredient {
	return &Ingredient{
		Name: name,
		Link: link,
	}
}

// Amount represents an ingredient amount
type Amount struct {
	gast.BaseBlock
	Factor string
	Unit   *string // pointer because it can be null
}

// Dump implements Node.Dump
func (n *Amount) Dump(source []byte, level int) {
	m := map[string]string{
		"Factor": n.Factor,
	}
	if n.Unit != nil {
		m["Unit"] = *n.Unit
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// Kind implements Node.Kind
func (n *Amount) Kind() gast.NodeKind {
	return KindAmount
}

// NewAmount returns a new Amount node
func NewAmount(factor string, unit *string) *Amount {
	return &Amount{
		Factor: factor,
		Unit:   unit,
	}
}

// Instructions represents the recipe instructions
type Instructions struct {
	gast.BaseBlock
	Content string
}

// Dump implements Node.Dump
func (n *Instructions) Dump(source []byte, level int) {
	m := map[string]string{
		"Content": n.Content,
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// Kind implements Node.Kind
func (n *Instructions) Kind() gast.NodeKind {
	return KindInstructions
}

// NewInstructions returns a new Instructions node
func NewInstructions(text string) *Instructions {
	return &Instructions{Content: text}
}
