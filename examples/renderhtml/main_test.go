package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	recipemd "github.com/xcapaldi/recipemd-go"
)

func TestRenderHTMLProducesArticle(t *testing.T) {
	data, err := os.ReadFile("../../testdata/golden/ing_simple.md")
	if err != nil {
		t.Fatal(err)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	got := p.RenderHTML(r, 3)

	if !strings.Contains(got, `class="recipemd-recipe"`) {
		t.Error("missing recipemd-recipe article element")
	}
	if !strings.Contains(got, "Recipe") {
		t.Errorf("missing title in output:\n%s", got)
	}
	if !strings.Contains(got, "salt") {
		t.Errorf("missing ingredient in output:\n%s", got)
	}
}

func TestRenderHTMLInvalidRecipeFails(t *testing.T) {
	p := recipemd.NewParser()
	_, err := p.Parse(bytes.NewReader([]byte("no heading here")))
	if err == nil {
		t.Error("expected parse error for invalid recipe")
	}
}
