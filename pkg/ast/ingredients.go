package ast

import (
	gast "github.com/yuin/goldmark/ast"
)

var (
	KindIngredients     = gast.NewNodeKind("Ingredients")
	KindIngredient      = gast.NewNodeKind("Ingredient")
	KindIngredientGroup = gast.NewNodeKind("IngredientGroup")
)

// Ingredients is the container for the ingredients section
type Ingredients struct {
	gast.BaseBlock
}

func NewIngredients() *Ingredients {
	return &Ingredients{}
}

func (n *Ingredients) Kind() gast.NodeKind {
	return KindIngredients
}

func (n *Ingredients) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Ingredient represents a single ingredient with optional amount
type Ingredient struct {
	gast.BaseBlock
	Amount string // parsed from leading emphasis, e.g., "2 cups"
	Name   string // the ingredient name
}

func NewIngredient() *Ingredient {
	return &Ingredient{}
}

func (n *Ingredient) Kind() gast.NodeKind {
	return KindIngredient
}

func (n *Ingredient) Dump(source []byte, level int) {
	m := map[string]string{}
	if n.Amount != "" {
		m["Amount"] = n.Amount
	}
	if n.Name != "" {
		m["Name"] = n.Name
	}
	gast.DumpHelper(n, source, level, m, nil)
}

// IngredientGroup is a named group of ingredients (H2/H3 heading + nested list)
type IngredientGroup struct {
	gast.BaseBlock
	Name string
}

func NewIngredientGroup() *IngredientGroup {
	return &IngredientGroup{}
}

func (n *IngredientGroup) Kind() gast.NodeKind {
	return KindIngredientGroup
}

func (n *IngredientGroup) Dump(source []byte, level int) {
	m := map[string]string{}
	if n.Name != "" {
		m["Name"] = n.Name
	}
	gast.DumpHelper(n, source, level, m, nil)
}
