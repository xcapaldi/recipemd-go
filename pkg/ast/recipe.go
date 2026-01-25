package ast

import (
	gast "github.com/yuin/goldmark/ast"
)

// Recipe node kinds
var (
	KindRecipe      = gast.NewNodeKind("Recipe")
	KindTitle       = gast.NewNodeKind("Title")
	KindDescription = gast.NewNodeKind("Description")
)

// Recipe is the root container for a RecipeMD document
type Recipe struct {
	gast.BaseBlock
}

func NewRecipe() *Recipe {
	return &Recipe{}
}

func (n *Recipe) Kind() gast.NodeKind {
	return KindRecipe
}

func (n *Recipe) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Title wraps the first H1 heading
type Title struct {
	gast.BaseBlock
}

func NewTitle() *Title {
	return &Title{}
}

func (n *Title) Kind() gast.NodeKind {
	return KindTitle
}

func (n *Title) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Description contains the description paragraphs
type Description struct {
	gast.BaseBlock
}

func NewDescription() *Description {
	return &Description{}
}

func (n *Description) Kind() gast.NodeKind {
	return KindDescription
}

func (n *Description) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
