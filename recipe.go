package recipemd

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
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

// RenderMarkdown formats a Recipe as RecipeMD markdown.
func (r *Recipe) RenderMarkdown(rounding int) string {
	var sb strings.Builder

	sb.WriteString("# ")
	sb.WriteString(r.Title)
	sb.WriteString("\n")

	if r.Description != nil && *r.Description != "" {
		sb.WriteString("\n")
		sb.WriteString(*r.Description)
		sb.WriteString("\n")
	}

	if len(r.Tags) > 0 {
		sb.WriteString("\n*")
		sb.WriteString(strings.Join(r.Tags, ", "))
		sb.WriteString("*\n")
	}

	if len(r.Yields) > 0 {
		sb.WriteString("\n**")
		yields := make([]string, len(r.Yields))
		for i, y := range r.Yields {
			yields[i] = y.Serialize(rounding)
		}
		sb.WriteString(strings.Join(yields, ", "))
		sb.WriteString("**\n")
	}

	sb.WriteString("\n---\n")

	renderMarkdownIngredientList(&sb, r.Ingredients, rounding)
	renderMarkdownIngredientGroups(&sb, r.IngredientGroups, 2, rounding)

	if r.Instructions != nil && *r.Instructions != "" {
		sb.WriteString("\n---\n\n")
		sb.WriteString(*r.Instructions)
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderMarkdownIngredientList(sb *strings.Builder, ingredients []Ingredient, rounding int) {
	if len(ingredients) == 0 {
		return
	}
	sb.WriteString("\n")
	for _, ing := range ingredients {
		sb.WriteString("- ")
		if ing.Amount != nil {
			sb.WriteString("*")
			sb.WriteString(ing.Amount.Serialize(rounding))
			sb.WriteString("* ")
		}
		if ing.Link != nil {
			sb.WriteString("[")
			sb.WriteString(ing.Name)
			sb.WriteString("](")
			sb.WriteString(*ing.Link)
			sb.WriteString(")")
		} else {
			sb.WriteString(ing.Name)
		}
		sb.WriteString("\n")
	}
}

func renderMarkdownIngredientGroups(sb *strings.Builder, groups []IngredientGroup, level int, rounding int) {
	for _, g := range groups {
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("#", level))
		sb.WriteString(" ")
		sb.WriteString(g.Title)
		sb.WriteString("\n")
		renderMarkdownIngredientList(sb, g.Ingredients, rounding)
		renderMarkdownIngredientGroups(sb, g.IngredientGroups, level+1, rounding)
	}
}

// Flatten resolves linked ingredients by parsing referenced recipe files
// and inlining their ingredients. Links resolved relative to recipeFile dir.
func (r *Recipe) Flatten(recipeFile string) error {
	baseDir := filepath.Dir(recipeFile)
  ingredients, err := flattenIngredients(r.Ingredients, baseDir)
  if err != nil {
    return fmt.Errorf("flattenIngredients: %w", err)
  }
	r.Ingredients = ingredients
  groups, err := flattenIngredientGroups(r.IngredientGroups, baseDir)
  if err != nil {
    return fmt.Errorf("flattenIngredientGroups: %w", err)
  }
	r.IngredientGroups = groups

	return nil
}

func flattenIngredients(ingredients []Ingredient, baseDir string) ([]Ingredient, error) {
  result := make([]Ingredient, 0, len(ingredients))
	for _, ing := range ingredients {
		if ing.Link != nil {
			resolved, err := resolveLinkedRecipe(*ing.Link, baseDir, &ing)
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

func flattenIngredientGroups(groups []IngredientGroup, baseDir string) ([]IngredientGroup, error) {
  result := make([]IngredientGroup, 0, len(groups))
	for _, g := range groups {
    ingredients, err := flattenIngredients(g.Ingredients, baseDir)
    if err != nil {
      return nil, fmt.Errorf("flattenIngredients:%w", err)
    }
    groups, err := flattenIngredientGroups(g.IngredientGroups, baseDir)
    if err != nil {
      return nil, fmt.Errorf("flattenIngredientGroups: %w" , err)
    }
		flat := IngredientGroup{
			Title:            g.Title,
			Ingredients:      ingredients,
			IngredientGroups: groups,
		}
		result = append(result, flat)
	}

	return result, nil
}

func resolveLinkedRecipe(link string, baseDir string, parent *Ingredient) ([]Ingredient, error) {
	if strings.Contains(link, "://") {
		return []Ingredient{*parent}, nil
	}

	path := filepath.Join(baseDir, link)
	data, err := os.ReadFile(path)
	if err != nil {
    return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	linked, err := ParseRecipe(data)
	if err != nil {
    return nil, fmt.Errorf("ParseRecipe: %w", err)
	}

	if parent.Amount != nil && len(linked.Yields) > 0 {
		if err := linked.ScaleForYield(*parent.Amount); err != nil {
      return nil, fmt.Errorf("linked.ScaleForYield: %w", err)
		}
	}

	linkedDir := filepath.Dir(path)
	flatIngredients, err := flattenIngredients(linked.Ingredients, linkedDir)
  if err != nil {
    return nil, fmt.Errorf("flattenIngredients: %w", err)
  }
	for _, g := range linked.IngredientGroups {
    ingredients, err := flattenIngredients(g.Ingredients, linkedDir)
    if err != nil {
      return nil, fmt.Errorf("flattenIngredients: %w", err)
    }
    flatIngredients = append(flatIngredients, ingredients...)
    groupIngredients, err := flattenGroupIngredients(g.IngredientGroups, linkedDir)
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

func flattenGroupIngredients(groups []IngredientGroup, baseDir string) ([]Ingredient, error) {
  result := make([]Ingredient, 0, len(groups))
	for _, g := range groups {
    ingredients, err := flattenIngredients(g.Ingredients, baseDir)
    if err != nil {
      return nil, fmt.Errorf("flattenIngredients: %w", err)
    }
    result = append(result, ingredients...)
    groupIngredients, err := flattenGroupIngredients(g.IngredientGroups, baseDir)
    if err != nil {
      return nil, fmt.Errorf("flattenGroupIngredients: %w", err)
    }
    result = append(result, groupIngredients...)
	}
	return result, nil
}
