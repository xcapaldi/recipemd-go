package recipemd

import (
	"bytes"
	"encoding/json"
	"strings"
	"text/template"
)

var markdownTmpl = template.Must(template.New("recipemd").Funcs(template.FuncMap{
	"join": strings.Join,
	"hashes": func(n int) string { return strings.Repeat("#", n) },
}).Parse(`# {{ .Recipe.Title }}
{{ if .HasDescription }}
{{ .Description }}
{{ end }}{{ if .Recipe.Tags }}
*{{ join .Recipe.Tags ", " }}*
{{ end }}{{ if .Recipe.Yields }}
**{{ join .SerializedYields ", " }}**
{{ end }}
---
{{ template "ingredients" . }}{{ template "groups" . }}{{ if .HasInstructions }}
---

{{ .Instructions }}
{{ end }}`))

var ingredientsTmpl = template.Must(markdownTmpl.New("ingredients").Parse(
	`{{ if .Ingredients }}
{{ range .Ingredients }}- {{ if .Amount }}*{{ .Amount }}* {{ end }}{{ if .Link }}[{{ .Name }}]({{ .Link }}){{ else }}{{ .Name }}{{ end }}
{{ end }}{{ end }}`))

var _ = template.Must(markdownTmpl.New("groups").Parse(
	`{{ range .Groups }}
{{ .Heading }} {{ .Title }}
{{ template "ingredients" .IngredientData }}{{ template "groups" .SubgroupData }}{{ end }}`))

type renderData struct {
	Recipe   *Recipe
	rounding int
	level    int
}

func (d renderData) HasDescription() bool {
	return d.Recipe.Description != nil && *d.Recipe.Description != ""
}

func (d renderData) Description() string {
	if d.Recipe.Description == nil {
		return ""
	}
	return *d.Recipe.Description
}

func (d renderData) HasInstructions() bool {
	return d.Recipe.Instructions != nil && *d.Recipe.Instructions != ""
}

func (d renderData) Instructions() string {
	if d.Recipe.Instructions == nil {
		return ""
	}
	return *d.Recipe.Instructions
}

func (d renderData) SerializedYields() []string {
	yields := make([]string, len(d.Recipe.Yields))
	for i, y := range d.Recipe.Yields {
		yields[i] = y.Serialize(d.rounding)
	}
	return yields
}

func (d renderData) Ingredients() []ingredientData {
	return makeIngredientData(d.Recipe.Ingredients, d.rounding)
}

func (d renderData) Groups() []groupData {
	return makeGroupData(d.Recipe.IngredientGroups, d.rounding, d.level)
}

type ingredientData struct {
	Name   string
	Amount string
	Link   string
}

func makeIngredientData(ingredients []Ingredient, rounding int) []ingredientData {
	result := make([]ingredientData, len(ingredients))
	for i, ing := range ingredients {
		d := ingredientData{Name: ing.Name}
		if ing.Amount != nil {
			d.Amount = ing.Amount.Serialize(rounding)
		}
		if ing.Link != nil {
			d.Link = *ing.Link
		}
		result[i] = d
	}
	return result
}

type groupData struct {
	Title    string
	rounding int
	level    int
	group    IngredientGroup
}

func (g groupData) Heading() string {
	return strings.Repeat("#", g.level)
}

func (g groupData) IngredientData() struct{ Ingredients []ingredientData } {
	return struct{ Ingredients []ingredientData }{
		Ingredients: makeIngredientData(g.group.Ingredients, g.rounding),
	}
}

func (g groupData) SubgroupData() struct{ Groups []groupData } {
	return struct{ Groups []groupData }{
		Groups: makeGroupData(g.group.IngredientGroups, g.rounding, g.level+1),
	}
}

func makeGroupData(groups []IngredientGroup, rounding int, level int) []groupData {
	result := make([]groupData, len(groups))
	for i, g := range groups {
		result[i] = groupData{
			Title:    g.Title,
			rounding: rounding,
			level:    level,
			group:    g,
		}
	}
	return result
}

// RenderMarkdown formats a Recipe as RecipeMD markdown.
func (p *Parser) RenderMarkdown(r *Recipe, rounding int) string {
	var buf bytes.Buffer
	data := renderData{Recipe: r, rounding: rounding, level: 2}
	_ = markdownTmpl.Execute(&buf, data)
	return buf.String()
}

// RenderJSON serializes a Recipe as indented JSON.
func (p *Parser) RenderJSON(r *Recipe) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
