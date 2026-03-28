package recipemd

import (
	"html"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// InlineIngredientsFormat controls where the injected amount appears relative
// to the ingredient name in the rendered output.
type InlineIngredientsFormat int

const (
	// InlineIngredientsBefore places the formatted amount before the matched
	// name: "add 3 eggs to the bowl".
	InlineIngredientsBefore InlineIngredientsFormat = iota

	// InlineIngredientsAfter places the formatted amount in parentheses after
	// the matched name: "add eggs (3) to the bowl".
	InlineIngredientsAfter
)

// InlineIngredientsOption is a functional option for configuring inline
// ingredient injection. Options are passed to [WithInlineIngredients].
type InlineIngredientsOption func(*inlineIngredientsConfig)

type inlineIngredientsConfig struct {
	format         InlineIngredientsFormat
	htmlHover      bool
	prepSeparators []string
}

// WithInlineFormat sets how the injected amount is positioned relative to the
// ingredient name. The default is [InlineIngredientsBefore].
func WithInlineFormat(f InlineIngredientsFormat) InlineIngredientsOption {
	return func(c *inlineIngredientsConfig) { c.format = f }
}

// WithInlineHTMLHover renders the amount as a tooltip (HTML title attribute)
// rather than inline visible text when using [Parser.RenderHTML]. The
// ingredient name receives a "recipemd-inline-ingredient" span whose title
// attribute holds the amount (and prep note when applicable). The display
// format option is ignored when hover is enabled.
func WithInlineHTMLHover() InlineIngredientsOption {
	return func(c *inlineIngredientsConfig) { c.htmlHover = true }
}

// WithInlinePrepSeparators registers one or more separator strings that split
// an ingredient name into a base name and a preparation note.
//
// For example, with separator "," the ingredient "3 cloves garlic, chopped"
// yields base="garlic" and prep="chopped". With separator "(" the ingredient
// "3 cloves garlic (chopped)" yields base="garlic" and prep="chopped" — the
// paired closing ")" is stripped automatically.
//
// Only the first matching separator is applied. The base name is used for
// matching in the instructions; the prep note is included in the injected text.
//
//   - [InlineIngredientsBefore]: "3 cloves chopped garlic"
//   - [InlineIngredientsAfter]:  "garlic (3 cloves chopped)"
//   - Hover:                     title="3 cloves chopped", visible text="garlic"
func WithInlinePrepSeparators(seps ...string) InlineIngredientsOption {
	return func(c *inlineIngredientsConfig) {
		c.prepSeparators = append(c.prepSeparators, seps...)
	}
}

// WithInlineIngredients returns an [Option] that enables inline ingredient
// injection at render time. Every occurrence of an ingredient name in the
// instructions is annotated with its amount by [Parser.RenderMarkdown] and
// [Parser.RenderHTML].
//
// Matching is case-insensitive and respects word boundaries so partial matches
// (e.g. "salt" inside "salted") are avoided. Both singular and plural forms
// of each ingredient name are matched (e.g. ingredient "egg" also matches
// "eggs" in the instructions). Multi-word names are matched as complete phrases
// and take priority over any shorter overlapping names.
//
// Sub-options control the display format, HTML hover behaviour, and optional
// preparation-note separators; see [WithInlineFormat], [WithInlineHTMLHover],
// and [WithInlinePrepSeparators].
//
//	p := recipemd.NewParser(
//	    recipemd.WithInlineIngredients(
//	        recipemd.WithInlineFormat(recipemd.InlineIngredientsBefore),
//	        recipemd.WithInlinePrepSeparators(",", "("),
//	    ),
//	)
func WithInlineIngredients(opts ...InlineIngredientsOption) Option {
	return func(p *Parser) {
		cfg := &inlineIngredientsConfig{}
		for _, o := range opts {
			o(cfg)
		}
		p.inlineIngredients = true
		p.inlineIngredientsCfg = *cfg
	}
}

// ---------------------------------------------------------------------------
// Internal injector
// ---------------------------------------------------------------------------

type inlineEntry struct {
	amount string // formatted amount string, e.g. "3 cups"
	prep   string // preparation note stripped from name, e.g. "chopped"
}

type inlineInjector struct {
	cfg    inlineIngredientsConfig
	re     *regexp.Regexp
	lookup map[string]*inlineEntry // lowercase match key → entry
}

// buildInjector constructs an inlineInjector for the given recipe. Returns nil
// when no ingredients with amounts are found.
func buildInjector(r *Recipe, cfg inlineIngredientsConfig, rounding int) *inlineInjector {
	ingredients := r.LeafIngredients()

	lookup := make(map[string]*inlineEntry, len(ingredients)*2)
	var alts []string

	for _, ing := range ingredients {
		if ing.Amount == nil {
			continue
		}

		name := ing.Name
		prep := ""

		// Split out preparation note using configured separators.
		for _, sep := range cfg.prepSeparators {
			if idx := strings.Index(name, sep); idx >= 0 {
				raw := strings.TrimSpace(name[idx+len(sep):])
				// Strip the paired closing bracket when separator is an opener.
				raw = stripClosingBracket(raw, sep)
				prep = strings.TrimSpace(raw)
				name = strings.TrimSpace(name[:idx])
				break
			}
		}

		entry := &inlineEntry{
			amount: ing.Amount.Serialize(rounding),
			prep:   prep,
		}

		// Register all word forms (singular + plural) so that "egg" matches
		// "eggs" in the instructions and vice-versa.
		for _, form := range wordForms(name) {
			if _, exists := lookup[form]; !exists {
				lookup[form] = entry
				alts = append(alts, regexp.QuoteMeta(form))
			}
		}
	}

	if len(alts) == 0 {
		return nil
	}

	// Sort longest alternatives first so that multi-word names (e.g. "brown
	// sugar") take precedence over shorter sub-names (e.g. "sugar") within
	// the combined alternation.
	sort.Slice(alts, func(i, j int) bool { return len(alts[i]) > len(alts[j]) })

	pattern := `(?i)\b(?:` + strings.Join(alts, "|") + `)\b`
	re := regexp.MustCompile(pattern)

	return &inlineInjector{cfg: cfg, re: re, lookup: lookup}
}

// stripClosingBracket removes the paired closing bracket from s when sep is
// an opening bracket character, e.g. "(" → strip trailing ")".
func stripClosingBracket(s, sep string) string {
	pairs := map[string]byte{"(": ')', "[": ']', "{": '}'}
	if close, ok := pairs[sep]; ok {
		s = strings.TrimRight(s, string(close))
	}
	return s
}

// injectText applies inline injection to a plain-text (markdown) string.
// The result can be used directly in markdown output.
func (inj *inlineInjector) injectText(text string) string {
	return inj.re.ReplaceAllStringFunc(text, func(match string) string {
		e := inj.lookup[strings.ToLower(match)]
		return inj.formatText(match, e)
	})
}

// injectHTML applies inline injection to an HTML string produced by goldmark,
// touching only text nodes (content between tags) and producing span elements.
func (inj *inlineInjector) injectHTML(htmlStr string) string {
	return injectHTMLTextNodes(htmlStr, inj.re, func(match string) string {
		e := inj.lookup[strings.ToLower(match)]
		if inj.cfg.htmlHover {
			return inj.formatHTMLHover(match, e)
		}
		return inj.formatHTMLSpan(match, e)
	})
}

// formatText builds the plain-text replacement for match.
func (inj *inlineInjector) formatText(match string, e *inlineEntry) string {
	if e.prep != "" {
		switch inj.cfg.format {
		case InlineIngredientsBefore:
			return e.amount + " " + e.prep + " " + match
		case InlineIngredientsAfter:
			return match + " (" + e.amount + " " + e.prep + ")"
		}
	}
	switch inj.cfg.format {
	case InlineIngredientsBefore:
		return e.amount + " " + match
	case InlineIngredientsAfter:
		return match + " (" + e.amount + ")"
	}
	return match
}

// formatHTMLSpan wraps the replacement in a recipemd-inline-ingredient span.
// match is raw text content from the HTML (already entity-encoded for simple
// ASCII names) and is reused verbatim to preserve entity encoding.
func (inj *inlineInjector) formatHTMLSpan(match string, e *inlineEntry) string {
	var inner string
	if e.prep != "" {
		switch inj.cfg.format {
		case InlineIngredientsBefore:
			inner = html.EscapeString(e.amount+" "+e.prep) + " " + match
		case InlineIngredientsAfter:
			inner = match + " (" + html.EscapeString(e.amount+" "+e.prep) + ")"
		}
	} else {
		switch inj.cfg.format {
		case InlineIngredientsBefore:
			inner = html.EscapeString(e.amount) + " " + match
		case InlineIngredientsAfter:
			inner = match + " (" + html.EscapeString(e.amount) + ")"
		}
	}
	return `<span class="recipemd-inline-ingredient">` + inner + `</span>`
}

// formatHTMLHover wraps match in a span whose title attribute holds the amount
// (and prep note). The visible text is the matched name only.
func (inj *inlineInjector) formatHTMLHover(match string, e *inlineEntry) string {
	title := e.amount
	if e.prep != "" {
		title += " " + e.prep
	}
	return `<span class="recipemd-inline-ingredient" title="` +
		html.EscapeString(title) + `">` + match + `</span>`
}

// injectHTMLTextNodes runs replacer over text segments of an HTML string,
// leaving all tag markup untouched. It does not parse HTML fully; it splits
// on '<' and '>' which is sufficient for well-formed goldmark output.
func injectHTMLTextNodes(htmlStr string, re *regexp.Regexp, replacer func(string) string) string {
	var b strings.Builder
	b.Grow(len(htmlStr))
	for len(htmlStr) > 0 {
		lt := strings.IndexByte(htmlStr, '<')
		if lt < 0 {
			b.WriteString(re.ReplaceAllStringFunc(htmlStr, replacer))
			break
		}
		if lt > 0 {
			b.WriteString(re.ReplaceAllStringFunc(htmlStr[:lt], replacer))
		}
		gt := strings.IndexByte(htmlStr[lt:], '>')
		if gt < 0 {
			b.WriteString(htmlStr[lt:])
			break
		}
		b.WriteString(htmlStr[lt : lt+gt+1])
		htmlStr = htmlStr[lt+gt+1:]
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Pluralization helpers
// ---------------------------------------------------------------------------

// wordForms returns the unique lowercase word forms (singular and plural) that
// should be matched for an ingredient name. For multi-word names each form is
// derived by transforming the last word only.
//
// The implementation covers standard English pluralization rules with a
// focused irregulars table for common food vocabulary, inspired by the
// go-pluralize package (github.com/gertd/go-pluralize).
func wordForms(word string) []string {
	lower := strings.ToLower(word)

	// For multi-word names derive forms from the last word only.
	parts := strings.Fields(lower)
	if len(parts) > 1 {
		prefix := strings.Join(parts[:len(parts)-1], " ") + " "
		last := parts[len(parts)-1]
		var forms []string
		for _, f := range singleWordForms(last) {
			forms = append(forms, prefix+f)
		}
		return dedup(forms)
	}

	return singleWordForms(lower)
}

func singleWordForms(lower string) []string {
	sg := toSingular(lower)
	pl := toPlural(lower)
	if sg == pl {
		return []string{sg}
	}
	return dedup([]string{sg, pl})
}

func dedup(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := ss[:0]
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// toPlural converts a lowercase singular word to its plural form.
func toPlural(word string) string {
	if pl, ok := irregularsToPlural[word]; ok {
		return pl
	}
	if _, ok := irregularsToSingular[word]; ok {
		return word // already plural
	}
	switch {
	case hasSuffix(word, "ies"): // already plural
		return word
	case hasSuffix(word, "y") && len(word) > 1 && isConsonant(rune(word[len(word)-2])):
		return word[:len(word)-1] + "ies"
	case hasSuffix(word, "fe"):
		return word[:len(word)-2] + "ves"
	case hasSuffix(word, "f") && !hasSuffix(word, "ff"):
		return word[:len(word)-1] + "ves"
	case hasSuffixAny(word, "ch", "sh", "x", "z", "s"):
		return word + "es"
	default:
		return word + "s"
	}
}

// toSingular converts a lowercase plural word to its singular form.
func toSingular(word string) string {
	if sg, ok := irregularsToSingular[word]; ok {
		return sg
	}
	if _, ok := irregularsToPlural[word]; ok {
		return word // already singular
	}
	switch {
	case hasSuffix(word, "ies") && len(word) > 3:
		return word[:len(word)-3] + "y"
	case hasSuffix(word, "ves") && len(word) > 3:
		// knives → knife, leaves → leaf: fall back to removing -ves and adding -fe/-f.
		// Use the irregulars table for exact known cases; approximation otherwise.
		return word[:len(word)-3] + "f"
	case hasSuffixAny(word, "shes", "ches", "xes", "zes") && len(word) > 4:
		return word[:len(word)-2]
	case hasSuffix(word, "ses") && len(word) > 3:
		return word[:len(word)-2]
	case hasSuffix(word, "s") && !hasSuffix(word, "ss"):
		return word[:len(word)-1]
	default:
		return word
	}
}

// irregularsToPlural maps singular → plural for words that don't follow the
// standard rules, with emphasis on food-relevant vocabulary.
var irregularsToPlural = map[string]string{
	// -o words that take -es
	"potato":  "potatoes",
	"tomato":  "tomatoes",
	"hero":    "heroes",
	"mango":   "mangoes",
	// -f/-fe words
	"leaf":  "leaves",
	"loaf":  "loaves",
	"half":  "halves",
	"shelf": "shelves",
	"knife": "knives",
	"wife":  "wives",
	"life":  "lives",
	"wolf":  "wolves",
	// Common English irregulars
	"man":    "men",
	"woman":  "women",
	"child":  "children",
	"person": "people",
	"tooth":  "teeth",
	"foot":   "feet",
	"goose":  "geese",
	"mouse":  "mice",
	// Invariant / uncountable culinary nouns
	"flour":   "flour",
	"sugar":   "sugar",
	"salt":    "salt",
	"pepper":  "pepper",
	"rice":    "rice",
	"garlic":  "garlic",
	"ginger":  "ginger",
	"honey":   "honey",
	"oil":     "oil",
	"butter":  "butter",
	"milk":    "milk",
	"water":   "water",
	"vinegar": "vinegar",
}

// irregularsToSingular is the reverse of irregularsToPlural.
var irregularsToSingular = func() map[string]string {
	m := make(map[string]string, len(irregularsToPlural))
	for sg, pl := range irregularsToPlural {
		if sg != pl { // skip invariants
			m[pl] = sg
		}
	}
	return m
}()

func hasSuffix(s, suffix string) bool { return strings.HasSuffix(s, suffix) }

func hasSuffixAny(s string, suffixes ...string) bool {
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}

func isConsonant(r rune) bool {
	return !strings.ContainsRune("aeiou", unicode.ToLower(r))
}
