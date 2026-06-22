package recipemd

import (
	"strings"
	"testing"
)

func TestRenderOKF(t *testing.T) {
	t.Parallel()
	p := NewParser()

	t.Run("minimal", func(t *testing.T) {
		t.Parallel()
		r := &Recipe{
			Title:            "Test",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "salt"}},
			IngredientGroups: []IngredientGroup{},
		}
		got := p.RenderOKF(r, 3)
		if !strings.HasPrefix(got, "---\n") {
			t.Fatalf("missing frontmatter fence: %s", got)
		}
		if !strings.Contains(got, "type: Recipe\n") {
			t.Error("missing OKF type field")
		}
		if !strings.Contains(got, `title: "Test"`) {
			t.Error("missing title field")
		}
		if !strings.Contains(got, "# Test") {
			t.Error("missing RecipeMD body title")
		}
		if !strings.Contains(got, "- salt") {
			t.Error("missing ingredient")
		}
	})

	t.Run("full recipe with special characters", func(t *testing.T) {
		t.Parallel()
		desc := "A \"great\" recipe.\nServe warm."
		r := &Recipe{
			Title:       "Guac",
			Description: &desc,
			Tags:        []string{"sauce", "vegan"},
			Yields:      []Amount{{Factor: 4, Unit: new("servings")}},
			Ingredients: []Ingredient{
				{Name: "avocado", Amount: &Amount{Factor: 1, Unit: nil}},
			},
			IngredientGroups: []IngredientGroup{},
		}
		got := p.RenderOKF(r, 3)
		if !strings.Contains(got, "description: |\n  A \"great\" recipe.\n  Serve warm.\n") {
			t.Errorf("description block scalar not rendered correctly:\n%s", got)
		}
		if !strings.Contains(got, `tags: ["sauce", "vegan"]`) {
			t.Errorf("tags not rendered: %s", got)
		}
	})

	t.Run("round trips through WithFrontmatter", func(t *testing.T) {
		t.Parallel()
		desc := "A great recipe."
		r := &Recipe{
			Title:       "Guac",
			Description: &desc,
			Tags:        []string{"sauce", "vegan"},
			Yields:      []Amount{{Factor: 4, Unit: new("servings")}},
			Ingredients: []Ingredient{
				{Name: "avocado", Amount: &Amount{Factor: 1, Unit: nil}},
				{Name: "salt"},
			},
			IngredientGroups: []IngredientGroup{},
		}
		doc := p.RenderOKF(r, 3)

		fp := NewParser(WithFrontmatter())
		got, err := fp.Parse(strings.NewReader(doc))
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if got.Title != r.Title {
			t.Errorf("Title = %q, want %q", got.Title, r.Title)
		}
		if got.Description == nil || *got.Description != *r.Description {
			t.Errorf("Description = %v, want %v", got.Description, r.Description)
		}
		if strings.Join(got.Tags, ",") != strings.Join(r.Tags, ",") {
			t.Errorf("Tags = %v, want %v", got.Tags, r.Tags)
		}
		if len(got.Ingredients) != len(r.Ingredients) {
			t.Fatalf("Ingredients = %v, want %v", got.Ingredients, r.Ingredients)
		}
	})

	t.Run("round trips OKF metadata: type, resource, timestamp, extra", func(t *testing.T) {
		t.Parallel()
		resource := "https://example.com/guac"
		timestamp := "2024-01-02T15:04:05Z"
		r := &Recipe{
			Title:            "Guac",
			Yields:           []Amount{},
			Tags:             []string{},
			Ingredients:      []Ingredient{{Name: "avocado"}},
			IngredientGroups: []IngredientGroup{},
			OKF: &OKFMetadata{
				Type:      "Playbook",
				Resource:  &resource,
				Timestamp: &timestamp,
				Extra:     map[string]string{"custom_field": "hello", "another": "value"},
			},
		}
		doc := p.RenderOKF(r, 3)
		if !strings.Contains(doc, "type: Playbook\n") {
			t.Errorf("missing custom type field:\n%s", doc)
		}
		if !strings.Contains(doc, `resource: "https://example.com/guac"`) {
			t.Errorf("missing resource field:\n%s", doc)
		}
		if !strings.Contains(doc, `timestamp: "2024-01-02T15:04:05Z"`) {
			t.Errorf("missing timestamp field:\n%s", doc)
		}
		if !strings.Contains(doc, `another: "value"`) || !strings.Contains(doc, `custom_field: "hello"`) {
			t.Errorf("missing extra fields:\n%s", doc)
		}

		fp := NewParser(WithFrontmatter())
		got, err := fp.Parse(strings.NewReader(doc))
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if got.OKF == nil {
			t.Fatal("OKF metadata not round-tripped")
		}
		if got.OKF.Type != "Playbook" {
			t.Errorf("Type = %q, want %q", got.OKF.Type, "Playbook")
		}
		if got.OKF.Resource == nil || *got.OKF.Resource != resource {
			t.Errorf("Resource = %v, want %q", got.OKF.Resource, resource)
		}
		if got.OKF.Timestamp == nil || *got.OKF.Timestamp != timestamp {
			t.Errorf("Timestamp = %v, want %q", got.OKF.Timestamp, timestamp)
		}
		if got.OKF.Extra["custom_field"] != "hello" || got.OKF.Extra["another"] != "value" {
			t.Errorf("Extra = %v, want custom_field=hello, another=value", got.OKF.Extra)
		}
	})
}
