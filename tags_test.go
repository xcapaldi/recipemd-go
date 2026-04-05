package recipemd

import (
	"strings"
	"testing"
)

func TestAnnotateTemperatures(t *testing.T) {
	t.Parallel()

	t.Run("degree celsius", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Bake at 180°C for a bit.", nil, 2)
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
		got := annotateTemperatures("Bake at 350°F.", nil, 2)
		if !strings.Contains(got, `data-unit="F"`) {
			t.Errorf("missing data-unit F in: %s", got)
		}
		if !strings.Contains(got, `data-value="350"`) {
			t.Errorf("missing data-value in: %s", got)
		}
	})

	t.Run("space before degree symbol", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Set oven to 200 °C.", nil, 2)
		if !strings.Contains(got, `data-unit="C"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `data-value="200"`) {
			t.Errorf("missing data-value in: %s", got)
		}
	})

	t.Run("word celsius", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Heat to 100 celsius.", nil, 2)
		if !strings.Contains(got, `data-unit="C"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
		if !strings.Contains(got, `>100 celsius</span>`) {
			t.Errorf("text not preserved in: %s", got)
		}
	})

	t.Run("word Fahrenheit", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Heat to 212 Fahrenheit.", nil, 2)
		if !strings.Contains(got, `data-unit="F"`) {
			t.Errorf("missing data-unit in: %s", got)
		}
	})

	t.Run("decimal temperature", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("Cook at 162.5°C.", nil, 2)
		if !strings.Contains(got, `data-value="162.5"`) {
			t.Errorf("missing decimal data-value in: %s", got)
		}
	})

	t.Run("convert C to F", func(t *testing.T) {
		t.Parallel()
		f := Fahrenheit
		got := annotateTemperatures("Bake at 100°C.", &f, 2)
		if !strings.Contains(got, `data-unit="F"`) {
			t.Errorf("should be converted to F in: %s", got)
		}
		if !strings.Contains(got, `data-value="212"`) {
			t.Errorf("100C should convert to 212F in: %s", got)
		}
		if !strings.Contains(got, `data-original-unit="C"`) {
			t.Errorf("missing original unit in: %s", got)
		}
		if !strings.Contains(got, `data-original-value="100"`) {
			t.Errorf("missing original value in: %s", got)
		}
		if !strings.Contains(got, `>212°F</span>`) {
			t.Errorf("displayed text should show converted value in: %s", got)
		}
	})

	t.Run("convert F to C", func(t *testing.T) {
		t.Parallel()
		c := Celsius
		got := annotateTemperatures("Bake at 212°F.", &c, 2)
		if !strings.Contains(got, `data-unit="C"`) {
			t.Errorf("should be converted to C in: %s", got)
		}
		if !strings.Contains(got, `data-value="100"`) {
			t.Errorf("212F should convert to 100C in: %s", got)
		}
		if !strings.Contains(got, `>100°C</span>`) {
			t.Errorf("displayed text should show converted value in: %s", got)
		}
	})

	t.Run("no conversion when same unit", func(t *testing.T) {
		t.Parallel()
		c := Celsius
		got := annotateTemperatures("Heat to 180°C.", &c, 2)
		if strings.Contains(got, `data-original`) {
			t.Errorf("should not have original attrs when unit matches in: %s", got)
		}
		if !strings.Contains(got, `>180°C</span>`) {
			t.Errorf("text should be preserved as-is in: %s", got)
		}
	})

	t.Run("inside HTML preserves tags", func(t *testing.T) {
		t.Parallel()
		input := `<p>Bake at 180°C</p>`
		got := annotateTemperatures(input, nil, 2)
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
		got := annotateTemperatures(input, nil, 2)
		if !strings.Contains(got, `data-temp="180°C"`) {
			t.Errorf("HTML attribute modified in: %s", got)
		}
	})

	t.Run("multiple temperatures", func(t *testing.T) {
		t.Parallel()
		got := annotateTemperatures("First 180°C then 200°F.", nil, 2)
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
		if !strings.Contains(got, `data-value="5"`) {
			t.Errorf("missing data-value in: %s", got)
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

func TestRenderHTMLWithOptions(t *testing.T) {
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
		got := p.RenderHTMLWithOptions(r, 2, HTMLOptions{})
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
		got := p.RenderHTMLWithOptions(r, 2, HTMLOptions{})
		if !strings.Contains(got, `class="recipemd-temperature"`) {
			t.Errorf("temperature not annotated in description: %s", got)
		}
	})

	t.Run("converts celsius to fahrenheit", func(t *testing.T) {
		t.Parallel()
		instructions := "Bake at 200°C."
		r := &Recipe{
			Title:            "Pie",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "apples"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		f := Fahrenheit
		got := p.RenderHTMLWithOptions(r, 0, HTMLOptions{ConvertTemperature: &f})
		if !strings.Contains(got, `data-unit="F"`) {
			t.Errorf("should convert to F in: %s", got)
		}
		if !strings.Contains(got, `data-value="392"`) {
			t.Errorf("200C = 392F, got: %s", got)
		}
		if !strings.Contains(got, `data-original-unit="C"`) {
			t.Errorf("missing original unit in: %s", got)
		}
		if !strings.Contains(got, `data-original-value="200"`) {
			t.Errorf("missing original value in: %s", got)
		}
	})

	t.Run("backward compatible with RenderHTML", func(t *testing.T) {
		t.Parallel()
		instructions := "Bake at 180°C for 30 minutes."
		r := &Recipe{
			Title:            "Test",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "flour"}},
			IngredientGroups: []IngredientGroup{},
			Instructions:     &instructions,
		}
		got := p.RenderHTML(r, 2)
		if !strings.Contains(got, `class="recipemd-temperature"`) {
			t.Errorf("RenderHTML should also annotate temperatures: %s", got)
		}
		if !strings.Contains(got, `class="recipemd-time"`) {
			t.Errorf("RenderHTML should also annotate times: %s", got)
		}
	})
}
