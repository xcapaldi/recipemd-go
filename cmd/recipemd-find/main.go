package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/exec"
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
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if args.version {
		fmt.Printf("recipemd-find %s\n", version)
		os.Exit(0)
	}

	if args.action == "" {
		fmt.Fprintf(os.Stderr, "Error: an action is required (recipes, tags, ingredients, units)\n")
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

type cliArgs struct {
	version     bool
	expression  string
	noMessages  bool
	outputMulti string // "no", "columns", "rows", or ""
	action      string
	folder      string
	count       bool
	searcher    string // external search program (e.g. "grep", "rg")
}

func parseArgs(raw []string) (cliArgs, error) {
	var args cliArgs
	args.folder = "."

	i := 0
	for i < len(raw) {
		arg := raw[i]
		switch arg {
		case "-v", "--version":
			args.version = true
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		case "-e", "--expression":
			i++
			if i >= len(raw) {
				return args, fmt.Errorf("missing value for %s", arg)
			}
			args.expression = raw[i]
		case "-s", "--no-messages":
			args.noMessages = true
		case "-1":
			args.outputMulti = "no"
		case "-C":
			args.outputMulti = "columns"
		case "-x":
			args.outputMulti = "rows"
		case "-c", "--count":
			args.count = true
		case "--searcher":
			i++
			if i >= len(raw) {
				return args, fmt.Errorf("missing value for %s", arg)
			}
			args.searcher = raw[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return args, fmt.Errorf("unknown option %q", arg)
			}
			if args.action == "" {
				args.action = arg
			} else {
				args.folder = arg
			}
		}
		i++
	}
	return args, nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: recipemd-find [options] <action> [folder]

Find recipes, ingredients and units by filter expression

Actions:
  recipes       list recipe paths
  tags          list used tags
  ingredients   list used ingredients
  units         list used units

Options:
  -v, --version       show version
  -h, --help          show help
  -e, --expression E  filter expression, e.g. "cake and vegan or ingr:cheese"
  -s, --no-messages   suppress error messages
  -1                  force output to be one entry per line
  -C                  force multi-column output
  -x                  multi-column output sorted across columns
  -c, --count         count number of uses (tags, ingredients, units only)
  --searcher PROG     use external search program (grep, rg) to pre-filter
                      files. The filter expression is translated to the
                      program's syntax. Candidate files are then parsed and
                      verified with the RecipeMD-aware filter.
`)
}

type parsedRecipe struct {
	recipe *recipemd.Recipe
	path   string
}

func getFilteredRecipes(args cliArgs) []parsedRecipe {
	// When a searcher is specified with an expression, use it to pre-filter
	// candidate files before doing full RecipeMD parsing.
	if args.searcher != "" && args.expression != "" {
		return getFilteredRecipesWithSearcher(args)
	}
	return getFilteredRecipesBuiltin(args)
}

func getFilteredRecipesBuiltin(args cliArgs) []parsedRecipe {
	var results []parsedRecipe

	folder := args.folder
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
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

		recipe, err := recipemd.NewParser().Parse(data)
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

func getFilteredRecipesWithSearcher(args cliArgs) []parsedRecipe {
	folder := args.folder

	// Parse expression into AST, translate to external searcher, get candidate files
	ast := parseExprAST(tokenize(args.expression))
	candidates, err := evalSearcherAST(ast, args.searcher, folder, args.noMessages)
	if err != nil {
		if !args.noMessages {
			fmt.Fprintf(os.Stderr, "Searcher error: %v\n", err)
		}
		return nil
	}

	// Parse candidates and verify with exact RecipeMD-aware filter
	var results []parsedRecipe
	for _, path := range candidates {
		fullPath := filepath.Join(folder, path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			if !args.noMessages {
				fmt.Fprintf(os.Stderr, "An error occurred, skipping %s: %v\n", path, err)
			}
			continue
		}

		recipe, err := recipemd.NewParser().Parse(data)
		if err != nil {
			if !args.noMessages {
				fmt.Fprintf(os.Stderr, "An error occurred, skipping %s: %v\n", path, err)
			}
			continue
		}

		// Exact filter verification — the external searcher is a text-level
		// pre-filter that doesn't understand RecipeMD structure, so we
		// confirm with the real filter.
		if !matchesFilter(recipe, args.expression) {
			continue
		}

		results = append(results, parsedRecipe{recipe: recipe, path: path})
	}
	return results
}

// --- Expression AST ---

type exprNodeKind int

const (
	nodeLeaf exprNodeKind = iota
	nodeAnd
	nodeOr
	nodeNot
)

type exprNode struct {
	kind     exprNodeKind
	term     string // for nodeLeaf: raw term like "cake", "ingr:cheese", "tag:vegan"
	children []*exprNode
}

// parseExprAST parses a tokenized filter expression into an AST.
func parseExprAST(tokens []string) *exprNode {
	node, _ := parseOrAST(tokens)
	return node
}

func parseOrAST(tokens []string) (*exprNode, []string) {
	left, tokens := parseAndAST(tokens)
	var orChildren []*exprNode
	orChildren = append(orChildren, left)
	for len(tokens) > 0 && strings.EqualFold(tokens[0], "or") {
		tokens = tokens[1:]
		right, rest := parseAndAST(tokens)
		orChildren = append(orChildren, right)
		tokens = rest
	}
	if len(orChildren) == 1 {
		return orChildren[0], tokens
	}
	return &exprNode{kind: nodeOr, children: orChildren}, tokens
}

func parseAndAST(tokens []string) (*exprNode, []string) {
	left, tokens := parsePrimaryAST(tokens)
	var andChildren []*exprNode
	andChildren = append(andChildren, left)
	for len(tokens) > 0 && strings.EqualFold(tokens[0], "and") {
		tokens = tokens[1:]
		right, rest := parsePrimaryAST(tokens)
		andChildren = append(andChildren, right)
		tokens = rest
	}
	if len(andChildren) == 1 {
		return andChildren[0], tokens
	}
	return &exprNode{kind: nodeAnd, children: andChildren}, tokens
}

func parsePrimaryAST(tokens []string) (*exprNode, []string) {
	if len(tokens) == 0 {
		return &exprNode{kind: nodeLeaf, term: ""}, tokens
	}
	token := tokens[0]
	tokens = tokens[1:]
	if strings.EqualFold(token, "not") {
		child, rest := parsePrimaryAST(tokens)
		return &exprNode{kind: nodeNot, children: []*exprNode{child}}, rest
	}
	return &exprNode{kind: nodeLeaf, term: token}, tokens
}

// --- External searcher evaluation ---

// evalSearcherAST evaluates an expression AST using the external search program.
// Returns relative paths of matching .md files.
func evalSearcherAST(node *exprNode, searcher, folder string, noMessages bool) ([]string, error) {
	switch node.kind {
	case nodeLeaf:
		if node.term == "" {
			return nil, nil
		}
		return searcherGrep(searcher, folder, extractSearchTerm(node.term), false, noMessages)

	case nodeNot:
		return searcherGrep(searcher, folder, extractSearchTerm(node.children[0].term), true, noMessages)

	case nodeAnd:
		// Intersection: start with first child's results, narrow down
		result, err := evalSearcherAST(node.children[0], searcher, folder, noMessages)
		if err != nil {
			return nil, err
		}
		resultSet := stringSet(result)
		for _, child := range node.children[1:] {
			childFiles, err := evalSearcherAST(child, searcher, folder, noMessages)
			if err != nil {
				return nil, err
			}
			resultSet = intersect(resultSet, stringSet(childFiles))
		}
		return setToSlice(resultSet), nil

	case nodeOr:
		// Union: combine all children's results
		resultSet := make(map[string]bool)
		for _, child := range node.children {
			childFiles, err := evalSearcherAST(child, searcher, folder, noMessages)
			if err != nil {
				return nil, err
			}
			for _, f := range childFiles {
				resultSet[f] = true
			}
		}
		return setToSlice(resultSet), nil
	}
	return nil, nil
}

// extractSearchTerm strips the ingr:/tag: prefix since the external tool
// searches raw text and can't distinguish RecipeMD sections structurally.
func extractSearchTerm(term string) string {
	lower := strings.ToLower(term)
	if strings.HasPrefix(lower, "ingr:") {
		return term[5:]
	}
	if strings.HasPrefix(lower, "tag:") {
		return term[4:]
	}
	return term
}

// searcherGrep runs the external search program and returns matching relative paths.
// If invert is true, returns files that do NOT match the pattern.
func searcherGrep(searcher, folder, pattern string, invert bool, noMessages bool) ([]string, error) {
	prog := filepath.Base(searcher)
	var cmdArgs []string

	switch prog {
	case "rg", "ripgrep":
		// rg -li pattern -g "*.md" folder
		// rg --files-without-match -i pattern -g "*.md" folder
		cmdArgs = append(cmdArgs, "-i")
		if invert {
			cmdArgs = append(cmdArgs, "--files-without-match")
		} else {
			cmdArgs = append(cmdArgs, "-l")
		}
		cmdArgs = append(cmdArgs, "-g", "*.md", "--", pattern, folder)

	case "grep":
		// grep -rli pattern --include="*.md" folder
		// grep -rLi pattern --include="*.md" folder
		cmdArgs = append(cmdArgs, "-r", "-i")
		if invert {
			cmdArgs = append(cmdArgs, "-L")
		} else {
			cmdArgs = append(cmdArgs, "-l")
		}
		cmdArgs = append(cmdArgs, "--include=*.md", "--", pattern, folder)

	default:
		// Generic: assume grep-compatible interface
		cmdArgs = append(cmdArgs, "-r", "-i")
		if invert {
			cmdArgs = append(cmdArgs, "-L")
		} else {
			cmdArgs = append(cmdArgs, "-l")
		}
		cmdArgs = append(cmdArgs, "--include=*.md", "--", pattern, folder)
	}

	cmd := exec.Command(searcher, cmdArgs...)
	cmd.Stderr = os.Stderr
	if noMessages {
		cmd.Stderr = nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start %s: %w", searcher, err)
	}

	var files []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Convert to relative path
		rel, err := filepath.Rel(folder, line)
		if err != nil {
			rel = line
		}
		files = append(files, rel)
	}

	// grep exits 1 when no matches found — that's not an error
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return files, nil
		}
		return files, nil // be lenient with searcher exit codes
	}

	return files, nil
}

func stringSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

func intersect(a, b map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for k := range a {
		if b[k] {
			result[k] = true
		}
	}
	return result
}

func setToSlice(s map[string]bool) []string {
	result := make([]string, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
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
