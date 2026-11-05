package recipemd

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// RenderOption is a functional option for configuring HTML rendering.
// Options are passed to [Recipe.RenderHTML].
type RenderOption func(*renderConfig)

// WithGFMRendering returns a [RenderOption] that enables GitHub Flavored
// Markdown (GFM) extensions when converting the raw markdown held in the
// Description and Instructions fields to HTML.
//
// Pass it when the recipe was parsed by a [Parser] created with
// [WithGithubFormattedMarkdown], so that GFM constructs such as tables and
// strikethrough survive the round trip to HTML.
func WithGFMRendering() RenderOption {
	return func(c *renderConfig) {
		c.goldmarkExtensions = append(c.goldmarkExtensions, extension.GFM)
	}
}

type renderConfig struct {
	goldmarkExtensions []goldmark.Extender
}

// defaultHTMLProcessor is the plain CommonMark processor used when
// [Recipe.RenderHTML] is called without options. A single instance is shared
// across calls: goldmark builds a fresh parse context per Convert and guards
// its lazy renderer initialisation internally.
var defaultHTMLProcessor = sync.OnceValue(func() goldmark.Markdown { return goldmark.New() })

// processor returns the markdown processor described by the config.
func (c *renderConfig) processor() goldmark.Markdown {
	if len(c.goldmarkExtensions) == 0 {
		return defaultHTMLProcessor()
	}
	return goldmark.New(goldmark.WithExtensions(c.goldmarkExtensions...))
}

type htmlGroupCtx struct {
	IngredientGroup
	Level int
}

func (g htmlGroupCtx) Subgroups() []htmlGroupCtx {
	out := make([]htmlGroupCtx, len(g.IngredientGroups))
	for i, sg := range g.IngredientGroups {
		out[i] = htmlGroupCtx{sg, g.Level + 1}
	}
	return out
}

const htmlMainTmpl = `<article class="recipemd-recipe">
<h1 class="recipemd-title">{{ .Title }}</h1>
{{- if or (deref .Description) .Tags .Yields }}
<div class="recipemd-preamble">
  {{- with deref .Description }}
  <div class="recipemd-description">{{ renderMD . }}</div>
  {{- end }}
  {{- if .Tags }}
  <ul class="recipemd-tags">
    {{- range .Tags }}
    <li class="recipemd-tag">{{ . }}</li>
    {{- end }}
  </ul>
  {{- end }}
  {{- if .Yields }}
  <p class="recipemd-yields"><strong>
    {{- range $i, $y := .Yields }}{{ if $i }}, {{ end -}}
    <span class="recipemd-yield" {{ template "amountAttrs" $y }}>{{ template "amountParts" $y }}</span>
    {{- end -}}
  </strong></p>
  {{- end }}
</div>
{{- end }}
<hr class="recipemd-separator">
<section class="recipemd-ingredients">
{{ template "ingredients" .Ingredients -}}
{{ template "groups" (topGroups .IngredientGroups) -}}
</section>
{{- with deref .Instructions }}
<hr class="recipemd-separator">
<section class="recipemd-instructions">{{ renderMD . }}</section>
{{- end }}
</article>`

const htmlIngredientsTmpl = `{{ if . -}}
<ul class="recipemd-ingredient-list">
{{ range . -}}
  <li class="recipemd-ingredient">
    {{- with .Amount }}<em class="recipemd-amount" {{ template "amountAttrs" . }}>{{ template "amountParts" . }}</em> {{ end -}}
    {{- if .Link }}<a class="recipemd-ingredient-link" href="{{ deref .Link }}">{{ .Name }}</a>
    {{- else }}<span class="recipemd-ingredient-name">{{ .Name }}</span>{{ end }}
  </li>
{{ end -}}
</ul>
{{ end }}`

// htmlAmountAttrsTmpl emits the machine-readable data attributes for an
// amount. data-factor always carries full float64 precision regardless of the
// rounding used for display, so client-side code can rescale without
// compounding rounding errors.
const htmlAmountAttrsTmpl = `data-factor="{{ rawFactor . }}"{{ with .Unit }} data-unit="{{ . }}"{{ end }}`

// htmlAmountPartsTmpl emits the human-readable amount, with factor and unit
// wrapped separately so either can be restyled or rewritten independently.
const htmlAmountPartsTmpl = `<span class="recipemd-amount-factor">{{ formatFactor . }}</span>
{{- with .Unit }} <span class="recipemd-amount-unit">{{ . }}</span>{{ end }}`

const htmlGroupsTmpl = `{{ range . -}}
<div class="recipemd-ingredient-group" data-level="{{ .Level }}">
{{ heading .Level .Title }}
{{ template "ingredients" .Ingredients -}}
{{ template "groups" .Subgroups }}
</div>
{{ end }}`

// RenderHTML renders the recipe as an HTML <article> element.
//
// Numeric amounts are rounded to rounding decimal places (trailing zeros are
// removed); pass a negative value to use full float64 precision.
//
// The Description and Instructions fields, which are stored as raw markdown
// strings, are converted to HTML with a plain CommonMark processor unless
// [WithGFMRendering] is supplied. Ingredient amounts are wrapped in <em> and
// yields in <strong>, mirroring the emphasis encoding used in RecipeMD source.
// All elements carry CSS class attributes with the prefix "recipemd-" for
// styling.
//
// The markup is also structured for client-side scripting:
//   - Each tag is its own <li class="recipemd-tag"> inside
//     <ul class="recipemd-tags">, so tags can be styled individually (e.g. as
//     pills).
//   - Each yield is wrapped in <span class="recipemd-yield">, and each
//     ingredient amount is <em class="recipemd-amount">. Both carry a
//     data-factor attribute holding the unrounded numeric factor and, when the
//     amount has a unit, a data-unit attribute. Their visible text is split
//     into <span class="recipemd-amount-factor"> (rounded per rounding) and
//     <span class="recipemd-amount-unit">. A script can therefore rescale a
//     recipe by selecting [data-factor], multiplying dataset.factor, and
//     rewriting the .recipemd-amount-factor text, without parsing any text.
//
// Ingredient groups are rendered as nested <div> blocks; the heading level
// starts at h2 for top-level groups and increments for each sub-level. Each
// group <div> carries a data-level attribute with that heading level.
//
// WARNING: This method is a work-in-progress and not yet ready for production
// use. Known limitations:
//   - Ingredient links reference raw .md file paths without any resolution or
//     conversion to HTML-friendly URLs.
//   - CommonMark reference-style links (e.g. [text][ref] with a separate
//     [ref]: url definition) only resolve correctly when the definition appears
//     in the same section (Description or Instructions) as the usage. Definitions
//     in one section are not visible when rendering the other, so cross-section
//     reflinks are silently left unresolved.
func (r *Recipe) RenderHTML(rounding int, opts ...RenderOption) string {
	cfg := &renderConfig{}
	for _, o := range opts {
		o(cfg)
	}

	return r.renderHTML(cfg.processor(), rounding)
}

// renderHTML renders the recipe using an explicit markdown processor for the
// Description and Instructions fields. It backs both [Recipe.RenderHTML] and
// the deprecated [Parser.RenderHTML], which supplies the parser's own
// processor so that its configuration carries over.
func (r *Recipe) renderHTML(md goldmark.Markdown, rounding int) string {
	funcs := htmlFuncMap(md, rounding)
	funcs["topGroups"] = func(groups []IngredientGroup) []htmlGroupCtx {
		out := make([]htmlGroupCtx, len(groups))
		for i, g := range groups {
			out[i] = htmlGroupCtx{g, 2}
		}
		return out
	}

	tmpl := template.Must(template.New("recipemd").Funcs(funcs).Parse(htmlMainTmpl))
	template.Must(tmpl.New("ingredients").Parse(htmlIngredientsTmpl))
	template.Must(tmpl.New("groups").Parse(htmlGroupsTmpl))
	template.Must(tmpl.New("amountAttrs").Parse(htmlAmountAttrsTmpl))
	template.Must(tmpl.New("amountParts").Parse(htmlAmountPartsTmpl))

	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, r)
	return buf.String()
}

func htmlFuncMap(md goldmark.Markdown, rounding int) template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"formatFactor": func(a Amount) string { return a.FormatFactor(rounding) },
		"rawFactor":    func(a Amount) string { return a.FormatFactor(-1) },
		"renderMD": func(src string) template.HTML {
			var buf bytes.Buffer
			_ = md.Convert([]byte(src), &buf)
			return template.HTML(buf.String())
		},
		"heading": func(level int, title string) template.HTML {
			escaped := template.HTMLEscapeString(title)
			return template.HTML(fmt.Sprintf(`<h%d class="recipemd-group-title">%s</h%d>`, level, escaped, level))
		},
	}
}
