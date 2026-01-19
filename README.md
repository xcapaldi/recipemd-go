# RecipeMD Go

A Go implementation of the RecipeMD 2.4.0 specification as a [goldmark](https://github.com/yuin/goldmark) extension.

## Overview

RecipeMD is a Markdown-based format for writing recipes. This library provides a goldmark extension that parses RecipeMD documents and outputs structured JSON data.

## Features

- ✅ Full RecipeMD 2.4.0 specification support
- ✅ Title, description, tags, and yields parsing
- ✅ Ingredient parsing with amounts and units
- ✅ Nested ingredient groups
- ✅ Ingredient links to other recipes
- ✅ Instructions section
- ✅ JSON output format matching canonical test cases

## Installation

```bash
go get github.com/xcapaldi/recipemd-go
```

## Usage

```go
package main

import (
	"bytes"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/xcapaldi/recipemd-go/pkg/extension"
)

func main() {
	markdown := `# Chocolate Chip Cookies

The best chocolate chip cookies!

*dessert, cookies*

**24 cookies**

---

- *2 cups* flour
- *1 cup* chocolate chips

---

Mix ingredients and bake at 375°F for 10 minutes.
`

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.NewRecipeMD(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		panic(err)
	}

	fmt.Println(buf.String())
}
```

## RecipeMD Format

RecipeMD documents follow this structure:

1. **Title**: First-level heading (`# Title`)
2. **Description** (optional): Paragraphs after the title
3. **Tags** (optional): Fully italicized paragraph (`*tag1, tag2, tag3*`)
4. **Yields** (optional): Fully bold paragraph (`**2 servings, 4 cups**`)
5. **Horizontal divider**: Separates metadata from ingredients (`---`)
6. **Ingredients**: Ungrouped list and/or grouped under headings
   - Format: `- *amount unit* name [link]`
   - Example: `- *100 g* flour`
   - Example with link: `- *1* [pie crust](./pie-crust.md)`
7. **Ingredient Groups** (optional): Organize ingredients under headings
8. **Horizontal divider** (optional): Separates ingredients from instructions (`---`)
9. **Instructions** (optional): All remaining content

### Example

```markdown
# Chocolate Chip Cookies

The best chocolate chip cookies you'll ever make!

*dessert, cookies, baking*

**24 cookies**

---

## Dry Ingredients

- *2 1/4 cups* all-purpose flour
- *1 tsp* baking soda
- *1 tsp* salt

## Wet Ingredients

- *1 cup* butter
- *2* eggs
- *2 tsp* vanilla extract

## Mix-ins

- *2 cups* chocolate chips

---

Preheat oven to 375°F.

Mix dry ingredients in one bowl, wet ingredients in another.
Combine, then fold in chocolate chips.

Bake for 9-11 minutes until golden brown.
```

## Output Format

The extension outputs JSON matching the RecipeMD canonical test case format:

```json
{
  "title": "Chocolate Chip Cookies",
  "description": "The best chocolate chip cookies!",
  "tags": ["dessert", "cookies"],
  "yields": [
    {
      "factor": "24",
      "unit": "cookies"
    }
  ],
  "ingredients": [],
  "ingredient_groups": [
    {
      "title": "Dry Ingredients",
      "ingredients": [
        {
          "name": "flour",
          "amount": {
            "factor": "2",
            "unit": "cups"
          },
          "link": null
        }
      ],
      "ingredient_groups": []
    }
  ],
  "instructions": "Mix ingredients and bake."
}
```

## Testing

```bash
go test ./pkg/extension/
```

## Specification

This implementation follows the [RecipeMD 2.4.0 specification](https://github.com/RecipeMD/RecipeMD/blob/master/specification.md).

## License

MIT License - see LICENSE file for details
