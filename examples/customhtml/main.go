// Customhtml shows how to write your own HTML renderer for a parsed recipe
// instead of using [recipemd.Recipe.RenderHTML].
//
// Usage: customhtml <recipe.md>
//
// The markup here is aimed at a page that styles tags as pills and rescales
// amounts in the browser: each tag is its own <li>, and every amount and
// yield carries its unrounded factor in a data-factor attribute.
//
//	customhtml recipe.md > recipe.html
package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/yuin/goldmark"

	recipemd "github.com/xcapaldi/recipemd-go"
)

// groupCtx pairs an ingredient group with its heading level, which the
// Recipe type does not store: it is implied by nesting depth.
type groupCtx struct {
	recipemd.IngredientGroup
	Level int
}

func (g groupCtx) Subgroups() []groupCtx { return withLevel(g.IngredientGroups, g.Level+1) }

func withLevel(groups []recipemd.IngredientGroup, level int) []groupCtx {
	out := make([]groupCtx, len(groups))
	for i, g := range groups {
		out[i] = groupCtx{g, min(level, 6)}
	}
	return out
}

const tmplSrc = `<article class="recipe">
<h1>{{ .Title }}</h1>
{{- with .Description }}
<div class="description">{{ markdown . }}</div>
{{- end }}
{{- if .Tags }}
<ul class="tags">
{{- range .Tags }}
  <li class="tag">{{ . }}</li>
{{- end }}
</ul>
{{- end }}
{{- if .Yields }}
<ul class="yields">
{{- range .Yields }}
  <li class="yield">{{ template "amount" . }}</li>
{{- end }}
</ul>
{{- end }}
{{ template "ingredients" .Ingredients }}
{{- range groups .IngredientGroups 2 }}{{ template "group" . }}{{ end }}
{{- with .Instructions }}
<section class="instructions">{{ markdown . }}</section>
{{- end }}
</article>
{{ define "amount" -}}
<span class="amount" data-factor="{{ .FormatFactor -1 }}"{{ with .Unit }} data-unit="{{ . }}"{{ end }}>
  {{- .FormatFactor rounding }}{{ with .Unit }} {{ . }}{{ end -}}
</span>
{{- end }}
{{ define "ingredients" }}{{ if . -}}
<ul class="ingredients">
{{- range . }}
  <li>{{ with .Amount }}{{ template "amount" . }} {{ end -}}
  {{ if .Link }}<a href="{{ .Link }}">{{ .Name }}</a>{{ else }}{{ .Name }}{{ end }}</li>
{{- end }}
</ul>
{{- end }}{{ end }}
{{ define "group" }}
<section class="group" data-level="{{ .Level }}">
{{ heading .Level .Title }}
{{ template "ingredients" .Ingredients }}
{{- range .Subgroups }}{{ template "group" . }}{{ end }}
</section>
{{- end }}`

// render renders r with a custom template. Description and Instructions
// hold raw markdown, so they go through md; amounts on screen are rounded to
// rounding decimal places.
func render(r *recipemd.Recipe, md goldmark.Markdown, rounding int) (string, error) {
	funcs := template.FuncMap{
		"rounding": func() int { return rounding },
		"groups":   withLevel,
		"markdown": func(src string) (template.HTML, error) {
			var buf bytes.Buffer
			if err := md.Convert([]byte(src), &buf); err != nil {
				return "", err
			}
			// goldmark drops raw HTML from the source unless configured
			// with html.WithUnsafe, so this output is safe to mark as HTML.
			return template.HTML(buf.String()), nil
		},
		"heading": func(level int, title string) template.HTML {
			return template.HTML(fmt.Sprintf("<h%d>%s</h%d>", level, template.HTMLEscapeString(title), level))
		},
	}
	tmpl, err := template.New("recipe").Funcs(funcs).Parse(tmplSrc)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: customhtml <recipe.md>")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	r, err := recipemd.NewParser().Parse(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out, err := render(r, goldmark.New(), 2)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(out)
}
