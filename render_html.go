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

// WARNING: WIP — not ready for production use.
// Known issues:
//   - Ingredient links point to raw .md file paths with no resolution or conversion.
//
// RenderHTML formats a Recipe as an HTML <article> element.
// Fields containing raw markdown (Description, Instructions) are parsed and
// rendered to HTML. Ingredient amounts and yields are wrapped in <em>/<strong>
// to restore the emphasis that markdown formatting conveys. All elements carry
// class attributes matching the RecipeMD types so they can be styled with CSS.
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
