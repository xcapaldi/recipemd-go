package recipemd

// This file holds the pre-v1.1.0 rendering API. Rendering moved from [Parser]
// to [Recipe] because the Parser receiver carried no rendering state: only
// RenderHTML ever read it, and only to reach the parser's markdown processor.
// These wrappers keep the v1 module line compatible; they are scheduled for
// removal in v2.

// RenderJSON serialises r as compact JSON.
//
// Deprecated: use [Recipe.RenderJSON] instead. This wrapper ignores its
// receiver and will be removed in v2.
func (p *Parser) RenderJSON(r *Recipe) ([]byte, error) {
	return r.RenderJSON()
}

// RenderMarkdown renders r as a RecipeMD-formatted markdown string.
//
// Deprecated: use [Recipe.RenderMarkdown] instead. This wrapper ignores its
// receiver and will be removed in v2.
func (p *Parser) RenderMarkdown(r *Recipe, rounding int) string {
	return r.RenderMarkdown(rounding)
}

// RenderHTML renders r as an HTML <article> element, converting the markdown
// in the Description and Instructions fields with the parser's own processor.
//
// Deprecated: use [Recipe.RenderHTML] instead, passing [WithGFMRendering] when
// the recipe was parsed with [WithGithubFormattedMarkdown]. This wrapper will
// be removed in v2.
func (p *Parser) RenderHTML(r *Recipe, rounding int) string {
	return r.renderHTML(p.goldmarkProcessor, rounding)
}
