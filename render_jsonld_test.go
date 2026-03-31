package recipemd

import (
	"encoding/json"
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

func TestRenderJSONLD(t *testing.T) {
	t.Parallel()
	p := NewParser()

	t.Run("minimal recipe", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Toast",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "bread"}},
			IngredientGroups: []IngredientGroup{},
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if parsed["@context"] != "https://schema.org/" {
			t.Errorf("@context = %v", parsed["@context"])
		}
		if parsed["@type"] != "Recipe" {
			t.Errorf("@type = %v", parsed["@type"])
		}
		if parsed["name"] != "Toast" {
			t.Errorf("name = %v", parsed["name"])
		}
		ings, ok := parsed["recipeIngredient"].([]any)
		if !ok || len(ings) != 1 {
			t.Fatalf("recipeIngredient = %v", parsed["recipeIngredient"])
		}
		if ings[0] != "bread" {
			t.Errorf("recipeIngredient[0] = %v", ings[0])
		}
	})

	t.Run("omits empty optional fields", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Empty",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		s := string(got)
		if strings.Contains(s, "description") {
			t.Error("should omit empty description")
		}
		if strings.Contains(s, "recipeYield") {
			t.Error("should omit empty recipeYield")
		}
		if strings.Contains(s, "keywords") {
			t.Error("should omit empty keywords")
		}
		if strings.Contains(s, "recipeInstructions") {
			t.Error("should omit empty recipeInstructions")
		}
		if strings.Contains(s, "recipeIngredient") {
			t.Error("should omit empty recipeIngredient")
		}
	})

	t.Run("keywords is comma-separated string", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Tagged",
			Tags:             []string{"vegan", "quick", "healthy"},
			Yields:           []Amount{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		kw, ok := parsed["keywords"].(string)
		if !ok {
			t.Fatalf("keywords should be a string, got %T: %v", parsed["keywords"], parsed["keywords"])
		}
		if kw != "vegan, quick, healthy" {
			t.Errorf("keywords = %q, want %q", kw, "vegan, quick, healthy")
		}
	})

	t.Run("description is plain text", func(t *testing.T) {
		t.Parallel()
		desc := "A recipe with **bold** and *italic* text."
		r := &Recipe{
			Title:            "Desc",
			Description:      &desc,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		d := parsed["description"].(string)
		if strings.Contains(d, "**") || strings.Contains(d, "*italic*") {
			t.Errorf("description should be plain text, got %q", d)
		}
		if !strings.Contains(d, "bold") || !strings.Contains(d, "italic") {
			t.Errorf("description should contain text content, got %q", d)
		}
	})

	t.Run("recipeYield from first yield", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Y",
			Yields:           []Amount{{Factor: 4, Unit: ptr("servings")}, {Factor: 500, Unit: ptr("g")}},
			Tags:             []string{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		if parsed["recipeYield"] != "4 servings" {
			t.Errorf("recipeYield = %v", parsed["recipeYield"])
		}
	})

	t.Run("ingredients include grouped", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:  "G",
			Yields: []Amount{},
			Tags:   []string{},
			Ingredients: []Ingredient{
				{Name: "salt"},
				{Name: "flour", Amount: &Amount{Factor: 200, Unit: ptr("g")}},
			},
			IngredientGroups: []IngredientGroup{
				{
					Title:            "Sauce",
					Ingredients:      []Ingredient{{Name: "tomato", Amount: &Amount{Factor: 3}}},
					IngredientGroups: []IngredientGroup{},
				},
			},
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		ings := parsed["recipeIngredient"].([]any)
		if len(ings) != 3 {
			t.Fatalf("expected 3 ingredients, got %d: %v", len(ings), ings)
		}
		if ings[0] != "salt" {
			t.Errorf("ingredient[0] = %v", ings[0])
		}
		if ings[1] != "200 g flour" {
			t.Errorf("ingredient[1] = %v", ings[1])
		}
		if ings[2] != "3 tomato" {
			t.Errorf("ingredient[2] = %v", ings[2])
		}
	})

	t.Run("instructions flat steps", func(t *testing.T) {
		t.Parallel()
		instructions := "Preheat oven to 200°C.\n\nMix ingredients.\n\nBake for 30 minutes."
		r := &Recipe{
			Title:            "Flat",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		steps := parsed["recipeInstructions"].([]any)
		if len(steps) != 3 {
			t.Fatalf("expected 3 steps, got %d: %v", len(steps), steps)
		}
		step0 := steps[0].(map[string]any)
		if step0["@type"] != "HowToStep" {
			t.Errorf("step[0] @type = %v", step0["@type"])
		}
		if step0["text"] != "Preheat oven to 200°C." {
			t.Errorf("step[0] text = %v", step0["text"])
		}
	})

	t.Run("instructions with sections", func(t *testing.T) {
		t.Parallel()
		instructions := "## Prepare Dough\n\nMix flour and water.\n\nKnead for 10 minutes.\n\n## Bake\n\nPreheat oven.\n\nBake at 200°C."
		r := &Recipe{
			Title:            "Sections",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		sections := parsed["recipeInstructions"].([]any)
		if len(sections) != 2 {
			t.Fatalf("expected 2 sections, got %d: %v", len(sections), sections)
		}

		sec0 := sections[0].(map[string]any)
		if sec0["@type"] != "HowToSection" {
			t.Errorf("section[0] @type = %v", sec0["@type"])
		}
		if sec0["name"] != "Prepare Dough" {
			t.Errorf("section[0] name = %v", sec0["name"])
		}
		items0 := sec0["itemListElement"].([]any)
		if len(items0) != 2 {
			t.Fatalf("section[0] expected 2 steps, got %d", len(items0))
		}

		sec1 := sections[1].(map[string]any)
		if sec1["name"] != "Bake" {
			t.Errorf("section[1] name = %v", sec1["name"])
		}
		items1 := sec1["itemListElement"].([]any)
		if len(items1) != 2 {
			t.Fatalf("section[1] expected 2 steps, got %d", len(items1))
		}
	})

	t.Run("instructions with image in paragraph", func(t *testing.T) {
		t.Parallel()
		instructions := "Mix well.\n\n![step photo](https://example.com/step1.jpg)\n\nBake it."
		r := &Recipe{
			Title:            "Img",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		steps := parsed["recipeInstructions"].([]any)
		// The image paragraph may become a step with image or a step with alt text.
		// At minimum we should have steps and one of them should reference the image URL.
		foundImage := false
		for _, s := range steps {
			step := s.(map[string]any)
			if img, ok := step["image"]; ok && img == "https://example.com/step1.jpg" {
				foundImage = true
			}
		}
		if !foundImage {
			t.Errorf("expected an image URL in steps, got: %s", got)
		}
	})

	t.Run("instructions strips markdown formatting", func(t *testing.T) {
		t.Parallel()
		instructions := "Mix **thoroughly** with a *wooden* spoon."
		r := &Recipe{
			Title:            "MD",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		steps := parsed["recipeInstructions"].([]any)
		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}
		step := steps[0].(map[string]any)
		txt := step["text"].(string)
		if strings.Contains(txt, "**") || strings.Contains(txt, "*wooden*") {
			t.Errorf("step text should be plain text, got %q", txt)
		}
		if !strings.Contains(txt, "thoroughly") || !strings.Contains(txt, "wooden") {
			t.Errorf("step text should preserve word content, got %q", txt)
		}
	})

	t.Run("instructions with list items", func(t *testing.T) {
		t.Parallel()
		instructions := "Follow these steps:\n\n- First do this\n- Then do that\n- Finally done"
		r := &Recipe{
			Title:            "ListInst",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		steps := parsed["recipeInstructions"].([]any)
		// Should have at least the paragraph + 3 list items = 4 steps.
		if len(steps) < 4 {
			t.Errorf("expected at least 4 steps, got %d: %s", len(steps), got)
		}
	})

	t.Run("full recipe", func(t *testing.T) {
		t.Parallel()
		desc := "A **delicious** recipe for testing."
		instructions := "## Prep\n\nChop vegetables.\n\n## Cook\n\nStir fry everything."
		r := &Recipe{
			Title:       "Stir Fry",
			Description: &desc,
			Yields:      []Amount{{Factor: 2, Unit: ptr("servings")}},
			Tags:        []string{"asian", "quick", "vegetarian"},
			Ingredients: []Ingredient{
				{Name: "tofu", Amount: &Amount{Factor: 200, Unit: ptr("g")}},
				{Name: "soy sauce", Amount: &Amount{Factor: 2, Unit: ptr("tbsp")}},
			},
			IngredientGroups: []IngredientGroup{
				{
					Title: "Veggies",
					Ingredients: []Ingredient{
						{Name: "carrot", Amount: &Amount{Factor: 1}},
						{Name: "broccoli", Amount: &Amount{Factor: 100, Unit: ptr("g")}},
					},
					IngredientGroups: []IngredientGroup{},
				},
			},
			Instructions: &instructions,
		}
		got, err := p.RenderJSONLD(r, 3)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}

		if parsed["@context"] != "https://schema.org/" {
			t.Errorf("@context = %v", parsed["@context"])
		}
		if parsed["@type"] != "Recipe" {
			t.Errorf("@type = %v", parsed["@type"])
		}
		if parsed["name"] != "Stir Fry" {
			t.Errorf("name = %v", parsed["name"])
		}

		// Description should be plain text.
		d := parsed["description"].(string)
		if strings.Contains(d, "**") {
			t.Errorf("description has markdown: %q", d)
		}

		if parsed["recipeYield"] != "2 servings" {
			t.Errorf("recipeYield = %v", parsed["recipeYield"])
		}

		// Keywords must be a string.
		kw := parsed["keywords"].(string)
		if kw != "asian, quick, vegetarian" {
			t.Errorf("keywords = %q", kw)
		}

		// All 4 ingredients (2 top-level + 2 grouped).
		ings := parsed["recipeIngredient"].([]any)
		if len(ings) != 4 {
			t.Errorf("expected 4 ingredients, got %d", len(ings))
		}

		// 2 sections.
		sections := parsed["recipeInstructions"].([]any)
		if len(sections) != 2 {
			t.Errorf("expected 2 instruction sections, got %d", len(sections))
		}
	})
}
