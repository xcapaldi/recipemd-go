# RecipeMD Goldmark Extension - Implementation Plan

## Overview
Implement a RecipeMD parser as a goldmark markdown extension that fully complies with the RecipeMD specification.

## RecipeMD Specification Summary

### Document Structure
1. **Title** (required): First-level heading (`# Recipe Name`)
2. **Description** (optional): One or more paragraphs
3. **Tags** (optional): Paragraph completely in italics (`*sauce, vegan*`)
4. **Yields** (optional): Paragraph completely in bold (`**4 Servings, 200g**`)
5. **First Horizontal Divider** (required): Separator (`---`)
6. **Ingredients** (required): List items with optional groupings via headings
7. **Second Horizontal Divider** (optional): Separator
8. **Instructions** (optional): Remaining content

### Ingredient Syntax
- **Amount in italics**: `*1* avocado` or `*1 1/2* cups flour`
- **Amount formats supported**:
  - Improper fractions: `1 1/5`
  - Proper fractions: `3/7`
  - Unicode vulgar fractions: `½`, `¼`, `¾`
  - Decimals with `.` or `,`: `1.5` or `1,5`
- **Units**: Text following the amount before ingredient name
- **Ingredient Links**: `[ingredient name](recipe-url)` for referencing other recipes
- **Ingredient Groups**: Headings create groups; nesting based on heading level

### Parsing Rules
- Tags are comma-separated (except between ASCII digits)
- Yields are comma-separated amounts
- Ingredients must be in a list (unordered or ordered)
- Groups can be nested based on heading hierarchy

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│ RecipeMD Extension (goldmark.Extender)                  │
├─────────────────────────────────────────────────────────┤
│ • Registers parser, renderer, AST transformer           │
└─────────────────────────────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                  ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Parser     │  │  AST Nodes   │  │   Renderer   │
├──────────────┤  ├──────────────┤  ├──────────────┤
│ • Identifies │  │ • Recipe     │  │ • HTML       │
│   sections   │  │ • Ingredient │  │ • JSON       │
│ • Parses     │  │ • Amount     │  │ • Markdown   │
│   amounts    │  │ • Group      │  │   (preserve) │
│ • Extracts   │  │ • Tags       │  │              │
│   metadata   │  │ • Yields     │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
```

## Implementation Components

### 1. AST Node Types

#### 1.1 RecipeNode
```go
type RecipeNode struct {
    ast.BaseBlock
    Title       string
    Description []ast.Node
    Tags        []string
    Yields      []Amount
    Ingredients *IngredientGroupNode
    Instructions []ast.Node
}
```

#### 1.2 IngredientGroupNode
```go
type IngredientGroupNode struct {
    ast.BaseBlock
    Title      string
    Level      int
    Children   []ast.Node // Can contain IngredientNode or IngredientGroupNode
}
```

#### 1.3 IngredientNode
```go
type IngredientNode struct {
    ast.BaseInline
    Amount     *Amount
    Name       string
    Link       string  // Recipe reference URL if present
}
```

#### 1.4 Amount
```go
type Amount struct {
    Value      float64
    Unit       string
    RawText    string  // Original representation
}
```

### 2. Parser Implementation

#### 2.1 RecipeMDParser (implements parser.BlockParser)
Responsible for identifying RecipeMD structure and coordinating parsing.

**Priority**: Higher than default parsers to intercept recipe documents

**Trigger**:
- Document starts with level-1 heading
- Contains first horizontal divider followed by list

**Functions**:
- `Trigger(parent ast.Node, reader text.Reader, pc parser.Context) bool`
- `Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State)`
- `Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State`
- `Close(node ast.Node, reader text.Reader, pc parser.Context)`
- `CanInterruptParagraph() bool`
- `CanAcceptIndentedLine() bool`

**Parsing Flow**:
1. Parse title (first H1)
2. Parse description paragraphs
3. Parse tags (italic paragraph)
4. Parse yields (bold paragraph)
5. Wait for first horizontal divider
6. Parse ingredients (lists with groups)
7. Parse instructions (after optional second divider)

#### 2.2 AmountParser
Utility for parsing ingredient amounts.

**Functions**:
```go
func ParseAmount(text string) (*Amount, int, error)
```

**Algorithm**:
1. Skip leading whitespace
2. Try to parse fraction (improper or proper)
   - Handle unicode vulgar fractions (½ → 0.5)
   - Handle improper fractions (1 1/2 → 1.5)
   - Handle proper fractions (3/4 → 0.75)
3. Try to parse decimal
   - Support both `.` and `,` as decimal separator
4. Extract unit (remaining non-numeric text before ingredient)
5. Return Amount and bytes consumed

**Fraction Support**:
- Unicode vulgar fractions: `½`, `⅓`, `¼`, `¾`, `⅕`, `⅙`, `⅐`, `⅛`, `⅑`, `⅒`
- ASCII fractions: `1/2`, `1 1/2`
- Range handling: Consider if needed

#### 2.3 IngredientParser
Utility for parsing individual ingredient lines.

**Functions**:
```go
func ParseIngredient(text string) (*IngredientNode, error)
```

**Algorithm**:
1. Check for italic text at start (amount)
2. Parse amount if present using AmountParser
3. Detect markdown link syntax `[name](url)`
   - Extract name and URL
   - Mark as recipe reference
4. Extract ingredient name (remaining text)
5. Return IngredientNode

#### 2.4 MetadataParser
Utility for parsing tags and yields.

**Functions**:
```go
func ParseTags(text string) []string
func ParseYields(text string) []Amount
```

**Tags Algorithm**:
1. Split by comma
2. Ignore commas between ASCII digits (decimal separators)
3. Trim whitespace
4. Return list of tags

**Yields Algorithm**:
1. Split by comma (same rules as tags)
2. Parse each part as Amount
3. Return list of amounts

### 3. AST Transformer

#### 3.1 RecipeMDTransformer (implements parser.ASTTransformer)
Post-processes the AST to:
- Validate recipe structure
- Build ingredient group hierarchy
- Link recipe references
- Add semantic information

**Functions**:
- `Transform(node *ast.Document, reader text.Reader, pc parser.Context)`

**Tasks**:
1. Find RecipeNode in AST
2. Validate required sections
3. Build ingredient group tree based on heading levels
4. Resolve recipe links if needed
5. Add CSS classes or data attributes for styling

### 4. Renderer Implementation

#### 4.1 RecipeMDHTMLRenderer (implements renderer.NodeRenderer)
Renders RecipeMD AST nodes to HTML.

**Functions**:
- `RegisterFuncs(reg renderer.NodeRendererFuncRegisterer)`

**Rendering Rules**:

**RecipeNode**:
```html
<article class="recipe">
  <header>
    <h1>{title}</h1>
    <div class="description">{description}</div>
    <div class="tags">{tags}</div>
    <div class="yields">{yields}</div>
  </header>
  <section class="ingredients">
    {ingredients}
  </section>
  <section class="instructions">
    {instructions}
  </section>
</article>
```

**IngredientGroupNode**:
```html
<div class="ingredient-group" data-level="{level}">
  <h{level+2}>{title}</h{level+2}>
  <ul>
    {children}
  </ul>
</div>
```

**IngredientNode**:
```html
<li class="ingredient">
  <span class="amount">{amount.value} {amount.unit}</span>
  <span class="name">
    <a href="{link}" class="recipe-link">{name}</a> <!-- if link -->
    {name} <!-- if no link -->
  </span>
</li>
```

**Tags**:
```html
<ul class="tags">
  <li>{tag1}</li>
  <li>{tag2}</li>
</ul>
```

**Yields**:
```html
<ul class="yields">
  <li>{amount1} {unit1}</li>
  <li>{amount2} {unit2}</li>
</ul>
```

#### 4.2 Alternative Renderers (Future)
- JSON renderer (for API consumption)
- Markdown renderer (for preservation/editing)

### 5. Extension Registration

#### 5.1 RecipeMDExtension (implements goldmark.Extender)
Main extension that ties everything together.

**Functions**:
```go
func (e *RecipeMDExtension) Extend(m goldmark.Markdown)
```

**Registration**:
1. Register RecipeMDParser with high priority
2. Register RecipeMDTransformer
3. Register RecipeMDHTMLRenderer
4. Add extension options if needed

**Usage**:
```go
md := goldmark.New(
    goldmark.WithExtensions(
        recipemd.NewRecipeMDExtension(),
    ),
)
html, err := md.Convert(source)
```

## Directory Structure

```
recipemd-go/
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── IMPLEMENTATION_PLAN.md
├── parser/
│   ├── recipe.go          # RecipeMDParser
│   ├── amount.go          # AmountParser
│   ├── ingredient.go      # IngredientParser
│   ├── metadata.go        # MetadataParser (tags, yields)
│   └── parser_test.go     # Unit tests
├── ast/
│   ├── recipe.go          # RecipeNode
│   ├── ingredient.go      # IngredientNode, IngredientGroupNode
│   └── amount.go          # Amount struct
├── renderer/
│   ├── html.go            # RecipeMDHTMLRenderer
│   └── renderer_test.go   # Renderer tests
├── extension.go           # RecipeMDExtension
├── transformer.go         # RecipeMDTransformer
├── testdata/
│   ├── valid/
│   │   ├── simple.md
│   │   ├── simple.json
│   │   ├── with-groups.md
│   │   └── with-groups.json
│   └── invalid/
│       ├── no-title.invalid.md
│       └── no-divider.invalid.md
└── examples/
    ├── basic/
    │   └── main.go
    └── custom-renderer/
        └── main.go
```

## Implementation Phases

### Phase 1: Foundation (AST + Basic Parser)
- [ ] Define AST node types
- [ ] Implement Amount struct and AmountParser
- [ ] Write tests for amount parsing (fractions, decimals, unicode)
- [ ] Implement basic RecipeMDParser structure
- [ ] Parse title and description

### Phase 2: Metadata Parsing
- [ ] Implement tags parser
- [ ] Implement yields parser
- [ ] Handle metadata detection (italic/bold paragraphs)
- [ ] Write tests for metadata parsing

### Phase 3: Ingredients Parsing
- [ ] Implement IngredientParser
- [ ] Handle ingredient lists
- [ ] Parse ingredient links
- [ ] Implement IngredientGroupNode
- [ ] Handle heading-based grouping
- [ ] Write tests for ingredient parsing

### Phase 4: Instructions & Structure
- [ ] Parse horizontal dividers
- [ ] Parse instructions section
- [ ] Validate complete recipe structure
- [ ] Write integration tests

### Phase 5: AST Transformer
- [ ] Implement RecipeMDTransformer
- [ ] Build ingredient group hierarchy
- [ ] Validate recipe structure
- [ ] Add semantic information

### Phase 6: HTML Renderer
- [ ] Implement RecipeMDHTMLRenderer
- [ ] Render all node types to HTML
- [ ] Add CSS classes and data attributes
- [ ] Write renderer tests
- [ ] Test with various recipes

### Phase 7: Extension Integration
- [ ] Implement RecipeMDExtension
- [ ] Register all components
- [ ] Write end-to-end tests
- [ ] Create examples

### Phase 8: Compliance & Polish
- [ ] Add specification test cases
- [ ] Handle edge cases
- [ ] Optimize performance
- [ ] Write documentation
- [ ] Create README with examples

## Testing Strategy

### Unit Tests
- Amount parsing (all fraction types, decimals, unicode)
- Ingredient parsing (with/without amounts, links)
- Metadata parsing (tags, yields)
- Individual component tests

### Integration Tests
- Complete recipe parsing
- Recipe with nested groups
- Recipe with links
- Edge cases (minimal recipe, maximal recipe)

### Specification Compliance Tests
- Use official RecipeMD test cases from specification
- Valid recipes (*.md → *.json comparison)
- Invalid recipes (*.invalid.md should fail)

### Benchmark Tests
- Parsing performance
- Large recipe handling
- Memory allocation

## Edge Cases to Handle

1. **Empty sections**: Recipe with no description, no tags, no yields
2. **No instructions**: Recipe with only ingredients
3. **Nested groups**: Multiple levels of ingredient groups
4. **Unicode**: Vulgar fractions, non-ASCII characters in names
5. **Links**: Recipe links vs regular links in instructions
6. **Malformed amounts**: Invalid fractions, missing units
7. **Divider detection**: Multiple horizontal rules
8. **List types**: Ordered vs unordered lists for ingredients
9. **Whitespace**: Leading/trailing spaces in amounts and names
10. **Decimal separators**: Both `.` and `,` in different locales

## Dependencies

```go
require (
    github.com/yuin/goldmark v1.6.0
)
```

## Configuration Options

Consider adding options for:
- Strict mode (fail on invalid recipes vs best-effort parsing)
- CSS class prefix customization
- Link validation
- Unit normalization
- Amount formatting

## Future Enhancements

1. **JSON Renderer**: Export recipes as structured JSON
2. **Markdown Renderer**: Preserve/normalize RecipeMD format
3. **Recipe Scaling**: API to scale ingredient amounts
4. **Unit Conversion**: Convert between metric/imperial
5. **Recipe Linking**: Resolve and inline recipe references
6. **Search Index**: Build searchable index of recipes
7. **Validation Tools**: CLI to validate RecipeMD files

## References

- [RecipeMD Specification](https://github.com/RecipeMD/RecipeMD/blob/master/specification.md)
- [RecipeMD Official Site](https://recipemd.org/specification.html)
- [Goldmark Documentation](https://github.com/yuin/goldmark)
- [RecipeMD Python Reference](https://github.com/RecipeMD/RecipeMD)
- [RecipeMD Rust Implementation](https://github.com/d-k-bo/recipemd-rs)

## Success Criteria

The implementation will be considered complete when:
1. All RecipeMD specification requirements are implemented
2. All official test cases pass
3. HTML rendering matches expected output
4. Code is well-documented and tested (>80% coverage)
5. Examples demonstrate common use cases
6. README provides clear usage instructions
