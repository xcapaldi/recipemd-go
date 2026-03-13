package recipemd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGolden(t *testing.T) {
	suites := []struct {
		name string
		dir  string
		opts []Option
	}{
		{"default", "testdata/golden", nil},
		{"gfm", "testdata/golden/gfm", []Option{WithGithubFormattedMarkdown()}},
		{"frontmatter", "testdata/golden/frontmatter", []Option{WithFrontmatter()}},
	}

	for _, suite := range suites {
		t.Run(suite.name, func(t *testing.T) {
			files, err := filepath.Glob(filepath.Join(suite.dir, "*.md"))
			if err != nil {
				t.Fatal(err)
			}
			if len(files) == 0 {
				t.Fatalf("no test files in %s", suite.dir)
			}

			for _, mdFile := range files {
				name := strings.TrimSuffix(filepath.Base(mdFile), ".md")
				isInvalid := strings.HasSuffix(name, ".invalid")

				t.Run(name, func(t *testing.T) {
					input, err := os.ReadFile(mdFile)
					if err != nil {
						t.Fatal(err)
					}

					recipe, parseErr := NewParser(suite.opts...).Parse(input)

					if isInvalid {
						if parseErr == nil {
							t.Errorf("expected parse error for invalid case")
						}
						return
					}

					if parseErr != nil {
						t.Fatalf("Parse error: %v", parseErr)
					}

					got, err := json.MarshalIndent(recipe, "", "  ")
					if err != nil {
						t.Fatal(err)
					}
					jsonFile := strings.TrimSuffix(mdFile, ".md") + ".json"
					expected, err := os.ReadFile(jsonFile)
					if err != nil {
						t.Fatal(err)
					}

					var expectedMap, gotMap map[string]any
					json.Unmarshal(expected, &expectedMap)
					json.Unmarshal(got, &gotMap)

					expectedNorm, _ := json.Marshal(expectedMap)
					gotNorm, _ := json.Marshal(gotMap)

					if string(expectedNorm) != string(gotNorm) {
						t.Errorf("mismatch\nexpected:\n%s\ngot:\n%s", expected, got)
					}
				})
			}
		})
	}
}
