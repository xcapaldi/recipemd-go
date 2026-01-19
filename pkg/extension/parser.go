package extension

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/xcapaldi/recipemd-go/pkg/extension/ast"
)

// recipeParser is an AST transformer that converts a standard markdown document
// into a RecipeMD structured document
type recipeParser struct {
}

// NewRecipeParser creates a new recipeParser
func NewRecipeParser() parser.ASTTransformer {
	return &recipeParser{}
}

// Transform transforms the AST to RecipeMD structure
func (p *recipeParser) Transform(node *gast.Document, reader text.Reader, pc parser.Context) {
	recipe := ast.NewRecipe()
	source := reader.Source()

	// Track current position in the document structure
	var title string
	var descriptionParts []string
	var tagsNode *ast.RecipeTags
	var yieldsNode *ast.RecipeYields
	var firstDividerFound bool
	var secondDividerFound bool
	var currentIngredientList *ast.IngredientList
	var ingredientGroupStack []*ast.IngredientGroup
	var topLevelGroups []*ast.IngredientGroup
	var instructionParts []string

	// Walk through all children of the document
	child := node.FirstChild()
	for child != nil {
		next := child.NextSibling()

		switch n := child.(type) {
		case *gast.Heading:
			if n.Level == 1 && title == "" {
				// First H1 is the title
				title = extractText(n, source)
			} else if firstDividerFound && !secondDividerFound {
				// This is an ingredient group heading
				groupTitle := extractText(n, source)
				group := ast.NewIngredientGroup(groupTitle, n.Level)

				// Handle nested groups based on heading level
				p.addIngredientGroup(recipe, group, &ingredientGroupStack, &topLevelGroups, &currentIngredientList)
			} else if secondDividerFound {
				// After second divider, everything is instructions
				instructionParts = append(instructionParts, renderNode(n, source))
			}

		case *gast.Paragraph:
			if title == "" {
				// Before title, skip
			} else if !firstDividerFound {
				// Check if this is tags (fully italic) or yields (fully bold)
				if isFullyItalic(n, source) {
					// This is tags
					text := extractText(n, source)
					tags := splitTags(text)
					tagsNode = ast.NewRecipeTags(tags)
				} else if isFullyBold(n, source) {
					// This is yields
					text := extractText(n, source)
					yields := parseYields(text)
					yieldsNode = ast.NewRecipeYields()
					for _, y := range yields {
						yieldsNode.AppendChild(yieldsNode, y)
					}
				} else {
					// This is description
					descriptionParts = append(descriptionParts, renderNode(n, source))
				}
			} else if secondDividerFound {
				// Instructions
				instructionParts = append(instructionParts, renderNode(n, source))
			}

		case *gast.List:
			if firstDividerFound && !secondDividerFound {
				// This is an ingredient list
				ingredients := parseIngredients(n, source)

				// Add to current group or ungrouped list
				if len(ingredientGroupStack) > 0 {
					currentGroup := ingredientGroupStack[len(ingredientGroupStack)-1]
					for _, ing := range ingredients {
						currentGroup.AppendChild(currentGroup, ing)
					}
				} else {
					if currentIngredientList == nil {
						currentIngredientList = ast.NewIngredientList()
					}
					for _, ing := range ingredients {
						currentIngredientList.AppendChild(currentIngredientList, ing)
					}
				}
			} else if secondDividerFound {
				// Instructions
				instructionParts = append(instructionParts, renderNode(n, source))
			}

		case *gast.ThematicBreak:
			if !firstDividerFound {
				firstDividerFound = true
			} else if !secondDividerFound {
				secondDividerFound = true
			}

		default:
			// Other nodes
			if secondDividerFound {
				instructionParts = append(instructionParts, renderNode(n, source))
			} else if !firstDividerFound && title != "" {
				// Part of description
				descriptionParts = append(descriptionParts, renderNode(n, source))
			}
		}

		child = next
	}

	// Build the recipe structure
	description := strings.TrimSpace(strings.Join(descriptionParts, "\n\n"))
	metadata := ast.NewRecipeMetadata(title, description)
	recipe.AppendChild(recipe, metadata)

	if tagsNode != nil {
		metadata.AppendChild(metadata, tagsNode)
	}
	if yieldsNode != nil {
		metadata.AppendChild(metadata, yieldsNode)
	}

	// Add ungrouped ingredients
	if currentIngredientList != nil && currentIngredientList.HasChildren() {
		recipe.AppendChild(recipe, currentIngredientList)
	}

	// Add ingredient groups (only top-level groups)
	for _, group := range topLevelGroups {
		recipe.AppendChild(recipe, group)
	}

	// Add instructions
	if len(instructionParts) > 0 {
		instructionsText := strings.TrimSpace(strings.Join(instructionParts, "\n\n"))
		instructions := ast.NewInstructions(instructionsText)
		recipe.AppendChild(recipe, instructions)
	}

	// Replace the document's children with our recipe node
	node.RemoveChildren(node)
	node.AppendChild(node, recipe)
}

// addIngredientGroup adds an ingredient group to the appropriate parent based on heading level
func (p *recipeParser) addIngredientGroup(recipe *ast.Recipe, group *ast.IngredientGroup, stack *[]*ast.IngredientGroup, topLevel *[]*ast.IngredientGroup, currentList **ast.IngredientList) {
	// Reset current ingredient list when we hit a new group
	*currentList = nil

	// Find the appropriate parent based on heading level
	for len(*stack) > 0 {
		parent := (*stack)[len(*stack)-1]
		if group.Level > parent.Level {
			// This group is a child of the parent
			parent.AppendChild(parent, group)
			*stack = append(*stack, group)
			return
		}
		// Pop the stack
		*stack = (*stack)[:len(*stack)-1]
	}

	// If we get here, this is a top-level group
	*topLevel = append(*topLevel, group)
	*stack = append(*stack, group)
}

// isFullyItalic checks if a paragraph is fully italic (for tags)
func isFullyItalic(para *gast.Paragraph, source []byte) bool {
	if !para.HasChildren() {
		return false
	}

	// Check if the first child is an emphasis and it's the only child
	child := para.FirstChild()
	if emph, ok := child.(*gast.Emphasis); ok && emph.Level == 1 {
		// Check if this emphasis contains all the text
		if child.NextSibling() == nil {
			return true
		}
	}

	return false
}

// isFullyBold checks if a paragraph is fully bold (for yields)
func isFullyBold(para *gast.Paragraph, source []byte) bool {
	if !para.HasChildren() {
		return false
	}

	// Check if the first child is an emphasis with level 2 (bold) and it's the only child
	child := para.FirstChild()
	if emph, ok := child.(*gast.Emphasis); ok && emph.Level == 2 {
		// Check if this emphasis contains all the text
		if child.NextSibling() == nil {
			return true
		}
	}

	return false
}

// extractText extracts plain text from a node
func extractText(node gast.Node, source []byte) string {
	var buf bytes.Buffer
	child := node.FirstChild()
	for child != nil {
		if text, ok := child.(*gast.Text); ok {
			buf.Write(text.Segment.Value(source))
		} else if child.HasChildren() {
			buf.WriteString(extractText(child, source))
		}
		child = child.NextSibling()
	}
	return buf.String()
}

// renderNode renders a node back to markdown-like text
func renderNode(node gast.Node, source []byte) string {
	var buf bytes.Buffer

	switch n := node.(type) {
	case *gast.Heading:
		buf.WriteString(extractText(n, source))
	case *gast.Paragraph:
		buf.WriteString(extractText(n, source))
	case *gast.Text:
		buf.Write(n.Segment.Value(source))
	case *gast.CodeBlock, *gast.FencedCodeBlock:
		// For code blocks, get the raw content
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}
	case *gast.Image:
		// Render image as markdown
		buf.WriteString("![")
		child := n.FirstChild()
		for child != nil {
			buf.WriteString(renderNode(child, source))
			child = child.NextSibling()
		}
		buf.WriteString("](")
		buf.Write(n.Destination)
		buf.WriteString(")")
	case *gast.RawHTML:
		// For raw HTML, preserve it
		lines := n.Segments
		for i := 0; i < lines.Len(); i++ {
			segment := lines.At(i)
			buf.Write(segment.Value(source))
		}
	default:
		// For other nodes, try to extract text
		if n.HasChildren() {
			child := n.FirstChild()
			for child != nil {
				buf.WriteString(renderNode(child, source))
				child = child.NextSibling()
			}
		}
	}

	return buf.String()
}

// splitTags splits a comma-separated tag string
func splitTags(text string) []string {
	// Per spec: comma not treated as divider if chars before and after are numerical
	var tags []string
	var current strings.Builder
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		if runes[i] == ',' {
			// Check if this comma should be treated as a separator
			before := i > 0 && unicode.IsDigit(runes[i-1])
			after := i < len(runes)-1 && unicode.IsDigit(runes[i+1])

			if before && after {
				// Not a separator, part of number
				current.WriteRune(runes[i])
			} else {
				// This is a separator
				tag := strings.TrimSpace(current.String())
				if tag != "" {
					tags = append(tags, tag)
				}
				current.Reset()
			}
		} else {
			current.WriteRune(runes[i])
		}
	}

	// Add the last tag
	tag := strings.TrimSpace(current.String())
	if tag != "" {
		tags = append(tags, tag)
	}

	return tags
}

// parseYields parses yield amounts from text
func parseYields(text string) []*ast.Yield {
	parts := splitTags(text) // Same splitting logic as tags
	var yields []*ast.Yield

	for _, part := range parts {
		part = strings.TrimSpace(part)
		factor, unit := parseAmount(part)
		yields = append(yields, ast.NewYield(factor, unit))
	}

	return yields
}

// parseAmount parses an amount string into factor and unit
// Handles formats like "5", "5 cups", "1.2 ml", "1 1/4 servings", "1,5 Tassen"
func parseAmount(text string) (factor string, unit *string) {
	text = strings.TrimSpace(text)

	// Pattern to match number at the start
	// Supports: decimals (1.2, 1,5), fractions (1/2, 1 1/2), whole numbers
	numberPattern := regexp.MustCompile(`^([\d.,/\s]+)`)
	matches := numberPattern.FindStringSubmatch(text)

	if len(matches) > 0 {
		factor = strings.TrimSpace(matches[1])
		remainder := strings.TrimSpace(text[len(matches[1]):])

		if remainder != "" {
			unit = &remainder
		}
	} else {
		// No number found, the whole thing is the factor
		factor = text
	}

	return factor, unit
}

// parseIngredients parses ingredients from a list
func parseIngredients(list *gast.List, source []byte) []*ast.Ingredient {
	var ingredients []*ast.Ingredient

	child := list.FirstChild()
	for child != nil {
		if item, ok := child.(*gast.ListItem); ok {
			ingredient := parseIngredient(item, source)
			if ingredient != nil {
				ingredients = append(ingredients, ingredient)
			}
		}
		child = child.NextSibling()
	}

	return ingredients
}

// parseIngredient parses a single ingredient from a list item
func parseIngredient(item *gast.ListItem, source []byte) *ast.Ingredient {
	// An ingredient can have:
	// - Optional amount (italic text at the start)
	// - Name (required, plain text or link)
	// - Optional link to another recipe

	var amount *ast.Amount
	var name string
	var link *string

	// Look at the children
	child := item.FirstChild()

	// List items can have either Paragraph or TextBlock children
	if para, ok := child.(*gast.Paragraph); ok {
		child = para.FirstChild()
	} else if textBlock, ok := child.(*gast.TextBlock); ok {
		child = textBlock.FirstChild()
	}

	// Check if first child is italic (amount)
	if emph, ok := child.(*gast.Emphasis); ok && emph.Level == 1 {
		amountText := extractText(emph, source)
		factor, unit := parseAmount(amountText)
		amount = ast.NewAmount(factor, unit)
		child = child.NextSibling()
	}

	// Collect the rest of the text/nodes for the name
	var nameParts []string
	for child != nil {
		if linkNode, ok := child.(*gast.Link); ok {
			// This is a link - use it as both name and link
			linkText := extractText(linkNode, source)
			if name == "" {
				name = linkText
			} else {
				nameParts = append(nameParts, linkText)
			}
			linkDest := string(linkNode.Destination)
			link = &linkDest
		} else if textNode, ok := child.(*gast.Text); ok {
			text := string(textNode.Segment.Value(source))
			nameParts = append(nameParts, text)
		} else {
			// Other node types - try to extract text
			text := extractText(child, source)
			if text != "" {
				nameParts = append(nameParts, text)
			}
		}
		child = child.NextSibling()
	}

	if name == "" && len(nameParts) > 0 {
		name = strings.Join(nameParts, "")
	}

	if name == "" {
		return nil
	}

	name = strings.TrimSpace(name)
	ingredient := ast.NewIngredient(name, link)
	if amount != nil {
		ingredient.AppendChild(ingredient, amount)
	}

	return ingredient
}
