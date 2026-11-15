package main

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark"

	recipemd "github.com/xcapaldi/recipemd-go"
)

const src = `# Cookies

Crisp *and* chewy.

*sweet, baking & treats*

**12 cookies**

---

- *1/3 cup* sugar
- [dough](dough.md)

## Topping

### Extra

- *2* sprinkles

---

Bake <script>x</script> it.
`

func TestRender(t *testing.T) {
	r, err := recipemd.NewParser().Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	got, err := render(r, goldmark.New(), 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<p>Crisp <em>and</em> chewy.</p>`,
		`<li class="tag">sweet</li>`,
		`<li class="tag">baking &amp; treats</li>`,
		`<span class="amount" data-factor="12" data-unit="cookies">12 cookies</span>`,
		`<span class="amount" data-factor="0.3333333333333333" data-unit="cup">0.33 cup</span> sugar`,
		`<a href="dough.md">dough</a>`,
		`<section class="group" data-level="2">` + "\n<h2>Topping</h2>",
		`<section class="group" data-level="3">` + "\n<h3>Extra</h3>",
		`<span class="amount" data-factor="2">2</span> sprinkles`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<script>") {
		t.Errorf("raw HTML from instructions leaked into output:\n%s", got)
	}
}
