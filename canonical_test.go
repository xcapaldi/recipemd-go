package recipemd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonical(t *testing.T) {
	files, err := filepath.Glob("testdata/canonical/*.md")
	if err != nil {
		t.Fatal(err)
	}

	for _, mdFile := range files {
		name := strings.TrimSuffix(filepath.Base(mdFile), ".md")
		isInvalid := strings.HasSuffix(name, ".invalid")

		t.Run(name, func(t *testing.T) {
			input, err := os.ReadFile(mdFile)
			if err != nil {
				t.Fatal(err)
			}

			recipe, parseErr := NewParser().Parse(input)

			if isInvalid {
				if parseErr == nil {
					t.Errorf("expected parse error for invalid case")
				}
				return
			}

			if parseErr != nil {
				t.Fatalf("Parse error: %v", parseErr)
			}

			jsonFile := strings.TrimSuffix(mdFile, ".md") + ".json"
			expected, err := os.ReadFile(jsonFile)
			if err != nil {
				t.Fatal(err)
			}

			got, err := json.MarshalIndent(recipe, "", "  ")
			if err != nil {
				t.Fatal(err)
			}

			// normalize for comparison
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
}
