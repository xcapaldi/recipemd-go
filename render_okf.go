package recipemd

import (
	"strings"
)

// RenderOKF renders r as an Open Knowledge Format (OKF) concept document.
//
// OKF (https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf)
// documents are Markdown files with YAML frontmatter carrying a required
// `type` field plus optional standardized metadata (title, description,
// tags, ...). The returned document declares `type: Recipe` and copies the
// recipe's title, description, and tags into the frontmatter; the body is
// the recipe rendered as standard RecipeMD via [Parser.RenderMarkdown], so
// the file remains a complete, self-contained recipe rather than just a
// metadata stub.
//
// Numeric amounts in the body are rounded to rounding decimal places, as in
// [Parser.RenderMarkdown].
//
// Because the body is plain RecipeMD, a [Parser] constructed with
// [WithFrontmatter] can parse the returned document back into an equivalent
// [Recipe].
func (p *Parser) RenderOKF(r *Recipe, rounding int) string {
	var fm strings.Builder
	fm.WriteString("---\n")
	fm.WriteString("type: Recipe\n")
	fm.WriteString("title: ")
	fm.WriteString(yamlQuote(r.Title))
	fm.WriteByte('\n')
	if r.Description != nil {
		fm.WriteString("description: ")
		fm.WriteString(yamlQuote(*r.Description))
		fm.WriteByte('\n')
	}
	if len(r.Tags) > 0 {
		fm.WriteString("tags: [")
		for i, t := range r.Tags {
			if i > 0 {
				fm.WriteString(", ")
			}
			fm.WriteString(yamlQuote(t))
		}
		fm.WriteString("]\n")
	}
	fm.WriteString("---\n\n")
	fm.WriteString(p.RenderMarkdown(r, rounding))
	return fm.String()
}

// yamlQuote renders s as a double-quoted YAML scalar, escaping backslashes,
// double quotes, and newlines so the value stays on a single logical line.
func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return `"` + s + `"`
}
