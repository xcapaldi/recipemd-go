package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	recipemd "github.com/xcapaldi/recipemd-go"
)

func TestParseProducesValidJSON(t *testing.T) {
	data, err := os.ReadFile("../../testdata/golden/ing_simple.md")
	if err != nil {
		t.Fatal(err)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out, err := p.RenderJSON(r)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var v map[string]any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if v["title"] != "Recipe" {
		t.Errorf("title = %q, want %q", v["title"], "Recipe")
	}
}

func TestParseInvalidRecipeFails(t *testing.T) {
	p := recipemd.NewParser()
	_, err := p.Parse(bytes.NewReader([]byte("no heading here")))
	if err == nil {
		t.Error("expected parse error for invalid recipe")
	}
}
