package recipemd

import (
	"fmt"
	"testing"

	"github.com/yuin/goldmark/ast"
)

var sampleRecipe = []byte(`# Guacamole

Some people call it guac.

*sauce, vegan*

**4 Servings, 200g**

---

- *1* avocado
- *.5 teaspoon* salt
- *1 1/2 pinches* red pepper flakes
- lemon juice

---

Remove flesh from avocado and roughly mash with fork. Season to taste
with salt, pepper and lemon juice.
`)

func TestParseAST(t *testing.T) {
	node := parseToAST(sampleRecipe)
	dumpAST(node, sampleRecipe, 0)
}

func dumpAST(n ast.Node, source []byte, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	fmt.Printf("%s%s", indent, n.Kind())
	if n.Type() == ast.TypeInline || n.Type() == ast.TypeBlock {
		if text := n.Text(source); len(text) > 0 {
			fmt.Printf(" %q", text)
		}
	}
	fmt.Println()
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		dumpAST(c, source, depth+1)
	}
}
