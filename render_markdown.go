package recipemd

import (
	"bytes"
	"strings"
	"text/template"
)

type mdGroupCtx struct {
	IngredientGroup
	Level int
}

func (g mdGroupCtx) Heading() string {
	return strings.Repeat("#", g.Level)
}

func (g mdGroupCtx) Subgroups() []mdGroupCtx {
	out := make([]mdGroupCtx, len(g.IngredientGroups))
	for i, sg := range g.IngredientGroups {
		out[i] = mdGroupCtx{sg, g.Level + 1}
	}
	return out
}

const mdMainTmpl = `# {{ .Title }}
{{ with deref .Description }}
{{ . }}
{{ end }}{{ if .Tags }}
*{{ join .Tags ", " }}*
{{ end }}{{ if .Yields }}
**{{ serializeYields .Yields }}**
{{ end }}
---
{{ template "ingredients" .Ingredients }}{{ template "groups" (topGroups .IngredientGroups) }}{{ with deref .Instructions }}
---

{{ . }}
{{ end }}`

const mdIngredientsTmpl = `{{ if . }}
{{ range . }}- {{ if .Amount }}*{{ serializeAmount .Amount }}* {{ end }}{{ if .Link }}[{{ .Name }}]({{ deref .Link }}){{ else }}{{ .Name }}{{ end }}
{{ end }}{{ end }}`

const mdGroupsTmpl = `{{ range . }}
{{ .Heading }} {{ .Title }}
{{ template "ingredients" .Ingredients }}{{ template "groups" (.Subgroups) }}{{ end }}`

// RenderMarkdown renders the recipe as a RecipeMD-formatted markdown string.
//
// Numeric amounts are rounded to rounding decimal places (trailing zeros are
// removed). Pass a negative rounding value to use full float64 precision.
//
// The returned string contains a complete, parseable RecipeMD document that
// [Parser.Parse] can round-trip back to an equivalent [Recipe].
func (r *Recipe) RenderMarkdown(rounding int) string {
	funcs := renderFuncMap(rounding)
	funcs["topGroups"] = func(groups []IngredientGroup) []mdGroupCtx {
		out := make([]mdGroupCtx, len(groups))
		for i, g := range groups {
			out[i] = mdGroupCtx{g, 2}
		}
		return out
	}

	tmpl := template.Must(template.New("recipemd").Funcs(funcs).Parse(mdMainTmpl))
	template.Must(tmpl.New("ingredients").Parse(mdIngredientsTmpl))
	template.Must(tmpl.New("groups").Parse(mdGroupsTmpl))

	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, r)
	return buf.String()
}
