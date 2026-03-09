package recipemd

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Commonmark compliant parser
var markdownParser = goldmark.DefaultParser()

// ParseRecipe converts a RecipeMD document into a Recipe struct.
// See: https://recipemd.org/specification.html#parsing-a-recipe
//
// The document is split into sections at thematic breaks (---), then each
// section is parsed independently: preamble (title, description, tags, yields),
// ingredients, and instructions.
func ParseRecipe(source []byte) (*Recipe, error) {
	preamble, ingredients, remaining := splitSections(source)

	recipe := &Recipe{
		Yields:           []Amount{},
		Tags:             []string{},
		Ingredients:      []Ingredient{},
		IngredientGroups: []IngredientGroup{},
	}

	if err := parsePreamble(preamble, recipe); err != nil {
    return nil, fmt.Errorf("parsePreamble: %w", err)
	}

	if ingredients == nil {
		return nil, fmt.Errorf("missing thematic break divider")
	}

	if err := parseIngredientsSection(ingredients, recipe); err != nil {
		return nil, err
	}

	if remaining != nil {
		instructions := strings.Trim(string(remaining), "\n")
		if instructions != "" {
			recipe.Instructions = &instructions
		}
	}

	return recipe, nil
}

// splitSections splits a RecipeMD source at thematic breaks (---) identified
// by goldmark (which correctly ignores --- inside fenced code blocks, setext
// H2 underlines, etc.). Returns up to three sections: preamble, ingredients,
// instructions. Sections that don't exist are returned as nil.
func splitSections(source []byte) (preamble, ingredients, instructions []byte) {
	document := markdownParser.Parse(text.NewReader(source))

	// Collect positions of real ThematicBreak nodes as identified by goldmark.
	// ThematicBreak nodes don't have Lines(), so we find position by scanning
	// forward. Track minPos to handle nodes with no valid bounds.
	var breakPositions []int
	minPos := 0
	for c := document.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == ast.KindThematicBreak {
			pos := findThematicBreakAfter(minPos, source)
			if pos >= 0 {
				breakPositions = append(breakPositions, pos)
				// Next search starts after this break's line
				minPos = pos
				for minPos < len(source) && source[minPos] != '\n' {
					minPos++
				}
				if minPos < len(source) {
					minPos++
				}
			}
		} else {
			// Update minPos based on this node's bounds
			_, end := getRecursiveSourceBounds(c, source)
			if end > minPos {
				minPos = end
			}
		}
	}
	if len(breakPositions) == 0 {
		return source, nil, nil
	}

	// Compute the byte just past each break's newline (start of next section).
	breakEnds := make([]int, len(breakPositions))
	for i, pos := range breakPositions {
		j := pos
		for j < len(source) && source[j] != '\n' {
			j++
		}
		if j < len(source) {
			j++ // include the newline
		}
		breakEnds[i] = j
	}

	switch len(breakPositions) {
	case 0:
		return source, nil, nil
	case 1:
		return source[:breakPositions[0]], source[breakEnds[0]:], nil
	default:
		return source[:breakPositions[0]],
			source[breakEnds[0]:breakPositions[1]],
			source[breakEnds[1]:]
	}
}

// findThematicBreakAfter finds the byte offset of the first thematic break
// line (3+ dashes) at or after minPos.
func findThematicBreakAfter(minPos int, source []byte) int {
	pos := minPos
	// Advance to line start if mid-line
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

// parsePreamble extracts title, description, tags, and yields from the preamble
// section (everything before the first thematic break).
// See: https://recipemd.org/specification.html#parsing-a-recipe steps 2–6
func parsePreamble(section []byte, recipe *Recipe) error {
	document := markdownParser.Parse(text.NewReader(section))

	// 2. Parse title: first block must be a level-1 heading.
	c := document.FirstChild()
	if c == nil {
		return fmt.Errorf("recipe must have a title")
	}
	h, ok := c.(*ast.Heading)
	if !ok {
		return fmt.Errorf("expected level 1 heading, got %T", c)
	}
	if h.Level != 1 {
		return fmt.Errorf("expected level 1 heading, got level %d", h.Level)
	}
	title, err := extractPlainText(h, section)
	if err != nil {
		return fmt.Errorf("extractPlainText: %w", err)
	}
	recipe.Title = title

	// 3. Description starts after the title line (plus setext underline if present).
	// We use the heading's own line end rather than the next sibling's start
	// because nodes like FencedCodeBlock have Lines() covering only their
	// interior content, not the fence delimiters.
	_, titleLineEnd := getDirectLineBounds(h)
	descStart := skipSetextUnderline(section, titleLineEnd)

	// 4–6. Walk remaining blocks to extract tags and yields.
	// Once either is seen, any subsequent non-tags/yields paragraph is an error.
	var excludeRanges [][2]int
	tagsFound, yieldsFound, tagsYieldsMode := false, false, false

	for c = c.NextSibling(); c != nil; c = c.NextSibling() {
		p, ok := c.(*ast.Paragraph)
		if !ok {
			// Non-paragraph blocks (headings, code, etc.) are part of the description.
			continue
		}

		if em, ok := isOnlyEmphasis(p, italic); ok {
			// 6. Italic-only paragraph → tags.
			if tagsFound {
				return fmt.Errorf("tags already set")
			}
			tagsText, err := extractPlainText(em, section)
			if err != nil {
				return fmt.Errorf("extractPlainText: %w", err)
			}
			recipe.Tags = parseTags(tagsText)
			tagsFound = true
			tagsYieldsMode = true
			if start, end := getDirectLineBounds(c); start >= 0 {
				excludeRanges = append(excludeRanges, [2]int{start, end})
			}
		} else if em, ok := isOnlyEmphasis(p, bold); ok {
			// 6. Bold-only paragraph → yields.
			if yieldsFound {
				return fmt.Errorf("yields already set")
			}
			yieldsText, err := extractPlainText(em, section)
			if err != nil {
				return fmt.Errorf("extractPlainText: %w", err)
			}
			yields, err := parseYields(yieldsText)
			if err != nil {
				return fmt.Errorf("parseYields: %w", err)
			}
			recipe.Yields = yields
			yieldsFound = true
			tagsYieldsMode = true
			if start, end := getDirectLineBounds(c); start >= 0 {
				excludeRanges = append(excludeRanges, [2]int{start, end})
			}
		} else if tagsYieldsMode {
			// Any other paragraph after tags/yields is invalid.
			return fmt.Errorf("unexpected content in tags/yields section")
		}
		// Otherwise: a regular description paragraph; included via exclusion logic below.
	}

	// 5. Build description: preamble from after title to end, minus tags/yields.
	if descStart < len(section) {
		desc := excludeRangesFromSource(section[descStart:], excludeRanges, descStart)
		desc = strings.Trim(desc, "\n")
		if desc != "" {
			recipe.Description = &desc
		}
	}

	return nil
}

// parseIngredientsSection extracts ingredients and ingredient groups from the
// section between the two thematic breaks.
// See: https://recipemd.org/specification.html#parsing-an-ingredient-list
func parseIngredientsSection(section []byte, recipe *Recipe) error {
	document := markdownParser.Parse(text.NewReader(section))

	c := document.FirstChild()

	// Paragraphs are not valid in the ingredients section.
	if _, ok := c.(*ast.Paragraph); ok {
		return fmt.Errorf("paragraph not valid in ingredients section")
	}

	c, err := parseIngredientList(c, section, &recipe.Ingredients)
	if err != nil {
		return err
	}
	_, err = parseIngredientGroup(c, section, &recipe.IngredientGroups, 0)
	return err
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
		c, err = parseIngredientList(c, source, &g.Ingredients)
		if err != nil {
			return nil, err
		}
		c, err = parseIngredientGroup(c, source, &g.IngredientGroups, l)
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
			ing, err := parseIngredient(c, source)
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
func parseIngredient(c ast.Node, source []byte) (Ingredient, error) {
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
		return Ingredient{Name: n}, nil
	}

	// 5. Examine c
	// Note: goldmark uses TextBlock for tight lists, Paragraph for loose lists
	var firstInline ast.Node
	if para, ok := c.(*ast.Paragraph); ok {
		firstInline = para.FirstChild()
	} else if tb, ok := c.(*ast.TextBlock); ok {
		firstInline = tb.FirstChild()
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
			// Set l to the link's destination
			dest := encodeURLPath(string(link.Destination))
			l = &dest
			// Set n to the link's text
			linkText, _ := extractPlainText(link, source)
			n = linkText
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

// findSingleLink checks if nodes from start consist only of whitespace and a single link
func findSingleLink(start ast.Node, source []byte) *ast.Link {
	var link *ast.Link
	for n := start; n != nil; n = n.NextSibling() {
		if l, ok := n.(*ast.Link); ok {
			if link != nil {
				return nil // multiple links
			}
			link = l
		} else if t, ok := n.(*ast.Text); ok {
			if strings.TrimSpace(string(t.Value(source))) != "" {
				return nil // non-whitespace text
			}
		} else {
			return nil // other inline element
		}
	}
	return link
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
