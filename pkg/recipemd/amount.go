package recipemd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// unicodeFractions maps Unicode vulgar fractions to their decimal values.
var unicodeFractions = map[rune]float64{
	'½': 0.5,
	'⅓': 1.0 / 3.0,
	'⅔': 2.0 / 3.0,
	'¼': 0.25,
	'¾': 0.75,
	'⅕': 0.2,
	'⅖': 0.4,
	'⅗': 0.6,
	'⅘': 0.8,
	'⅙': 1.0 / 6.0,
	'⅚': 5.0 / 6.0,
	'⅐': 1.0 / 7.0,
	'⅛': 0.125,
	'⅜': 0.375,
	'⅝': 0.625,
	'⅞': 0.875,
	'⅑': 1.0 / 9.0,
	'⅒': 0.1,
}

// numberPattern matches various number formats, including those starting with decimal point
var numberPattern = regexp.MustCompile(`^(\d*[.,]?\d+)\s*(.*)$`)

// fractionPattern matches fractions like 1/2, 3/4
var fractionPattern = regexp.MustCompile(`^(\d+)\s*/\s*(\d+)\s*(.*)$`)

// mixedNumberPattern matches mixed numbers like "1 1/2"
var mixedNumberPattern = regexp.MustCompile(`^(\d+)\s+(\d+)\s*/\s*(\d+)\s*(.*)$`)

// parseAmount parses an amount string and returns the factor and unit.
// It handles various number formats:
// - Simple integers: "5"
// - Decimals: "1.5" or "1,5"
// - Fractions: "1/2", "3/4"
// - Mixed numbers: "1 1/2"
// - Unicode fractions: "½", "¼"
// - Combined: "1½"
func parseAmount(s string) (factor string, unit *string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil, fmt.Errorf("empty amount string")
	}

	var value float64
	var rest string
	var parsed bool

	// Try parsing unicode fractions at the start
	runes := []rune(s)
	if frac, ok := unicodeFractions[runes[0]]; ok {
		value = frac
		rest = strings.TrimSpace(string(runes[1:]))
		parsed = true
	}

	// Try mixed number with unicode fraction: "1½"
	if !parsed {
		for i, r := range runes {
			if frac, ok := unicodeFractions[r]; ok {
				prefix := strings.TrimSpace(string(runes[:i]))
				if prefix != "" {
					wholeNum, err := parseNumber(prefix)
					if err == nil {
						value = wholeNum + frac
						rest = strings.TrimSpace(string(runes[i+1:]))
						parsed = true
						break
					}
				}
			}
		}
	}

	// Try mixed number: "1 1/2 cups"
	if !parsed {
		if m := mixedNumberPattern.FindStringSubmatch(s); m != nil {
			whole, _ := strconv.ParseFloat(m[1], 64)
			num, _ := strconv.ParseFloat(m[2], 64)
			denom, _ := strconv.ParseFloat(m[3], 64)
			if denom != 0 {
				value = whole + num/denom
				rest = strings.TrimSpace(m[4])
				parsed = true
			}
		}
	}

	// Try fraction: "1/2 cups"
	if !parsed {
		if m := fractionPattern.FindStringSubmatch(s); m != nil {
			num, _ := strconv.ParseFloat(m[1], 64)
			denom, _ := strconv.ParseFloat(m[2], 64)
			if denom != 0 {
				value = num / denom
				rest = strings.TrimSpace(m[3])
				parsed = true
			}
		}
	}

	// Try simple number or decimal: "1.5 cups" or "1,5 cups"
	if !parsed {
		if m := numberPattern.FindStringSubmatch(s); m != nil {
			numStr := strings.Replace(m[1], ",", ".", 1)
			val, err := strconv.ParseFloat(numStr, 64)
			if err == nil {
				value = val
				rest = strings.TrimSpace(m[2])
				parsed = true
			}
		}
	}

	if !parsed {
		return "", nil, fmt.Errorf("could not parse amount: %s", s)
	}

	// Format the factor
	factor = formatFactor(value)

	// Set unit if there's remaining text
	if rest != "" {
		unit = &rest
	}

	return factor, unit, nil
}

// parseNumber parses a simple number (integer or decimal with . or ,)
func parseNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.Replace(s, ",", ".", 1)
	return strconv.ParseFloat(s, 64)
}

// formatFactor formats a float64 as a string for JSON output.
// It uses the minimum number of decimal places needed.
func formatFactor(f float64) string {
	// Check if it's effectively an integer
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}

	// Use sufficient precision, then trim trailing zeros
	s := strconv.FormatFloat(f, 'f', 10, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// parseYieldOrAmount parses a yield or amount string that may or may not have a number.
// For yields like "4 Servings", "200g", or just text.
func parseYieldOrAmount(s string) (factor string, unit *string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil, fmt.Errorf("empty string")
	}

	// Try to parse as amount first
	factor, unit, err = parseAmount(s)
	if err == nil {
		return factor, unit, nil
	}

	// If that fails, check if there's any number at the start
	// This handles cases where the entire string is a unit without amount
	return "", nil, fmt.Errorf("no amount found in: %s", s)
}

// splitByComma splits a string by commas, but ignores commas between digits
// (to handle European decimal notation like "1,5 cups").
func splitByComma(s string) []string {
	var result []string
	var current strings.Builder
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		if runes[i] == ',' {
			// Check if comma is between digits (European decimal)
			prevIsDigit := i > 0 && unicode.IsDigit(runes[i-1])
			nextIsDigit := i+1 < len(runes) && unicode.IsDigit(runes[i+1])

			if prevIsDigit && nextIsDigit {
				// This is a decimal comma, keep it
				current.WriteRune(runes[i])
			} else {
				// This is a separator, split here
				item := strings.TrimSpace(current.String())
				if item != "" {
					result = append(result, item)
				}
				current.Reset()
			}
		} else {
			current.WriteRune(runes[i])
		}
	}

	// Add the last item
	item := strings.TrimSpace(current.String())
	if item != "" {
		result = append(result, item)
	}

	return result
}
