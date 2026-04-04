package recipemd

import (
	"strings"
	"text/template"
)

func renderFuncMap(rounding int) template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"serializeAmount": func(a *Amount) string {
			if a == nil {
				return ""
			}
			return a.Serialize(rounding)
		},
		"serializeIngredientAmount": func(ing Ingredient) string {
			if len(ing.CompoundAmount) > 0 {
				return SerializeCompound(ing.CompoundAmount)
			}
			if ing.Amount != nil {
				return ing.Amount.Serialize(rounding)
			}
			return ""
		},
		"serializeYields": func(yields []Amount) string {
			s := make([]string, len(yields))
			for i, y := range yields {
				s[i] = y.Serialize(rounding)
			}
			return strings.Join(s, ", ")
		},
	}
}
