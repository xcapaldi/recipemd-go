package recipemd

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Due to the nature of walking the AST with a goldmark ast.Walker, we cannot
// chain function calls as proposed in the RecipeMD parsing strategy:
// https://recipemd.org/specification.html#recipemd-parsing-strategy
// Instead we need a state machine to track what we have parsed and what could
// be the next parse-able element.
type parserState int

const (
	stateStart parserState = iota
	stateDescription
	stateTagsYields
	stateIngredients
	stateInstructions
)

// ParseRecipe converts a RecipeMD document into a Recipe struct.
// See: https://recipemd.org/specification.html#parsing-a-recipe
func ParseRecipe(source []byte) (*Recipe, error) {
	reader := text.NewReader(source)
	parser := goldmark.DefaultParser()
	document := parser.Parse(reader)
	thematicBreaks := findThematicBreaks(source)

	recipe := &Recipe{
		Yields:           []Amount{},
		Tags:             []string{},
		Ingredients:      []Ingredient{},
		IngredientGroups: []IngredientGroup{},
	}

	// State machine variables
	state := stateStart
	ingredientsParsed := false
	var descriptionStart int
	var excludeRanges [][2]int
	var firstBreakPos, secondBreakPos int
	breakIdx := 0

	excludeNodeRange := func(n ast.Node) {
		start, end := getNodeBounds(n)
		if start >= 0 {
			excludeRanges = append(excludeRanges, [2]int{start, end})
		}
	}

	extractRecipe := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch state {
		// 2. Parse title
		case stateStart:
			h, ok := n.(*ast.Heading)
			if !ok {
				return ast.WalkContinue, nil
			}
			if h.Level != 1 {
				return ast.WalkStop, fmt.Errorf("expected level 1 heading, got level %d", h.Level)
			}
			title, err := extractText(h, source)
			if err != nil {
				return ast.WalkStop, fmt.Errorf("extractText: %w", err)
			}
			recipe.Title = title
			// 3. Let descriptionStart be the index of the starting line of c (after title)
			_, end := getNodeBounds(n)
			descriptionStart = skipSetextUnderline(source, end)
			state = stateDescription
			return ast.WalkSkipChildren, nil

		// 4. Parse the description
		case stateDescription:
			// If c is a thematic break, go to 7 (stateIngredients)
			if n.Kind() == ast.KindThematicBreak {
				if breakIdx < len(thematicBreaks) {
					firstBreakPos = thematicBreaks[breakIdx]
					breakIdx++
				}
				state = stateIngredients
				return ast.WalkSkipChildren, nil
			}
			p, ok := n.(*ast.Paragraph)
			if !ok {
				// Not a paragraph - include in description (handled later using firstBreakPos)
				return ast.WalkSkipChildren, nil
			}
			// If c is a paragraph whose contents are a single emphasis, go to 5 (stateTagsYields)
			if em, ok := isOnlyEmphasis(p, italic); ok {
				// 6. Parse tags
				tagsText, err := extractText(em, source)
				if err != nil {
					return ast.WalkStop, fmt.Errorf("extractText: %w", err)
				}
				recipe.Tags = parseTags(tagsText)
				excludeNodeRange(n)
				state = stateTagsYields
				return ast.WalkSkipChildren, nil
			}
			// If c is a paragraph whose contents are a single strong emphasis, go to 5 (stateTagsYields)
			if em, ok := isOnlyEmphasis(p, bold); ok {
				// 6. Parse yields
				yieldsText, err := extractText(em, source)
				if err != nil {
					return ast.WalkStop, fmt.Errorf("extractText: %w", err)
				}
				yields, err := parseYields(yieldsText)
				if err != nil {
					return ast.WalkStop, fmt.Errorf("parseYields: %w", err)
				}
				recipe.Yields = yields
				excludeNodeRange(n)
				state = stateTagsYields
				return ast.WalkSkipChildren, nil
			}
			// Regular paragraph - include in description (handled later using firstBreakPos)
			return ast.WalkSkipChildren, nil

		// 6. Parse tags and yields (continued)
		case stateTagsYields:
			// If c is a thematic break, go to 7 (stateIngredients)
			if n.Kind() == ast.KindThematicBreak {
				if breakIdx < len(thematicBreaks) {
					firstBreakPos = thematicBreaks[breakIdx]
					breakIdx++
				}
				state = stateIngredients
				return ast.WalkSkipChildren, nil
			}
			p, ok := n.(*ast.Paragraph)
			if !ok {
				return ast.WalkSkipChildren, nil
			}
			if em, ok := isOnlyEmphasis(p, italic); ok {
				if len(recipe.Tags) > 0 {
					return ast.WalkStop, fmt.Errorf("tags already set")
				}
				tagsText, err := extractText(em, source)
				if err != nil {
					return ast.WalkStop, fmt.Errorf("extractText: %w", err)
				}
				recipe.Tags = parseTags(tagsText)
				excludeNodeRange(n)
				return ast.WalkSkipChildren, nil
			}
			if em, ok := isOnlyEmphasis(p, bold); ok {
				if len(recipe.Yields) > 0 {
					return ast.WalkStop, fmt.Errorf("yields already set")
				}
				yieldsText, err := extractText(em, source)
				if err != nil {
					return ast.WalkStop, fmt.Errorf("extractText: %w", err)
				}
				yields, err := parseYields(yieldsText)
				if err != nil {
					return ast.WalkStop, fmt.Errorf("parseYields: %w", err)
				}
				recipe.Yields = yields
				excludeNodeRange(n)
				return ast.WalkSkipChildren, nil
			}
			return ast.WalkStop, fmt.Errorf("unexpected content in tags/yields section")

		// 8. Parse ingredients and ingredient groups
		case stateIngredients:
			// 9. Find instruction divider
			if n.Kind() == ast.KindThematicBreak {
				if breakIdx < len(thematicBreaks) {
					secondBreakPos = thematicBreaks[breakIdx]
					breakIdx++
				}
				state = stateInstructions
				return ast.WalkSkipChildren, nil
			}
			if ingredientsParsed {
				return ast.WalkSkipChildren, nil
			}
			// Paragraphs are not valid in ingredients section
			if _, ok := n.(*ast.Paragraph); ok {
				return ast.WalkStop, fmt.Errorf("paragraph not valid in ingredients section")
			}
			// Run parsing ingredient list and groups
			c, err := parseIngredientList(n, source, &recipe.Ingredients)
			if err != nil {
				return ast.WalkStop, err
			}
			_, err = parseIngredientGroup(c, source, &recipe.IngredientGroups, 0)
			if err != nil {
				return ast.WalkStop, err
			}
			ingredientsParsed = true
			return ast.WalkSkipChildren, nil

		// 10. Instructions handled after walk
		case stateInstructions:
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	}

	if err := ast.Walk(document, extractRecipe); err != nil {
		return nil, fmt.Errorf("ast.Walk: %w", err)
	}

	// Validate
	if recipe.Title == "" {
		return nil, fmt.Errorf("recipe must have a title")
	}
	if state != stateIngredients && state != stateInstructions {
		return nil, fmt.Errorf("missing thematic break divider")
	}

	// 5. Set the description (from title end to first thematic break)
	if firstBreakPos > descriptionStart {
		descBytes := source[descriptionStart:firstBreakPos]
		desc := excludeRangesFromSource(descBytes, excludeRanges, descriptionStart)
		desc = strings.Trim(desc, "\n")
		if desc != "" {
			recipe.Description = &desc
		}
	}

	// 10. Set the recipe's instructions to the remainder of the document
	if secondBreakPos > 0 {
		// Skip past the thematic break line
		instrPos := secondBreakPos
		for instrPos < len(source) && source[instrPos] != '\n' {
			instrPos++
		}
		if instrPos < len(source) {
			instrPos++
		}
		instr := strings.Trim(string(source[instrPos:]), "\n")
		if instr != "" {
			recipe.Instructions = &instr
		}
	}

	return recipe, nil
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
		title, err := extractText(h, source)
		if err != nil {
			return nil, fmt.Errorf("extractText: %w", err)
		}
		g := IngredientGroup{
			Title:            title,
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
		}
		c = c.NextSibling()
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
			if ing.Name == "" {
				return nil, fmt.Errorf("ingredient must have a name")
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
		n = nodeSource(c, source)
	} else {
		// Parse the amount
		var r string
		afterAmount := firstInline

		if em, ok := firstInline.(*ast.Emphasis); ok && em.Level == 1 {
			// If c's contents start with an emphasis inline
			emText, err := extractText(em, source)
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
			r = getTextFrom(afterAmount, source)
		} else {
			// Let r be the verbatim contents of c
			r = nodeSource(c, source)
		}

		// Parse the link
		isOnlyChild := c.NextSibling() == nil
		link := findSingleLink(afterAmount, source)

		if isOnlyChild && link != nil {
			// Set l to the link's destination
			dest := string(link.Destination)
			dest = strings.ReplaceAll(dest, " ", "%20")
			l = &dest
			// Set n to the link's text
			linkText, _ := extractText(link, source)
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
		n += sep + nodeSource(c, source)
		prevBlock = c
	}

	// 7. Leave c (implicit)

	// 8. Let i be an ingredient with amount a, name n, link l
	n = strings.TrimSpace(n)
	return Ingredient{Amount: a, Name: n, Link: l}, nil
}

// getTextFrom extracts text from a sequence of inline nodes
func getTextFrom(start ast.Node, source []byte) string {
	var parts []string
	for n := start; n != nil; n = n.NextSibling() {
		parts = append(parts, inlineToText(n, source))
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}

// inlineToText converts an inline node to its text representation
func inlineToText(n ast.Node, source []byte) string {
	if t, ok := n.(*ast.Text); ok {
		return string(t.Value(source))
	}
	text, _ := extractText(n, source)
	if n.Kind() == ast.KindEmphasis {
		return "*" + text + "*"
	}
	if link, ok := n.(*ast.Link); ok {
		return "[" + text + "](" + string(link.Destination) + ")"
	}
	return text
}

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
	// Extract blank line content (everything between first and last newline)
	if _, rest, ok := bytes.Cut(between, []byte{'\n'}); ok {
		if blankContent, _, ok := bytes.Cut(rest, []byte{'\n'}); ok {
			return "\n" + string(blankContent) + "\n  "
		}
	}
	return "\n\n  "
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
	if v == nil {
		v, remaining = parseImproperFraction(s)
	}
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
		return Amount{Factor: formatDecimal(val), Unit: unit}, nil
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
	whole, _ := strconv.ParseFloat(string(runes[start:i]), 64)

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
	num, _ := strconv.ParseFloat(string(runes[start:i]), 64)

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
	denom, _ := strconv.ParseFloat(string(runes[start:i]), 64)

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
	whole, _ := strconv.ParseFloat(string(runes[start:i]), 64)

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
	num, _ := strconv.ParseFloat(string(runes[start:i]), 64)

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
	denom, _ := strconv.ParseFloat(string(runes[start:i]), 64)

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
	v, _ := strconv.ParseFloat(numStr, 64)
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
	v, _ := strconv.ParseFloat(string(runes[:i]), 64)
	return &v, string(runes[i:])
}

func findThematicBreaks(source []byte) []int {
	var positions []int
	lines := bytes.Split(source, []byte("\n"))
	pos := 0
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) >= 3 && len(bytes.Trim(trimmed, "-")) == 0 {
			positions = append(positions, pos)
		}
		pos += len(line) + 1
	}
	return positions
}

func getNodeBounds(n ast.Node) (start, end int) {
	lines := n.Lines()
	if lines.Len() == 0 {
		return -1, -1
	}
	return lines.At(0).Start, lines.At(lines.Len() - 1).Stop
}

// extractText recursively extracts plain text from a node, stripping markdown.
func extractText(node ast.Node, source []byte) (string, error) {
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
			childText, err := extractText(child, source)
			if err != nil {
				return "", err
			}
			buf.WriteString(childText)
		}
	}
	return buf.String(), nil
}

// nodeSource extracts raw source for a block node, preserving markdown syntax.
func nodeSource(node ast.Node, source []byte) string {
	start, end := nodeSourceBounds(node, source)
	if start < 0 {
		return ""
	}
	return strings.TrimRight(string(source[start:end]), "\n")
}

// nodeSourceBounds returns the byte range for a node's source.
func nodeSourceBounds(node ast.Node, source []byte) (start, end int) {
	if node.Type() == ast.TypeBlock {
		if lines := node.Lines(); lines.Len() > 0 {
			return lines.At(0).Start, lines.At(lines.Len() - 1).Stop
		}
	}
	start, end = -1, -1
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		childStart, childEnd := nodeSourceBounds(child, source)
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

func formatDecimal(f float64) string {
	s := fmt.Sprintf("%.3f", f)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
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
