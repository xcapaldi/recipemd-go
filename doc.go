// Package recipemd parses, scales, and renders recipes in the RecipeMD format.
//
// RecipeMD is a Markdown-based recipe format defined at https://recipemd.org.
// A recipe document begins with an H1 title, optional description paragraphs,
// optional italic tags and bold yields, a thematic break (---), and then an
// ingredient section made up of unordered lists and headings. An optional
// second thematic break separates free-form instructions.
//
// # Parsing
//
// Create a [Parser] with [NewParser] and call [Parser.Parse] to convert a
// RecipeMD document into a [Recipe]. Parse accepts any [io.Reader]:
//
//	p := recipemd.NewParser()
//	f, _ := os.Open("carbonara.md")
//	recipe, err := p.Parse(f)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(recipe.Title)
//
// Parse accumulates all structural and value-level problems via [errors.Join],
// so a single call can surface multiple [*ParseError] values at once.
//
// Options such as [WithFrontmatter] and [WithGithubFormattedMarkdown] can be
// passed to [NewParser] to handle YAML/TOML front matter and GitHub Flavored
// Markdown extensions respectively.
//
// # Scaling
//
// Recipes can be scaled by a numeric factor with [Recipe.Scale], or by a
// desired yield amount with [Recipe.ScaleForYield]:
//
//	// Double the recipe.
//	recipe.Scale(2)
//
//	// Scale to 8 servings.
//	desired, _ := recipemd.ParseAmountString("8 servings")
//	if err := recipe.ScaleForYield(desired); err != nil {
//	    log.Fatal(err)
//	}
//
// # Rendering
//
// A [Recipe] can be rendered back to RecipeMD markdown with
// [Recipe.RenderMarkdown], as compact JSON with [Recipe.RenderJSON], or as an
// HTML article element with [Recipe.RenderHTML]:
//
//	md := recipe.RenderMarkdown(2) // rounding to 2 decimal places
//	data, err := recipe.RenderJSON()
//	html := recipe.RenderHTML(2)
//
// [Recipe.RenderHTML] converts the raw markdown in the Description and
// Instructions fields with a plain CommonMark processor. Pass
// [WithGFMRendering] when the recipe was parsed with
// [WithGithubFormattedMarkdown], so that GFM constructs are preserved:
//
//	html := recipe.RenderHTML(2, recipemd.WithGFMRendering())
//
// Through v1.0.0 these were methods on [Parser], taking the recipe as an
// argument. Those methods still exist and still behave as they did, but they
// are deprecated and will be removed in v2:
//
//	md := p.RenderMarkdown(recipe, 2) // deprecated
package recipemd
