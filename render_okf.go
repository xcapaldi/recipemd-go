package recipemd

import (
	"bytes"
	"sort"
	"strings"
)

// OKFMetadata holds Open Knowledge Format frontmatter fields that have no
// equivalent in the core RecipeMD model. It is populated by [Parser.Parse]
// when [WithFrontmatter] is set and the document's YAML frontmatter declares
// a non-empty `type` field (the one field the OKF spec requires), and is
// read back by [Parser.RenderOKF] to round-trip those fields.
//
// Title, description, and tags are deliberately not duplicated here: when
// frontmatter supplies them, [Parser.Parse] merges them directly into
// [Recipe.Description] and [Recipe.Tags] as a fallback for whichever the
// RecipeMD body itself leaves unset. [Recipe.Title] always comes from the
// body's required H1 heading.
//
// See https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md.
type OKFMetadata struct {
	// Type classifies the concept (e.g. "Recipe", "BigQuery Table",
	// "Playbook"). Always non-empty when OKFMetadata is non-nil.
	Type string `json:"type"`
	// Resource is the canonical URI of the underlying asset, if any (e.g. a
	// link to the page a recipe was sourced from). Nil when absent.
	Resource *string `json:"resource"`
	// Timestamp is the ISO 8601 last-modified time, if any. Nil when absent.
	Timestamp *string `json:"timestamp"`
	// Extra holds any additional frontmatter keys not otherwise recognised.
	// The OKF spec requires consumers to tolerate and preserve unknown keys,
	// so these are kept (as strings; YAML lists are flattened to a
	// comma-separated string) rather than discarded.
	Extra map[string]string `json:"extra,omitempty"`
}

// RenderOKF renders r as an Open Knowledge Format (OKF) concept document.
//
// OKF documents are Markdown files with a YAML frontmatter block followed by
// a free-form body. The only required frontmatter field is `type`; this
// renders `type` (from [Recipe.OKF] if set, else "Recipe") along with the
// spec's recommended `title`, `description`, `resource`, `tags`, and
// `timestamp` fields, plus any [OKFMetadata.Extra] keys, all copied from r.
// The body is the recipe rendered as standard RecipeMD via
// [Parser.RenderMarkdown], so the file remains a complete, self-contained
// recipe rather than just a metadata stub.
//
// Numeric amounts in the body are rounded to rounding decimal places, as in
// [Parser.RenderMarkdown].
//
// Because the body is plain RecipeMD, a [Parser] constructed with
// [WithFrontmatter] can parse the returned document back into an equivalent
// [Recipe], reconstructing [Recipe.OKF] from the frontmatter.
func (p *Parser) RenderOKF(r *Recipe, rounding int) string {
	var fm strings.Builder
	fm.WriteString("---\n")

	typ := "Recipe"
	if r.OKF != nil && r.OKF.Type != "" {
		typ = r.OKF.Type
	}
	fm.WriteString("type: ")
	fm.WriteString(typ)
	fm.WriteByte('\n')

	fm.WriteString("title: ")
	fm.WriteString(yamlQuote(r.Title))
	fm.WriteByte('\n')

	if r.Description != nil {
		fm.WriteString(yamlBlockScalar("description", *r.Description))
	}

	if r.OKF != nil && r.OKF.Resource != nil {
		fm.WriteString("resource: ")
		fm.WriteString(yamlQuote(*r.OKF.Resource))
		fm.WriteByte('\n')
	}

	if len(r.Tags) > 0 {
		fm.WriteString("tags: [")
		for i, t := range r.Tags {
			if i > 0 {
				fm.WriteString(", ")
			}
			fm.WriteString(yamlQuote(t))
		}
		fm.WriteString("]\n")
	}

	if r.OKF != nil && r.OKF.Timestamp != nil {
		fm.WriteString("timestamp: ")
		fm.WriteString(yamlQuote(*r.OKF.Timestamp))
		fm.WriteByte('\n')
	}

	if r.OKF != nil && len(r.OKF.Extra) > 0 {
		keys := make([]string, 0, len(r.OKF.Extra))
		for k := range r.OKF.Extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fm.WriteString(k)
			fm.WriteString(": ")
			fm.WriteString(yamlQuote(r.OKF.Extra[k]))
			fm.WriteByte('\n')
		}
	}

	fm.WriteString("---\n\n")
	fm.WriteString(p.RenderMarkdown(r, rounding))
	return fm.String()
}

// yamlQuote renders s as a double-quoted YAML scalar, escaping backslashes,
// double quotes, and newlines so the value stays on a single logical line.
func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return `"` + s + `"`
}

// yamlBlockScalar renders key's value as a YAML literal block scalar
// ("key: |"), indenting each line by two spaces. Unlike a quoted scalar,
// this preserves embedded newlines and special characters (quotes,
// backslashes, markdown/HTML) verbatim and human-readably, which matters
// for RecipeMD descriptions that may span multiple paragraphs.
func yamlBlockScalar(key, value string) string {
	var b strings.Builder
	b.WriteString(key)
	b.WriteString(": |\n")
	for _, line := range strings.Split(value, "\n") {
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// frontmatterFields is the result of a best-effort parse of a YAML
// frontmatter block's top-level keys, limited to the subset of YAML syntax
// the OKF spec and [Parser.RenderOKF] use: plain, single-, and
// double-quoted scalars; flow sequences ("[a, b]"); block sequences
// ("- a\n- b"); and literal/folded block scalars ("|" and ">").
type frontmatterFields struct {
	Type        string
	Title       *string
	Description *string
	Tags        []string
	Resource    *string
	Timestamp   *string
	Extra       map[string]string
}

// splitFrontmatter locates a YAML (---) or TOML (+++) frontmatter block at
// the start of source. It returns the frontmatter's raw content (excluding
// the fence lines), the remaining document body, the fence delimiter used,
// and whether a frontmatter block was found at all.
//
// A closing fence must start at column 0 (no leading whitespace). This
// mirrors YAML's own rule that document markers cannot be indented, and
// ensures a literal "---" line inside an indented block scalar value (e.g.
// inside a multi-paragraph description rendered by [Parser.RenderOKF]) is
// never mistaken for the closing fence.
func splitFrontmatter(source []byte) (content, body, fence []byte, found bool) {
	if len(source) < 3 {
		return nil, source, nil, false
	}
	switch {
	case bytes.HasPrefix(source, []byte("---")):
		fence = []byte("---")
	case bytes.HasPrefix(source, []byte("+++")):
		fence = []byte("+++")
	default:
		return nil, source, nil, false
	}

	firstNL := bytes.IndexByte(source, '\n')
	if firstNL < 0 {
		return nil, source, nil, false
	}
	if len(bytes.TrimSpace(source[:firstNL])) != len(fence) {
		return nil, source, nil, false
	}

	rest := source[firstNL+1:]
	pos := 0
	for pos < len(rest) {
		lineEnd := bytes.IndexByte(rest[pos:], '\n')
		var line []byte
		var next int
		if lineEnd < 0 {
			line = rest[pos:]
			next = len(rest)
		} else {
			line = rest[pos : pos+lineEnd]
			next = pos + lineEnd + 1
		}
		if bytes.Equal(bytes.TrimRight(line, " \t\r"), fence) {
			return rest[:pos], rest[next:], fence, true
		}
		pos = next
		if lineEnd < 0 {
			break
		}
	}
	return nil, source, nil, false
}

// stripFrontmatter removes YAML (---) or TOML (+++) frontmatter from the
// beginning of source. Returns source unchanged if no frontmatter is found.
func stripFrontmatter(source []byte) []byte {
	_, body, _, found := splitFrontmatter(source)
	if !found {
		return source
	}
	return body
}

// parseOKFFrontmatter performs a best-effort parse of a YAML frontmatter
// block's top-level keys. It returns nil if the content has no non-empty
// `type` key, since that is the one field that marks frontmatter as OKF
// metadata rather than arbitrary frontmatter from another tool (e.g.
// Denote) that this package otherwise leaves untouched.
func parseOKFFrontmatter(content []byte) *frontmatterFields {
	lines := strings.Split(string(content), "\n")
	fields := &frontmatterFields{}
	i := 0
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" || line[0] == ' ' || line[0] == '\t' {
			i++
			continue
		}
		key, value, ok := splitYAMLKeyValue(line)
		if !ok {
			i++
			continue
		}
		switch {
		case value == "|" || value == "|-" || value == "|+":
			block, consumed := collectBlockScalar(lines[i+1:], false)
			setOKFField(fields, key, block)
			i += 1 + consumed
		case value == ">" || value == ">-" || value == ">+":
			block, consumed := collectBlockScalar(lines[i+1:], true)
			setOKFField(fields, key, block)
			i += 1 + consumed
		case value == "" && i+1 < len(lines) && isBlockSequenceItem(lines[i+1]):
			items, consumed := collectBlockSequence(lines[i+1:])
			setOKFListField(fields, key, items)
			i += 1 + consumed
		case strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]"):
			setOKFListField(fields, key, parseFlowList(value))
			i++
		default:
			setOKFField(fields, key, unquoteYAMLScalar(value))
			i++
		}
	}
	if fields.Type == "" {
		return nil
	}
	return fields
}

// splitYAMLKeyValue splits a top-level "key: value" line on its first colon.
// The key must consist solely of letters, digits, underscores, and hyphens;
// ok is false if the line doesn't match that shape.
func splitYAMLKeyValue(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx <= 0 {
		return "", "", false
	}
	key = line[:idx]
	for _, r := range key {
		if !isYAMLKeyRune(r) {
			return "", "", false
		}
	}
	return key, strings.TrimSpace(line[idx+1:]), true
}

func isYAMLKeyRune(r rune) bool {
	return r == '_' || r == '-' ||
		(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// isBlockSequenceItem reports whether line is a YAML block sequence item
// ("- value" or a bare "-"), ignoring leading indentation.
func isBlockSequenceItem(line string) bool {
	t := strings.TrimLeft(line, " \t")
	return t == "-" || strings.HasPrefix(t, "- ")
}

// collectBlockSequence gathers consecutive block sequence items from rest,
// returning the parsed (unquoted) item values and the number of lines consumed.
func collectBlockSequence(rest []string) (items []string, consumed int) {
	for _, line := range rest {
		if !isBlockSequenceItem(line) {
			break
		}
		t := strings.TrimLeft(line, " \t")
		val := strings.TrimSpace(strings.TrimPrefix(t, "-"))
		items = append(items, unquoteYAMLScalar(val))
		consumed++
	}
	return items, consumed
}

// collectBlockScalar gathers a YAML literal ("|") or folded (">") block
// scalar's content from rest (the lines following the "key: |" line),
// stopping at the first line that starts at column 0 (a new top-level key
// or the end of the frontmatter content). It returns the assembled value
// and the number of lines consumed.
func collectBlockScalar(rest []string, folded bool) (string, int) {
	var collected []string
	consumed := 0
	indent := -1
	for _, line := range rest {
		if strings.TrimSpace(line) == "" {
			collected = append(collected, "")
			consumed++
			continue
		}
		lineIndent := len(line) - len(strings.TrimLeft(line, " "))
		if lineIndent == 0 {
			break
		}
		if indent == -1 || lineIndent < indent {
			indent = lineIndent
		}
		collected = append(collected, line[indent:])
		consumed++
	}
	for len(collected) > 0 && collected[len(collected)-1] == "" {
		collected = collected[:len(collected)-1]
	}
	if folded {
		return foldLines(collected), consumed
	}
	return strings.Join(collected, "\n"), consumed
}

// foldLines applies a simplified YAML folded-scalar (">") join: consecutive
// non-blank lines are joined with a single space, and blank lines introduce
// a paragraph break.
func foldLines(lines []string) string {
	var b strings.Builder
	prevBlank := true
	for _, line := range lines {
		if line == "" {
			b.WriteString("\n")
			prevBlank = true
			continue
		}
		if !prevBlank {
			b.WriteString(" ")
		}
		b.WriteString(line)
		prevBlank = false
	}
	return b.String()
}

// parseFlowList parses a YAML flow sequence ("[a, b, \"c, d\"]") into its
// (unquoted) item values, splitting on top-level commas only.
func parseFlowList(value string) []string {
	inner := strings.TrimSpace(value)
	inner = strings.TrimSuffix(strings.TrimPrefix(inner, "["), "]")
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return nil
	}
	var items []string
	var cur strings.Builder
	var inQuote byte
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		switch {
		case inQuote != 0:
			cur.WriteByte(c)
			if c == inQuote && inner[i-1] != '\\' {
				inQuote = 0
			}
		case c == '"' || c == '\'':
			inQuote = c
			cur.WriteByte(c)
		case c == ',':
			items = append(items, unquoteYAMLScalar(strings.TrimSpace(cur.String())))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	items = append(items, unquoteYAMLScalar(strings.TrimSpace(cur.String())))
	return items
}

// unquoteYAMLScalar strips and unescapes a single- or double-quoted YAML
// scalar, or returns value unchanged if it isn't quoted.
func unquoteYAMLScalar(value string) string {
	v := strings.TrimSpace(value)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return unescapeDoubleQuoted(v[1 : len(v)-1])
	}
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'")
	}
	return v
}

// unescapeDoubleQuoted processes backslash escapes (\n, \t, \", \\) in the
// inner content of a double-quoted YAML scalar.
func unescapeDoubleQuoted(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(s[i])
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// setOKFField assigns a scalar value to the matching known field of fields,
// or stores it under key in Extra if unrecognised.
func setOKFField(fields *frontmatterFields, key, value string) {
	switch key {
	case "type":
		fields.Type = value
	case "title":
		fields.Title = &value
	case "description":
		fields.Description = &value
	case "resource":
		fields.Resource = &value
	case "timestamp":
		fields.Timestamp = &value
	default:
		if fields.Extra == nil {
			fields.Extra = map[string]string{}
		}
		fields.Extra[key] = value
	}
}

// setOKFListField assigns a list value to the matching known field of
// fields, or stores it (comma-joined) under key in Extra if unrecognised.
func setOKFListField(fields *frontmatterFields, key string, items []string) {
	switch key {
	case "tags":
		fields.Tags = items
	default:
		if fields.Extra == nil {
			fields.Extra = map[string]string{}
		}
		fields.Extra[key] = strings.Join(items, ", ")
	}
}
