package recipemd

import (
	"fmt"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/xcapaldi/recipemd-go/pkg/goldmark-recipemd/ast"
)

// Yield represents a serving amount (e.g., "4 servings")
type Yield struct {
	Amount string
	Unit   string
}

func (y Yield) String() string {
	if y.Unit == "" {
		return y.Amount
	}
	return fmt.Sprintf("%s %s", y.Amount, y.Unit)
}

// Amount represents an ingredient quantity
type Amount struct {
	Quantity string
	Unit     string
}

func (a Amount) String() string {
	if a.Unit == "" {
		return a.Quantity
	}
	return fmt.Sprintf("%s %s", a.Quantity, a.Unit)
}

// Ingredient represents a single ingredient
type Ingredient struct {
	Amount *Amount
	Name   string
}

// IngredientGroup is a named collection of ingredients
type IngredientGroup struct {
	Name        string
	Ingredients []Ingredient
}

// IngredientEntry is either an Ingredient or IngredientGroup
type IngredientEntry interface {
	isIngredientEntry()
}

func (Ingredient) isIngredientEntry()      {}
func (IngredientGroup) isIngredientEntry() {}

// Recipe is the structured representation of a RecipeMD document
type Recipe struct {
	Title        string
	Description  string
	Tags         []string
	Yields       []Yield
	Ingredients  []IngredientEntry
	Instructions string
}

// ExtractRecipe walks the AST and extracts a Recipe struct
func ExtractRecipe(doc *gast.Document, source []byte) *Recipe {
	recipe := &Recipe{}

	var recipeNode *ast.Recipe
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if r, ok := child.(*ast.Recipe); ok {
			recipeNode = r
			break
		}
	}

	if recipeNode == nil {
		return recipe
	}

	for child := recipeNode.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Title:
			recipe.Title = extractText(n, source)

		case *ast.Description:
			var parts []string
			for desc := n.FirstChild(); desc != nil; desc = desc.NextSibling() {
				parts = append(parts, strings.TrimSpace(extractText(desc, source)))
			}
			recipe.Description = strings.Join(parts, "\n\n")

		case *ast.Tags:
			tagsText := extractText(n, source)
			for _, tag := range strings.Split(tagsText, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					recipe.Tags = append(recipe.Tags, tag)
				}
			}

		case *ast.Yields:
			yieldsText := extractText(n, source)
			for _, y := range strings.Split(yieldsText, ",") {
				y = strings.TrimSpace(y)
				if y == "" {
					continue
				}
				yield := parseYield(y)
				recipe.Yields = append(recipe.Yields, yield)
			}

		case *ast.Ingredients:
			recipe.Ingredients = extractIngredients(n, source)

		case *ast.Instructions:
			recipe.Instructions = extractInstructions(n, source)
		}
	}

	return recipe
}

func extractIngredients(node *ast.Ingredients, source []byte) []IngredientEntry {
	var entries []IngredientEntry

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Ingredient:
			ing := Ingredient{Name: n.Name}
			if n.Amount != "" {
				amt := parseAmount(n.Amount)
				ing.Amount = &amt
			}
			entries = append(entries, ing)

		case *ast.IngredientGroup:
			group := IngredientGroup{Name: n.Name}
			for gc := n.FirstChild(); gc != nil; gc = gc.NextSibling() {
				if ing, ok := gc.(*ast.Ingredient); ok {
					i := Ingredient{Name: ing.Name}
					if ing.Amount != "" {
						amt := parseAmount(ing.Amount)
						i.Amount = &amt
					}
					group.Ingredients = append(group.Ingredients, i)
				}
			}
			entries = append(entries, group)
		}
	}

	return entries
}

func extractInstructions(node *ast.Instructions, source []byte) string {
	var parts []string
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		parts = append(parts, strings.TrimSpace(extractText(child, source)))
	}
	return strings.Join(parts, "\n\n")
}

func parseYield(s string) Yield {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 1 {
		return Yield{Amount: parts[0]}
	}
	return Yield{Amount: parts[0], Unit: parts[1]}
}

func parseAmount(s string) Amount {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 1 {
		return Amount{Quantity: parts[0]}
	}
	return Amount{Quantity: parts[0], Unit: parts[1]}
}

// BuildAST creates a new goldmark Document AST from a Recipe struct
func BuildAST(recipe *Recipe) *gast.Document {
	doc := gast.NewDocument()
	recipeNode := ast.NewRecipe()
	doc.AppendChild(doc, recipeNode)

	// Title
	if recipe.Title != "" {
		title := ast.NewTitle()
		heading := gast.NewHeading(1)
		heading.AppendChild(heading, gast.NewString([]byte(recipe.Title)))
		title.AppendChild(title, heading)
		recipeNode.AppendChild(recipeNode, title)
	}

	// Description
	if recipe.Description != "" {
		desc := ast.NewDescription()
		for _, p := range strings.Split(recipe.Description, "\n\n") {
			para := gast.NewParagraph()
			para.AppendChild(para, gast.NewString([]byte(p)))
			desc.AppendChild(desc, para)
		}
		recipeNode.AppendChild(recipeNode, desc)
	}

	// Tags
	if len(recipe.Tags) > 0 {
		tags := ast.NewTags()
		para := gast.NewParagraph()
		em := gast.NewEmphasis(1)
		em.AppendChild(em, gast.NewString([]byte(strings.Join(recipe.Tags, ", "))))
		para.AppendChild(para, em)
		tags.AppendChild(tags, para)
		recipeNode.AppendChild(recipeNode, tags)
	}

	// Yields
	if len(recipe.Yields) > 0 {
		yields := ast.NewYields()
		para := gast.NewParagraph()
		strong := gast.NewEmphasis(2)
		var yieldStrs []string
		for _, y := range recipe.Yields {
			yieldStrs = append(yieldStrs, y.String())
		}
		strong.AppendChild(strong, gast.NewString([]byte(strings.Join(yieldStrs, ", "))))
		para.AppendChild(para, strong)
		yields.AppendChild(yields, para)
		recipeNode.AppendChild(recipeNode, yields)
	}

	// Ingredients
	if len(recipe.Ingredients) > 0 {
		ings := ast.NewIngredients()
		for _, entry := range recipe.Ingredients {
			switch e := entry.(type) {
			case Ingredient:
				ing := ast.NewIngredient()
				if e.Amount != nil {
					ing.Amount = e.Amount.String()
				}
				ing.Name = e.Name
				ings.AppendChild(ings, ing)
			case IngredientGroup:
				group := ast.NewIngredientGroup()
				group.Name = e.Name
				for _, i := range e.Ingredients {
					ing := ast.NewIngredient()
					if i.Amount != nil {
						ing.Amount = i.Amount.String()
					}
					ing.Name = i.Name
					group.AppendChild(group, ing)
				}
				ings.AppendChild(ings, group)
			}
		}
		recipeNode.AppendChild(recipeNode, ings)
	}

	// Instructions
	if recipe.Instructions != "" {
		instrs := ast.NewInstructions()
		for _, p := range strings.Split(recipe.Instructions, "\n\n") {
			para := gast.NewParagraph()
			para.AppendChild(para, gast.NewString([]byte(p)))
			instrs.AppendChild(instrs, para)
		}
		recipeNode.AppendChild(recipeNode, instrs)
	}

	return doc
}

// UpdateAST modifies an existing AST based on changes in the Recipe struct
func UpdateAST(doc *gast.Document, recipe *Recipe, source []byte) {
	var recipeNode *ast.Recipe
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if r, ok := child.(*ast.Recipe); ok {
			recipeNode = r
			break
		}
	}

	if recipeNode == nil {
		return
	}

	for child := recipeNode.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Title:
			updateTextContent(n, recipe.Title)

		case *ast.Tags:
			if len(recipe.Tags) > 0 {
				updateTextContent(n, strings.Join(recipe.Tags, ", "))
			}

		case *ast.Yields:
			if len(recipe.Yields) > 0 {
				var yieldStrs []string
				for _, y := range recipe.Yields {
					yieldStrs = append(yieldStrs, y.String())
				}
				updateTextContent(n, strings.Join(yieldStrs, ", "))
			}
		}
	}
}

func updateTextContent(node gast.Node, newText string) {
	var findText func(n gast.Node) *gast.Text
	findText = func(n gast.Node) *gast.Text {
		if t, ok := n.(*gast.Text); ok {
			return t
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t := findText(c); t != nil {
				return t
			}
		}
		return nil
	}

	if t := findText(node); t != nil {
		t.Segment = text.NewSegment(0, len(newText))
	}
}

func extractText(node gast.Node, source []byte) string {
	var sb strings.Builder
	extractTextInto(&sb, node, source)
	return sb.String()
}

func extractTextInto(sb *strings.Builder, node gast.Node, source []byte) {
	switch n := node.(type) {
	case *gast.Text:
		sb.Write(n.Segment.Value(source))
	case *gast.String:
		sb.Write(n.Value)
	default:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			extractTextInto(sb, child, source)
		}
	}
}
