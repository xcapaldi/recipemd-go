package main

import (
	"os"
	"strings"
	"testing"

	recipemd "github.com/xcapaldi/recipemd-go"
)

func TestScaleByFactor(t *testing.T) {
	data, err := os.ReadFile("../../testdata/golden/amount_with_unit.md")
	if err != nil {
		t.Fatal(err)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	r.Scale(2)

	out := p.RenderMarkdown(r, 2)
	if !strings.Contains(out, "2 cup") {
		t.Errorf("expected scaled amount \"2 cup\" in output:\n%s", out)
	}
}

func TestScaleForYield(t *testing.T) {
	data, err := os.ReadFile("../../testdata/golden/yields_single.md")
	if err != nil {
		t.Fatal(err)
	}

	p := recipemd.NewParser()
	r, err := p.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	amount, err := recipemd.ParseAmountString("8 servings")
	if err != nil || amount.Factor == 0 {
		t.Fatal("failed to parse amount")
	}

	if err := r.ScaleForYield(amount); err != nil {
		t.Fatalf("ScaleForYield: %v", err)
	}

	if len(r.Yields) == 0 {
		t.Fatal("no yields after scaling")
	}
	if r.Yields[0].Factor != 8 {
		t.Errorf("yield factor = %v, want 8", r.Yields[0].Factor)
	}
}
