package ast

import (
	gast "github.com/yuin/goldmark/ast"
)

var (
	KindTags   = gast.NewNodeKind("Tags")
	KindYields = gast.NewNodeKind("Yields")
)

// Tags wraps a paragraph that is entirely italicized (comma-separated tags)
type Tags struct {
	gast.BaseBlock
}

func NewTags() *Tags {
	return &Tags{}
}

func (n *Tags) Kind() gast.NodeKind {
	return KindTags
}

func (n *Tags) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Yields wraps a paragraph that is entirely bold (comma-separated yields)
type Yields struct {
	gast.BaseBlock
}

func NewYields() *Yields {
	return &Yields{}
}

func (n *Yields) Kind() gast.NodeKind {
	return KindYields
}

func (n *Yields) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
