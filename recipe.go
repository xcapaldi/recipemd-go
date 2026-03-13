package recipemd

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type (
	Recipe struct {
		Title            string            `json:"title"`
		Description      *string           `json:"description"`
		Yields           []Amount          `json:"yields"`
		Tags             []string          `json:"tags"`
		Ingredients      []Ingredient      `json:"ingredients"`
		IngredientGroups []IngredientGroup `json:"ingredient_groups"`
		Instructions     *string           `json:"instructions"`
	}

	Ingredient struct {
		Name   string  `json:"name"`
		Amount *Amount `json:"amount"`
		Link   *string `json:"link"`
	}

	IngredientGroup struct {
		Title            string            `json:"title"`
		Ingredients      []Ingredient      `json:"ingredients"`
		IngredientGroups []IngredientGroup `json:"ingredient_groups"`
	}

	Amount struct {
		Factor float64  `json:"factor"`
		Unit   *string `json:"unit"`
	}
)

// MarshalJSON is a custom marshaler for an Amount.
func (a Amount) MarshalJSON() ([]byte, error) {
	s := a.FormatFactor(3)
	if a.Unit != nil {
		return fmt.Appendf([]byte{}, `{"factor":%q,"unit":%q}`, s, *a.Unit), nil
	}
	return fmt.Appendf([]byte{}, `{"factor":%q,"unit":null}`, s), nil
}

// FormatFactor formats the factor as a string. rounding < 0 means no rounding.
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

// ScaleForYield tries to find a matching yield in the recipe and uses that to
// find the overall scaling factor. If the desired yield is unitless, first try
// to match a recipe yield that also has no unit. If there is none, assume the
// scaling factor is for the implicit 1x recipe yield. For example, scale the 
// whole recipe by 2x.
func (r *Recipe) ScaleForYield(desiredYield Amount) error {
  for _, y := range r.Yields {
    if y.Unit == nil && desiredYield.Unit == nil {
      r.Scale(desiredYield.Factor/y.Factor)
      return nil
    }
    if y.Unit != nil && desiredYield.Unit != nil && *y.Unit == *desiredYield.Unit {
      r.Scale(desiredYield.Factor/y.Factor)
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

// Scale Recipe by factor
func (r *Recipe) Scale(factor float64) {
  for i := range(r.Yields) {
    r.Yields[i].Scale(factor)
  }
  for j := range(r.Ingredients) {
    r.Ingredients[j].Scale(factor)
  }
  for k := range(r.IngredientGroups) {
    r.IngredientGroups[k].Scale(factor)
  }
}

// Scale Amount by factor
func (a *Amount) Scale(factor float64) {
  a.Factor *= factor
}

// Scale Ingredient by factor
func (i *Ingredient) Scale(factor float64) {
  if i.Amount != nil {
    i.Amount.Scale(factor)
  }
}

// Scale IngredientGroup by factor
func (g *IngredientGroup) Scale(factor float64) {
  for i := range(g.Ingredients) {
    g.Ingredients[i].Scale(factor)
  }
  for j := range(g.IngredientGroups) {
    g.IngredientGroups[j].Scale(factor)
  }
}

// Serialize formats an Amount as a string.
func (a Amount) Serialize(rounding int) string {
	s := a.FormatFactor(rounding)
	if a.Unit != nil {
		return s + " " + *a.Unit
	}
	return s
}

// Serialize formats an Ingredient as a string.
func (i Ingredient) Serialize(rounding int) string {
	if i.Amount != nil {
		return i.Amount.Serialize(rounding) + " " + i.Name
	}
	return i.Name
}

// LeafIngredients returns all ingredients including those in groups.
func (r *Recipe) LeafIngredients() []Ingredient {
	var result []Ingredient
	result = append(result, r.Ingredients...)
	for _, g := range r.IngredientGroups {
		result = append(result, g.LeafIngredients()...)
	}
	return result
}

// LeafIngredients returns all ingredients in the group and subgroups.
func (g *IngredientGroup) LeafIngredients() []Ingredient {
	var result []Ingredient
	result = append(result, g.Ingredients...)
	for _, sub := range g.IngredientGroups {
		result = append(result, sub.LeafIngredients()...)
	}
	return result
}


