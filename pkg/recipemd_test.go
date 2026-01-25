package recipemd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

const sampleRecipe = `# Peanut Butter Cookies

A simple classic cookie recipe.

*cookies, dessert, easy*

**24 cookies**

---

- *2 cups* flour
- *1 cup* peanut butter
- *1 cup* sugar
- *1* egg

---

Mix all ingredients. Form into balls. Bake at 350°F for 12 minutes.
`

func TestParseRecipe(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(New()),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(sampleRecipe), &buf); err != nil {
		t.Fatal(err)
	}

	html := buf.String()

	checks := []string{
		`<article class="recipe">`,
		`<h1>Peanut Butter Cookies</h1>`,
		`<div class="description">`,
		`<div class="tags">`,
		`<span class="tag">cookies</span>`,
		`<div class="yields">`,
		`<section class="ingredients">`,
		`<li><em>2 cups</em> flour</li>`,
		`<section class="instructions">`,
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q\ngot:\n%s", check, html)
		}
	}
}

func TestParseRecipeWithSchemaOrg(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(New(WithSchemaOrg())),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(sampleRecipe), &buf); err != nil {
		t.Fatal(err)
	}

	html := buf.String()

	checks := []string{
		`itemscope itemtype="https://schema.org/Recipe"`,
		`itemprop="name"`,
		`itemprop="description"`,
		`itemprop="keywords"`,
		`itemprop="recipeYield"`,
		`itemprop="recipeIngredient"`,
		`itemprop="recipeInstructions"`,
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q\ngot:\n%s", check, html)
		}
	}
}

func TestExtractRecipe(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(New()),
	)

	source := []byte(sampleRecipe)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	recipe := ExtractRecipe(doc.OwnerDocument(), source)

	if recipe.Title != "Peanut Butter Cookies" {
		t.Errorf("expected title 'Peanut Butter Cookies', got %q", recipe.Title)
	}

	if len(recipe.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(recipe.Tags))
	}

	if len(recipe.Yields) != 1 {
		t.Errorf("expected 1 yield, got %d", len(recipe.Yields))
	}

	if len(recipe.Ingredients) != 4 {
		t.Errorf("expected 4 ingredients, got %d", len(recipe.Ingredients))
	}
}

func TestBuildAST(t *testing.T) {
	recipe := &Recipe{
		Title:       "Test Recipe",
		Description: "A test description.",
		Tags:        []string{"test", "example"},
		Yields:      []Yield{{Amount: "4", Unit: "servings"}},
		Ingredients: []IngredientEntry{
			Ingredient{Amount: &Amount{Quantity: "1", Unit: "cup"}, Name: "flour"},
			Ingredient{Amount: &Amount{Quantity: "2", Unit: "tbsp"}, Name: "sugar"},
		},
		Instructions: "Mix and bake.",
	}

	doc := BuildAST(recipe)

	if doc.FirstChild() == nil {
		t.Error("expected document to have children")
	}
}
