package recipemd

import (
	"regexp"
	"sort"
	"strings"
)

// InlineIngredients returns a copy of the recipe's Instructions with ingredient
// amounts injected before each matching ingredient name.
//
// For each ingredient that has an amount, every occurrence of the ingredient's
// name in the instructions (matched case-insensitively at word boundaries) is
// replaced with "<amount> <name>". For example, if an ingredient is
// "*1/2 tsp* cinnamon" and the instructions contain "add cinnamon and mix",
// the result is "add 1/2 tsp cinnamon and mix". The original casing of the
// matched word(s) in the instructions text is preserved.
//
// Multi-word ingredient names (e.g. "olive oil") are matched as complete
// phrases. When one ingredient name is a substring of another (e.g. "sugar"
// and "brown sugar"), the longer name is matched and replaced first so that
// the shorter name is not spuriously injected inside the already-replaced text.
//
// Amounts are formatted with the given rounding via [Amount.Serialize].
// Ingredients without an amount are not injected.
//
// Returns nil when r.Instructions is nil.
func InlineIngredients(r *Recipe, rounding int) *string {
	if r.Instructions == nil {
		return nil
	}

	type entry struct {
		name   string
		amount string
	}

	ingredients := r.LeafIngredients()
	entries := make([]entry, 0, len(ingredients))
	for _, ing := range ingredients {
		if ing.Amount == nil {
			continue
		}
		entries = append(entries, entry{
			name:   ing.Name,
			amount: ing.Amount.Serialize(rounding),
		})
	}

	if len(entries) == 0 {
		s := *r.Instructions
		return &s
	}

	// Sort longest first so that "brown sugar" is listed before "sugar" in the
	// alternation, ensuring the longer phrase wins when both could match.
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].name) > len(entries[j].name)
	})

	// Build a single combined regex so all replacements happen in one pass.
	// A single pass prevents a short ingredient (e.g. "sugar") from matching
	// inside text that was already injected for a longer one ("brown sugar").
	alts := make([]string, len(entries))
	for i, e := range entries {
		alts[i] = regexp.QuoteMeta(e.name)
	}
	pattern := `(?i)\b(?:` + strings.Join(alts, "|") + `)\b`
	re := regexp.MustCompile(pattern)

	// Build a lookup map keyed by lowercased ingredient name.
	amountFor := make(map[string]string, len(entries))
	for _, e := range entries {
		amountFor[strings.ToLower(e.name)] = e.amount
	}

	result := re.ReplaceAllStringFunc(*r.Instructions, func(match string) string {
		amount := amountFor[strings.ToLower(match)]
		return amount + " " + match
	})

	return &result
}
