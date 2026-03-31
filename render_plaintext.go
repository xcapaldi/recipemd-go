package recipemd

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// RenderPlainText converts a raw markdown string to plain text by parsing it
// and walking the AST, stripping all formatting. This is useful when markdown
// content needs to be embedded in formats that do not support markup, such as
// JSON-LD structured data.
func (p *Parser) RenderPlainText(markdown string) string {
	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := p.goldmarkProcessor.Parser().Parse(reader)
	var buf bytes.Buffer
	renderPlainTextNode(&buf, doc, source, true)
	return strings.TrimSpace(buf.String())
}

// renderPlainTextNode recursively walks the AST and writes plain text to buf.
// topLevel indicates whether we're iterating direct children of a block-level
// container (document, blockquote, list item) and should add paragraph spacing.
func renderPlainTextNode(buf *bytes.Buffer, node ast.Node, source []byte, topLevel bool) {
	switch n := node.(type) {
	case *ast.Document:
		renderPlainTextChildren(buf, n, source, true)

	case *ast.Heading:
		renderPlainTextChildren(buf, n, source, false)
		buf.WriteString("\n\n")

	case *ast.Paragraph:
		renderPlainTextChildren(buf, n, source, false)
		buf.WriteString("\n\n")

	case *ast.TextBlock:
		renderPlainTextChildren(buf, n, source, false)
		buf.WriteString("\n")

	case *ast.Text:
		buf.Write(n.Value(source))
		if n.SoftLineBreak() {
			buf.WriteByte(' ')
		}
		if n.HardLineBreak() {
			buf.WriteByte('\n')
		}

	case *ast.String:
		buf.Write(n.Value)

	case *ast.CodeSpan:
		renderPlainTextChildren(buf, n, source, false)

	case *ast.CodeBlock:
		renderCodeBlockLines(buf, n, source)
		buf.WriteString("\n")

	case *ast.FencedCodeBlock:
		renderCodeBlockLines(buf, n, source)
		buf.WriteString("\n")

	case *ast.Emphasis:
		renderPlainTextChildren(buf, n, source, false)

	case *ast.Link:
		renderPlainTextChildren(buf, n, source, false)

	case *ast.AutoLink:
		buf.Write(n.Label(source))

	case *ast.Image:
		// Emit alt text if available.
		renderPlainTextChildren(buf, n, source, false)

	case *ast.List:
		renderPlainTextChildren(buf, n, source, true)

	case *ast.ListItem:
		buf.WriteString("- ")
		renderPlainTextChildren(buf, n, source, false)
		if !strings.HasSuffix(buf.String(), "\n") {
			buf.WriteString("\n")
		}

	case *ast.Blockquote:
		renderPlainTextChildren(buf, n, source, true)

	case *ast.ThematicBreak:
		// Skip thematic breaks in plain text output.

	case *ast.HTMLBlock:
		// Skip HTML blocks.

	case *ast.RawHTML:
		// Skip inline HTML.

	default:
		// Handle GFM table types and any other unknown nodes by
		// attempting to render their children.
		switch {
		case n.Kind() == east.KindTable:
			renderTablePlainText(buf, n, source)
		case n.Kind() == east.KindTableHeader:
			renderTableRowPlainText(buf, n, source)
		case n.Kind() == east.KindTableRow:
			renderTableRowPlainText(buf, n, source)
		case n.Kind() == east.KindTableCell:
			renderPlainTextChildren(buf, n, source, false)
		default:
			renderPlainTextChildren(buf, n, source, topLevel)
		}
	}
}

// renderPlainTextChildren iterates over all children of node and renders each.
func renderPlainTextChildren(buf *bytes.Buffer, node ast.Node, source []byte, topLevel bool) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		renderPlainTextNode(buf, child, source, topLevel)
	}
}

// renderCodeBlockLines writes the raw lines of a code block.
func renderCodeBlockLines(buf *bytes.Buffer, node ast.Node, source []byte) {
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		buf.Write(seg.Value(source))
	}
}

// renderTablePlainText renders a GFM table as plain text with cells separated
// by tab characters and rows separated by newlines.
func renderTablePlainText(buf *bytes.Buffer, node ast.Node, source []byte) {
	// Table children are TableHeader (the header row) and TableRow nodes.
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		renderTableRowPlainText(buf, child, source)
	}
	buf.WriteString("\n")
}

// renderTableRowPlainText renders a single table row.
func renderTableRowPlainText(buf *bytes.Buffer, row ast.Node, source []byte) {
	first := true
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		if !first {
			buf.WriteByte('\t')
		}
		first = false
		renderPlainTextChildren(buf, cell, source, false)
	}
	buf.WriteByte('\n')
}
