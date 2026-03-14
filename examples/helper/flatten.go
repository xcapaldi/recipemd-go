package helper

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	recipemd "github.com/xcapaldi/recipemd-go"
)

// Flatten resolves all locally-linked ingredients in r, inlining them in place.
// recipeFile is the path to the recipe file, used to resolve relative links.
// HTTP(S) links are left as-is.
func Flatten(p *recipemd.Parser, r *recipemd.Recipe, recipeFile string) error {
	baseDir := filepath.Dir(recipeFile)
	ingredients, err := flattenIngredients(p, r.Ingredients, baseDir)
	if err != nil {
		return fmt.Errorf("flattenIngredients: %w", err)
	}
	r.Ingredients = ingredients
	groups, err := flattenIngredientGroups(p, r.IngredientGroups, baseDir)
	if err != nil {
		return fmt.Errorf("flattenIngredientGroups: %w", err)
	}
	r.IngredientGroups = groups
	return nil
}

func flattenIngredients(p *recipemd.Parser, ingredients []recipemd.Ingredient, baseDir string) ([]recipemd.Ingredient, error) {
	result := make([]recipemd.Ingredient, 0, len(ingredients))
	for _, ing := range ingredients {
		if ing.Link != nil {
			resolved, err := resolveLinkedRecipe(p, *ing.Link, baseDir, &ing)
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

func flattenIngredientGroups(p *recipemd.Parser, groups []recipemd.IngredientGroup, baseDir string) ([]recipemd.IngredientGroup, error) {
	result := make([]recipemd.IngredientGroup, 0, len(groups))
	for _, g := range groups {
		ingredients, err := flattenIngredients(p, g.Ingredients, baseDir)
		if err != nil {
			return nil, fmt.Errorf("flattenIngredients: %w", err)
		}
		subGroups, err := flattenIngredientGroups(p, g.IngredientGroups, baseDir)
		if err != nil {
			return nil, fmt.Errorf("flattenIngredientGroups: %w", err)
		}
		result = append(result, recipemd.IngredientGroup{
			Title:            g.Title,
			Ingredients:      ingredients,
			IngredientGroups: subGroups,
		})
	}
	return result, nil
}

func resolveLinkedRecipe(p *recipemd.Parser, link string, baseDir string, parent *recipemd.Ingredient) ([]recipemd.Ingredient, error) {
	if strings.Contains(link, "://") {
		return []recipemd.Ingredient{*parent}, nil
	}

	path := filepath.Join(baseDir, link)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	linked, err := p.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Parse: %w", err)
	}

	if parent.Amount != nil && len(linked.Yields) > 0 {
		if err := linked.ScaleForYield(*parent.Amount); err != nil {
			return nil, fmt.Errorf("linked.ScaleForYield: %w", err)
		}
	}

	linkedDir := filepath.Dir(path)
	flatIngredients, err := flattenIngredients(p, linked.Ingredients, linkedDir)
	if err != nil {
		return nil, fmt.Errorf("flattenIngredients: %w", err)
	}
	for _, g := range linked.IngredientGroups {
		ingredients, err := flattenIngredients(p, g.Ingredients, linkedDir)
		if err != nil {
			return nil, fmt.Errorf("flattenIngredients: %w", err)
		}
		flatIngredients = append(flatIngredients, ingredients...)
		groupIngredients, err := flattenGroupIngredients(p, g.IngredientGroups, linkedDir)
		if err != nil {
			return nil, fmt.Errorf("flattenGroupIngredients: %w", err)
		}
		flatIngredients = append(flatIngredients, groupIngredients...)
	}

	if len(flatIngredients) == 0 {
		return []recipemd.Ingredient{*parent}, nil
	}
	return flatIngredients, nil
}

func flattenGroupIngredients(p *recipemd.Parser, groups []recipemd.IngredientGroup, baseDir string) ([]recipemd.Ingredient, error) {
	result := make([]recipemd.Ingredient, 0, len(groups))
	for _, g := range groups {
		ingredients, err := flattenIngredients(p, g.Ingredients, baseDir)
		if err != nil {
			return nil, fmt.Errorf("flattenIngredients: %w", err)
		}
		result = append(result, ingredients...)
		groupIngredients, err := flattenGroupIngredients(p, g.IngredientGroups, baseDir)
		if err != nil {
			return nil, fmt.Errorf("flattenGroupIngredients: %w", err)
		}
		result = append(result, groupIngredients...)
	}
	return result, nil
}
