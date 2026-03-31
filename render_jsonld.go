package recipemd

import (
	"encoding/json"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// jsonLDRecipe is the Schema.org Recipe type serialised as JSON-LD.
// Fields use omitempty so that absent optional values are not emitted.
type jsonLDRecipe struct {
	Context            string `json:"@context"`
	Type               string `json:"@type"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	RecipeYield        string `json:"recipeYield,omitempty"`
	Keywords           string `json:"keywords,omitempty"`
	RecipeIngredient   []string `json:"recipeIngredient,omitempty"`
	RecipeInstructions []any    `json:"recipeInstructions,omitempty"`
}

// howToStep represents a Schema.org HowToStep in JSON-LD.
type howToStep struct {
	Type  string `json:"@type"`
	Text  string `json:"text"`
	Image string `json:"image,omitempty"`
}

// howToSection represents a Schema.org HowToSection in JSON-LD.
type howToSection struct {
	Type            string      `json:"@type"`
	Name            string      `json:"name"`
	ItemListElement []howToStep `json:"itemListElement"`
}

// RenderJSONLD renders r as a Schema.org Recipe in JSON-LD format.
//
// The output is a JSON object with "@context": "https://schema.org/" and
// "@type": "Recipe". Field mapping:
//
//   - [Recipe.Title] → "name"
//   - [Recipe.Description] → "description" (converted to plain text)
//   - [Recipe.Yields] → "recipeYield" (first yield, serialised)
//   - [Recipe.Tags] → "keywords" (comma-separated string, NOT an array)
//   - All ingredients (including grouped) → "recipeIngredient" (array of strings)
//   - [Recipe.Instructions] → "recipeInstructions" (HowToStep / HowToSection)
//
// Numeric amounts are rounded to rounding decimal places. Pass a negative
// value for full float64 precision.
//
// Optional Schema.org fields that cannot be derived from the RecipeMD format
// (prepTime, cookTime, nutrition, etc.) are omitted.
func (p *Parser) RenderJSONLD(r *Recipe, rounding int) ([]byte, error) {
	ld := jsonLDRecipe{
		Context: "https://schema.org/",
		Type:    "Recipe",
		Name:    r.Title,
	}

	if r.Description != nil && *r.Description != "" {
		ld.Description = p.RenderPlainText(*r.Description)
	}

	if len(r.Yields) > 0 {
		ld.RecipeYield = r.Yields[0].Serialize(rounding)
	}

	if len(r.Tags) > 0 {
		ld.Keywords = strings.Join(r.Tags, ", ")
	}

	ingredients := r.LeafIngredients()
	if len(ingredients) > 0 {
		ld.RecipeIngredient = make([]string, len(ingredients))
		for i, ing := range ingredients {
			ld.RecipeIngredient[i] = ing.Serialize(rounding)
		}
	}

	if r.Instructions != nil && *r.Instructions != "" {
		ld.RecipeInstructions = p.parseInstructions(*r.Instructions)
	}

	return json.MarshalIndent(ld, "", "  ")
}

// parseInstructions converts a markdown instructions string into a slice of
// howToStep and/or howToSection values suitable for JSON-LD recipeInstructions.
//
// Headings start new HowToSections. Paragraphs become HowToSteps. Images
// found inside paragraphs are attached to the step's image field.
func (p *Parser) parseInstructions(markdown string) []any {
	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := p.goldmarkProcessor.Parser().Parse(reader)

	var result []any
	var currentSection *howToSection

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Heading:
			// Flush any open section.
			if currentSection != nil {
				result = append(result, *currentSection)
			}
			headingText := plainTextInline(n, source)
			currentSection = &howToSection{
				Type:            "HowToSection",
				Name:            headingText,
				ItemListElement: []howToStep{},
			}

		case *ast.Paragraph:
			step := paragraphToStep(n, source)
			if step.Text == "" {
				continue
			}
			if currentSection != nil {
				currentSection.ItemListElement = append(currentSection.ItemListElement, step)
			} else {
				result = append(result, step)
			}

		case *ast.List:
			steps := listToSteps(n, source)
			if currentSection != nil {
				currentSection.ItemListElement = append(currentSection.ItemListElement, steps...)
			} else {
				for _, s := range steps {
					result = append(result, s)
				}
			}

		case *ast.ThematicBreak:
			// Skip thematic breaks.

		default:
			// For code blocks, blockquotes, tables, etc., render as plain text step.
			txt := blockToPlainText(child, source)
			if txt == "" {
				continue
			}
			step := howToStep{Type: "HowToStep", Text: txt}
			if currentSection != nil {
				currentSection.ItemListElement = append(currentSection.ItemListElement, step)
			} else {
				result = append(result, step)
			}
		}
	}

	// Flush the last open section.
	if currentSection != nil {
		result = append(result, *currentSection)
	}

	return result
}

// paragraphToStep converts a paragraph AST node into a howToStep,
// extracting any image destination for the step's image field.
// When a paragraph contains only an image, the alt text is used as the step
// text so that the step is not empty.
func paragraphToStep(para *ast.Paragraph, source []byte) howToStep {
	var textBuf strings.Builder
	var imageURL string

	for child := para.FirstChild(); child != nil; child = child.NextSibling() {
		if img, ok := child.(*ast.Image); ok {
			if imageURL == "" {
				imageURL = string(img.Destination)
			}
			// Don't include image in text output.
			continue
		}
		writeInlinePlainText(&textBuf, child, source)
	}

	txt := strings.TrimSpace(textBuf.String())

	// If the paragraph was image-only, use the alt text so the step is not
	// silently dropped.
	if txt == "" && imageURL != "" {
		txt = plainTextInline(para, source)
	}

	return howToStep{
		Type:  "HowToStep",
		Text:  txt,
		Image: imageURL,
	}
}

// listToSteps converts a list node into a slice of howToSteps, one per item.
func listToSteps(list *ast.List, source []byte) []howToStep {
	var steps []howToStep
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		txt := blockToPlainText(item, source)
		if txt == "" {
			continue
		}
		steps = append(steps, howToStep{Type: "HowToStep", Text: txt})
	}
	return steps
}

// plainTextInline extracts plain text from a block node's inline children,
// stripping all markdown formatting.
func plainTextInline(node ast.Node, source []byte) string {
	var buf strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		writeInlinePlainText(&buf, child, source)
	}
	return strings.TrimSpace(buf.String())
}

// writeInlinePlainText writes the plain text of an inline node to buf.
func writeInlinePlainText(buf *strings.Builder, node ast.Node, source []byte) {
	switch n := node.(type) {
	case *ast.Text:
		buf.Write(n.Value(source))
		if n.SoftLineBreak() {
			buf.WriteByte(' ')
		}
	case *ast.String:
		buf.Write(n.Value)
	case *ast.CodeSpan, *ast.Emphasis, *ast.Link:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			writeInlinePlainText(buf, child, source)
		}
	case *ast.AutoLink:
		buf.Write(n.Label(source))
	case *ast.Image:
		// Emit alt text.
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			writeInlinePlainText(buf, child, source)
		}
	case *ast.RawHTML:
		// Skip inline HTML.
	default:
		// Unknown inline — try children.
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			writeInlinePlainText(buf, child, source)
		}
	}
}

// blockToPlainText renders a block-level node as trimmed plain text.
func blockToPlainText(node ast.Node, source []byte) string {
	var buf strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == ast.TypeBlock {
			// Recurse into nested blocks (e.g. list items contain paragraphs).
			txt := blockToPlainText(child, source)
			if buf.Len() > 0 && txt != "" {
				buf.WriteByte(' ')
			}
			buf.WriteString(txt)
		} else {
			writeInlinePlainText(&buf, child, source)
		}
	}
	return strings.TrimSpace(buf.String())
}
