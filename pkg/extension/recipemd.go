package extension

import (
	"github.com/yuin/goldmark"
)

type recipemd struct {
}

// recipemd is an extension that provides RecipeMD markdown functionalities.
var RecipeMD = &recipemd{}

func (e *recipemd) Extend(m goldmark.Markdown) {
	// noop for now

}
