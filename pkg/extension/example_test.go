package extension_test

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/xcapaldi/recipemd-go/pkg/extension"
)

func Example() {
	markdown := `# Chocolate Chip Cookies

The best chocolate chip cookies you'll ever make!

*dessert, cookies, baking*

**24 cookies, 2 dozen**

---

- *2 1/4 cups* all-purpose flour
- *1 tsp* baking soda
- *1 tsp* salt
- *1 cup* butter, softened
- *3/4 cup* granulated sugar
- *3/4 cup* packed brown sugar
- *2* large eggs
- *2 tsp* vanilla extract
- *2 cups* chocolate chips

---

Preheat oven to 375°F (190°C).

Mix flour, baking soda, and salt in a bowl. In another bowl, beat butter and sugars until creamy.
Add eggs and vanilla. Gradually blend in flour mixture. Stir in chocolate chips.

Drop rounded tablespoons onto ungreased baking sheets. Bake 9-11 minutes or until golden brown.

Cool on baking sheets for 2 minutes, then move to wire racks.
`

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.NewRecipeMD(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		panic(err)
	}

	fmt.Println(buf.String())
}
