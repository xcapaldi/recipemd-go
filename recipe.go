package recipemd

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
		Factor string  `json:"factor"`
		Unit   *string `json:"unit"`
	}
)
