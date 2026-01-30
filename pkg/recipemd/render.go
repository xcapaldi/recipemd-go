package recipemd

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"
)

// ToMarkdown renders the recipe back to RecipeMD markdown format.
func (r *Recipe) ToMarkdown() ([]byte, error) {
	var buf bytes.Buffer

	// Title
	buf.WriteString("# ")
	buf.WriteString(r.Title)
	buf.WriteString("\n")

	// Description
	if r.Description != nil && *r.Description != "" {
		buf.WriteString("\n")
		buf.WriteString(*r.Description)
		buf.WriteString("\n")
	}

	// Tags
	if len(r.Tags) > 0 {
		buf.WriteString("\n*")
		buf.WriteString(strings.Join(r.Tags, ", "))
		buf.WriteString("*\n")
	}

	// Yields
	if len(r.Yields) > 0 {
		buf.WriteString("\n**")
		var yieldStrs []string
		for _, y := range r.Yields {
			if y.Unit != nil {
				yieldStrs = append(yieldStrs, y.Factor+" "+*y.Unit)
			} else {
				yieldStrs = append(yieldStrs, y.Factor)
			}
		}
		buf.WriteString(strings.Join(yieldStrs, ", "))
		buf.WriteString("**\n")
	}

	// First thematic break
	buf.WriteString("\n---\n")

	// Ingredients
	r.renderIngredients(&buf, r.Ingredients, 0)

	// Ingredient groups
	r.renderIngredientGroups(&buf, r.IngredientGroups, 2)

	// Second thematic break (only if there are instructions)
	if r.Instructions != nil && *r.Instructions != "" {
		buf.WriteString("\n---\n\n")
		buf.WriteString(*r.Instructions)
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// renderIngredients renders a list of ingredients to the buffer.
func (r *Recipe) renderIngredients(buf *bytes.Buffer, ingredients []Ingredient, indent int) {
	prefix := strings.Repeat("  ", indent)
	for _, ing := range ingredients {
		buf.WriteString("\n")
		buf.WriteString(prefix)
		buf.WriteString("- ")
		if ing.Amount != nil {
			buf.WriteString("*")
			buf.WriteString(ing.Amount.Factor)
			if ing.Amount.Unit != nil {
				buf.WriteString(" ")
				buf.WriteString(*ing.Amount.Unit)
			}
			buf.WriteString("* ")
		}
		if ing.Link != nil {
			buf.WriteString("[")
			buf.WriteString(ing.Name)
			buf.WriteString("](")
			buf.WriteString(*ing.Link)
			buf.WriteString(")")
		} else {
			buf.WriteString(ing.Name)
		}
	}
}

// renderIngredientGroups renders ingredient groups to the buffer.
func (r *Recipe) renderIngredientGroups(buf *bytes.Buffer, groups []IngredientGroup, headingLevel int) {
	for _, group := range groups {
		buf.WriteString("\n\n")
		buf.WriteString(strings.Repeat("#", headingLevel))
		buf.WriteString(" ")
		buf.WriteString(group.Title)

		r.renderIngredients(buf, group.Ingredients, 0)
		r.renderIngredientGroups(buf, group.IngredientGroups, headingLevel+1)
	}
}

// ToJSON renders the recipe to JSON format.
func (r *Recipe) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// htmlTemplate is the HTML template for rendering recipes.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script type="application/ld+json">
    {
        "@context": "https://schema.org/",
        "@type": "Recipe",
        "name": "{{.Title}}"{{if .Description}},
        "description": "{{js .Description}}"{{end}}{{if .Tags}},
        "keywords": "{{range $i, $t := .Tags}}{{if $i}}, {{end}}{{$t}}{{end}}"{{end}}{{if .Yields}},
        "recipeYield": "{{range $i, $y := .Yields}}{{if $i}}, {{end}}{{$y.Factor}}{{if $y.Unit}} {{$y.Unit}}{{end}}{{end}}"{{end}}{{if .Instructions}},
        "recipeInstructions": "{{js .Instructions}}"{{end}},
        "recipeIngredient": [{{range $i, $ing := .AllIngredients}}{{if $i}},{{end}}
            "{{if $ing.Amount}}{{$ing.Amount.Factor}}{{if $ing.Amount.Unit}} {{$ing.Amount.Unit}}{{end}} {{end}}{{$ing.Name}}"{{end}}
        ]
    }
    </script>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; line-height: 1.6; }
        h1 { color: #333; border-bottom: 2px solid #e74c3c; padding-bottom: 10px; }
        .description { color: #666; font-style: italic; margin-bottom: 20px; }
        .meta { display: flex; gap: 20px; margin-bottom: 20px; flex-wrap: wrap; }
        .tags { background: #f0f0f0; padding: 5px 10px; border-radius: 4px; }
        .tags span { background: #e74c3c; color: white; padding: 2px 8px; border-radius: 3px; margin-right: 5px; font-size: 0.85em; }
        .yields { background: #f0f0f0; padding: 5px 10px; border-radius: 4px; }
        .ingredients { background: #fafafa; padding: 15px 20px; border-radius: 8px; margin-bottom: 20px; }
        .ingredients h2 { margin-top: 0; color: #e74c3c; }
        .ingredients ul { list-style-type: disc; }
        .ingredients li { margin: 8px 0; }
        .amount { font-weight: bold; color: #e74c3c; }
        .ingredient-group { margin-left: 20px; margin-top: 15px; }
        .ingredient-group h3 { color: #666; margin-bottom: 10px; }
        .instructions { background: #fff; }
        .instructions h2 { color: #e74c3c; }
    </style>
</head>
<body>
    <article itemscope itemtype="https://schema.org/Recipe">
        <h1 itemprop="name">{{.Title}}</h1>

        {{if .Description}}
        <div class="description" itemprop="description">{{.Description}}</div>
        {{end}}

        <div class="meta">
            {{if .Tags}}
            <div class="tags">
                {{range .Tags}}<span>{{.}}</span>{{end}}
            </div>
            {{end}}

            {{if .Yields}}
            <div class="yields" itemprop="recipeYield">
                <strong>Yields:</strong>
                {{range $i, $y := .Yields}}{{if $i}}, {{end}}{{$y.Factor}}{{if $y.Unit}} {{$y.Unit}}{{end}}{{end}}
            </div>
            {{end}}
        </div>

        <div class="ingredients">
            <h2>Ingredients</h2>
            <ul>
                {{range .Ingredients}}
                <li itemprop="recipeIngredient">
                    {{if .Amount}}<span class="amount">{{.Amount.Factor}}{{if .Amount.Unit}} {{.Amount.Unit}}{{end}}</span> {{end}}{{if .Link}}<a href="{{.Link}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}
                </li>
                {{end}}
            </ul>
            {{range .IngredientGroups}}
            {{template "ingredientGroup" .}}
            {{end}}
        </div>

        {{if .Instructions}}
        <div class="instructions">
            <h2>Instructions</h2>
            <div itemprop="recipeInstructions">{{.Instructions}}</div>
        </div>
        {{end}}
    </article>
</body>
</html>

{{define "ingredientGroup"}}
<div class="ingredient-group">
    <h3>{{.Title}}</h3>
    <ul>
        {{range .Ingredients}}
        <li itemprop="recipeIngredient">
            {{if .Amount}}<span class="amount">{{.Amount.Factor}}{{if .Amount.Unit}} {{.Amount.Unit}}{{end}}</span> {{end}}{{if .Link}}<a href="{{.Link}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}
        </li>
        {{end}}
    </ul>
    {{range .IngredientGroups}}
    {{template "ingredientGroup" .}}
    {{end}}
</div>
{{end}}`

// htmlTemplateData wraps the Recipe for HTML template rendering.
type htmlTemplateData struct {
	*Recipe
	AllIngredients []Ingredient
}

// ToHTML renders the recipe to HTML format with schema.org markup.
func (r *Recipe) ToHTML() ([]byte, error) {
	tmpl, err := template.New("recipe").Parse(htmlTemplate)
	if err != nil {
		return nil, err
	}

	// Collect all ingredients (including from groups) for schema.org
	allIngredients := r.collectAllIngredients()

	data := htmlTemplateData{
		Recipe:         r,
		AllIngredients: allIngredients,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// collectAllIngredients collects all ingredients including from nested groups.
func (r *Recipe) collectAllIngredients() []Ingredient {
	var all []Ingredient
	all = append(all, r.Ingredients...)
	all = append(all, collectGroupIngredients(r.IngredientGroups)...)
	return all
}

// collectGroupIngredients recursively collects ingredients from groups.
func collectGroupIngredients(groups []IngredientGroup) []Ingredient {
	var all []Ingredient
	for _, g := range groups {
		all = append(all, g.Ingredients...)
		all = append(all, collectGroupIngredients(g.IngredientGroups)...)
	}
	return all
}
