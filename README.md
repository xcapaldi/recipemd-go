# recipemd-go

A Go library for parsing, manipulating, and rendering [RecipeMD](https://recipemd.org/) files.

RecipeMD is a Markdown-based format for writing recipes that's both human-readable and machine-parseable.

## Installation

```bash
go get github.com/xcapaldi/recipemd-go/pkg/recipemd
```

## Usage

### Library Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/xcapaldi/recipemd-go/pkg/recipemd"
)

func main() {
    source := []byte(`# Guacamole

A delicious avocado dip.

*mexican, vegan*

**4 Servings**

---

- *2* ripe avocados
- *.5 teaspoon* salt
- *1* lime, juiced
- fresh cilantro

---

Mash avocados, add salt and lime juice, mix well.
`)

    // Parse the recipe
    recipe, err := recipemd.Parse(source)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Title:", recipe.Title)
    fmt.Println("Tags:", recipe.Tags)

    // Scale the recipe (double all amounts)
    recipe.Scale(2.0)

    // Convert to different formats
    json, _ := recipe.ToJSON()
    fmt.Println(string(json))

    html, _ := recipe.ToHTML()
    // HTML includes Schema.org markup for SEO

    md, _ := recipe.ToMarkdown()
    // Reconstruct RecipeMD format
}
```

### CLI Usage

Build the CLI tool:

```bash
go build -o recipemd ./cmd/recipemd
```

Parse and convert recipes:

```bash
# Output as JSON (default)
./recipemd recipe.md

# Output as HTML with Schema.org markup
./recipemd -format html recipe.md

# Scale recipe and output as markdown
./recipemd -scale 2.0 -format markdown recipe.md

# Validate a recipe
./recipemd -validate recipe.md

# Save output to file
./recipemd -format html -output recipe.html recipe.md
```

## RecipeMD Format

RecipeMD follows a specific structure:

```markdown
# Recipe Title

Optional description paragraph(s).

*tag1, tag2, tag3*

**4 Servings, 200g**

---

- *2* ingredient with amount
- *.5 teaspoon* ingredient with amount and unit
- *1 1/2 cups* ingredient with mixed number
- *1/4 kg* ingredient with unicode fraction
- ingredient without amount
- [linked ingredient](./other-recipe.md)

## Ingredient Group (optional)

- *1* grouped ingredient

---

Instructions go here. Can be paragraphs or lists.
```

### Supported Amount Formats

The parser handles various amount formats:

| Input | Parsed Factor |
|-------|--------------|
| `5` | 5 |
| `1.5` | 1.5 |
| `1,5` (European) | 1.5 |
| `.5` | 0.5 |
| `1/2` | 0.5 |
| `3/4` | 0.75 |
| `1 1/2` | 1.5 |
| `1/2` | 0.5 |
| `1/4` | 0.25 |
| `11/2` | 1.5 |

## API Reference

### Parse Functions

```go
// Parse a RecipeMD source
func Parse(source []byte) (*Recipe, error)

// Parse from file
func ParseFile(path string) (*Recipe, error)

// Parse and panic on error (for testing/static definitions)
func MustParse(source []byte) *Recipe
```

### Recipe Type

```go
type Recipe struct {
    Title            string
    Description      *string
    Tags             []string
    Yields           []Yield
    Ingredients      []Ingredient
    IngredientGroups []IngredientGroup
    Instructions     *string
}

// Scale all amounts by a factor
func (r *Recipe) Scale(factor float64)

// Validate the recipe structure
func (r *Recipe) Validate() error

// Deep copy the recipe
func (r *Recipe) Clone() *Recipe

// Convert to JSON
func (r *Recipe) ToJSON() ([]byte, error)

// Convert to RecipeMD markdown
func (r *Recipe) ToMarkdown() ([]byte, error)

// Convert to HTML with Schema.org markup
func (r *Recipe) ToHTML() ([]byte, error)
```

### Ingredient Type

```go
type Ingredient struct {
    Name   string  // e.g., "avocados"
    Amount *Amount // nil if no amount specified
    Link   *string // URL if ingredient is a recipe link
}

type Amount struct {
    Factor string  // e.g., "1.5"
    Unit   *string // e.g., "cups", nil if no unit
}
```

### Yield Type

```go
type Yield struct {
    Factor string  // e.g., "4"
    Unit   *string // e.g., "Servings"
}
```

### Ingredient Groups

Recipes can have nested ingredient groups:

```go
type IngredientGroup struct {
    Title            string
    Ingredients      []Ingredient
    IngredientGroups []IngredientGroup // nested groups
}
```

## Features

- Full [RecipeMD specification](https://recipemd.org/specification.html) support
- Unicode vulgar fraction parsing
- Mixed number support (1 1/2)
- European decimal notation (1,5)
- Nested ingredient groups
- Linked ingredients (references to other recipes)
- Recipe scaling
- Multi-format output (JSON, Markdown, HTML)
- Schema.org Recipe markup in HTML output
- Deep cloning for safe manipulation

## License

MIT
