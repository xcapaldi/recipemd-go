package recipemd

import (
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// ParseMarkdownStructure parses a RecipeMD markdown document and extracts
// the structural components into a MarkdownRecipe.
func ParseMarkdownStructure(source []byte) (*MarkdownRecipe, error) {
	// Create parser with HTML blocks enabled
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	mr := &MarkdownRecipe{
		RawSource: source,
	}

	// Collect all top-level blocks
	var blocks []ast.Node
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		blocks = append(blocks, child)
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("empty document")
	}

	// State machine for parsing
	idx := 0

	// 1. First block must be H1
	if idx >= len(blocks) {
		return nil, fmt.Errorf("recipe must have a title")
	}

	heading, ok := blocks[idx].(*ast.Heading)
	if !ok || heading.Level != 1 {
		return nil, fmt.Errorf("recipe must start with a level-1 heading (title)")
	}
	mr.TitleNode = heading
	idx++

	// 2. Collect description paragraphs until we hit tags/yield/thematic break
	for idx < len(blocks) {
		block := blocks[idx]

		// Stop at thematic break
		if _, ok := block.(*ast.ThematicBreak); ok {
			break
		}

		// Check if this is a tags paragraph (single emphasis)
		if isTagsParagraph(block, source) {
			mr.TagsNode = block
			idx++
			continue
		}

		// Check if this is a yield paragraph (single strong emphasis)
		if isYieldParagraph(block, source) {
			mr.YieldNode = block
			idx++
			continue
		}

		// Otherwise, it's a description paragraph
		if _, ok := block.(*ast.Paragraph); ok {
			mr.DescriptionNodes = append(mr.DescriptionNodes, block)
		} else {
			// Non-paragraph blocks in description area are also allowed
			mr.DescriptionNodes = append(mr.DescriptionNodes, block)
		}
		idx++
	}

	// 3. Find first thematic break (start of ingredients)
	if idx >= len(blocks) {
		// No thematic break means no ingredients section
		return mr, nil
	}

	if _, ok := blocks[idx].(*ast.ThematicBreak); !ok {
		return nil, fmt.Errorf("expected thematic break before ingredients")
	}
	idx++ // Skip the first thematic break

	// 4. Collect ingredient nodes until second thematic break
	for idx < len(blocks) {
		block := blocks[idx]

		// Stop at second thematic break
		if _, ok := block.(*ast.ThematicBreak); ok {
			idx++ // Skip the second thematic break
			break
		}

		// Accept lists and headings in ingredient section
		switch block.(type) {
		case *ast.List:
			mr.IngredientNodes = append(mr.IngredientNodes, block)
		case *ast.Heading:
			mr.IngredientNodes = append(mr.IngredientNodes, block)
		default:
			// Other block types in ingredients section are invalid
			return nil, fmt.Errorf("unexpected block type in ingredients section: %T", block)
		}
		idx++
	}

	// 5. Everything after second thematic break is instructions
	for idx < len(blocks) {
		mr.InstructionNodes = append(mr.InstructionNodes, blocks[idx])
		idx++
	}

	return mr, nil
}

// isTagsParagraph checks if a node is a paragraph containing only emphasized text (tags).
func isTagsParagraph(node ast.Node, source []byte) bool {
	p, ok := node.(*ast.Paragraph)
	if !ok {
		return false
	}

	// Check if paragraph has exactly one child that is an Emphasis node
	child := p.FirstChild()
	if child == nil {
		return false
	}

	// Should be only one child (the emphasis)
	if child.NextSibling() != nil {
		return false
	}

	// Must be emphasis (not strong)
	emph, ok := child.(*ast.Emphasis)
	if !ok {
		return false
	}

	// Emphasis level 1 = italics, level 2 = bold
	return emph.Level == 1
}

// isYieldParagraph checks if a node is a paragraph containing only strong emphasized text (yield).
func isYieldParagraph(node ast.Node, source []byte) bool {
	p, ok := node.(*ast.Paragraph)
	if !ok {
		return false
	}

	// Check if paragraph has exactly one child that is a Strong/Emphasis node
	child := p.FirstChild()
	if child == nil {
		return false
	}

	// Should be only one child
	if child.NextSibling() != nil {
		return false
	}

	// Check for emphasis with level 2 (strong/bold)
	emph, ok := child.(*ast.Emphasis)
	if !ok {
		return false
	}

	return emph.Level == 2
}

// getNodeText extracts the text content from an AST node.
func getNodeText(node ast.Node, source []byte) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *ast.Text:
		return string(n.Segment.Value(source))
	case *ast.String:
		return string(n.Value)
	default:
		// Recursively get text from children
		var result string
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			result += getNodeText(child, source)
		}
		return result
	}
}

// getInnerText extracts inner text from an emphasis/strong node.
func getInnerText(node ast.Node, source []byte) string {
	p, ok := node.(*ast.Paragraph)
	if !ok {
		return ""
	}

	child := p.FirstChild()
	if child == nil {
		return ""
	}

	emph, ok := child.(*ast.Emphasis)
	if !ok {
		return ""
	}

	return getNodeText(emph, source)
}
