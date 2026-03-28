---
name: recipemd-format
description: Read, write, and edit recipes in RecipeMD format. Use when working with .md recipe files that follow the RecipeMD specification or when creating new recipes in structured Markdown.
license: MIT
---

# RecipeMD Format

[RecipeMD](https://recipemd.org/specification.html) is a Markdown-based format for structured recipes. 
A recipe is a valid CommonMark document with a defined three-section layout.

## When to use

- Editing or reviewing `.md` files that contain recipes
- Creating new recipes in plain Markdown
- Converting recipes from other formats to RecipeMD

## Document structure

Three sections separated by `---` (thematic breaks):

```
# Title          <-- Preamble (required)
...
---
- ingredients    <-- Ingredients (required)
---
instructions     <-- Instructions (optional)
```

## Preamble

The preamble is everything before the first `---`. It must start with an H1 title.

| Element | Format | Required |
|---------|--------|----------|
| Title | `# Recipe Name` (H1 heading) | Yes |
| Description | Markdown paragraphs after the title | No |
| Tags | `*tag1, tag2, tag3*` (italic paragraph) | No |
| Yields | `**4 servings**` (bold paragraph) | No |

Tags and yields each occupy their own paragraph. Multiple yields are comma-separated within one bold span: `**4 servings, 800 ml**`.

## Ingredients

Everything between the first and second `---`. Ingredients are unordered list items.

### Basic ingredient

```markdown
- ingredient name
```

### With amount

Wrap the amount in emphasis (single `*`):

```markdown
- *200 g* spaghetti
- *2* eggs
- *1 tbsp* olive oil
```

The amount is always the first element in the list item. The format is `*[number][unit]*` where unit is optional.

### Ingredient groups

Use H2 or deeper headings to create named groups:

```markdown
## Marinade

- *2 tbsp* soy sauce
- *1 tbsp* sesame oil

## Stir Fry

- *200 g* tofu
- *1* bell pepper, sliced
```

Groups can nest (H3 inside H2, etc).

### Linked ingredients

Link an ingredient name to another recipe file:

```markdown
- *200 ml* [tomato sauce](./tomato-sauce.md)
```

## Instructions

Everything after the second `---`. Free-form Markdown -- any valid CommonMark content (paragraphs, lists, images, tables, etc).

The second `---` and instructions section are optional. A recipe with only a preamble and ingredients is valid.

## Amount number formats

Amounts support several number formats:

| Format | Examples |
|--------|----------|
| Integer | `3`, `12` |
| Decimal | `1.5`, `0.75` |
| Decimal (comma) | `1,5` |
| Fraction | `1/2`, `3/4` |
| Mixed number | `1 1/2`, `2 3/4` |
| Vulgar fraction | `½`, `¼`, `¾`, `⅓`, `⅔` |

## Complete example

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

## Common mistakes

- Missing `---` between sections (preamble, ingredients, instructions must be separated by thematic breaks)
- Using `**bold**` for amounts instead of `*italic*` (amounts use single emphasis)
- Placing amounts outside emphasis markers: `200 g spaghetti` instead of `*200 g* spaghetti`
- Using ordered lists for ingredients (must be unordered `- `)
- Forgetting that tags use italic (`*...*`) and yields use bold (`**...**`)

## Frontmatter (non-standard extension)
  
Some tools support YAML (`---`) or TOML (`+++`) frontmatter before the recipe content.
This is not part of the official spec but my be useful when combining RecipeMD with note management systems.
Parsers that support this strip the frontmatter before applying the standard rules.

## Reference

Full specification: https://recipemd.org/specification.html
