package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	recipemd "github.com/xcapaldi/recipemd-go"
)

func TestRenderOKFProducesFrontmatter(t *testing.T) {
	data, err := os.ReadFile("../../testdata/golden/ing_simple.md")
	if err != nil {
		t.Fatal(err)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	got := p.RenderOKF(r, 2)

	if !strings.HasPrefix(got, "---\n") {
		t.Errorf("missing frontmatter fence:\n%s", got)
	}
	if !strings.Contains(got, "type: Recipe\n") {
		t.Errorf("missing OKF type field:\n%s", got)
	}
	if !strings.Contains(got, "salt") {
		t.Errorf("missing ingredient in output:\n%s", got)
	}
}
