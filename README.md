# recipemd-go

[![Go Reference](https://pkg.go.dev/badge/github.com/xcapaldi/recipemd-go.svg)](https://pkg.go.dev/github.com/xcapaldi/recipemd-go)
![GitHub Release](https://img.shields.io/github/v/release/xcapaldi/recipemd-go)

A Go library for parsing, scaling, and rendering recipes in the [RecipeMD](https://recipemd.org) format.
This format builds on top of structured Markdown such that both humans and programs can digest it.

*Go, parser, RecipeMD, Markdown*

**1 Go module**

---

- *1 pkg* `github.com/xcapaldi/recipemd-go`
- *1 pkg* `github.com/yuin/goldmark` Markdown parser

## Examples

- *1* `parse` — parse a recipe and output JSON
- *1* `scale` — scale a recipe by factor or target yield
- *1* `flatten` — inline all linked sub-recipe ingredients
- *1* `renderhtml` — render a recipe as an HTML `<article>`

---

## Installation

```bash
go get github.com/xcapaldi/recipemd-go
```

## What is RecipeMD?

[RecipeMD](https://recipemd.org/specification.html) is a Markdown-based format for writing recipes.
A recipe file is plain Markdown with a defined structure:

```markdown
# Carbonara

A classic Roman pasta.

*Italian, pasta*

**2 servings**

---

- *200 g* spaghetti
- *100 g* guanciale
- *2* eggs
- *50 g* Pecorino Romano

## Sauce

- *1 tbsp* black pepper, coarsely ground

---

Boil pasta. Render guanciale. Whisk eggs with cheese and pepper.
Toss together off the heat.
```

The document has three sections divided by `---` thematic breaks:

| Section | Content |
|---|---|
| Preamble | H1 title, optional description, optional *tags* (italic), optional **yields** (bold) |
| Ingredients | Unordered lists; H2+ headings introduce named ingredient groups |
| Instructions | Free-form Markdown text |

Amounts are wrapped in emphasis: `*2 tbsp*`. Supported number formats:
integers (`3`), decimals (`1.5`), fractions (`1/2`), improper fractions
(`1 1/2`), and Unicode vulgar fractions (`½ ¼ ¾`).

## Why RecipeMD?

Shockingly there are multiple competing specifications in the world of plaintext cooking:

- [Open Recipe Format](https://open-recipe-format.readthedocs.io/en/latest/) -- YAML
- [hrecipe](https://microformats.org/wiki/hrecipe) -- XML
- [schema.org Recipe](https://schema.org/Recipe)
- [Cooklang](https://cooklang.org) -- DSL

Of the above, the only one that goes beyond a strictly formatted schema is Cooklang.
It is an extensive project with a CLI, web server and thoughtful features like inline ingredent declarations (these are extracted at render time).
Despite this, it lacks one fundamental advantage working in plain text -- it's not very readable in it's raw format.
This may seem like a minor inconvenience but it makes reading and writing recipes less accessible to humans and (increasingly important) AI agents.
The beauty of RecipeMD is that it's a ruleset on standard Markdown syntax with all of that format's inherent flexibility (images/tables/etc).
The nature of Cooklang's strict DSL enables some advanced features but I believe they can be acheived in RecipeMD through contextual analysis.

As an example working in the raw formats, here is a moderately complex recipe in both:

### RecipeMD

```markdown
# Chicken Tikka Masala

Tender charred chicken in a rich, spiced tomato-cream sauce.

*Indian, chicken, curry*

**4 servings**

---

## Marinade

- *700 g* boneless chicken thighs, cut into chunks
- *150 g* plain yogurt
- *2 tbsp* lemon juice
- *3 cloves* garlic, minced
- *1 tsp* fresh ginger, grated
- *1 tsp* garam masala
- *1 tsp* cumin
- *1/2 tsp* turmeric
- *1/2 tsp* cayenne pepper
- *1 tsp* salt

## Sauce

- *2 tbsp* ghee
- *1* large onion, finely diced
- *4 cloves* garlic, minced
- *1 tbsp* fresh ginger, grated
- *1 tbsp* garam masala
- *1 tsp* cumin
- *1 tsp* coriander
- *1/2 tsp* turmeric
- *1/2 tsp* cayenne pepper
- *400 g* canned crushed tomatoes
- *200 ml* heavy cream
- *1 tsp* salt

## Serving

- *300 g* basmati rice
- *1* handful fresh cilantro, roughly chopped

---

Combine chicken with all marinade ingredients in a bowl. Cover and refrigerate for at least 1 hour, ideally overnight.

Preheat a broiler or grill to high. Thread chicken onto skewers and cook for 10–12 minutes, turning once, until charred in spots and just cooked through. Set aside.

Cook rice according to package directions.

Heat ghee in a large saucepan over medium heat. Add onion and cook for 8–10 minutes until deep golden. Add garlic and ginger; cook 2 minutes until fragrant.

Stir in garam masala, cumin, coriander, turmeric, and cayenne. Toast 1 minute. Add crushed tomatoes and simmer uncovered for 15 minutes, stirring occasionally, until sauce thickens.

Pour in cream and stir to combine. Add the grilled chicken and simmer 5 minutes to meld flavors. Season with salt.

Serve over rice, garnished with fresh cilantro.
```

### Cooklang

```
---
title: Chicken Tikka Masala
description: Tender charred chicken in a rich, spiced tomato-cream sauce.
tags: Indian, chicken, curry
servings: 4
---

== Marinate ==

Combine @chicken thighs{700%g}(boneless, cut into chunks) with @plain yogurt{150%g},
@lemon juice{2%tbsp}, @garlic{3%cloves}(minced), @fresh ginger{1%tsp}(grated),
@garam masala{1%tsp}, @cumin{1%tsp}, @turmeric{1/2%tsp}, @cayenne pepper{1/2%tsp},
and @salt{1%tsp} in a #large bowl{}.

Cover and refrigerate for ~marinating{1%hour} (or overnight for best results).

== Grill Chicken ==

Preheat a #grill or broiler to high. Thread chicken onto #skewers and cook for
~grilling{12%minutes}, turning once, until charred in spots and cooked through. Set aside.

== Cook Rice ==

Cook @basmati rice{300%g} in a #saucepan{} according to package directions (~{18%minutes}).

== Make Sauce ==

Heat @ghee{2%tbsp} in a #large saucepan{} over medium heat.
Add @onion{1%large}(finely diced) and cook for ~onion{10%minutes} until deep golden.

Add @garlic{4%cloves}(minced) and @fresh ginger{1%tbsp}(grated); cook ~{2%minutes} until fragrant.

Stir in @garam masala{1%tbsp}, @cumin{1%tsp}, @coriander{1%tsp}, @turmeric{1/2%tsp}, and @cayenne pepper{1/2%tsp}.
Toast ~{1%minute}, then add @crushed tomatoes{400%g} and simmer uncovered for ~{15%minutes},
stirring occasionally, until thickened.

Pour in @heavy cream{200%ml} and stir to combine.
Add grilled chicken and simmer ~finishing{5%minutes} to meld flavors.
Season with @salt{1%tsp}.

== Serve ==

Plate over rice and garnish with @fresh cilantro{1%handful}(roughly chopped).
```

## Library Usage

### Parsing

`Parse` accepts any `io.Reader`:

```go
import recipemd "github.com/xcapaldi/recipemd-go"

p := recipemd.NewParser()

f, _ := os.Open("carbonara.md")
defer f.Close()

recipe, err := p.Parse(f)
if err != nil {
    log.Fatal(err)
}

fmt.Println(recipe.Title)   // "Carbonara"
fmt.Println(recipe.Tags)    // ["Italian", "pasta"]
fmt.Println(recipe.Yields)  // [{Factor:2 Unit:"servings"}]
```

Parse collects all structural and value-level problems via `errors.Join`,
so a single call surfaces every issue at once. Individual errors carry line
and column information:

```go
if parseError, ok := errors.AsType[*recipemd.ParseError](err); ok {
    fmt.Printf("line %d, col %d: %s\n", parseError.Line, parseError.Column, parseError.Message)
}
```

#### Parser options

While not part of the official spec, this implementation supports some parser options to make the format more ergonomic.
In particular `WithFrontmatter` will strip YAML/TOML frontmatter before parsing the remaining content as spec-complient RecipeMD.
This is particularly useful if you combine this format with another note management system (like [Denote](https://protesilaos.com/emacs/denote)) that relies on frontmatter.
In addition `WithGithubFormattedMarkdown` enables support for GFM features like tables, autolinks, task lists (in ingredients as well) and strikethrough.

```go
parser := recipemd.NewParser(
    recipemd.WithFrontmatter(),             // strip YAML/TOML front matter
    recipemd.WithGithubFormattedMarkdown(), // enable GFM (tables, task lists, …)
)
```

### Scaling

```go
// Multiply all amounts by a factor.
recipe.Scale(2)

// Scale to a specific yield.
desired, _ := recipemd.ParseAmountString("6 servings")
if err := recipe.ScaleForYield(desired); err != nil {
    log.Fatal(err)
}
```

### Rendering

```go
// RecipeMD Markdown (amounts rounded to 2 decimal places)
fmt.Print(p.RenderMarkdown(recipe, 2))

// Compact JSON
data, err := p.RenderJSON(recipe)

// HTML <article> element (amounts rounded to 3 decimal places) (still WIP)
fmt.Println(p.RenderHTML(recipe, 3))
```

## Examples

The `examples/` directory contains small, self-contained programs that
demonstrate common use cases:

| Example | Description |
|---|---|
| [`examples/parse`](examples/parse) | Parse a recipe and write compact JSON to stdout |
| [`examples/scale`](examples/scale) | Scale a recipe by factor or target yield, write RecipeMD to stdout |
| [`examples/flatten`](examples/flatten) | Inline all linked sub-recipes, write RecipeMD to stdout |
| [`examples/renderhtml`](examples/renderhtml) | Flatten and render as an HTML `<article>` |

Run any example directly:

```bash
go run ./examples/parse   carbonara.md
go run ./examples/scale   carbonara.md "4 servings"
go run ./examples/flatten main_dish.md
go run ./examples/renderhtml carbonara.md
```

## Running tests

```bash
go test ./...
```

The suite includes the [RecipeMD canonical test suite](https://github.com/tstehr/RecipeMD/tree/master/test),
extensive golden tests and unit tests.

## Performance

This library is quite performant compared to the reference implementation but not due to any clever optimizations.
Go is simply much faster and this translates directly.
In practice, the difference should be almost unnoticeable for any level of personal use.
Nevertheless, here is a very crude benchmark parsing and scaling a short recipe:

| Implementation | Command | Runs | Total | Avg/run | Speedup |
|----------------|---------|-----:|------:|--------:|--------:|
| Python recipemd 5.0.0 | `recipemd -y "10 ml" testdata/canonical/recipe.md` | 100 | 13,956 ms | 139.5 ms | 1x |
| Go (examples/scale) | `scale testdata/canonical/recipe.md "10 ml"` | 100 | 434 ms | 4.3 ms | **32x** |

## License

MIT — see [LICENSE](LICENSE).
