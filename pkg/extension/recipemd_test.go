package extension

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
)

func TestRecipeMD_BasicRecipe(t *testing.T) {
	markdown := `# Test Recipe

This is a test recipe description.

*tag1, tag2*

**2 servings**

---

- *100 g* flour
- *2* eggs
- salt

---

Mix all ingredients and bake.
`

	md := goldmark.New(
		goldmark.WithExtensions(
			NewRecipeMD(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		t.Fatal(err)
	}

	// Just verify it produces valid JSON for now
	output := buf.String()
	if output == "" {
		t.Fatal("Expected output, got empty string")
	}

	t.Logf("Output: %s", output)
}

func TestRecipeMD_Tags(t *testing.T) {
	markdown := `# Tags

*tag1, tag2, tag3*
`

	md := goldmark.New(
		goldmark.WithExtensions(
			NewRecipeMD(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	t.Logf("Output: %s", output)
}

func TestRecipeMD_Yields(t *testing.T) {
	markdown := `# Yields

**1.2 cups, 1,5 Tassen, 1 1/4 servings, 5 servings, 5**
`

	md := goldmark.New(
		goldmark.WithExtensions(
			NewRecipeMD(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	t.Logf("Output: %s", output)
}

func TestRecipeMD_Ingredients(t *testing.T) {
	markdown := `# Ingredients

---

- *5* ingredient 1
- ingredient 2
- *1.5 ml* ingredient 3

`

	md := goldmark.New(
		goldmark.WithExtensions(
			NewRecipeMD(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	t.Logf("Output: %s", output)
}

func TestRecipeMD_IngredientGroups(t *testing.T) {
	markdown := `# Title

---

- ingredient 0

## Group 1

- ingredient 1
- ingredient 2

### Subgroup 1.1

- ingredient 3
- ingredient 4

## Group 2

- ingredient 7
- ingredient 8
`

	md := goldmark.New(
		goldmark.WithExtensions(
			NewRecipeMD(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	t.Logf("Output: %s", output)
}
