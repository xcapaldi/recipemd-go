package recipemd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnnotateTemperatures(t *testing.T) {
	t.Parallel()

	t.Run("degree celsius", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Bake at 180°C for a bit.", 2)
		if !strings.Contains(got, `class="recipemd-temperature"`) {
			t.Fatalf("missing temperature class in: %s", got)
		}
		if !strings.Contains(got, `data-unit="C"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `data-value="180"`) {
			t.Errorf("missing data-value in: %s", got)
		}
		if !strings.Contains(got, `>180°C</span>`) {
			t.Errorf("original text not preserved in: %s", got)
		}
	})

	t.Run("degree fahrenheit", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Bake at 350°F.", 2)
		if !strings.Contains(got, `data-unit="F"`) {
			t.Errorf("missing data-unit F in: %s", got)
		}
		if !strings.Contains(got, `data-value="350"`) {
			t.Errorf("missing data-value in: %s", got)
		}
	})

	t.Run("space before degree symbol", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Set oven to 200 °C.", 2)
		if !strings.Contains(got, `data-unit="C"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `data-value="200"`) {
			t.Errorf("missing data-value in: %s", got)
		}
	})

	t.Run("word celsius", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Heat to 100 celsius.", 2)
		if !strings.Contains(got, `data-unit="C"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `>100 celsius</span>`) {
			t.Errorf("text not preserved in: %s", got)
		}
	})

	t.Run("word Fahrenheit", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Heat to 212 Fahrenheit.", 2)
		if !strings.Contains(got, `data-unit="F"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
	})

	t.Run("decimal temperature", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Cook at 162.5°C.", 2)
		if !strings.Contains(got, `data-value="162.5"`) {
			t.Errorf("missing decimal data-value in: %s", got)
		}
	})

	t.Run("inside HTML preserves tags", func(t *testing.T) {
		t.Parallel()
		input := `<p>Bake at 180°C</p>`
		got := annotateTemperatures(input, 2)
		if !strings.Contains(got, "<p>") {
			t.Errorf("HTML tags damaged in: %s", got)
		}
		if !strings.Contains(got, `class="recipemd-temperature"`) {
			t.Errorf("temperature not annotated in: %s", got)
		}
	})

	t.Run("does not modify HTML attributes", func(t *testing.T) {
		t.Parallel()
		input := `<div data-temp="180°C">some text</div>`
		got := annotateTemperatures(input, 2)
		if !strings.Contains(got, `data-temp="180°C"`) {
			t.Errorf("HTML attribute modified in: %s", got)
		}
	})

	t.Run("multiple temperatures", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("First 180°C then 200°F.", 2)
		if strings.Count(got, `class="recipemd-temperature"`) != 2 {
			t.Errorf("expected 2 temperature annotations in: %s", got)
		}
	})
}

func TestAnnotateTimes(t *testing.T) {
	t.Parallel()

	t.Run("minutes", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Cook for 30 minutes.")
		if !strings.Contains(got, `class="recipemd-time"`) {
			t.Fatalf("missing time class in: %s", got)
		}
		if !strings.Contains(got, `data-unit="min"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `data-value="30"`) {
			t.Errorf("missing data-value in: %s", got)
		}
		if !strings.Contains(got, `>30 minutes</span>`) {
			t.Errorf("text not preserved in: %s", got)
		}
	})

	t.Run("min abbreviation", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Wait 5 min.")
		if !strings.Contains(got, `data-unit="min"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
	})

	t.Run("mins abbreviation", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Wait 15 mins.")
		if !strings.Contains(got, `data-unit="min"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
	})

	t.Run("hours", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Rest for 2 hours.")
		if !strings.Contains(got, `data-unit="h"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `data-value="2"`) {
			t.Errorf("missing data-value in: %s", got)
		}
	})

	t.Run("hr abbreviation", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Wait 1 hr.")
		if !strings.Contains(got, `data-unit="h"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
	})

	t.Run("hrs abbreviation", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Marinate 4 hrs.")
		if !strings.Contains(got, `data-unit="h"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
	})

	t.Run("seconds", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Microwave 90 seconds.")
		if !strings.Contains(got, `data-unit="s"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `data-value="90"`) {
			t.Errorf("missing data-value in: %s", got)
		}
	})

	t.Run("sec abbreviation", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Pulse for 10 secs.")
		if !strings.Contains(got, `data-unit="s"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
	})

	t.Run("no space before unit", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Bake 45min.")
		if !strings.Contains(got, `data-unit="min"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `data-value="45"`) {
			t.Errorf("missing data-value in: %s", got)
		}
	})

	t.Run("decimal time", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Cook for 1.5 hours.")
		if !strings.Contains(got, `data-value="1.5"`) {
			t.Errorf("missing decimal data-value in: %s", got)
		}
	})

	t.Run("inside HTML preserves tags", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes(`<p>Cook for 30 minutes.</p>`)
		if !strings.Contains(got, "<p>") {
			t.Errorf("HTML tags damaged in: %s", got)
		}
		if !strings.Contains(got, `class="recipemd-time"`) {
			t.Errorf("time not annotated in: %s", got)
		}
	})

	t.Run("multiple times", func(t *testing.T) {
		t.Parallel()
		got := annotateTimes("Boil 10 minutes, rest 2 hours.")
		if strings.Count(got, `class="recipemd-time"`) != 2 {
			t.Errorf("expected 2 time annotations in: %s", got)
		}
	})
}

func TestConvertTemperaturesInText(t *testing.T) {
	t.Parallel()

	t.Run("C to F", func(t *testing.T) {
		t.Parallel()
		got := convertTemperaturesInText("Bake at 100°C.", Fahrenheit, 2)
		if got != "Bake at 212°F." {
			t.Errorf("got %q, want %q", got, "Bake at 212°F.")
		}
	})

	t.Run("F to C", func(t *testing.T) {
		t.Parallel()
		got := convertTemperaturesInText("Bake at 212°F.", Celsius, 2)
		if got != "Bake at 100°C." {
			t.Errorf("got %q, want %q", got, "Bake at 100°C.")
		}
	})

	t.Run("same unit unchanged", func(t *testing.T) {
		t.Parallel()
		got := convertTemperaturesInText("Bake at 180°C.", Celsius, 2)
		if got != "Bake at 180°C." {
			t.Errorf("got %q, want %q", got, "Bake at 180°C.")
		}
	})

	t.Run("word fahrenheit to celsius", func(t *testing.T) {
		t.Parallel()
		got := convertTemperaturesInText("Heat to 392 Fahrenheit.", Celsius, 2)
		if got != "Heat to 200°C." {
			t.Errorf("got %q, want %q", got, "Heat to 200°C.")
		}
	})

	t.Run("multiple temperatures", func(t *testing.T) {
		t.Parallel()
		got := convertTemperaturesInText("First 100°C then 212°F.", Celsius, 2)
		if got != "First 100°C then 100°C." {
			t.Errorf("got %q, want %q", got, "First 100°C then 100°C.")
		}
	})
}

func TestRecipeConvertTemperatures(t *testing.T) {
	t.Parallel()

	t.Run("converts description and instructions", func(t *testing.T) {
		t.Parallel()
		desc := "Preheat oven to 200°C."
		instructions := "Bake at 200°C for 30 minutes."
		r := &Recipe{
			Title:            "Cake",
			Description:      &desc,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		r.ConvertTemperatures(Fahrenheit, 0)
		if !strings.Contains(*r.Description, "392°F") {
			t.Errorf("description not converted: %s", *r.Description)
		}
		if !strings.Contains(*r.Instructions, "392°F") {
			t.Errorf("instructions not converted: %s", *r.Instructions)
		}
	})

	t.Run("nil fields are safe", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Simple",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "salt"}},
			IngredientGroups: []IngredientGroup{},
		}
		r.ConvertTemperatures(Fahrenheit, 2) // should not panic
	})

	t.Run("same unit is no-op", func(t *testing.T) {
		t.Parallel()
		instructions := "Bake at 180°C."
		r := &Recipe{
			Title:            "Test",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		r.ConvertTemperatures(Celsius, 2)
		if *r.Instructions != "Bake at 180°C." {
			t.Errorf("same-unit conversion should be no-op: %s", *r.Instructions)
		}
	})

	t.Run("works with markdown rendering", func(t *testing.T) {
		t.Parallel()
		p := NewParser()
		instructions := "Bake at 100°C for 30 minutes."
		r := &Recipe{
			Title:            "Cake",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		r.ConvertTemperatures(Fahrenheit, 2)
		got := p.RenderMarkdown(r, 2)
		if !strings.Contains(got, "212°F") {
			t.Errorf("converted temperature not in markdown output: %s", got)
		}
		if strings.Contains(got, "100°C") {
			t.Errorf("original temperature should be gone: %s", got)
		}
	})

	t.Run("works with JSON rendering", func(t *testing.T) {
		t.Parallel()
		p := NewParser()
		instructions := "Bake at 100°C."
		r := &Recipe{
			Title:            "Test",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		r.ConvertTemperatures(Fahrenheit, 2)
		got, err := p.RenderJSON(r)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatal(err)
		}
		instr, ok := parsed["instructions"].(string)
		if !ok {
			t.Fatal("instructions missing from JSON")
		}
		if !strings.Contains(instr, "212°F") {
			t.Errorf("converted temperature not in JSON: %s", instr)
		}
	})

	t.Run("works with HTML rendering", func(t *testing.T) {
		t.Parallel()
		p := NewParser()
		instructions := "Bake at 100°C for 30 minutes."
		r := &Recipe{
			Title:            "Cake",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		r.ConvertTemperatures(Fahrenheit, 0)
		got := p.RenderHTML(r, 0)
		// Should show converted value in a temperature span
		if !strings.Contains(got, `class="recipemd-temperature"`) {
			t.Errorf("temperature not annotated in HTML: %s", got)
		}
		if !strings.Contains(got, `data-unit="F"`) {
			t.Errorf("should show F after conversion: %s", got)
		}
		// Time annotation should still work
		if !strings.Contains(got, `class="recipemd-time"`) {
			t.Errorf("time not annotated in HTML: %s", got)
		}
	})
}

func TestRenderHTMLAnnotation(t *testing.T) {
	t.Parallel()
	p := NewParser()

	t.Run("annotates temperatures in instructions", func(t *testing.T) {
		t.Parallel()
		instructions := "Bake at 180°C for 30 minutes."
		r := &Recipe{
			Title:            "Cake",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got := p.RenderHTML(r, 2)
		if !strings.Contains(got, `class="recipemd-temperature"`) {
			t.Errorf("temperature not annotated in: %s", got)
		}
		if !strings.Contains(got, `class="recipemd-time"`) {
			t.Errorf("time not annotated in: %s", got)
		}
	})

	t.Run("annotates temperatures in description", func(t *testing.T) {
		t.Parallel()
		desc := "A bread baked at 220°C."
		r := &Recipe{
			Title:            "Bread",
			Description:      &desc,
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
		}
		got := p.RenderHTML(r, 2)
		if !strings.Contains(got, `class="recipemd-temperature"`) {
			t.Errorf("temperature not annotated in description: %s", got)
		}
	})
}
