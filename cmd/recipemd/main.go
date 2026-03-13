package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	recipemd "github.com/xcapaldi/recipemd-go"
)

const version = "0.1.0"

func main() {
	// Custom usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: recipemd [options] <file>\n\nRead and process recipemd recipes\n\nOptions:\n")
		flag.PrintDefaults()
	}

	showVersion := flag.Bool("v", false, "show version")
	showVersionLong := flag.Bool("version", false, "show version")
	showTitle := flag.Bool("t", false, "display recipe title")
	showTitleLong := flag.Bool("title", false, "display recipe title")
	showIngredients := flag.Bool("i", false, "display recipe ingredients")
	showIngredientsLong := flag.Bool("ingredients", false, "display recipe ingredients")
	showJSON := flag.Bool("j", false, "display recipe as JSON")
	showJSONLong := flag.Bool("json", false, "display recipe as JSON")
	roundStr := flag.String("r", "2", "round amount to n digits after decimal point. Default is \"2\", use \"no\" to disable rounding")
	roundStrLong := flag.String("round", "2", "round amount to n digits after decimal point. Default is \"2\", use \"no\" to disable rounding")
	multiply := flag.String("m", "", "multiply recipe by N")
	multiplyLong := flag.String("multiply", "", "multiply recipe by N")
	yield := flag.String("y", "", "scale the recipe for yield Y, e.g. \"5 servings\"")
	yieldLong := flag.String("yield", "", "scale the recipe for yield Y, e.g. \"5 servings\"")
	flatten := flag.Bool("f", false, "flatten ingredients and instructions of linked recipes into main recipe")
	flattenLong := flag.Bool("flatten", false, "flatten ingredients and instructions of linked recipes into main recipe")

	flag.Parse()

	if *showVersion || *showVersionLong {
		fmt.Printf("recipemd %s\n", version)
		os.Exit(0)
	}

	title := *showTitle || *showTitleLong
	ingredients := *showIngredients || *showIngredientsLong
	jsonOut := *showJSON || *showJSONLong
	flat := *flatten || *flattenLong

	// Mutual exclusivity check for display options
	displayCount := 0
	if title {
		displayCount++
	}
	if ingredients {
		displayCount++
	}
	if jsonOut {
		displayCount++
	}
	if displayCount > 1 {
		fmt.Fprintf(os.Stderr, "Error: -t/--title, -i/--ingredients, and -j/--json are mutually exclusive\n")
		os.Exit(1)
	}

	// Resolve round value (prefer short flag if set)
	rStr := *roundStr
	if isFlagSet("round") {
		rStr = *roundStrLong
	}
	rounding := 2
	if strings.EqualFold(rStr, "no") {
		rounding = -1
	} else {
		n, err := strconv.Atoi(rStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid rounding value %q\n", rStr)
			os.Exit(1)
		}
		rounding = n
	}

	// Resolve multiply/yield (prefer short flag if set)
	multiplyVal := *multiply
	if multiplyVal == "" {
		multiplyVal = *multiplyLong
	}
	yieldVal := *yield
	if yieldVal == "" {
		yieldVal = *yieldLong
	}

	if multiplyVal != "" && yieldVal != "" {
		fmt.Fprintf(os.Stderr, "Error: -m/--multiply and -y/--yield are mutually exclusive\n")
		os.Exit(1)
	}

	// Need a file argument
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: a recipe file is required\n")
		flag.Usage()
		os.Exit(1)
	}
	filePath := flag.Arg(0)

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse recipe
	p := recipemd.NewParser()
	recipe, err := p.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing recipe: %v\n", err)
		os.Exit(1)
	}

	// Flatten linked recipes
	if flat {
		if err := p.Flatten(recipe, filePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	// Scale recipe
	if yieldVal != "" {
		requiredYield, err := recipemd.ParseAmountString(yieldVal)
		if err != nil || requiredYield.Factor == 0 {
			fmt.Fprintf(os.Stderr, "Error: given yield is not valid\n")
			os.Exit(1)
		}
		if err := recipe.ScaleForYield(requiredYield); err != nil {
			fmt.Fprintf(os.Stderr, "Error: recipe does not have a yield with matching unit\n")
			units := make([]string, 0)
			for _, y := range recipe.Yields {
				if y.Unit != nil {
					units = append(units, fmt.Sprintf("%q", *y.Unit))
				}
			}
			if len(units) > 0 {
				fmt.Fprintf(os.Stderr, "Available units: %s\n", strings.Join(units, ", "))
			}
			os.Exit(1)
		}
	} else if multiplyVal != "" {
		mult, err := recipemd.ParseAmountString(multiplyVal)
		if err != nil || mult.Factor == 0 {
			fmt.Fprintf(os.Stderr, "Error: given multiplier is not valid\n")
			os.Exit(1)
		}
		if mult.Unit != nil {
			fmt.Fprintf(os.Stderr, "Error: a recipe can only be multiplied with a unitless amount\n")
			os.Exit(1)
		}
		recipe.Scale(mult.Factor)
	}

	// Output
	if title {
		fmt.Println(recipe.Title)
	} else if ingredients {
		for _, ing := range recipe.LeafIngredients() {
			fmt.Println(ing.Serialize(rounding))
		}
	} else if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(recipe); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Print(p.RenderMarkdown(recipe, rounding))
	}
}

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
