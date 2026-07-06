package recipemd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Option is a functional option for configuring a [Parser].
// Options are passed to [NewParser] and applied before the first parse.
type Option func(*Parser)

// WithFrontmatter returns an [Option] that instructs the parser to strip
// YAML (---) or TOML (+++) front matter from the source before parsing.
//
// Front matter is identified by a fence of three dashes or plus signs on its
// own line at the very start of the document. The content up to and including
// the closing fence is removed before the RecipeMD document is parsed.
func WithFrontmatter() Option { return func(p *Parser) { p.Frontmatter = true } }

// WithOKF returns an [Option] that instructs the parser to parse YAML (---)
// front matter as Google Open Knowledge Format (OKF) metadata, exposed via
// [Recipe.OKF].
//
// The OKF specification names one required field ("type") and five
// recommended fields ("title", "description", "resource", "tags",
// "timestamp"); any other frontmatter key is preserved in [OKF.Extensions].
// A document without frontmatter parses normally with a nil [Recipe.OKF].
// Frontmatter that is not valid YAML, or that omits the required "type"
// field, is reported as a parse error.
//
// WithOKF implies stripping the YAML frontmatter before the remaining
// content is parsed as RecipeMD, so it does not need to be combined with
// [WithFrontmatter] (though it may be, e.g. to also tolerate TOML front
// matter).
//
// See https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
func WithOKF() Option { return func(p *Parser) { p.okf = true } }

// WithGithubFormattedMarkdown returns an [Option] that enables GitHub Flavored
// Markdown (GFM) extensions in the underlying markdown processor.
//
// This adds support for tables, strikethrough, autolinks, task lists, and
// other GFM features. Task-list checkboxes in ingredient items are
// transparently skipped so that ingredient parsing is unaffected.
func WithGithubFormattedMarkdown() Option {
	return func(p *Parser) {
		p.goldmarkExtensions = append(p.goldmarkExtensions, extension.GFM)
		p.hasTaskList = true
	}
}

// Parser parses RecipeMD documents and renders [Recipe] values back to
// markdown or JSON.
//
// Create a Parser with [NewParser]. A single Parser instance is safe to reuse
// across multiple calls to [Parser.Parse] and the render methods.
//
// The exported Frontmatter field reflects whether the [WithFrontmatter] option
// was supplied at construction time. It should not be modified after the
// Parser is created.
type Parser struct {
	// Frontmatter reports whether the parser strips YAML/TOML front matter
	// before parsing. Set via [WithFrontmatter].
	Frontmatter        bool
	okf                bool
	hasTaskList        bool
	goldmarkProcessor  goldmark.Markdown
	goldmarkExtensions []goldmark.Extender
}

// NewParser creates a new Parser, applying any supplied options.
//
// Available options are [WithFrontmatter] and [WithGithubFormattedMarkdown].
// If no options are provided a plain CommonMark parser is used.
//
//	p := recipemd.NewParser(recipemd.WithFrontmatter())
func NewParser(opts ...Option) (p *Parser) {
	p = &Parser{}
	for _, o := range opts {
		o(p)
	}

	if len(p.goldmarkExtensions) > 0 {
		p.goldmarkProcessor = goldmark.New(goldmark.WithExtensions(p.goldmarkExtensions...))
		return
	}

	p.goldmarkProcessor = goldmark.New()
	return
}

// Parse converts a RecipeMD document into a [Recipe].
//
// r is an [io.Reader] providing a UTF-8-encoded RecipeMD document. The
// document structure that Parse expects is:
//
//  1. An H1 heading containing the recipe title (required).
//  2. An optional preamble: description paragraphs, an italic tags paragraph,
//     and/or a bold yields paragraph, in any order.
//  3. A thematic break (---) separating the preamble from the ingredients.
//  4. An ingredient section: unordered lists of ingredients and optional
//     sub-headings that introduce named [IngredientGroup] sections.
//  5. An optional second thematic break followed by free-form instructions.
//
// Parse collects all structural and value-level errors via [errors.Join],
// returning a non-nil error that may wrap one or more [*ParseError] values.
// Non-fatal errors are accumulated rather than halting the parse, so all
// problems are reported at once. A nil error means the document was valid.
//
// See https://recipemd.org/specification.html#parsing-a-recipe for the full
// specification.
func (p *Parser) Parse(r io.Reader) (*Recipe, error) {
	source, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}

	var okfMeta *OKF
	var okfErr error
	if p.okf {
		if fence, frontmatter, rest, ok := splitFrontmatter(source); ok && fence == yamlFence {
			okfMeta, okfErr = ParseOKF(frontmatter)
			source = rest
		}
	}
	if p.Frontmatter {
		source = stripFrontmatter(source)
	}

	document := p.goldmarkProcessor.Parser().Parse(text.NewReader(source))

	recipe := &Recipe{
		OKF:              okfMeta,
		Yields:           []Amount{},
		Tags:             []string{},
		Ingredients:      []Ingredient{},
		IngredientGroups: []IngredientGroup{},
	}

	var errs error
	if okfErr != nil {
		errs = errors.Join(errs, &ParseError{Message: okfErr.Error(), Offset: 0, Line: 1, Column: 1})
	}

	c := document.FirstChild()
	if c == nil {
		return nil, newParseError(source, 0, "recipe must have a title")
	}

	// --- Preamble: title ---
	h, ok := c.(*ast.Heading)
	if !ok {
		return nil, newParseError(source, nodeStartOffset(c), fmt.Sprintf("expected level 1 heading, got %T", c))
	}
	if h.Level != 1 {
		errs = errors.Join(errs, newParseError(source, nodeStartOffset(h), fmt.Sprintf("expected level 1 heading, got level %d", h.Level)))
	}
	title, err := extractPlainText(h, source)
	if err != nil {
		return nil, newParseError(source, nodeStartOffset(h), err.Error())
	}
	recipe.Title = title

	_, titleLineEnd := getDirectLineBounds(h)
	descStart := skipSetextUnderline(source, titleLineEnd)
	c = c.NextSibling()

	// --- Preamble: description, tags, yields ---
	var excludeRanges [][2]int
	tagsFound, yieldsFound, tagsYieldsMode := false, false, false

	lastPreBreakEnd := descStart
	for c != nil && c.Kind() != ast.KindThematicBreak {
		para, isPara := c.(*ast.Paragraph)
		if isPara {
			if em, ok := isOnlyEmphasis(para, italic); ok {
				if tagsFound {
					errs = errors.Join(errs, newParseError(source, nodeStartOffset(c), "tags already set"))
					c = c.NextSibling()
					continue
				}
				tagsText, err := extractPlainText(em, source)
				if err != nil {
					return nil, newParseError(source, nodeStartOffset(c), err.Error())
				}
				recipe.Tags = parseTags(tagsText)
				tagsFound = true
				tagsYieldsMode = true
				if start, end := getDirectLineBounds(c); start >= 0 {
					excludeRanges = append(excludeRanges, [2]int{start, end})
				}
				c = c.NextSibling()
				continue
			}
			if em, ok := isOnlyEmphasis(para, bold); ok {
				if yieldsFound {
					errs = errors.Join(errs, newParseError(source, nodeStartOffset(c), "yields already set"))
					c = c.NextSibling()
					continue
				}
				yieldsText, err := extractPlainText(em, source)
				if err != nil {
					return nil, newParseError(source, nodeStartOffset(c), err.Error())
				}
				yields, yieldErrs := parseYields(yieldsText)
				if yieldErrs != nil {
					errs = errors.Join(errs, newParseError(source, nodeStartOffset(c), yieldErrs.Error()))
				}
				recipe.Yields = yields
				yieldsFound = true
				tagsYieldsMode = true
				if start, end := getDirectLineBounds(c); start >= 0 {
					excludeRanges = append(excludeRanges, [2]int{start, end})
				}
				c = c.NextSibling()
				continue
			}
			if tagsYieldsMode {
				errs = errors.Join(errs, newParseError(source, nodeStartOffset(c), "unexpected content in tags/yields section"))
				c = c.NextSibling()
				continue
			}
		}
		if _, end := getRecursiveSourceBounds(c, source); end > lastPreBreakEnd {
			lastPreBreakEnd = end
		}
		c = c.NextSibling()
	}

	// --- First thematic break ---
	if c == nil || c.Kind() != ast.KindThematicBreak {
		return nil, newParseError(source, lastPreBreakEnd, "missing thematic break divider")
	}
	firstBreakPos := findDashLine(source, lastPreBreakEnd)

	// Build description: source from after title to first break, minus tags/yields.
	if firstBreakPos > descStart {
		desc := excludeRangesFromSource(source[descStart:firstBreakPos], excludeRanges, descStart)
		desc = strings.Trim(desc, "\n")
		if desc != "" {
			recipe.Description = &desc
		}
	}

	c = c.NextSibling()

	// --- Ingredients ---
	for c != nil {
		if para, ok := c.(*ast.Paragraph); ok {
			errs = errors.Join(errs, newParseError(source, nodeStartOffset(para), "paragraph not valid in ingredients section"))
			c = c.NextSibling()
			continue
		}
		break
	}
	var listErrs, groupErrs error
	c, listErrs = parseIngredientList(c, source, &recipe.Ingredients, p.hasTaskList)
	c, groupErrs = parseIngredientGroup(c, source, &recipe.IngredientGroups, 0, p.hasTaskList)
	errs = errors.Join(errs, listErrs, groupErrs)

	// --- Second thematic break (optional) → instructions ---
	if c != nil && c.Kind() == ast.KindThematicBreak {
		breakPos := findDashLine(source, firstBreakPos+1)
		breakEnd := skipLine(source, breakPos)
		instructions := strings.Trim(string(source[breakEnd:]), "\n")
		if instructions != "" {
			recipe.Instructions = &instructions
		}
	}

	if errs != nil {
		return nil, errs
	}
	return recipe, nil
}

// findDashLine finds the byte offset of the first line of 3+ dashes at or
// after minPos, aligning to line boundaries.
func findDashLine(source []byte, minPos int) int {
	pos := minPos
	if pos > 0 && pos < len(source) && source[pos-1] != '\n' {
		for pos < len(source) && source[pos] != '\n' {
			pos++
		}
		if pos < len(source) {
			pos++
		}
	}
	for pos < len(source) {
		lineStart := pos
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		line := bytes.TrimSpace(source[lineStart:lineEnd])
		if len(line) >= 3 && len(bytes.Trim(line, "-")) == 0 {
			return lineStart
		}
		pos = lineEnd + 1
	}
	return -1
}

// skipLine returns the byte offset just past the newline at pos.
func skipLine(source []byte, pos int) int {
	if pos < 0 {
		return len(source)
	}
	for pos < len(source) && source[pos] != '\n' {
		pos++
	}
	if pos < len(source) {
		pos++
	}
	return pos
}

// parseIngredientGroup parses headings and lists in the ingredient section.
// It accepts a block c, modifies groups to append ingredient groups found,
// and returns the current block.
// See: https://recipemd.org/specification.html#parsing-ingredient-groups
func parseIngredientGroup(
	c ast.Node,
	source []byte,
	groups *[]IngredientGroup,
	parentLevel int,
	skipCheckbox bool,
) (ast.Node, error) {
	var errs error
	for {
		h, ok := c.(*ast.Heading)
		if !ok {
			return c, errs
		}
		l := h.Level
		if l <= parentLevel {
			return c, errs
		}
		title, err := extractPlainText(h, source)
		if err != nil {
			return nil, errors.Join(errs, newParseError(source, nodeStartOffset(h), err.Error()))
		}
		g := IngredientGroup{
			Title:            title,
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
		}
		c = c.NextSibling()
		var listErrs, groupErrs error
		if c != nil {
			c, listErrs = parseIngredientList(c, source, &g.Ingredients, skipCheckbox)
			c, groupErrs = parseIngredientGroup(c, source, &g.IngredientGroups, l, skipCheckbox)
		}
		errs = errors.Join(errs, listErrs, groupErrs)
		*groups = append(*groups, g)
		if c == nil {
			return nil, errs
		}
	}
}

// parseIngredientList parses a list of ingredients.
// It accepts a block c, modifies ingredients to append ingredients found,
// and returns the current block.
// See: https://recipemd.org/specification.html#parsing-an-ingredient-list
func parseIngredientList(
	c ast.Node,
	source []byte,
	ingredients *[]Ingredient,
	skipCheckbox bool,
) (ast.Node, error) {
	var errs error
	for {
		// 1. Examine c
		list, ok := c.(*ast.List)
		if !ok {
			return c, errs
		}
		// Enter c
		c = list.FirstChild()
		if c == nil {
			c = list.NextSibling()
			continue
		}

		// 2. Collect ingredients
		for {
			ing, err := parseIngredient(c, source, skipCheckbox)
			if err != nil {
				offset, _ := getRecursiveSourceBounds(c, source)
				if offset < 0 {
					offset = 0
				}
				errs = errors.Join(errs, newParseError(source, offset, err.Error()))
			} else {
				*ingredients = append(*ingredients, ing)
			}
			if c.NextSibling() != nil {
				c = c.NextSibling()
			} else {
				c = list.NextSibling()
				break
			}
		}
	}
}

// parseIngredient parses a block c into an ingredient.
// See: https://recipemd.org/specification.html#parsing-an-ingredient
func parseIngredient(c ast.Node, source []byte, skipCheckbox bool) (Ingredient, error) {
	// 1. Examine c: If c is a list item, enter c
	li, ok := c.(*ast.ListItem)
	if !ok {
		return Ingredient{}, fmt.Errorf("expected list item")
	}
	c = li.FirstChild()

	// 2. Let a be the amount, set to unset
	var a *Amount
	// 3. Let n be the name, set to empty string
	n := ""
	// 4. Let l be a link, set to unset
	var l *string

	if c == nil {
		return Ingredient{}, fmt.Errorf("ingredient must have a name")
	}

	// 5. Examine c
	// Note: goldmark uses TextBlock for tight lists, Paragraph for loose lists
	var firstInline ast.Node
	if para, ok := c.(*ast.Paragraph); ok {
		firstInline = para.FirstChild()
	} else if tb, ok := c.(*ast.TextBlock); ok {
		firstInline = tb.FirstChild()
	}
	if skipCheckbox && firstInline != nil && firstInline.Kind() == east.KindTaskCheckBox {
		firstInline = firstInline.NextSibling()
		// skip whitespace text node after checkbox
		if t, ok := firstInline.(*ast.Text); ok && strings.TrimSpace(string(t.Value(source))) == "" {
			firstInline = firstInline.NextSibling()
		}
	}
	if firstInline == nil {
		// If c is not a paragraph, set n to verbatim contents of c
		n = extractRawMarkdown(c, source)
	} else {
		// Parse the amount
		var r string
		afterAmount := firstInline

		if em, ok := firstInline.(*ast.Emphasis); ok && em.Level == 1 {
			// If c's contents start with an emphasis inline
			emText, err := extractPlainText(em, source)
			if err != nil {
				return Ingredient{}, err
			}
			amt, err := parseAmount(emText)
			if err != nil {
				return Ingredient{}, err
			}
			a = &amt
			// Let r be the remaining contents of c after the emphasis
			afterAmount = em.NextSibling()
			r = extractInlineSequenceText(afterAmount, source)
		} else {
			// Let r be the verbatim contents of c
			r = extractRawMarkdown(c, source)
		}

		// Parse the link
		isOnlyChild := c.NextSibling() == nil
		link := findSingleLink(afterAmount, source)

		if isOnlyChild && link != nil {
			dest := encodeURLPath(link.destination)
			l = &dest
			n = link.text
		} else {
			n = r
		}
	}

	// 6. Parse the following blocks of the list item
	prevBlock := c
	for c.NextSibling() != nil {
		c = c.NextSibling()
		// Append c's verbatim contents to n, preserving blank lines
		sep := getBlockSeparator(prevBlock, c, source)
		n += sep + extractRawMarkdown(c, source)
		prevBlock = c
	}

	// 7. Leave c (implicit)

	// 8. Let i be an ingredient with amount a, name n, link l
	n = strings.TrimSpace(n)
	if n == "" {
		return Ingredient{}, fmt.Errorf("ingredient must have a name")
	}

	return Ingredient{Amount: a, Name: n, Link: l}, nil
}

// extractInlineSequenceText extracts text from a sequence of sibling inline nodes,
// preserving some markdown syntax (emphasis markers, link syntax).
func extractInlineSequenceText(start ast.Node, source []byte) string {
	var parts []string
	for n := start; n != nil; n = n.NextSibling() {
		parts = append(parts, convertInlineNodeToText(n, source))
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}

// convertInlineNodeToText converts a single inline node to text,
// preserving markdown syntax for emphasis and links.
func convertInlineNodeToText(n ast.Node, source []byte) string {
	if t, ok := n.(*ast.Text); ok {
		return string(t.Value(source))
	}
	if al, ok := n.(*ast.AutoLink); ok {
		return string(al.URL(source))
	}
	text, _ := extractPlainText(n, source)
	if n.Kind() == ast.KindEmphasis {
		return "*" + text + "*"
	}
	if link, ok := n.(*ast.Link); ok {
		return "[" + text + "](" + string(link.Destination) + ")"
	}
	return text
}

const listItemContinuationIndent = "  "

// getBlockSeparator returns the whitespace between two blocks, preserving blank lines
func getBlockSeparator(prev, curr ast.Node, source []byte) string {
	prevLines := prev.Lines()
	currLines := curr.Lines()
	if prevLines.Len() == 0 || currLines.Len() == 0 {
		return "\n"
	}
	prevEnd := prevLines.At(prevLines.Len() - 1).Stop
	currStart := currLines.At(0).Start
	between := source[prevEnd:currStart]
	// Check for blank line (more than one newline)
	newlineCount := bytes.Count(between, []byte{'\n'})
	if newlineCount <= 1 {
		return "\n"
	}
	// Extract blank line content (everything between first and last newline).
	// Trailing 2-space indent matches markdown list item continuation.
	if _, rest, ok := bytes.Cut(between, []byte{'\n'}); ok {
		if blankContent, _, ok := bytes.Cut(rest, []byte{'\n'}); ok {
			return "\n" + string(blankContent) + "\n" + listItemContinuationIndent
		}
	}
	return "\n\n" + listItemContinuationIndent
}

type linkInfo struct {
	destination string
	text        string
}

// findSingleLink checks if nodes from start consist only of whitespace and a
// single link (explicit or autolink). Returns nil if no link or multiple links.
func findSingleLink(start ast.Node, source []byte) *linkInfo {
	var found *linkInfo
	for n := start; n != nil; n = n.NextSibling() {
		if l, ok := n.(*ast.Link); ok {
			if found != nil {
				return nil
			}
			text, _ := extractPlainText(l, source)
			found = &linkInfo{destination: string(l.Destination), text: text}
		} else if al, ok := n.(*ast.AutoLink); ok {
			if found != nil {
				return nil
			}
			url := string(al.URL(source))
			found = &linkInfo{destination: url, text: url}
		} else if t, ok := n.(*ast.Text); ok {
			if strings.TrimSpace(string(t.Value(source))) != "" {
				return nil
			}
		} else {
			return nil
		}
	}
	return found
}

// ParseAmountString parses a human-readable amount string into an [Amount].
//
// The following number formats are recognised (case-insensitive):
//   - Mixed number:       "1 1/2" or "1 ½"
//   - Proper fraction:    "1/2"
//   - Vulgar fraction:    "½", "¾", etc. (Unicode fraction characters)
//   - Decimal:            "1.5" or "1,5"
//   - Integer:            "3"
//
// An optional sign (- for negative) may precede the number. Any non-numeric
// text following the number is interpreted as the unit (e.g. "1.5 cups" →
// Factor=1.5, Unit="cups").
//
// ParseAmountString returns an error if a unit is present without a numeric
// value, or if the input cannot be parsed as any recognised format.
func ParseAmountString(s string) (Amount, error) {
	return parseAmount(s)
}

// parseAmount parses an amount string into value and unit.
// See: https://recipemd.org/specification.html#parsing-an-amount
func parseAmount(s string) (Amount, error) {
	// 1. Trim whitespace at beginning
	s = strings.TrimLeftFunc(s, unicode.IsSpace)

	// 2. Check for negative
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
		s = strings.TrimLeftFunc(s, unicode.IsSpace)
	}

	// 3. Let v be a number, set to unset
	var v *float64
	var remaining string

	// Try improper fraction: a w+ b w* / w* c
	v, remaining = parseImproperFraction(s)
	// Try improper with vulgar: a w+ b (vulgar)
	if v == nil {
		v, remaining = parseImproperVulgar(s)
	}
	// Try proper fraction: a w* / w* b
	if v == nil {
		v, remaining = parseProperFraction(s)
	}
	// Try vulgar fraction alone
	if v == nil {
		v, remaining = parseVulgarAlone(s)
	}
	// Try decimal: a [.,] b
	if v == nil {
		v, remaining = parseDecimalNumber(s)
	}
	// Try integer
	if v == nil {
		v, remaining = parseIntegerNumber(s)
	}

	// 4. Let u be remainder, stripped of whitespace
	u := strings.TrimSpace(remaining)
	var unit *string
	if u != "" {
		unit = &u
	}

	// 5. Return result
	if v != nil {
		val := *v
		if negative {
			val = -val
		}
		return Amount{Factor: val, Unit: unit}, nil
	} else if unit != nil {
		return Amount{}, fmt.Errorf("unit without value: %q", s)
	}
	return Amount{}, nil
}

// parseImproperFraction parses "a b/c" (e.g., "1 1/2")
func parseImproperFraction(s string) (*float64, string) {
	runes := []rune(s)
	// Match: integer, whitespace+, integer, whitespace*, /, whitespace*, integer
	i := 0
	// Parse whole part
	start := i
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	if i == start {
		return nil, s
	}
	whole := mustParseFloat(string(runes[start:i]))

	// Need at least one whitespace
	if i >= len(runes) || !unicode.IsSpace(runes[i]) {
		return nil, s
	}
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}

	// Parse numerator
	start = i
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	if i == start {
		return nil, s
	}
	num := mustParseFloat(string(runes[start:i]))

	// Skip whitespace, expect /
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	if i >= len(runes) || runes[i] != '/' {
		return nil, s
	}
	i++ // skip /
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}

	// Parse denominator
	start = i
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	if i == start {
		return nil, s
	}
	denom := mustParseFloat(string(runes[start:i]))

	if denom == 0 {
		return nil, s
	}
	v := whole + num/denom
	return &v, string(runes[i:])
}

// parseImproperVulgar parses "a b" where b is a vulgar fraction (e.g., "1 ½")
func parseImproperVulgar(s string) (*float64, string) {
	runes := []rune(s)
	i := 0
	// Parse whole part
	start := i
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	if i == start {
		return nil, s
	}
	whole := mustParseFloat(string(runes[start:i]))

	// Need at least one whitespace
	if i >= len(runes) || !unicode.IsSpace(runes[i]) {
		return nil, s
	}
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}

	// Check for vulgar fraction
	if i >= len(runes) {
		return nil, s
	}
	frac, ok := vulgarFractionMap[runes[i]]
	if !ok {
		return nil, s
	}
	i++

	v := whole + frac
	return &v, string(runes[i:])
}

// parseProperFraction parses "a/b" (e.g., "1/2")
func parseProperFraction(s string) (*float64, string) {
	runes := []rune(s)
	i := 0
	// Parse numerator
	start := i
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	if i == start {
		return nil, s
	}
	num := mustParseFloat(string(runes[start:i]))

	// Skip whitespace, expect /
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	if i >= len(runes) || runes[i] != '/' {
		return nil, s
	}
	i++ // skip /
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}

	// Parse denominator
	start = i
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	if i == start {
		return nil, s
	}
	denom := mustParseFloat(string(runes[start:i]))

	if denom == 0 {
		return nil, s
	}
	v := num / denom
	return &v, string(runes[i:])
}

// parseVulgarAlone parses a single vulgar fraction (e.g., "½")
func parseVulgarAlone(s string) (*float64, string) {
	runes := []rune(s)
	if len(runes) == 0 {
		return nil, s
	}
	frac, ok := vulgarFractionMap[runes[0]]
	if !ok {
		return nil, s
	}
	return &frac, string(runes[1:])
}

// parseDecimalNumber parses "a.b", "a,b", or ".b" (e.g., "1.5" or ".5")
func parseDecimalNumber(s string) (*float64, string) {
	runes := []rune(s)
	i := 0
	start := i

	// Parse optional integer part
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}

	// Need decimal point
	if i >= len(runes) || (runes[i] != '.' && runes[i] != ',') {
		return nil, s
	}
	i++ // skip decimal point

	// Parse fractional part (required)
	fracStart := i
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	if i == fracStart {
		return nil, s
	}

	numStr := string(runes[start:i])
	numStr = strings.Replace(numStr, ",", ".", 1)
	v := mustParseFloat(numStr)
	return &v, string(runes[i:])
}

// parseIntegerNumber parses an integer (e.g., "5")
func parseIntegerNumber(s string) (*float64, string) {
	runes := []rune(s)
	i := 0
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	if i == 0 {
		return nil, s
	}
	v := mustParseFloat(string(runes[:i]))
	return &v, string(runes[i:])
}

// mustParseFloat parses a string to float64, panicking on error.
// Only use when input is pre-validated to contain valid numeric characters.
func mustParseFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(fmt.Sprintf("mustParseFloat: invalid input %q: %v", s, err))
	}
	return v
}

// encodeURLPath properly encodes special characters in a URL path.
func encodeURLPath(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	return u.String()
}

// getDirectLineBounds returns the byte range from a node's own Lines() property.
// Does not recurse into children. Returns (-1, -1) if node has no lines.
func getDirectLineBounds(n ast.Node) (start, end int) {
	lines := n.Lines()
	if lines.Len() == 0 {
		return -1, -1
	}
	return lines.At(0).Start, lines.At(lines.Len() - 1).Stop
}

// extractPlainText recursively extracts plain text from a node, stripping all markdown.
func extractPlainText(node ast.Node, source []byte) (string, error) {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			if buf.Len() > 0 {
				if _, err := buf.WriteRune(' '); err != nil {
					return "", fmt.Errorf("buf.WriteRune: %w", err)
				}
			}
			if _, err := buf.Write(bytes.TrimSpace(text.Value(source))); err != nil {
				return "", fmt.Errorf("buf.Write: %w", err)
			}
		} else {
			childText, err := extractPlainText(child, source)
			if err != nil {
				return "", err
			}
			buf.WriteString(childText)
		}
	}
	return buf.String(), nil
}

// extractRawMarkdown extracts raw source for a block node, preserving markdown syntax.
func extractRawMarkdown(node ast.Node, source []byte) string {
	start, end := getRecursiveSourceBounds(node, source)
	if start < 0 {
		return ""
	}
	return strings.TrimRight(string(source[start:end]), "\n")
}

// getRecursiveSourceBounds returns the byte range for a node's source,
// recursively including all children. Handles list markers specially.
func getRecursiveSourceBounds(node ast.Node, source []byte) (start, end int) {
	if node.Type() == ast.TypeBlock {
		if lines := node.Lines(); lines.Len() > 0 {
			return lines.At(0).Start, lines.At(lines.Len() - 1).Stop
		}
	}
	start, end = -1, -1
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		childStart, childEnd := getRecursiveSourceBounds(child, source)
		if childStart < 0 {
			continue
		}
		if node.Kind() == ast.KindList || node.Kind() == ast.KindListItem {
			for childStart > 0 && source[childStart-1] != '\n' {
				childStart--
			}
		}
		if start < 0 || childStart < start {
			start = childStart
		}
		if childEnd > end {
			end = childEnd
		}
	}
	return start, end
}

// offsetToLineCol converts a 0-based byte offset into 1-based line and column numbers.
func offsetToLineCol(source []byte, offset int) (line, col int) {
	line, col = 1, 1
	for i := 0; i < offset && i < len(source); i++ {
		if source[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return
}

// newParseError constructs a *ParseError from a byte offset in source and a message.
func newParseError(source []byte, offset int, msg string) *ParseError {
	line, col := offsetToLineCol(source, offset)
	return &ParseError{Message: msg, Offset: offset, Line: line, Column: col}
}

// nodeStartOffset returns the 0-based byte offset of the first byte of an AST node.
// Returns 0 if the node has no line info.
func nodeStartOffset(n ast.Node) int {
	lines := n.Lines()
	if lines.Len() > 0 {
		return lines.At(0).Start
	}
	return 0
}

type emphasisLevel int

const (
	italic emphasisLevel = iota + 1
	bold
)

// isOnlyEmphasis returns the emphasis if paragraph has exactly one Emphasis
// child with given level (*italics* and **bold**)
func isOnlyEmphasis(p *ast.Paragraph, level emphasisLevel) (*ast.Emphasis, bool) {
	first := p.FirstChild()
	if first == nil || first.NextSibling() != nil || first.Kind() != ast.KindEmphasis {
		return nil, false
	}
	em := first.(*ast.Emphasis)
	if em.Level != int(level) {
		return nil, false
	}
	return em, true
}

// splitList splits on commas but not decimal commas (digit,digit).
func splitList(list string) []string {
	parts := make([]string, 0, strings.Count(list, ",")+1)
	var start, search int
	for {
		index := strings.IndexByte(list[search:], ',')
		if index == -1 {
			break
		}
		position := search + index
		if position > 0 &&
			position < len(list)-1 &&
			unicode.IsDigit(rune(list[position-1])) &&
			unicode.IsDigit(rune(list[position+1])) {
			search = position + 1
			continue
		}
		if t := strings.TrimSpace(list[start:position]); t != "" {
			parts = append(parts, t)
		}
		start = position + 1
		search = start
	}
	if t := strings.TrimSpace(list[start:]); t != "" {
		parts = append(parts, t)
	}
	return parts
}

func parseTags(s string) []string {
	return splitList(s)
}

func parseYields(s string) (yields []Amount, errs error) {
	for _, yield := range splitList(s) {
		amount, err := parseAmount(yield)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		yields = append(yields, amount)
	}
	return yields, errs
}

var vulgarFractionMap = map[rune]float64{
	'¼': 1.0 / 4, '½': 1.0 / 2, '¾': 3.0 / 4,
	'⅐': 1.0 / 7, '⅑': 1.0 / 9, '⅒': 1.0 / 10,
	'⅓': 1.0 / 3, '⅔': 2.0 / 3,
	'⅕': 1.0 / 5, '⅖': 2.0 / 5, '⅗': 3.0 / 5, '⅘': 4.0 / 5,
	'⅙': 1.0 / 6, '⅚': 5.0 / 6,
	'⅛': 1.0 / 8, '⅜': 3.0 / 8, '⅝': 5.0 / 8, '⅞': 7.0 / 8,
}

func skipSetextUnderline(source []byte, pos int) int {
	if pos >= len(source) || source[pos] != '\n' {
		return pos
	}
	next := pos + 1
	if next >= len(source) || source[next] != '=' {
		return pos
	}
	for next < len(source) && source[next] != '\n' {
		next++
	}
	return next
}

// Frontmatter fence styles recognised by splitFrontmatter.
const (
	yamlFence = "---"
	tomlFence = "+++"
)

// stripFrontmatter removes YAML (---) or TOML (+++) frontmatter from the
// beginning of source. Returns source unchanged if no frontmatter is found.
func stripFrontmatter(source []byte) []byte {
	if _, _, rest, ok := splitFrontmatter(source); ok {
		return rest
	}
	return source
}

// splitFrontmatter splits YAML (---) or TOML (+++) frontmatter from the
// beginning of source. It returns the fence style, the frontmatter content
// between the fences, and the remaining source after the closing fence.
// found is false when source does not start with a complete frontmatter block.
func splitFrontmatter(source []byte) (fence string, frontmatter, rest []byte, found bool) {
	if len(source) < 3 {
		return "", nil, nil, false
	}
	if bytes.HasPrefix(source, []byte(yamlFence)) {
		fence = yamlFence
	} else if bytes.HasPrefix(source, []byte(tomlFence)) {
		fence = tomlFence
	} else {
		return "", nil, nil, false
	}

	// Opening fence must be alone on the line (optional trailing whitespace)
	firstNL := bytes.IndexByte(source, '\n')
	if firstNL < 0 {
		return "", nil, nil, false
	}
	if len(bytes.TrimSpace(source[:firstNL])) != len(fence) {
		return "", nil, nil, false
	}

	// Find closing fence
	body := source[firstNL+1:]
	pos := 0
	for pos < len(body) {
		lineEnd := bytes.IndexByte(body[pos:], '\n')
		var line []byte
		if lineEnd < 0 {
			line = body[pos:]
		} else {
			line = body[pos : pos+lineEnd]
		}
		if string(bytes.TrimSpace(line)) == fence {
			if lineEnd < 0 {
				return fence, body[:pos], nil, true
			}
			return fence, body[:pos], body[pos+lineEnd+1:], true
		}
		if lineEnd < 0 {
			break
		}
		pos += lineEnd + 1
	}
	return "", nil, nil, false
}

func excludeRangesFromSource(src []byte, ranges [][2]int, offset int) string {
	if len(ranges) == 0 {
		return string(src)
	}
	var result strings.Builder
	pos := 0
	for _, r := range ranges {
		start := r[0] - offset
		end := r[1] - offset
		if start < 0 || end > len(src) {
			continue
		}
		if start > pos {
			result.Write(src[pos:start])
		}
		pos = end
	}
	if pos < len(src) {
		result.Write(src[pos:])
	}
	return result.String()
}
