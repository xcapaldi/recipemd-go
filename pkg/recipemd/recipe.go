package recipemd

import (
	"fmt"
	"strconv"
)

// Scale scales all ingredient amounts and yields by the given factor.
// For example, Scale(2.0) doubles all amounts.
func (r *Recipe) Scale(factor float64) {
	// Scale top-level ingredients
	for i := range r.Ingredients {
		scaleIngredient(&r.Ingredients[i], factor)
	}

	// Scale ingredient groups
	scaleIngredientGroups(r.IngredientGroups, factor)

	// Scale yields
	for i := range r.Yields {
		scaleYield(&r.Yields[i], factor)
	}
}

// scaleIngredient scales a single ingredient's amount.
func scaleIngredient(ing *Ingredient, factor float64) {
	if ing.Amount == nil {
		return
	}

	currentFactor, err := strconv.ParseFloat(ing.Amount.Factor, 64)
	if err != nil {
		return
	}

	newFactor := currentFactor * factor
	ing.Amount.Factor = formatFactor(newFactor)
}

// scaleYield scales a single yield's factor.
func scaleYield(y *Yield, factor float64) {
	currentFactor, err := strconv.ParseFloat(y.Factor, 64)
	if err != nil {
		return
	}

	newFactor := currentFactor * factor
	y.Factor = formatFactor(newFactor)
}

// scaleIngredientGroups recursively scales all ingredients in groups.
func scaleIngredientGroups(groups []IngredientGroup, factor float64) {
	for i := range groups {
		for j := range groups[i].Ingredients {
			scaleIngredient(&groups[i].Ingredients[j], factor)
		}
		scaleIngredientGroups(groups[i].IngredientGroups, factor)
	}
}

// Validate validates the recipe structure.
// Returns nil if valid, or an error describing the validation failure.
func (r *Recipe) Validate() error {
	// Title is required
	if r.Title == "" {
		return fmt.Errorf("recipe must have a title")
	}

	// Validate all ingredient amounts
	for _, ing := range r.Ingredients {
		if err := validateIngredient(ing); err != nil {
			return err
		}
	}

	// Validate ingredient groups
	if err := validateIngredientGroups(r.IngredientGroups); err != nil {
		return err
	}

	// Validate yields
	for _, y := range r.Yields {
		if err := validateYield(y); err != nil {
			return err
		}
	}

	return nil
}

// validateIngredient validates a single ingredient.
func validateIngredient(ing Ingredient) error {
	if ing.Name == "" {
		return fmt.Errorf("ingredient has no name")
	}

	if ing.Amount != nil {
		if ing.Amount.Factor == "" {
			return fmt.Errorf("ingredient '%s' has amount with no factor", ing.Name)
		}
		_, err := strconv.ParseFloat(ing.Amount.Factor, 64)
		if err != nil {
			return fmt.Errorf("ingredient '%s' has invalid factor '%s'", ing.Name, ing.Amount.Factor)
		}
	}

	return nil
}

// validateYield validates a single yield.
func validateYield(y Yield) error {
	if y.Factor == "" {
		return fmt.Errorf("yield has no factor")
	}
	_, err := strconv.ParseFloat(y.Factor, 64)
	if err != nil {
		return fmt.Errorf("yield has invalid factor '%s'", y.Factor)
	}
	return nil
}

// validateIngredientGroups recursively validates ingredient groups.
func validateIngredientGroups(groups []IngredientGroup) error {
	for _, g := range groups {
		if g.Title == "" {
			return fmt.Errorf("ingredient group has no title")
		}

		for _, ing := range g.Ingredients {
			if err := validateIngredient(ing); err != nil {
				return err
			}
		}

		if err := validateIngredientGroups(g.IngredientGroups); err != nil {
			return err
		}
	}
	return nil
}

// Clone creates a deep copy of the recipe.
func (r *Recipe) Clone() *Recipe {
	clone := &Recipe{
		Title: r.Title,
		Tags:  make([]string, len(r.Tags)),
	}

	copy(clone.Tags, r.Tags)

	if r.Description != nil {
		desc := *r.Description
		clone.Description = &desc
	}

	if r.Instructions != nil {
		instr := *r.Instructions
		clone.Instructions = &instr
	}

	clone.Yields = make([]Yield, len(r.Yields))
	for i, y := range r.Yields {
		clone.Yields[i] = Yield{Factor: y.Factor}
		if y.Unit != nil {
			unit := *y.Unit
			clone.Yields[i].Unit = &unit
		}
	}

	clone.Ingredients = cloneIngredients(r.Ingredients)
	clone.IngredientGroups = cloneIngredientGroups(r.IngredientGroups)

	return clone
}

// cloneIngredients creates a deep copy of ingredients.
func cloneIngredients(ingredients []Ingredient) []Ingredient {
	result := make([]Ingredient, len(ingredients))
	for i, ing := range ingredients {
		result[i] = Ingredient{Name: ing.Name}
		if ing.Amount != nil {
			result[i].Amount = &Amount{Factor: ing.Amount.Factor}
			if ing.Amount.Unit != nil {
				unit := *ing.Amount.Unit
				result[i].Amount.Unit = &unit
			}
		}
		if ing.Link != nil {
			link := *ing.Link
			result[i].Link = &link
		}
	}
	return result
}

// cloneIngredientGroups creates a deep copy of ingredient groups.
func cloneIngredientGroups(groups []IngredientGroup) []IngredientGroup {
	result := make([]IngredientGroup, len(groups))
	for i, g := range groups {
		result[i] = IngredientGroup{
			Title:            g.Title,
			Ingredients:      cloneIngredients(g.Ingredients),
			IngredientGroups: cloneIngredientGroups(g.IngredientGroups),
		}
	}
	return result
}
