package recipemd

import (
    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/text"
)

func parseToAST(source []byte) ast.Node {
	reader := text.NewReader(source)
	parser := goldmark.DefaultParser()
	return parser.Parse(reader)	
}
