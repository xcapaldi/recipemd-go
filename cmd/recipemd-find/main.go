package main

import (
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode"
	"unsafe"

	recipemd "github.com/xcapaldi/recipemd-go"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("v", false, "show version")
	showVersionLong := flag.Bool("version", false, "show version")
	expression := flag.String("e", "", "filter expression")
	expressionLong := flag.String("expression", "", "filter expression")
	noMessages := flag.Bool("s", false, "suppress error messages")
	noMessagesLong := flag.Bool("no-messages", false, "suppress error messages")
	outputOne := flag.Bool("1", false, "force output one entry per line")
	outputCols := flag.Bool("C", false, "force multi-column output")
	outputRows := flag.Bool("x", false, "multi-column output sorted across columns")
	count := flag.Bool("c", false, "count number of uses")
	countLong := flag.Bool("count", false, "count number of uses")
	gfm := flag.Bool("gfm", false, "enable GitHub Flavored Markdown extensions")
	frontmatter := flag.Bool("frontmatter", false, "strip YAML/TOML frontmatter before parsing")

	flag.Parse()

	if *showVersion || *showVersionLong {
		fmt.Printf("recipemd-find %s\n", version)
		os.Exit(0)
	}

	args := cliArgs{
		expression:  coalesce(*expression, *expressionLong),
		noMessages:  *noMessages || *noMessagesLong,
		count:       *count || *countLong,
		gfm:         *gfm,
		frontmatter: *frontmatter,
		folder:      ".",
	}

	if *outputOne {
		args.outputMulti = "no"
	} else if *outputRows {
		args.outputMulti = "rows"
	} else if *outputCols {
		args.outputMulti = "columns"
	}

	positional := flag.Args()
	if len(positional) < 1 {
		fmt.Fprintf(os.Stderr, "Error: an action is required (recipes, tags, ingredients, units)\n")
		os.Exit(1)
	}
	args.action = positional[0]
	if len(positional) > 1 {
		args.folder = positional[1]
	}

	if args.count && args.action == "recipes" {
		fmt.Fprintf(os.Stderr, "Error: -c/--count cannot be used with recipes action\n")
		os.Exit(1)
	}

	switch args.action {
	case "recipes":
		listRecipes(args)
	case "tags":
		listTags(args)
	case "ingredients":
		listIngredients(args)
	case "units":
		listUnits(args)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown action %q\n", args.action)
		os.Exit(1)
	}
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

type cliArgs struct {
	expression  string
	noMessages  bool
	outputMulti string // "no", "columns", "rows", or ""
	action      string
	folder      string
	count       bool
	gfm         bool
	frontmatter bool
}

func parserOpts(args cliArgs) []recipemd.Option {
	var opts []recipemd.Option
	if args.gfm {
		opts = append(opts, recipemd.WithGithubFormattedMarkdown())
	}
	if args.frontmatter {
		opts = append(opts, recipemd.WithFrontmatter())
	}
	return opts
}

type parsedRecipe struct {
	recipe *recipemd.Recipe
	path   string
}

func getFilteredRecipes(args cliArgs) []parsedRecipe {
	var results []parsedRecipe

	folder := args.folder
	err := filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if !args.noMessages {
				relPath, _ := filepath.Rel(folder, path)
				fmt.Fprintf(os.Stderr, "An error occurred, skipping %s: %v\n", relPath, err)
			}
			return nil
		}

		recipe, err := recipemd.NewParser(parserOpts(args)...).Parse(data)
		if err != nil {
			if !args.noMessages {
				relPath, _ := filepath.Rel(folder, path)
				fmt.Fprintf(os.Stderr, "An error occurred, skipping %s: %v\n", relPath, err)
			}
			return nil
		}

		if args.expression != "" && !matchesFilter(recipe, args.expression) {
			return nil
		}

		relPath, _ := filepath.Rel(folder, path)
		results = append(results, parsedRecipe{recipe: recipe, path: relPath})
		return nil
	})

	if err != nil && !args.noMessages {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
	}

	return results
}

// matchesFilter evaluates a simple boolean filter expression against a recipe.
// Supports: word matching (in title/tags), "ingr:word" for ingredient matching,
// "and", "or", "not" operators.
func matchesFilter(r *recipemd.Recipe, expr string) bool {
	tokens := tokenize(expr)
	result, _ := parseOr(tokens, r)
	return result
}

func tokenize(expr string) []string {
	var tokens []string
	current := strings.Builder{}
	for _, ch := range expr {
		if unicode.IsSpace(ch) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func parseOr(tokens []string, r *recipemd.Recipe) (bool, []string) {
	left, tokens := parseAnd(tokens, r)
	for len(tokens) > 0 && strings.EqualFold(tokens[0], "or") {
		tokens = tokens[1:]
		right, rest := parseAnd(tokens, r)
		left = left || right
		tokens = rest
	}
	return left, tokens
}

func parseAnd(tokens []string, r *recipemd.Recipe) (bool, []string) {
	left, tokens := parsePrimary(tokens, r)
	for len(tokens) > 0 && strings.EqualFold(tokens[0], "and") {
		tokens = tokens[1:]
		right, rest := parsePrimary(tokens, r)
		left = left && right
		tokens = rest
	}
	return left, tokens
}

func parsePrimary(tokens []string, r *recipemd.Recipe) (bool, []string) {
	if len(tokens) == 0 {
		return false, tokens
	}

	token := tokens[0]
	tokens = tokens[1:]

	if strings.EqualFold(token, "not") {
		result, rest := parsePrimary(tokens, r)
		return !result, rest
	}

	if strings.HasPrefix(strings.ToLower(token), "ingr:") {
		needle := strings.ToLower(token[5:])
		for _, ing := range r.LeafIngredients() {
			if strings.Contains(strings.ToLower(ing.Name), needle) {
				return true, tokens
			}
		}
		return false, tokens
	}

	if strings.HasPrefix(strings.ToLower(token), "tag:") {
		needle := strings.ToLower(token[4:])
		for _, tag := range r.Tags {
			if strings.Contains(strings.ToLower(tag), needle) {
				return true, tokens
			}
		}
		return false, tokens
	}

	// Default: match against title and tags
	needle := strings.ToLower(token)
	if strings.Contains(strings.ToLower(r.Title), needle) {
		return true, tokens
	}
	for _, tag := range r.Tags {
		if strings.Contains(strings.ToLower(tag), needle) {
			return true, tokens
		}
	}
	return false, tokens
}

func listRecipes(args cliArgs) {
	recipes := getFilteredRecipes(args)
	items := make([]string, len(recipes))
	for i, r := range recipes {
		items[i] = r.path
	}
	sort.Strings(items)
	printResult(items, args.outputMulti)
}

func listTags(args cliArgs) {
	listElements(args, func(r *recipemd.Recipe) []string {
		return r.Tags
	})
}

func listIngredients(args cliArgs) {
	listElements(args, func(r *recipemd.Recipe) []string {
		ings := r.LeafIngredients()
		names := make([]string, len(ings))
		for i, ing := range ings {
			names[i] = ing.Name
		}
		return names
	})
}

func listUnits(args cliArgs) {
	listElements(args, func(r *recipemd.Recipe) []string {
		var units []string
		for _, ing := range r.LeafIngredients() {
			if ing.Amount != nil && ing.Amount.Unit != nil {
				units = append(units, *ing.Amount.Unit)
			}
		}
		for _, y := range r.Yields {
			if y.Unit != nil {
				units = append(units, *y.Unit)
			}
		}
		return units
	})
}

func listElements(args cliArgs, extractor func(*recipemd.Recipe) []string) {
	recipes := getFilteredRecipes(args)
	counter := make(map[string]int)

	for _, pr := range recipes {
		seen := make(map[string]bool)
		for _, item := range extractor(pr.recipe) {
			if !seen[item] {
				counter[item]++
				seen[item] = true
			}
		}
	}

	if args.count {
		type pair struct {
			name  string
			count int
		}
		pairs := make([]pair, 0, len(counter))
		maxCount := 0
		for name, count := range counter {
			pairs = append(pairs, pair{name, count})
			if count > maxCount {
				maxCount = count
			}
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].count > pairs[j].count
		})
		maxWidth := len(fmt.Sprintf("%d", maxCount))
		items := make([]string, len(pairs))
		for i, p := range pairs {
			items[i] = fmt.Sprintf("%*d %s", maxWidth, p.count, p.name)
		}
		printResult(items, args.outputMulti)
	} else {
		items := make([]string, 0, len(counter))
		for name := range counter {
			items = append(items, name)
		}
		sort.Slice(items, func(i, j int) bool {
			return strings.ToLower(items[i]) < strings.ToLower(items[j])
		})
		printResult(items, args.outputMulti)
	}
}

func printResult(items []string, outputMulti string) {
	if outputMulti == "" {
		if isTerminal() {
			outputMulti = "columns"
		} else {
			outputMulti = "no"
		}
	}

	switch outputMulti {
	case "columns":
		printColumns(items, false)
	case "rows":
		printColumns(items, true)
	default:
		for _, item := range items {
			fmt.Println(item)
		}
	}
}

func isTerminal() bool {
	_, _, err := getTerminalSize(int(os.Stdout.Fd()))
	return err == nil
}

func printColumns(items []string, across bool) {
	if len(items) == 0 {
		return
	}

	maxWidth := 0
	for _, item := range items {
		if len(item) > maxWidth {
			maxWidth = len(item)
		}
	}
	colWidth := maxWidth + 2
	lineWidth := getTerminalWidth()

	colCount := lineWidth / colWidth
	if colCount == 0 {
		colCount = 1
	}
	rowCount := int(math.Ceil(float64(len(items)) / float64(colCount)))

	if across {
		for r := 0; r < rowCount; r++ {
			var row []string
			for c := 0; c < colCount; c++ {
				idx := r*colCount + c
				if idx < len(items) {
					row = append(row, items[idx])
				}
			}
			printRow(row, colWidth)
		}
	} else {
		for r := 0; r < rowCount; r++ {
			var row []string
			for c := 0; c < colCount; c++ {
				idx := c*rowCount + r
				if idx < len(items) {
					row = append(row, items[idx])
				}
			}
			printRow(row, colWidth)
		}
	}
}

func printRow(row []string, colWidth int) {
	if len(row) == 0 {
		return
	}
	var parts []string
	for i, val := range row {
		if i < len(row)-1 {
			parts = append(parts, fmt.Sprintf("%-*s", colWidth, val))
		} else {
			parts = append(parts, val)
		}
	}
	fmt.Println(strings.Join(parts, ""))
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getTerminalSize(fd int) (int, int, error) {
	var ws winsize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}

func getTerminalWidth() int {
	width, _, err := getTerminalSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}
