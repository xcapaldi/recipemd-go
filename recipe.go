package recipemd

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type (
	// Recipe is the top-level representation of a parsed RecipeMD document.
	//
	// Title is always present. Description, Instructions, and individual
	// Amount.Unit values are optional and represented as pointers; a nil
	// pointer means the field was absent in the source document.
	// Yields, Tags, Ingredients, and IngredientGroups are initialised to
	// empty (non-nil) slices by [Parser.Parse].
	Recipe struct {
		// Title is the recipe name, taken from the H1 heading.
		Title string `json:"title"`
		// Description is the optional free-form text between the title and
		// the tags/yields lines, preserved as raw markdown. Nil when absent.
		Description *string `json:"description"`
		// Yields lists the recipe's yield amounts (e.g. "12 cookies",
		// "1 loaf"). Multiple yields with different units are allowed.
		Yields []Amount `json:"yields"`
		// Tags is the comma-separated list of category tags from the italic
		// paragraph in the preamble (e.g. "vegan, gluten-free").
		Tags []string `json:"tags"`
		// Ingredients holds the flat, top-level ingredient list that appears
		// directly after the first thematic break.
		Ingredients []Ingredient `json:"ingredients"`
		// IngredientGroups holds named sections of ingredients introduced by
		// headings in the ingredient section. Groups may be nested.
		IngredientGroups []IngredientGroup `json:"ingredient_groups"`
		// Instructions is the optional free-form text after the second
		// thematic break, preserved as raw markdown. Nil when absent.
		Instructions *string `json:"instructions"`
	}

	// Ingredient represents a single item in a recipe's ingredient list.
	//
	// Every ingredient must have a Name. Amount and Link are optional: Amount
	// is present when the ingredient line starts with an italic quantity (e.g.
	// "*200 g* flour"), and Link is present when the entire ingredient name is
	// a hyperlink to another recipe file.
	Ingredient struct {
		// Name is the ingredient's display name (e.g. "flour", "olive oil").
		Name string `json:"name"`
		// Amount is the optional quantity for this ingredient. Nil when the
		// ingredient has no amount specified.
		Amount *Amount `json:"amount"`
		// Link is the optional URL or relative file path of a linked recipe.
		// Nil when the ingredient is not a link.
		Link *string `json:"link"`
	}

	// IngredientGroup is a named section of ingredients within a recipe,
	// introduced by a heading in the ingredient part of the document.
	//
	// Groups may contain both direct ingredients and nested sub-groups,
	// mirroring the heading hierarchy in the source document.
	IngredientGroup struct {
		// Title is the heading text that names this group.
		Title string `json:"title"`
		// Ingredients is the flat list of ingredients directly inside this group.
		Ingredients []Ingredient `json:"ingredients"`
		// IngredientGroups holds any nested sub-groups whose headings are at a
		// deeper level than this group's heading.
		IngredientGroups []IngredientGroup `json:"ingredient_groups"`
	}

	// Amount represents a measured quantity consisting of a numeric factor and
	// an optional unit of measurement.
	//
	// The Factor is always set. Unit is nil when the amount is unitless
	// (e.g. "3 eggs" has Factor=3 and Unit=nil).
	Amount struct {
		// Factor is the numeric value of the amount (e.g. 1.5 for "1.5 cups").
		Factor float64 `json:"factor"`
		// Unit is the optional measurement unit (e.g. "cups", "g", "ml").
		// Nil when the amount has no unit.
		Unit *string `json:"unit"`
	}
)

// MarshalJSON implements [encoding/json.Marshaler] for Amount.
//
// The numeric factor is encoded as a quoted string rounded to three decimal
// places with trailing zeros removed (e.g. "1.5"), so that JSON consumers
// receive a human-readable value rather than a raw float64. The unit field is
// always present, set to null when [Amount.Unit] is nil.
func (a Amount) MarshalJSON() ([]byte, error) {
	s := a.FormatFactor(3)
	if a.Unit != nil {
		return fmt.Appendf([]byte{}, `{"factor":%q,"unit":%q}`, s, *a.Unit), nil
	}
	return fmt.Appendf([]byte{}, `{"factor":%q,"unit":null}`, s), nil
}

// FormatFactor formats the numeric factor as a decimal string.
//
// When rounding is zero or positive the value is rounded to that many decimal
// places and trailing zeros (and a trailing decimal point) are removed.
// When rounding is negative the full precision of the underlying float64 is
// used. For example, FormatFactor(2) on a factor of 1.500 returns "1.5".
func (a Amount) FormatFactor(rounding int) string {
	if rounding < 0 {
		return strconv.FormatFloat(a.Factor, 'f', -1, 64)
	}
	rounded := math.Round(a.Factor*math.Pow(10, float64(rounding))) / math.Pow(10, float64(rounding))
	s := strconv.FormatFloat(rounded, 'f', rounding, 64)
	if rounding > 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// ScaleForYield scales the recipe so that its yield matches desiredYield.
//
// The method searches [Recipe.Yields] for an entry whose unit matches the unit
// of desiredYield, then calls [Recipe.Scale] with the derived ratio. Unit
// matching is case-sensitive and exact.
//
// If desiredYield is unitless (Unit == nil) the method first looks for a
// unitless entry in Yields. When none exists it falls back to treating
// desiredYield.Factor as a direct multiplier (i.e. it scales the whole recipe
// by that factor).
//
// ScaleForYield returns an error when desiredYield has a unit that does not
// match any yield in the recipe.
func (r *Recipe) ScaleForYield(desiredYield Amount) error {
	for _, y := range r.Yields {
		if y.Unit == nil && desiredYield.Unit == nil {
			r.Scale(desiredYield.Factor / y.Factor)
			return nil
		}
		if y.Unit != nil && desiredYield.Unit != nil && *y.Unit == *desiredYield.Unit {
			r.Scale(desiredYield.Factor / y.Factor)
			return nil
		}
	}

	// fallback on scaling the whole recipe
	if desiredYield.Unit == nil {
		r.Scale(desiredYield.Factor)
		return nil
	}

	return errors.New("no matching yield unit found")
}

// Scale multiplies every ingredient amount and every yield in the recipe by
// factor. The recipe title, description, tags, and instructions are unchanged.
func (r *Recipe) Scale(factor float64) {
	for i := range r.Yields {
		r.Yields[i].Scale(factor)
	}
	for j := range r.Ingredients {
		r.Ingredients[j].Scale(factor)
	}
	for k := range r.IngredientGroups {
		r.IngredientGroups[k].Scale(factor)
	}
}

// Scale multiplies the amount's factor by factor.
func (a *Amount) Scale(factor float64) {
	a.Factor *= factor
}

// Scale scales the ingredient's amount by factor. It is a no-op when the
// ingredient has no amount.
func (i *Ingredient) Scale(factor float64) {
	if i.Amount != nil {
		i.Amount.Scale(factor)
	}
}

// Scale recursively scales all ingredients and nested sub-groups by factor.
func (g *IngredientGroup) Scale(factor float64) {
	for i := range g.Ingredients {
		g.Ingredients[i].Scale(factor)
	}
	for j := range g.IngredientGroups {
		g.IngredientGroups[j].Scale(factor)
	}
}

// Serialize formats the amount as a human-readable string.
//
// The factor is formatted with [Amount.FormatFactor] using the given rounding.
// When a unit is present it is appended after a space (e.g. "1.5 cups").
// When there is no unit only the formatted number is returned (e.g. "3").
func (a Amount) Serialize(rounding int) string {
	s := a.FormatFactor(rounding)
	if a.Unit != nil {
		return s + " " + *a.Unit
	}
	return s
}

// Serialize formats the ingredient as a human-readable string.
//
// When an amount is present it is serialised (via [Amount.Serialize]) and
// prepended to the name, separated by a space (e.g. "200 g flour").
// When there is no amount only the name is returned (e.g. "salt").
func (i Ingredient) Serialize(rounding int) string {
	if i.Amount != nil {
		return i.Amount.Serialize(rounding) + " " + i.Name
	}
	return i.Name
}

// LeafIngredients returns a flat list of every ingredient in the recipe,
// including those nested inside ingredient groups and sub-groups. The
// top-level [Recipe.Ingredients] slice is listed first, followed by the
// ingredients from each [Recipe.IngredientGroups] entry in order.
func (r *Recipe) LeafIngredients() []Ingredient {
	var result []Ingredient
	result = append(result, r.Ingredients...)
	for _, g := range r.IngredientGroups {
		result = append(result, g.LeafIngredients()...)
	}
	return result
}

// LeafIngredients returns a flat list of every ingredient in the group,
// including those in nested sub-groups, in depth-first order.
func (g *IngredientGroup) LeafIngredients() []Ingredient {
	var result []Ingredient
	result = append(result, g.Ingredients...)
	for _, sub := range g.IngredientGroups {
		result = append(result, sub.LeafIngredients()...)
	}
	return result
}
