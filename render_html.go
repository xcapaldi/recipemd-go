package recipemd

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

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
  <p class="recipemd-tags"><em>{{ join .Tags ", " }}</em></p>
  {{- end }}
  {{- if .Yields }}
  <p class="recipemd-yields"><strong>{{ serializeYields .Yields }}</strong></p>
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
    {{- if .Amount }}<em class="recipemd-amount">{{ serializeAmount .Amount }}</em> {{ end -}}
    {{- if .Link }}<a class="recipemd-ingredient-link" href="{{ deref .Link }}">{{ .Name }}</a>
    {{- else }}<span class="recipemd-ingredient-name">{{ .Name }}</span>{{ end }}
  </li>
{{ end -}}
</ul>
{{ end }}`

const htmlGroupsTmpl = `{{ range . -}}
<div class="recipemd-ingredient-group">
{{ heading .Level .Title }}
{{ template "ingredients" .Ingredients -}}
{{ template "groups" .Subgroups }}
</div>
{{ end }}`

// RenderHTML renders r as an HTML <article> element.
//
// Numeric amounts are rounded to rounding decimal places (trailing zeros are
// removed); pass a negative value to use full float64 precision.
//
// The Description and Instructions fields, which are stored as raw markdown
// strings, are converted to HTML using the same markdown processor that was
// configured on the [Parser]. Ingredient amounts are wrapped in <em> and
// yields in <strong>, mirroring the emphasis encoding used in RecipeMD source.
// All elements carry CSS class attributes with the prefix "recipemd-" for
// styling.
//
// Ingredient groups are rendered as nested <div> blocks; the heading level
// starts at h2 for top-level groups and increments for each sub-level.
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
func (p *Parser) RenderHTML(r *Recipe, rounding int) string {
	funcs := htmlFuncMap(p, rounding)
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

	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, r)
	return buf.String()
}

func htmlFuncMap(p *Parser, rounding int) template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"serializeAmount": func(a *Amount) string {
			if a == nil {
				return ""
			}
			return a.Serialize(rounding)
		},
		"serializeYields": func(yields []Amount) string {
			s := make([]string, len(yields))
			for i, y := range yields {
				s[i] = y.Serialize(rounding)
			}
			return strings.Join(s, ", ")
		},
		"renderMD": func(md string) template.HTML {
			var buf bytes.Buffer
			_ = p.goldmarkProcessor.Convert([]byte(md), &buf)
			return template.HTML(buf.String())
		},
		"heading": func(level int, title string) template.HTML {
			escaped := template.HTMLEscapeString(title)
			return template.HTML(fmt.Sprintf(`<h%d class="recipemd-group-title">%s</h%d>`, level, escaped, level))
		},
	}
}
