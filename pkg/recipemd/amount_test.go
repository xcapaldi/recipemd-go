package recipemd

import (
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFactor string
		wantUnit   *string
		wantErr    bool
	}{
		// Simple integers
		{
			name:       "simple integer",
			input:      "5",
			wantFactor: "5",
			wantUnit:   nil,
		},
		{
			name:       "integer with unit",
			input:      "5 cups",
			wantFactor: "5",
			wantUnit:   strPtr("cups"),
		},

		// Decimals with dot
		{
			name:       "decimal with dot",
			input:      "1.5",
			wantFactor: "1.5",
			wantUnit:   nil,
		},
		{
			name:       "decimal with dot and unit",
			input:      "1.5 cups",
			wantFactor: "1.5",
			wantUnit:   strPtr("cups"),
		},
		{
			name:       "decimal starting with dot",
			input:      ".5 teaspoon",
			wantFactor: "0.5",
			wantUnit:   strPtr("teaspoon"),
		},

		// Decimals with comma (European)
		{
			name:       "decimal with comma",
			input:      "1,5 Tassen",
			wantFactor: "1.5",
			wantUnit:   strPtr("Tassen"),
		},

		// Fractions
		{
			name:       "simple fraction",
			input:      "1/2",
			wantFactor: "0.5",
			wantUnit:   nil,
		},
		{
			name:       "fraction with unit",
			input:      "3/4 cup",
			wantFactor: "0.75",
			wantUnit:   strPtr("cup"),
		},

		// Mixed numbers
		{
			name:       "mixed number",
			input:      "1 1/2 cups",
			wantFactor: "1.5",
			wantUnit:   strPtr("cups"),
		},
		{
			name:       "mixed number with larger fraction",
			input:      "1 1/4 servings",
			wantFactor: "1.25",
			wantUnit:   strPtr("servings"),
		},

		// Unicode fractions
		{
			name:       "unicode half",
			input:      "½ cup",
			wantFactor: "0.5",
			wantUnit:   strPtr("cup"),
		},
		{
			name:       "unicode quarter",
			input:      "¼ kg",
			wantFactor: "0.25",
			wantUnit:   strPtr("kg"),
		},
		{
			name:       "unicode three-quarters",
			input:      "¾ teaspoon",
			wantFactor: "0.75",
			wantUnit:   strPtr("teaspoon"),
		},
		{
			name:       "mixed with unicode fraction",
			input:      "1½ cups",
			wantFactor: "1.5",
			wantUnit:   strPtr("cups"),
		},

		// Edge cases
		{
			name:       "number only no space",
			input:      "20ml",
			wantFactor: "20",
			wantUnit:   strPtr("ml"),
		},
		{
			name:       "large number",
			input:      "200 g",
			wantFactor: "200",
			wantUnit:   strPtr("g"),
		},

		// Errors
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factor, unit, err := parseAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAmount(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if factor != tt.wantFactor {
				t.Errorf("parseAmount(%q) factor = %q, want %q", tt.input, factor, tt.wantFactor)
			}
			if (unit == nil) != (tt.wantUnit == nil) {
				t.Errorf("parseAmount(%q) unit = %v, want %v", tt.input, unit, tt.wantUnit)
			}
			if unit != nil && tt.wantUnit != nil && *unit != *tt.wantUnit {
				t.Errorf("parseAmount(%q) unit = %q, want %q", tt.input, *unit, *tt.wantUnit)
			}
		})
	}
}

func TestSplitByComma(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple split",
			input: "tag1, tag2, tag3",
			want:  []string{"tag1", "tag2", "tag3"},
		},
		{
			name:  "preserves decimal comma",
			input: "1,5 cups, 2 servings",
			want:  []string{"1,5 cups", "2 servings"},
		},
		{
			name:  "multiple decimal commas",
			input: "1,2 cups, 3,4 servings, 5,6 Tassen",
			want:  []string{"1,2 cups", "3,4 servings", "5,6 Tassen"},
		},
		{
			name:  "no commas",
			input: "single item",
			want:  []string{"single item"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace handling",
			input: "  tag1  ,  tag2  ,  tag3  ",
			want:  []string{"tag1", "tag2", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitByComma(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitByComma(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitByComma(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFormatFactor(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{1, "1"},
		{1.5, "1.5"},
		{0.25, "0.25"},
		{0.333333333, "0.333333333"},
		{10, "10"},
		{100.0, "100"},
		{1.0, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatFactor(tt.input)
			if got != tt.want {
				t.Errorf("formatFactor(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
