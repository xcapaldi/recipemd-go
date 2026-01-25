package ast

import (
	gast "github.com/yuin/goldmark/ast"
)

var KindInstructions = gast.NewNodeKind("Instructions")

// Instructions is the container for everything after the second thematic break
type Instructions struct {
	gast.BaseBlock
}

func NewInstructions() *Instructions {
	return &Instructions{}
}

func (n *Instructions) Kind() gast.NodeKind {
	return KindInstructions
}

func (n *Instructions) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
