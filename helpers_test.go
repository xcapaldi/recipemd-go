package recipemd

func ptr[T any](v T) *T { return &v }
