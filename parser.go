package recipemd

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type Option func(*Parser)

func WithFrontmatter() Option   { return func(p *Parser) { p.Frontmatter = true } }
func WithGithubFormattedMarkdown() Option { return func(p *Parser) {
	p.goldmarkExtensions = append(p.goldmarkExtensions, extension.GFM)
	p.hasTaskList = true
} }

type Parser struct {
	Frontmatter    bool
	hasTaskList    bool
	goldmarkProcessor  goldmark.Markdown
	goldmarkExtensions []goldmark.Extender
}

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

// Flatten resolves linked ingredients by parsing referenced recipe files
// and inlining their ingredients. Links resolved relative to recipeFile dir.
func (p *Parser) Flatten(r *Recipe, recipeFile string) error {
	baseDir := filepath.Dir(recipeFile)
	ingredients, err := p.flattenIngredients(r.Ingredients, baseDir)
	if err != nil {
		return fmt.Errorf("flattenIngredients: %w", err)
	}
	r.Ingredients = ingredients
	groups, err := p.flattenIngredientGroups(r.IngredientGroups, baseDir)
	if err != nil {
		return fmt.Errorf("flattenIngredientGroups: %w", err)
	}
	r.IngredientGroups = groups
	return nil
}

func (p *Parser) flattenIngredients(ingredients []Ingredient, baseDir string) ([]Ingredient, error) {
	result := make([]Ingredient, 0, len(ingredients))
	for _, ing := range ingredients {
		if ing.Link != nil {
			resolved, err := p.resolveLinkedRecipe(*ing.Link, baseDir, &ing)
			if err != nil {
				return nil, fmt.Errorf("resolveLinkedRecipe: %w", err)
			}
			result = append(result, resolved...)
		} else {
			result = append(result, ing)
		}
	}
	return result, nil
}

func (p *Parser) flattenIngredientGroups(groups []IngredientGroup, baseDir string) ([]IngredientGroup, error) {
	result := make([]IngredientGroup, 0, len(groups))
	for _, g := range groups {
		ingredients, err := p.flattenIngredients(g.Ingredients, baseDir)
		if err != nil {
			return nil, fmt.Errorf("flattenIngredients: %w", err)
		}
		groups, err := p.flattenIngredientGroups(g.IngredientGroups, baseDir)
		if err != nil {
			return nil, fmt.Errorf("flattenIngredientGroups: %w", err)
		}
		result = append(result, IngredientGroup{
			Title:            g.Title,
			Ingredients:      ingredients,
			IngredientGroups: groups,
		})
	}
	return result, nil
}

func (p *Parser) resolveLinkedRecipe(link string, baseDir string, parent *Ingredient) ([]Ingredient, error) {
	if strings.Contains(link, "://") {
		return []Ingredient{*parent}, nil
	}

	path := filepath.Join(baseDir, link)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	linked, err := p.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("Parse: %w", err)
	}

	if parent.Amount != nil && len(linked.Yields) > 0 {
		if err := linked.ScaleForYield(*parent.Amount); err != nil {
			return nil, fmt.Errorf("linked.ScaleForYield: %w", err)
		}
	}

	linkedDir := filepath.Dir(path)
	flatIngredients, err := p.flattenIngredients(linked.Ingredients, linkedDir)
	if err != nil {
		return nil, fmt.Errorf("flattenIngredients: %w", err)
	}
	for _, g := range linked.IngredientGroups {
		ingredients, err := p.flattenIngredients(g.Ingredients, linkedDir)
		if err != nil {
			return nil, fmt.Errorf("flattenIngredients: %w", err)
		}
		flatIngredients = append(flatIngredients, ingredients...)
		groupIngredients, err := p.flattenGroupIngredients(g.IngredientGroups, linkedDir)
		if err != nil {
			return nil, fmt.Errorf("flattenGroupIngredients: %w", err)
		}
		flatIngredients = append(flatIngredients, groupIngredients...)
	}

	if len(flatIngredients) == 0 {
		return []Ingredient{*parent}, nil
	}
	return flatIngredients, nil
}

func (p *Parser) flattenGroupIngredients(groups []IngredientGroup, baseDir string) ([]Ingredient, error) {
	result := make([]Ingredient, 0, len(groups))
	for _, g := range groups {
		ingredients, err := p.flattenIngredients(g.Ingredients, baseDir)
		if err != nil {
			return nil, fmt.Errorf("flattenIngredients: %w", err)
		}
		result = append(result, ingredients...)
		groupIngredients, err := p.flattenGroupIngredients(g.IngredientGroups, baseDir)
		if err != nil {
			return nil, fmt.Errorf("flattenGroupIngredients: %w", err)
		}
		result = append(result, groupIngredients...)
	}
	return result, nil
}

// Parse converts a RecipeMD document into a Recipe struct via a single
// goldmark parse and linear AST walk.
// See: https://recipemd.org/specification.html#parsing-a-recipe
func (p *Parser) Parse(source []byte) (*Recipe, error) {
	if p.Frontmatter {
		source = stripFrontmatter(source)
	}

	document := p.goldmarkProcessor.Parser().Parse(text.NewReader(source))

	recipe := &Recipe{
		Yields:           []Amount{},
		Tags:             []string{},
		Ingredients:      []Ingredient{},
		IngredientGroups: []IngredientGroup{},
	}

	c := document.FirstChild()
	if c == nil {
		return nil, fmt.Errorf("recipe must have a title")
	}

	// --- Preamble: title ---
	h, ok := c.(*ast.Heading)
	if !ok {
		return nil, fmt.Errorf("expected level 1 heading, got %T", c)
	}
	if h.Level != 1 {
		return nil, fmt.Errorf("expected level 1 heading, got level %d", h.Level)
	}
	title, err := extractPlainText(h, source)
	if err != nil {
		return nil, fmt.Errorf("extractPlainText: %w", err)
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
		p, isPara := c.(*ast.Paragraph)
		if isPara {
			if em, ok := isOnlyEmphasis(p, italic); ok {
				if tagsFound {
					return nil, fmt.Errorf("tags already set")
				}
				tagsText, err := extractPlainText(em, source)
				if err != nil {
					return nil, fmt.Errorf("extractPlainText: %w", err)
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
			if em, ok := isOnlyEmphasis(p, bold); ok {
				if yieldsFound {
					return nil, fmt.Errorf("yields already set")
				}
				yieldsText, err := extractPlainText(em, source)
				if err != nil {
					return nil, fmt.Errorf("extractPlainText: %w", err)
				}
				yields, err := parseYields(yieldsText)
				if err != nil {
					return nil, fmt.Errorf("parseYields: %w", err)
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
				return nil, fmt.Errorf("unexpected content in tags/yields section")
			}
		}
		if _, end := getRecursiveSourceBounds(c, source); end > lastPreBreakEnd {
			lastPreBreakEnd = end
		}
		c = c.NextSibling()
	}

	// --- First thematic break ---
	if c == nil || c.Kind() != ast.KindThematicBreak {
		return nil, fmt.Errorf("missing thematic break divider")
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
	if _, ok := c.(*ast.Paragraph); ok {
		return nil, fmt.Errorf("paragraph not valid in ingredients section")
	}
	c, err = parseIngredientList(c, source, &recipe.Ingredients, p.hasTaskList)
	if err != nil {
		return nil, err
	}
	c, err = parseIngredientGroup(c, source, &recipe.IngredientGroups, 0, p.hasTaskList)
	if err != nil {
		return nil, err
	}

	// --- Second thematic break (optional) → instructions ---
	if c != nil && c.Kind() == ast.KindThematicBreak {
		breakPos := findDashLine(source, firstBreakPos+1)
		breakEnd := skipLine(source, breakPos)
		instructions := strings.Trim(string(source[breakEnd:]), "\n")
		if instructions != "" {
			recipe.Instructions = &instructions
		}
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
	for {
		h, ok := c.(*ast.Heading)
		if !ok {
			return c, nil
		}
		l := h.Level
		if l <= parentLevel {
			return c, nil
		}
		title, err := extractPlainText(h, source)
		if err != nil {
			return nil, fmt.Errorf("extractPlainText: %w", err)
		}
		g := IngredientGroup{
			Title:            title,
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
		}
		c = c.NextSibling()
		if c == nil {
			*groups = append(*groups, g)
			return nil, nil
		}
		c, err = parseIngredientList(c, source, &g.Ingredients, skipCheckbox)
		if err != nil {
			return nil, err
		}
		c, err = parseIngredientGroup(c, source, &g.IngredientGroups, l, skipCheckbox)
		if err != nil {
			return nil, err
		}
		*groups = append(*groups, g)
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
	for {
		// 1. Examine c
		list, ok := c.(*ast.List)
		if !ok {
			return c, nil
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
				return nil, fmt.Errorf("parseIngredient: %w", err)
			}
			*ingredients = append(*ingredients, ing)
			// Go to next item
			if c.NextSibling() != nil {
				c = c.NextSibling()
			} else {
				// Leave c and go to 1
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

// ParseAmountString parses an amount string into value and unit.
// This is the exported version of parseAmount for CLI use.
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

func parseYields(s string) (yields []Amount, err error) {
	for _, yield := range splitList(s) {
		amount, err := parseAmount(yield)
		if err != nil {
			return nil, fmt.Errorf("parseAmount: %w", err)
		}
		yields = append(yields, amount)
	}
	return yields, nil
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

// stripFrontmatter removes YAML (---) or TOML (+++) frontmatter from the
// beginning of source. Returns source unchanged if no frontmatter is found.
func stripFrontmatter(source []byte) []byte {
	if len(source) < 3 {
		return source
	}
	var fence []byte
	if bytes.HasPrefix(source, []byte("---")) {
		fence = []byte("---")
	} else if bytes.HasPrefix(source, []byte("+++")) {
		fence = []byte("+++")
	} else {
		return source
	}

	// Opening fence must be alone on the line (optional trailing whitespace)
	firstNL := bytes.IndexByte(source, '\n')
	if firstNL < 0 {
		return source
	}
	if len(bytes.TrimSpace(source[:firstNL])) != len(fence) {
		return source
	}

	// Find closing fence
	rest := source[firstNL+1:]
	for len(rest) > 0 {
		lineEnd := bytes.IndexByte(rest, '\n')
		var line []byte
		if lineEnd < 0 {
			line = rest
		} else {
			line = rest[:lineEnd]
		}
		if bytes.Equal(bytes.TrimSpace(line), fence) {
			if lineEnd < 0 {
				return nil
			}
			return rest[lineEnd+1:]
		}
		if lineEnd < 0 {
			break
		}
		rest = rest[lineEnd+1:]
	}
	return source
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
