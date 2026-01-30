package recipemd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// ToRecipe converts the MarkdownRecipe AST representation to the Recipe domain model.
func (mr *MarkdownRecipe) ToRecipe() (*Recipe, error) {
	if mr.TitleNode == nil {
		return nil, fmt.Errorf("recipe must have a title")
	}

	recipe := &Recipe{
		Tags:             []string{},
		Yields:           []Yield{},
		Ingredients:      []Ingredient{},
		IngredientGroups: []IngredientGroup{},
	}

	// Extract title
	recipe.Title = getNodeText(mr.TitleNode, mr.RawSource)

	// Extract description
	if len(mr.DescriptionNodes) > 0 {
		desc := mr.extractDescription()
		if desc != "" {
			recipe.Description = &desc
		}
	}

	// Extract tags
	if mr.TagsNode != nil {
		tagsText := getInnerText(mr.TagsNode, mr.RawSource)
		recipe.Tags = splitByComma(tagsText)
	}

	// Extract yields
	if mr.YieldNode != nil {
		yieldsText := getInnerText(mr.YieldNode, mr.RawSource)
		yields, err := parseYields(yieldsText)
		if err != nil {
			return nil, fmt.Errorf("error parsing yields: %w", err)
		}
		recipe.Yields = yields
	}

	// Extract ingredients (with groups)
	ingredients, groups, err := mr.extractIngredients()
	if err != nil {
		return nil, fmt.Errorf("error parsing ingredients: %w", err)
	}
	recipe.Ingredients = ingredients
	recipe.IngredientGroups = groups

	// Extract instructions
	if len(mr.InstructionNodes) > 0 {
		instr := mr.extractInstructions()
		if instr != "" {
			recipe.Instructions = &instr
		}
	}

	return recipe, nil
}

// extractDescription combines description nodes into a single string.
func (mr *MarkdownRecipe) extractDescription() string {
	var parts []string
	for _, node := range mr.DescriptionNodes {
		text := mr.nodeToMarkdown(node)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// extractInstructions combines instruction nodes into a single string.
func (mr *MarkdownRecipe) extractInstructions() string {
	var parts []string
	for _, node := range mr.InstructionNodes {
		text := mr.nodeToMarkdown(node)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// nodeToMarkdown reconstructs markdown from an AST node.
func (mr *MarkdownRecipe) nodeToMarkdown(node ast.Node) string {
	if node == nil {
		return ""
	}

	// Handle lists specially to preserve structure
	if list, ok := node.(*ast.List); ok {
		return mr.renderList(list)
	}

	// For paragraphs and other block nodes, extract the raw text
	if node.Type() == ast.TypeBlock {
		return mr.extractBlockText(node)
	}

	return getNodeText(node, mr.RawSource)
}

// renderList renders a list node back to markdown.
func (mr *MarkdownRecipe) renderList(list *ast.List) string {
	var buf bytes.Buffer
	itemNum := 1

	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}

		// Add list marker
		if list.IsOrdered() {
			buf.WriteString(fmt.Sprintf("%d. ", itemNum))
			itemNum++
		} else {
			buf.WriteString("- ")
		}

		// Render item content
		for itemChild := item.FirstChild(); itemChild != nil; itemChild = itemChild.NextSibling() {
			buf.WriteString(mr.extractBlockText(itemChild))
		}
		buf.WriteString("\n")
	}

	return strings.TrimSpace(buf.String())
}

// extractBlockText extracts text from a block node, preserving inline markdown.
func (mr *MarkdownRecipe) extractBlockText(node ast.Node) string {
	var buf bytes.Buffer
	mr.renderInlineMarkdown(node, &buf)
	return strings.TrimSpace(buf.String())
}

// renderInlineMarkdown renders inline nodes back to markdown.
func (mr *MarkdownRecipe) renderInlineMarkdown(node ast.Node, buf *bytes.Buffer) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			buf.Write(n.Segment.Value(mr.RawSource))
			if n.SoftLineBreak() {
				buf.WriteByte('\n')
			}
		case *ast.String:
			buf.Write(n.Value)
		case *ast.Emphasis:
			marker := "*"
			if n.Level == 2 {
				marker = "**"
			}
			buf.WriteString(marker)
			mr.renderInlineMarkdown(n, buf)
			buf.WriteString(marker)
		case *ast.Link:
			buf.WriteByte('[')
			mr.renderInlineMarkdown(n, buf)
			buf.WriteString("](")
			buf.Write(n.Destination)
			buf.WriteByte(')')
		case *ast.Image:
			buf.WriteString("![")
			mr.renderInlineMarkdown(n, buf)
			buf.WriteString("](")
			buf.Write(n.Destination)
			buf.WriteByte(')')
		case *ast.CodeSpan:
			buf.WriteByte('`')
			mr.renderInlineMarkdown(n, buf)
			buf.WriteByte('`')
		case *ast.RawHTML:
			for i := 0; i < n.Segments.Len(); i++ {
				seg := n.Segments.At(i)
				buf.Write(seg.Value(mr.RawSource))
			}
		case *ast.AutoLink:
			buf.WriteByte('<')
			buf.Write(n.URL(mr.RawSource))
			buf.WriteByte('>')
		default:
			// For other nodes, recurse into children
			mr.renderInlineMarkdown(child, buf)
		}
	}
}

// parseYields parses the yields string into a list of Yield structs.
func parseYields(text string) ([]Yield, error) {
	items := splitByComma(text)
	var yields []Yield

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		factor, unit, err := parseAmount(item)
		if err != nil {
			return nil, fmt.Errorf("invalid yield '%s': %w", item, err)
		}

		yields = append(yields, Yield{
			Factor: factor,
			Unit:   unit,
		})
	}

	return yields, nil
}

// extractIngredients extracts ingredients and ingredient groups from the AST.
func (mr *MarkdownRecipe) extractIngredients() ([]Ingredient, []IngredientGroup, error) {
	var ingredients []Ingredient
	var groups []IngredientGroup

	// Track heading levels for group nesting
	type groupContext struct {
		group       *IngredientGroup
		headingLevel int
	}
	var groupStack []groupContext

	for _, node := range mr.IngredientNodes {
		switch n := node.(type) {
		case *ast.Heading:
			// Create a new ingredient group
			title := getNodeText(n, mr.RawSource)
			newGroup := IngredientGroup{
				Title:            title,
				Ingredients:      []Ingredient{},
				IngredientGroups: []IngredientGroup{},
			}

			// Determine where this group belongs based on heading level
			level := n.Level

			// Pop groups that are at the same or lower level
			for len(groupStack) > 0 && groupStack[len(groupStack)-1].headingLevel >= level {
				// Pop the group and add it to its parent
				popped := groupStack[len(groupStack)-1]
				groupStack = groupStack[:len(groupStack)-1]

				if len(groupStack) > 0 {
					parent := groupStack[len(groupStack)-1].group
					parent.IngredientGroups = append(parent.IngredientGroups, *popped.group)
				} else {
					groups = append(groups, *popped.group)
				}
			}

			// Push the new group onto the stack
			groupStack = append(groupStack, groupContext{
				group:       &newGroup,
				headingLevel: level,
			})

		case *ast.List:
			// Parse ingredients from list items
			listIngredients, err := mr.parseListIngredients(n)
			if err != nil {
				return nil, nil, err
			}

			// Add ingredients to the current group or top level
			if len(groupStack) > 0 {
				currentGroup := groupStack[len(groupStack)-1].group
				currentGroup.Ingredients = append(currentGroup.Ingredients, listIngredients...)
			} else {
				ingredients = append(ingredients, listIngredients...)
			}
		}
	}

	// Pop remaining groups from the stack
	for len(groupStack) > 0 {
		popped := groupStack[len(groupStack)-1]
		groupStack = groupStack[:len(groupStack)-1]

		if len(groupStack) > 0 {
			parent := groupStack[len(groupStack)-1].group
			parent.IngredientGroups = append(parent.IngredientGroups, *popped.group)
		} else {
			groups = append(groups, *popped.group)
		}
	}

	return ingredients, groups, nil
}

// parseListIngredients parses a list node into Ingredient structs.
func (mr *MarkdownRecipe) parseListIngredients(list *ast.List) ([]Ingredient, error) {
	var ingredients []Ingredient

	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}

		ing, err := mr.parseListItem(item)
		if err != nil {
			return nil, err
		}
		ingredients = append(ingredients, ing)
	}

	return ingredients, nil
}

// parseListItem parses a single list item into an Ingredient.
func (mr *MarkdownRecipe) parseListItem(item *ast.ListItem) (Ingredient, error) {
	ing := Ingredient{}

	// Get the text block inside the list item
	textBlock := item.FirstChild()
	if textBlock == nil {
		return ing, fmt.Errorf("empty list item")
	}

	// Check for amount (emphasis at the start)
	firstInline := textBlock.FirstChild()
	var amountNode *ast.Emphasis
	var restStart ast.Node

	if emph, ok := firstInline.(*ast.Emphasis); ok && emph.Level == 1 {
		amountNode = emph
		restStart = emph.NextSibling()
	} else {
		restStart = firstInline
	}

	// Parse amount if present
	if amountNode != nil {
		amountText := getNodeText(amountNode, mr.RawSource)
		factor, unit, err := parseAmount(amountText)
		if err != nil {
			return ing, fmt.Errorf("invalid amount '%s': %w", amountText, err)
		}
		ing.Amount = &Amount{
			Factor: factor,
			Unit:   unit,
		}
	}

	// Parse the rest as ingredient name (and possibly link)
	var nameParts []string
	var linkDest *string

	for node := restStart; node != nil; node = node.NextSibling() {
		switch n := node.(type) {
		case *ast.Text:
			// Preserve the text as-is (with leading/trailing spaces)
			text := string(n.Segment.Value(mr.RawSource))
			nameParts = append(nameParts, text)
		case *ast.Link:
			// Extract link text as name and destination as link
			linkText := getNodeText(n, mr.RawSource)
			dest := string(n.Destination)
			linkDest = &dest
			nameParts = append(nameParts, linkText)
		case *ast.Emphasis:
			// Markdown in ingredient name
			marker := "*"
			if n.Level == 2 {
				marker = "**"
			}
			nameParts = append(nameParts, marker+getNodeText(n, mr.RawSource)+marker)
		case *ast.CodeSpan:
			nameParts = append(nameParts, "`"+getNodeText(n, mr.RawSource)+"`")
		}
	}

	ing.Name = strings.TrimSpace(strings.Join(nameParts, ""))
	ing.Link = linkDest

	// Validate: ingredient must have a name
	if ing.Name == "" {
		return ing, fmt.Errorf("ingredient has no name")
	}

	return ing, nil
}
