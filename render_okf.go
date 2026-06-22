package recipemd

import (
	"strings"
)

// RenderOKF renders r as an Open Knowledge Format (OKF) concept document.
//
// OKF (https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
// documents are Markdown files with a YAML frontmatter block followed by a
// free-form body. The only required frontmatter field is `type`; this
// renders `type: Recipe` along with the spec's recommended `title`,
// `description`, and `tags` fields, copied from the recipe. The body is the
// recipe rendered as standard RecipeMD via [Parser.RenderMarkdown], so the
// file remains a complete, self-contained recipe rather than just a metadata
// stub.
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
		fm.WriteString(yamlBlockScalar("description", *r.Description))
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

// yamlBlockScalar renders key's value as a YAML literal block scalar
// ("key: |"), indenting each line by two spaces. Unlike a quoted scalar,
// this preserves embedded newlines and special characters (quotes,
// backslashes, markdown/HTML) verbatim and human-readably, which matters
// for RecipeMD descriptions that may span multiple paragraphs.
func yamlBlockScalar(key, value string) string {
	var b strings.Builder
	b.WriteString(key)
	b.WriteString(": |\n")
	for _, line := range strings.Split(value, "\n") {
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
