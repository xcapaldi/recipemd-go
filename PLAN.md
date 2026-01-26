# RecipeMD-Go Implementation Plan

A Go implementation of the [RecipeMD specification](https://recipemd.org/specification.html) with a web UI, intelligent scaling, and MCP server integration.

## Table of Contents

- [Overview](#overview)
- [Architecture Decisions](#architecture-decisions)
- [Data Structures](#data-structures)
- [Milestone 1: CLI Feature Parity](#milestone-1-cli-feature-parity)
- [Milestone 2: Web UI & Advanced Features](#milestone-2-web-ui--advanced-features)
- [Milestone 3: Performance & Storage](#milestone-3-performance--storage)
- [Milestone 4: Enhanced Metadata](#milestone-4-enhanced-metadata)
- [Milestone 5: Recipe Editing](#milestone-5-recipe-editing)
- [Infrastructure](#infrastructure)
- [Dependencies](#dependencies)
- [Future Considerations](#future-considerations)

---

## Overview

### What is RecipeMD?

RecipeMD is a Markdown-based format for recipes with a strict header structure followed by free-form instructions. A recipe consists of:

```markdown
# Recipe Title

Short description paragraph(s).

*tag1, tag2, tag3*

**4 servings, 500g**

---

- *1* avocado
- *.5 teaspoon* salt
- *1 1/2 pinches* red pepper flakes
- lemon juice

## Ingredient Group (optional)

- *200g* flour
- *100ml* water

---

Free-form markdown instructions go here.
Can include any valid CommonMark: headers, lists, images, etc.
```

### Key Characteristics

1. **Strict header structure**: Title → Description → Tags → Yields → Ingredients (in that order)
2. **Amounts in italics**: `*1 cup*` within ingredient list items
3. **Tags in italics paragraph**: `*italian, vegetarian*`
4. **Yields in bold paragraph**: `**4 servings**`
5. **Horizontal rules**: Delimit the ingredient section
6. **Free-form instructions**: Standard CommonMark after the ingredient section

### Project Goals

1. Parse RecipeMD files into structured Go types
2. Render recipes to HTML with customizable templates
3. Round-trip back to valid RecipeMD markdown
4. Scale recipes with intelligent unit conversion
5. Provide CLI, web UI, and MCP server interfaces

---

## Architecture Decisions

### Parser Strategy: Hybrid Approach

**Decision**: Pre-parse the header section for structured data extraction, use goldmark for rendering description and instructions.

**Rationale**:
- The RecipeMD header has strict ordering requirements that goldmark's extension model isn't designed to enforce
- Goldmark excels at *adding* syntax, not *constraining* it
- The header parsing is about data extraction, not rendering
- Instructions and descriptions are standard CommonMark where goldmark shines

**Implementation**:
```
┌─────────────────────────────────────────────────────────┐
│                    Raw .md file                         │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Custom Header Parser                       │
│  - Extract title (# heading)                            │
│  - Extract description (paragraphs before tags)         │
│  - Extract tags (*italics paragraph*)                   │
│  - Extract yields (**bold paragraph**)                  │
│  - Extract ingredients (list items between ---)         │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                 Recipe struct                           │
│  - Title: string                                        │
│  - Description: string (raw markdown)                   │
│  - Tags: []string                                       │
│  - Yields: []Amount                                     │
│  - IngredientGroups: []IngredientGroup                  │
│  - Instructions: string (raw markdown)                  │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Goldmark Rendering                         │
│  - Render Description to HTML                           │
│  - Render Instructions to HTML                          │
│  - Custom renderer for recipe-specific elements         │
└─────────────────────────────────────────────────────────┘
```

### Why Not Full Goldmark Integration?

We considered creating custom AST nodes (`RecipeNode`, `IngredientNode`, etc.) but rejected this because:

1. **Validation complexity**: RecipeMD requires elements in a specific order. Goldmark's parser would accept them in any order, requiring post-parse validation anyway.

2. **Separation of concerns**: Parsing structured data is fundamentally different from rendering markdown. The hybrid approach keeps these concerns separate.

3. **Round-trip fidelity**: By storing raw markdown strings for description/instructions, we preserve formatting when writing back to `.md` files.

4. **Simpler testing**: Unit tests for header parsing don't need goldmark. Unit tests for rendering don't need recipe parsing.

### Web UI: Standard Library Templates + HTML

**Decision**: Use `html/template` with plain HTML and AIP-style URL navigation.

**Rationale**:
- Zero external dependencies for the web layer
- AIP-160 filtering naturally maps to query parameters
- Progressive enhancement possible later (htmx, etc.)
- Simpler deployment (single binary)

### Storage: SQLite

**Decision**: Use SQLite for recipe indexing and metadata.

**Rationale**:
- Excellent for filtering and complex queries (supports AIP-160 implementation)
- Single file, portable, embedded
- Good performance for read-heavy workloads
- WAL mode for concurrent reads
- Recipes remain as source-of-truth `.md` files; SQLite is an index/cache

### Unit Conversion: Start Simple

**Decision**: Implement basic scaling first, document intelligent conversion as enhancement.

**Rationale**:
- Core functionality (multiply amounts) is straightforward
- Intelligent conversion (tbsp → cups) requires unit ontology
- Can add later without breaking changes

**Future options documented**:
- [go-units](https://github.com/bcicen/go-units) - General unit conversion
- [measurementcommon](https://github.com/forgedsoftware/measurementcommon) concepts - Cooking-specific
- Custom implementation with cooking unit hierarchy

### MCP Server: Official SDK

**Decision**: Use `github.com/modelcontextprotocol/go-sdk`

**Rationale**:
- Maintained by Anthropic/Google collaboration
- Full MCP spec implementation
- Long-term support expected

---

## Data Structures

### Core Types

```go
// Amount represents a quantity with optional unit
// Examples: "1", "1.5 cups", "1/2 teaspoon"
type Amount struct {
    Quantity *float64 // nil if no quantity (e.g., "salt to taste")
    Unit     string   // empty if no unit (e.g., "2 eggs")

    // Original representation for round-trip fidelity
    OriginalString string
}

// Ingredient represents a single ingredient line
type Ingredient struct {
    Amount *Amount // nil if no amount specified
    Name   string  // "avocado", "salt", etc.
}

// IngredientGroup represents a named group of ingredients
// The default group (no header) has an empty name
type IngredientGroup struct {
    Name        string
    Ingredients []Ingredient
}

// Recipe represents a complete parsed RecipeMD document
type Recipe struct {
    // Filesystem metadata
    Path     string    // Absolute path to source file
    Modified time.Time // Last modification time

    // Recipe content
    Title            string
    Description      string // Raw markdown
    Tags             []string
    Yields           []Amount
    IngredientGroups []IngredientGroup
    Instructions     string // Raw markdown

    // Extended metadata (Milestone 4)
    LastCooked *time.Time
    Rating     *int // 1-5
    Notes      string
}
```

### Amount Parsing

RecipeMD amounts can be expressed as:
- Decimals with `.` or `,`: `1.5`, `1,5`
- Fractions: `1/2`, `3/4`
- Mixed numbers: `1 1/2`
- Ranges (future): `1-2`

```go
// ParseAmount parses an amount string into structured data
// Examples:
//   "1" → Amount{Quantity: 1.0, Unit: ""}
//   "1.5 cups" → Amount{Quantity: 1.5, Unit: "cups"}
//   "1/2 teaspoon" → Amount{Quantity: 0.5, Unit: "teaspoon"}
//   "1 1/2 pinches" → Amount{Quantity: 1.5, Unit: "pinches"}
func ParseAmount(s string) (*Amount, error)
```

### Comma Handling in Lists

Per the spec: "a comma is not treated as a list divider if the characters directly before and after are numerical."

This allows `**4 servings, 1,5 kg**` to parse as two yields: `4 servings` and `1,5 kg` (not three).

```go
// SplitRespectingDecimals splits on commas but not decimal commas
// "4 servings, 1,5 kg" → ["4 servings", "1,5 kg"]
// "tag1, tag2, tag3" → ["tag1", "tag2", "tag3"]
func SplitRespectingDecimals(s string) []string
```

---

## Milestone 1: CLI Feature Parity

**Goal**: Match [RecipeMD CLI](https://recipemd.org/cli.html) functionality.

### Features

| CLI Command | Description | Implementation |
|-------------|-------------|----------------|
| `recipemd <file>` | Display recipe | Parse and render to terminal |
| `recipemd -j <file>` | JSON output | Marshal Recipe struct |
| `recipemd -r <dir>` | List recipes | Recursive glob `**/*.md` |
| `recipemd --multiply <n>` | Scale recipe | Multiply all amounts |
| `recipemd --yield <amount>` | Scale to yield | Calculate multiplier from yield |
| `recipemd --export html` | HTML export | Goldmark render |

### Package Structure

```
recipemd-go/
├── cmd/
│   └── recipemd/
│       └── main.go           # CLI entry point
├── pkg/
│   ├── parser/
│   │   ├── parser.go         # Main Parse() function
│   │   ├── amount.go         # Amount parsing
│   │   ├── amount_test.go
│   │   ├── header.go         # Header section parsing
│   │   ├── header_test.go
│   │   └── ingredients.go    # Ingredient list parsing
│   ├── recipe/
│   │   ├── recipe.go         # Recipe type definition
│   │   ├── scale.go          # Scaling operations
│   │   └── scale_test.go
│   ├── render/
│   │   ├── html.go           # HTML rendering via goldmark
│   │   ├── markdown.go       # Round-trip markdown output
│   │   ├── terminal.go       # Terminal/text output
│   │   └── json.go           # JSON serialization
│   └── extension/
│       └── recipemd.go       # Goldmark extension (existing)
├── go.mod
├── go.sum
└── PLAN.md
```

### Implementation Steps

#### 1.1 Amount Parser
- [ ] Define `Amount` struct with `Quantity`, `Unit`, `OriginalString`
- [ ] Implement decimal parsing (`.` and `,` separators)
- [ ] Implement fraction parsing (`1/2`, `3/4`)
- [ ] Implement mixed number parsing (`1 1/2`)
- [ ] Implement `SplitRespectingDecimals` for comma handling
- [ ] Write comprehensive tests with edge cases

#### 1.2 Header Parser
- [ ] Extract title from first `#` heading
- [ ] Extract description paragraphs (before tags/yields)
- [ ] Detect and parse tags paragraph (fully italicized)
- [ ] Detect and parse yields paragraph (fully bold)
- [ ] Handle optional elements (tags and yields are optional)
- [ ] Validate ordering (title must come first, etc.)

#### 1.3 Ingredient Parser
- [ ] Detect ingredient section boundaries (`---` horizontal rules)
- [ ] Parse unordered list items as ingredients
- [ ] Extract amount from italicized prefix
- [ ] Handle ingredient groups (headings within ingredient section)
- [ ] Handle ingredients without amounts

#### 1.4 Recipe Assembly
- [ ] Combine parsers into unified `Parse([]byte) (*Recipe, error)`
- [ ] Extract instructions (everything after ingredient section)
- [ ] Preserve raw markdown for description and instructions

#### 1.5 Rendering
- [ ] HTML renderer using goldmark for description/instructions
- [ ] Custom HTML template for full recipe layout
- [ ] Terminal renderer with ANSI formatting
- [ ] JSON marshaling
- [ ] Markdown round-trip renderer

#### 1.6 CLI
- [ ] Flag parsing with standard library `flag` package
- [ ] Recursive directory listing with `filepath.WalkDir`
- [ ] Implement `--multiply` scaling
- [ ] Implement `--yield` scaling (requires yield parsing)
- [ ] Output format selection (text/json/html)

### Testing Strategy

```go
// Test fixtures as embedded files
//go:embed testdata/*.md
var testRecipes embed.FS

// Table-driven tests for amount parsing
func TestParseAmount(t *testing.T) {
    tests := []struct {
        input    string
        expected Amount
    }{
        {"1", Amount{Quantity: ptr(1.0), Unit: ""}},
        {"1.5 cups", Amount{Quantity: ptr(1.5), Unit: "cups"}},
        {"1/2 teaspoon", Amount{Quantity: ptr(0.5), Unit: "teaspoon"}},
        // ... more cases
    }
    // ...
}
```

---

## Milestone 2: Web UI & Advanced Features

**Goal**: Web interface with filtering, unit switching, and recipe scaling.

### Features

1. **Recipe browsing**: List all recipes with pagination
2. **AIP-160 filtering**: `?filter=tags:"vegetarian" AND yields.quantity>2`
3. **Unit switching**: Toggle between imperial and metric display
4. **Intelligent scaling**: Scale with UI controls
5. **MCP server**: Expose recipes to LLM applications

### Web UI Design

#### URL Structure (AIP-style)

```
GET /recipes                           # List recipes
GET /recipes?filter=tags:"vegan"       # Filtered list
GET /recipes?filter=title:"pasta"      # Search by title
GET /recipes/{path}                    # View single recipe
GET /recipes/{path}?scale=2            # Scaled view
GET /recipes/{path}?units=metric       # Unit preference
```

#### Template Structure

```
web/
├── templates/
│   ├── layout.html        # Base layout
│   ├── recipe_list.html   # Recipe listing
│   ├── recipe_view.html   # Single recipe view
│   └── partials/
│       ├── ingredient.html
│       ├── pagination.html
│       └── filter_form.html
├── static/
│   └── style.css          # Minimal CSS
└── handlers/
    ├── recipes.go         # HTTP handlers
    └── middleware.go      # Logging, etc.
```

#### HTML Example

```html
<!-- recipe_view.html -->
{{define "content"}}
<article class="recipe">
  <h1>{{.Recipe.Title}}</h1>

  <div class="description">
    {{.DescriptionHTML}}
  </div>

  {{if .Recipe.Tags}}
  <ul class="tags">
    {{range .Recipe.Tags}}
    <li><a href="/recipes?filter=tags:{{.}}">{{.}}</a></li>
    {{end}}
  </ul>
  {{end}}

  {{if .Recipe.Yields}}
  <div class="yields">
    {{range .Recipe.Yields}}
    <span class="yield">{{.}}</span>
    {{end}}

    <form class="scale-form" method="get">
      <label>Scale:
        <input type="number" name="scale" value="{{.Scale}}" step="0.5" min="0.25">
      </label>
      <button type="submit">Apply</button>
    </form>
  </div>
  {{end}}

  <section class="ingredients">
    {{range .Recipe.IngredientGroups}}
    {{if .Name}}<h2>{{.Name}}</h2>{{end}}
    <ul>
      {{range .Ingredients}}
      <li>
        {{if .Amount}}<span class="amount">{{.Amount.Display $.Units}}</span>{{end}}
        {{.Name}}
      </li>
      {{end}}
    </ul>
    {{end}}
  </section>

  <section class="instructions">
    {{.InstructionsHTML}}
  </section>
</article>
{{end}}
```

### AIP-160 Filtering

**Implementation approach**: Use [go.einride.tech/aip/filtering](https://pkg.go.dev/go.einride.tech/aip/filtering) for parsing, implement custom evaluator.

#### Supported Filter Fields

| Field | Type | Example |
|-------|------|---------|
| `title` | string | `title:"pasta"` |
| `tags` | repeated string | `tags:"vegan"` |
| `yields.quantity` | number | `yields.quantity >= 4` |
| `yields.unit` | string | `yields.unit:"servings"` |
| `ingredients.name` | string | `ingredients.name:"garlic"` |

#### Filter Examples

```
# Vegan recipes
tags:"vegan"

# Recipes with "chicken" in title
title:"chicken"

# Recipes serving 4 or more
yields.quantity >= 4

# Italian pasta dishes
tags:"italian" AND title:"pasta"

# Recipes containing garlic
ingredients.name:"garlic"
```

#### Implementation

```go
// FilterRecipes applies an AIP-160 filter to a recipe collection
func FilterRecipes(recipes []*Recipe, filter string) ([]*Recipe, error) {
    if filter == "" {
        return recipes, nil
    }

    // Parse filter using einride/aip
    expr, err := filtering.ParseFilter(filter)
    if err != nil {
        return nil, fmt.Errorf("invalid filter: %w", err)
    }

    // Evaluate against each recipe
    var results []*Recipe
    for _, r := range recipes {
        if evaluateFilter(expr, r) {
            results = append(results, r)
        }
    }
    return results, nil
}
```

### Unit Conversion

#### Phase 1: Display Toggle (Milestone 2)

Store original amounts, display in preferred unit system:

```go
type UnitSystem string

const (
    UnitSystemOriginal UnitSystem = "original"
    UnitSystemMetric   UnitSystem = "metric"
    UnitSystemImperial UnitSystem = "imperial"
)

// Display renders amount in the specified unit system
func (a *Amount) Display(system UnitSystem) string {
    if system == UnitSystemOriginal || a.Unit == "" {
        return a.OriginalString
    }
    // Convert and format
    converted := convertUnit(a.Quantity, a.Unit, system)
    return formatAmount(converted)
}
```

#### Phase 2: Intelligent Scaling (Future)

Upgrade amounts when scaling makes sense:

```go
// Example: 12 tablespoons → 3/4 cup
// Requires unit hierarchy:
//   teaspoon < tablespoon < fluid ounce < cup < pint < quart < gallon

type UnitHierarchy struct {
    Units []UnitDefinition
}

type UnitDefinition struct {
    Name       string   // "tablespoon"
    Aliases    []string // ["tbsp", "T", "Tbsp"]
    System     UnitSystem
    Category   string   // "volume", "weight", "count"
    BaseAmount float64  // In base units (e.g., mL for volume)
}
```

### MCP Server

Expose recipe functionality to LLM applications.

#### Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `list_recipes` | List available recipes | `filter?: string` |
| `get_recipe` | Get recipe details | `path: string, scale?: number` |
| `search_recipes` | Full-text search | `query: string` |
| `scale_recipe` | Get scaled ingredients | `path: string, scale: number` |

#### Implementation

```go
package mcp

import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewRecipeServer(store RecipeStore) *mcp.Server {
    server := mcp.NewServer("recipemd", "1.0.0")

    server.AddTool("list_recipes", listRecipesTool(store))
    server.AddTool("get_recipe", getRecipeTool(store))
    server.AddTool("scale_recipe", scaleRecipeTool(store))

    return server
}
```

### Package Structure Addition

```
recipemd-go/
├── ... (existing)
├── internal/
│   ├── web/
│   │   ├── server.go
│   │   ├── handlers.go
│   │   ├── templates.go
│   │   └── middleware.go
│   ├── filter/
│   │   ├── filter.go      # AIP-160 implementation
│   │   └── filter_test.go
│   └── units/
│       ├── convert.go     # Unit conversion
│       └── convert_test.go
├── mcp/
│   └── server.go          # MCP server implementation
└── web/
    ├── templates/
    └── static/
```

### Implementation Steps

#### 2.1 Web Server Foundation
- [ ] HTTP server setup with `net/http`
- [ ] Template loading and rendering
- [ ] Static file serving
- [ ] Request logging middleware
- [ ] Route definitions

#### 2.2 Recipe Listing
- [ ] Handler for `GET /recipes`
- [ ] Template for recipe list
- [ ] Pagination support

#### 2.3 Recipe View
- [ ] Handler for `GET /recipes/{path}`
- [ ] Goldmark rendering for description/instructions
- [ ] Template for single recipe

#### 2.4 Scaling UI
- [ ] Query parameter parsing for `scale`
- [ ] Amount scaling in display
- [ ] Form for scale input

#### 2.5 AIP-160 Filtering
- [ ] Integrate einride/aip filtering package
- [ ] Implement filter evaluator for Recipe type
- [ ] Filter form in UI
- [ ] Error handling for invalid filters

#### 2.6 Unit Switching
- [ ] Query parameter for `units`
- [ ] Cookie/preference storage
- [ ] Display conversion logic

#### 2.7 MCP Server
- [ ] Server setup with official SDK
- [ ] `list_recipes` tool
- [ ] `get_recipe` tool
- [ ] `scale_recipe` tool
- [ ] Transport configuration (stdio)

---

## Milestone 3: Performance & Storage

**Goal**: SQLite-based indexing, caching, meal planning, grocery lists.

### SQLite Schema

```sql
-- Recipe index (source of truth remains .md files)
CREATE TABLE recipes (
    id INTEGER PRIMARY KEY,
    path TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    instructions TEXT,
    modified_at DATETIME NOT NULL,
    indexed_at DATETIME NOT NULL
);

-- Tags (many-to-many)
CREATE TABLE recipe_tags (
    recipe_id INTEGER REFERENCES recipes(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (recipe_id, tag)
);
CREATE INDEX idx_recipe_tags_tag ON recipe_tags(tag);

-- Yields
CREATE TABLE recipe_yields (
    id INTEGER PRIMARY KEY,
    recipe_id INTEGER REFERENCES recipes(id) ON DELETE CASCADE,
    quantity REAL,
    unit TEXT,
    original_string TEXT NOT NULL
);

-- Ingredients (flattened for searching)
CREATE TABLE recipe_ingredients (
    id INTEGER PRIMARY KEY,
    recipe_id INTEGER REFERENCES recipes(id) ON DELETE CASCADE,
    group_name TEXT,
    quantity REAL,
    unit TEXT,
    name TEXT NOT NULL,
    original_string TEXT NOT NULL
);
CREATE INDEX idx_recipe_ingredients_name ON recipe_ingredients(name);

-- Full-text search
CREATE VIRTUAL TABLE recipes_fts USING fts5(
    title,
    description,
    instructions,
    content='recipes',
    content_rowid='id'
);
```

### Indexing Strategy

```go
// Indexer watches recipe directory and maintains SQLite index
type Indexer struct {
    db        *sql.DB
    recipesDir string
    parser    *parser.Parser
}

// Sync scans directory and updates index
// - Adds new recipes
// - Updates modified recipes (based on mtime)
// - Removes deleted recipes
func (idx *Indexer) Sync(ctx context.Context) error {
    // Walk directory
    // Compare mtimes with indexed_at
    // Update as needed
}
```

### File Watching (Optional)

```go
// For real-time updates, use fsnotify
import "github.com/fsnotify/fsnotify"

func (idx *Indexer) Watch(ctx context.Context) error {
    watcher, err := fsnotify.NewWatcher()
    // ...
}
```

### Meal Planning

Store meal plans as markdown files for git-friendliness:

```markdown
# Week of 2024-01-15

## Monday
- Dinner: [[recipes/pasta-carbonara.md]]

## Tuesday
- Lunch: [[recipes/salad-nicoise.md]]
- Dinner: [[recipes/chicken-stir-fry.md]]

## Wednesday
...
```

```sql
-- Or in SQLite for querying
CREATE TABLE meal_plans (
    id INTEGER PRIMARY KEY,
    date DATE NOT NULL,
    meal_type TEXT NOT NULL, -- breakfast, lunch, dinner, snack
    recipe_id INTEGER REFERENCES recipes(id),
    notes TEXT,
    UNIQUE(date, meal_type)
);
```

### Grocery Lists

Generated from meal plan, stored as markdown:

```markdown
# Grocery List - Week of 2024-01-15

## Produce
- [ ] 2 avocados
- [ ] 1 lb chicken breast
- [ ] Mixed greens

## Dairy
- [ ] 1 cup parmesan cheese
- [ ] 4 eggs

## Pantry
- [ ] Olive oil (have)
- [ ] Salt (have)
```

```go
// GenerateGroceryList aggregates ingredients from meal plan
func GenerateGroceryList(plan *MealPlan, recipes []*Recipe) *GroceryList {
    // Aggregate ingredients
    // Combine same ingredients (2 recipes need onions → combined amount)
    // Categorize by type (produce, dairy, etc.)
    // Check pantry inventory (future feature)
}
```

### Implementation Steps

#### 3.1 SQLite Setup
- [ ] Schema definition and migrations
- [ ] Connection pooling with `database/sql`
- [ ] WAL mode configuration

#### 3.2 Indexer
- [ ] Directory scanning
- [ ] Differential sync (only update changed files)
- [ ] Transaction handling

#### 3.3 Query Layer
- [ ] Recipe retrieval by path
- [ ] Filter translation to SQL
- [ ] Full-text search

#### 3.4 File Watching (Optional)
- [ ] fsnotify integration
- [ ] Debouncing for rapid changes

#### 3.5 Meal Planning
- [ ] Data model
- [ ] CRUD handlers
- [ ] Calendar UI

#### 3.6 Grocery Lists
- [ ] Ingredient aggregation
- [ ] Category assignment
- [ ] Markdown export

---

## Milestone 4: Enhanced Metadata

**Goal**: Track cooking history, ratings, git integration.

### Extended Metadata

```sql
-- Cooking history
CREATE TABLE cooking_log (
    id INTEGER PRIMARY KEY,
    recipe_id INTEGER REFERENCES recipes(id) ON DELETE CASCADE,
    cooked_at DATETIME NOT NULL,
    scale REAL DEFAULT 1.0,
    notes TEXT
);

-- Ratings
CREATE TABLE recipe_ratings (
    recipe_id INTEGER PRIMARY KEY REFERENCES recipes(id) ON DELETE CASCADE,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    updated_at DATETIME NOT NULL
);
```

### Git History Integration

```go
// RecipeHistory retrieves git history for a recipe file
func RecipeHistory(repoPath, recipePath string) ([]Commit, error) {
    // Use go-git or shell out to git
    // git log --oneline -- <path>
}

type Commit struct {
    Hash    string
    Message string
    Author  string
    Date    time.Time
}

// RecipeAtCommit retrieves recipe content at a specific commit
func RecipeAtCommit(repoPath, recipePath, commitHash string) ([]byte, error) {
    // git show <commit>:<path>
}
```

### Sorting Options

```
GET /recipes?sort=last_cooked       # Most recently cooked first
GET /recipes?sort=-last_cooked      # Least recently cooked first
GET /recipes?sort=rating            # Highest rated first
GET /recipes?sort=title             # Alphabetical
GET /recipes?sort=modified          # Recently modified first
```

### Implementation Steps

#### 4.1 Cooking Log
- [ ] Database schema
- [ ] "I made this" button/handler
- [ ] History view per recipe

#### 4.2 Ratings
- [ ] Star rating component
- [ ] Update handler
- [ ] Sort by rating

#### 4.3 Git Integration
- [ ] go-git dependency or git CLI wrapper
- [ ] History endpoint
- [ ] Diff view between versions

#### 4.4 Enhanced Sorting
- [ ] Sort parameter parsing
- [ ] SQL ORDER BY generation
- [ ] UI sort controls

---

## Milestone 5: Recipe Editing

**Goal**: Edit recipes in the web UI, attempted parsing for new recipes.

### Edit Flow

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   View Recipe   │────▶│   Edit Form     │────▶│  Parse Preview  │
│                 │     │   (Markdown)    │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                │                       │
                                │                       │
                                ▼                       ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │   Validation    │     │   Show Errors   │
                        │   (Parse OK?)   │────▶│   (if invalid)  │
                        └─────────────────┘     └─────────────────┘
                                │
                                │ (valid)
                                ▼
                        ┌─────────────────┐
                        │   Save to File  │
                        │   (Re-index)    │
                        └─────────────────┘
```

### Live Preview

JavaScript-free approach using form submissions:

```html
<form method="post" action="/recipes/edit">
  <textarea name="content">{{.Content}}</textarea>

  <div class="actions">
    <button type="submit" name="action" value="preview">Preview</button>
    <button type="submit" name="action" value="save">Save</button>
  </div>
</form>

{{if .Preview}}
<div class="preview">
  {{if .ParseError}}
    <div class="error">{{.ParseError}}</div>
  {{else}}
    <article class="recipe">
      <!-- Rendered preview -->
    </article>
  {{end}}
</div>
{{end}}
```

### Attempted Parsing

For pasted/imported content that doesn't follow RecipeMD format:

```go
// AttemptParse tries to extract recipe structure from non-standard markdown
func AttemptParse(content []byte) (*Recipe, []ParseWarning, error) {
    // Try standard parse first
    recipe, err := Parse(content)
    if err == nil {
        return recipe, nil, nil
    }

    // Fallback: heuristic extraction
    var warnings []ParseWarning

    // Find first heading as title
    // Find lists as potential ingredients
    // Everything else as instructions

    return recipe, warnings, nil
}

type ParseWarning struct {
    Line    int
    Message string
    Suggestion string
}
```

### Implementation Steps

#### 5.1 Edit Handler
- [ ] Edit form with current content
- [ ] POST handler for updates
- [ ] File writing with backup

#### 5.2 Validation
- [ ] Parse before save
- [ ] Error display with line numbers
- [ ] Suggestions for common issues

#### 5.3 Preview
- [ ] Server-side preview rendering
- [ ] Side-by-side view option

#### 5.4 Attempted Parsing
- [ ] Heuristic title extraction
- [ ] Ingredient list detection
- [ ] Warning generation

#### 5.5 New Recipe
- [ ] Create form
- [ ] Template/skeleton option
- [ ] File naming conventions

---

## Infrastructure

### Container Setup

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o recipemd ./cmd/recipemd

FROM alpine:latest
RUN apk add --no-cache sqlite-libs
COPY --from=builder /app/recipemd /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["recipemd"]
CMD ["serve"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  recipemd:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./recipes:/recipes:ro      # Recipe markdown files (read-only)
      - recipemd-data:/data        # SQLite database, meal plans, etc.
    environment:
      - RECIPEMD_RECIPES_DIR=/recipes
      - RECIPEMD_DATA_DIR=/data
      - RECIPEMD_PORT=8080

volumes:
  recipemd-data:
```

### Configuration

```go
type Config struct {
    RecipesDir string `env:"RECIPEMD_RECIPES_DIR" default:"./recipes"`
    DataDir    string `env:"RECIPEMD_DATA_DIR" default:"./data"`
    Port       int    `env:"RECIPEMD_PORT" default:"8080"`

    // Optional
    ReadOnly   bool   `env:"RECIPEMD_READ_ONLY" default:"false"`
    GitEnabled bool   `env:"RECIPEMD_GIT_ENABLED" default:"true"`
}
```

### Deployment Options

#### Option 1: Direct Binary

```bash
# Build for Raspberry Pi
GOOS=linux GOARCH=arm64 go build -o recipemd ./cmd/recipemd

# Run
./recipemd serve --recipes-dir=/path/to/recipes
```

#### Option 2: Docker Compose

```bash
docker-compose up -d
```

#### Option 3: Systemd Service

```ini
# /etc/systemd/system/recipemd.service
[Unit]
Description=RecipeMD Server
After=network.target

[Service]
Type=simple
User=recipemd
ExecStart=/usr/local/bin/recipemd serve
Environment=RECIPEMD_RECIPES_DIR=/home/recipemd/recipes
Environment=RECIPEMD_DATA_DIR=/var/lib/recipemd
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Reverse Proxy (Optional)

Not required for local network access. Consider adding if:
- You want HTTPS (even self-signed)
- Running multiple services on same machine
- Want automatic certificate management

Caddy example (if desired):

```
# Caddyfile
recipes.local {
    reverse_proxy localhost:8080
}
```

---

## Dependencies

### Required

| Package | Purpose | Version |
|---------|---------|---------|
| `github.com/yuin/goldmark` | Markdown parsing/rendering | v1.7+ |
| `modernc.org/sqlite` | Pure Go SQLite driver | Latest |

### Milestone 2

| Package | Purpose |
|---------|---------|
| `go.einride.tech/aip/filtering` | AIP-160 filter parsing |
| `github.com/modelcontextprotocol/go-sdk` | MCP server |

### Milestone 3 (Optional)

| Package | Purpose |
|---------|---------|
| `github.com/fsnotify/fsnotify` | File watching |

### Milestone 4 (Optional)

| Package | Purpose |
|---------|---------|
| `github.com/go-git/go-git/v5` | Git integration |

### Development

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Linting |
| `go test` | Testing |

---

## Future Considerations

### Intelligent Unit Conversion

Full implementation would require:

```go
var cookingUnits = UnitHierarchy{
    Units: []UnitDefinition{
        // Volume (US)
        {Name: "teaspoon", Aliases: []string{"tsp"}, System: Imperial, Category: "volume", BaseAmount: 4.92892},
        {Name: "tablespoon", Aliases: []string{"tbsp", "T"}, System: Imperial, Category: "volume", BaseAmount: 14.7868},
        {Name: "fluid ounce", Aliases: []string{"fl oz"}, System: Imperial, Category: "volume", BaseAmount: 29.5735},
        {Name: "cup", Aliases: []string{"c"}, System: Imperial, Category: "volume", BaseAmount: 236.588},
        // ... more units

        // Volume (Metric)
        {Name: "milliliter", Aliases: []string{"ml", "mL"}, System: Metric, Category: "volume", BaseAmount: 1.0},
        {Name: "liter", Aliases: []string{"l", "L"}, System: Metric, Category: "volume", BaseAmount: 1000.0},

        // Weight
        {Name: "gram", Aliases: []string{"g"}, System: Metric, Category: "weight", BaseAmount: 1.0},
        {Name: "kilogram", Aliases: []string{"kg"}, System: Metric, Category: "weight", BaseAmount: 1000.0},
        {Name: "ounce", Aliases: []string{"oz"}, System: Imperial, Category: "weight", BaseAmount: 28.3495},
        {Name: "pound", Aliases: []string{"lb"}, System: Imperial, Category: "weight", BaseAmount: 453.592},
    },
}
```

### Recipe Import

Potential sources:
- URL scraping with recipe schema detection
- Import from other formats (Paprika, Nextcloud Cookbook)
- OCR from images (stretch goal)

### Multi-user Support

If needed:
- User accounts
- Per-user ratings and cooking log
- Shared vs. private recipes

### Nutrition Information

- Integration with nutrition APIs
- Per-ingredient nutrition data
- Automatic calculation for recipes

---

## References

- [RecipeMD Specification](https://recipemd.org/specification.html)
- [RecipeMD CLI Documentation](https://recipemd.org/cli.html)
- [RecipeMD Python Implementation](https://github.com/RecipeMD/RecipeMD)
- [Goldmark Documentation](https://github.com/yuin/goldmark)
- [AIP-160: Filtering](https://google.aip.dev/160)
- [go.einride.tech/aip/filtering](https://pkg.go.dev/go.einride.tech/aip/filtering)
- [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk)
