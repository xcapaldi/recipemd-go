package parser

import (
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/xcapaldi/recipemd-go/pkg/ast"
)

type recipeTransformer struct{}

// NewRecipeTransformer returns an ASTTransformer that adds RecipeMD semantic structure
func NewRecipeTransformer() parser.ASTTransformer {
	return &recipeTransformer{}
}

func (t *recipeTransformer) Transform(doc *gast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()

	// Find thematic breaks to split sections
	var breaks []*gast.ThematicBreak
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if tb, ok := child.(*gast.ThematicBreak); ok {
			breaks = append(breaks, tb)
		}
	}

	// Create root Recipe node
	recipe := ast.NewRecipe()

	// Collect all children first
	var children []gast.Node
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		children = append(children, child)
	}

	// Determine section boundaries
	firstBreakIdx := -1
	secondBreakIdx := -1
	for i, child := range children {
		if _, ok := child.(*gast.ThematicBreak); ok {
			if firstBreakIdx == -1 {
				firstBreakIdx = i
			} else if secondBreakIdx == -1 {
				secondBreakIdx = i
				break
			}
		}
	}

	// Process metadata section (before first break)
	metaEnd := len(children)
	if firstBreakIdx != -1 {
		metaEnd = firstBreakIdx
	}
	t.processMetadata(recipe, children[:metaEnd], source)

	// Process ingredients section (between first and second break)
	if firstBreakIdx != -1 {
		ingredientsStart := firstBreakIdx + 1
		ingredientsEnd := len(children)
		if secondBreakIdx != -1 {
			ingredientsEnd = secondBreakIdx
		}
		if ingredientsStart < ingredientsEnd {
			t.processIngredients(recipe, children[ingredientsStart:ingredientsEnd], source)
		}
	}

	// Process instructions section (after second break)
	if secondBreakIdx != -1 && secondBreakIdx+1 < len(children) {
		t.processInstructions(recipe, children[secondBreakIdx+1:], source)
	}

	// Replace document children with recipe node
	for doc.FirstChild() != nil {
		doc.RemoveChild(doc, doc.FirstChild())
	}
	doc.AppendChild(doc, recipe)
}

func (t *recipeTransformer) processMetadata(recipe *ast.Recipe, nodes []gast.Node, source []byte) {
	var description *ast.Description

	for _, node := range nodes {
		switch n := node.(type) {
		case *gast.Heading:
			if n.Level == 1 {
				title := ast.NewTitle()
				n.Parent().RemoveChild(n.Parent(), n)
				title.AppendChild(title, n)
				recipe.AppendChild(recipe, title)
			}
		case *gast.Paragraph:
			if isAllEmphasis(n) {
				tags := ast.NewTags()
				n.Parent().RemoveChild(n.Parent(), n)
				tags.AppendChild(tags, n)
				recipe.AppendChild(recipe, tags)
			} else if isAllStrong(n) {
				yields := ast.NewYields()
				n.Parent().RemoveChild(n.Parent(), n)
				yields.AppendChild(yields, n)
				recipe.AppendChild(recipe, yields)
			} else {
				if description == nil {
					description = ast.NewDescription()
					recipe.AppendChild(recipe, description)
				}
				n.Parent().RemoveChild(n.Parent(), n)
				description.AppendChild(description, n)
			}
		}
	}
}

func (t *recipeTransformer) processIngredients(recipe *ast.Recipe, nodes []gast.Node, source []byte) {
	ingredients := ast.NewIngredients()
	recipe.AppendChild(recipe, ingredients)

	var currentGroup *ast.IngredientGroup

	for _, node := range nodes {
		switch n := node.(type) {
		case *gast.Heading:
			// Start a new ingredient group
			currentGroup = ast.NewIngredientGroup()
			currentGroup.Name = extractText(n, source)
			n.Parent().RemoveChild(n.Parent(), n)
			currentGroup.AppendChild(currentGroup, n)
			ingredients.AppendChild(ingredients, currentGroup)

		case *gast.List:
			n.Parent().RemoveChild(n.Parent(), n)
			t.processIngredientList(n, currentGroup, ingredients, source)
		}
	}
}

func (t *recipeTransformer) processIngredientList(list *gast.List, group *ast.IngredientGroup, ingredients *ast.Ingredients, source []byte) {
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		listItem, ok := item.(*gast.ListItem)
		if !ok {
			continue
		}

		ingredient := ast.NewIngredient()
		t.parseIngredientContent(ingredient, listItem, source)

		if group != nil {
			group.AppendChild(group, ingredient)
		} else {
			ingredients.AppendChild(ingredients, ingredient)
		}
	}
}

func (t *recipeTransformer) parseIngredientContent(ingredient *ast.Ingredient, item *gast.ListItem, source []byte) {
	// Look for TextBlock or Paragraph as first child
	var textContainer gast.Node
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*gast.TextBlock); ok {
			textContainer = child
			break
		}
		if _, ok := child.(*gast.Paragraph); ok {
			textContainer = child
			break
		}
	}

	if textContainer == nil {
		return
	}

	var amountParts []string
	var nameParts []string
	inAmount := true

	for child := textContainer.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *gast.Emphasis:
			if inAmount && n.Level == 1 {
				amountParts = append(amountParts, extractText(n, source))
			} else {
				nameParts = append(nameParts, extractText(n, source))
			}
		case *gast.Text:
			txt := string(n.Segment.Value(source))
			if inAmount {
				inAmount = false
			}
			nameParts = append(nameParts, txt)
		default:
			nameParts = append(nameParts, extractText(child, source))
		}
	}

	ingredient.Amount = strings.TrimSpace(strings.Join(amountParts, " "))
	ingredient.Name = strings.TrimSpace(strings.Join(nameParts, ""))
}

func (t *recipeTransformer) processInstructions(recipe *ast.Recipe, nodes []gast.Node, source []byte) {
	instructions := ast.NewInstructions()
	recipe.AppendChild(recipe, instructions)

	for _, node := range nodes {
		node.Parent().RemoveChild(node.Parent(), node)
		instructions.AppendChild(instructions, node)
	}
}

// isAllEmphasis returns true if paragraph contains only a single Emphasis child
func isAllEmphasis(p *gast.Paragraph) bool {
	if p.ChildCount() != 1 {
		return false
	}
	em, ok := p.FirstChild().(*gast.Emphasis)
	return ok && em.Level == 1
}

// isAllStrong returns true if paragraph contains only a single Strong (Emphasis level 2) child
func isAllStrong(p *gast.Paragraph) bool {
	if p.ChildCount() != 1 {
		return false
	}
	em, ok := p.FirstChild().(*gast.Emphasis)
	return ok && em.Level == 2
}

// extractText recursively extracts text content from a node
func extractText(node gast.Node, source []byte) string {
	var sb strings.Builder
	extractTextInto(&sb, node, source)
	return sb.String()
}

func extractTextInto(sb *strings.Builder, node gast.Node, source []byte) {
	if t, ok := node.(*gast.Text); ok {
		sb.Write(t.Segment.Value(source))
		return
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		extractTextInto(sb, child, source)
	}
}
