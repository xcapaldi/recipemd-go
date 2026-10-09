package recipemd

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// TemperatureUnit represents a temperature measurement system.
type TemperatureUnit int

const (
	// Celsius represents the Celsius temperature scale.
	Celsius TemperatureUnit = iota
	// Fahrenheit represents the Fahrenheit temperature scale.
	Fahrenheit
)

// temperatureRe matches patterns like "180°C", "350 °F", "200 celsius", "375 fahrenheit".
// It requires either the ° symbol or the full word to avoid false positives on bare C/F.
var temperatureRe = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(°\s*[CcFf]|[Cc]elsius|[Ff]ahrenheit)`)

// timeRe matches patterns like "30 minutes", "2 hours", "1 hr", "45min", "10 secs".
var timeRe = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(minutes?|mins?|hours?|hrs?|seconds?|secs?)\b`)

func parseTemperatureUnit(unitStr string) TemperatureUnit {
	u := strings.ToLower(strings.ReplaceAll(unitStr, " ", ""))
	if strings.Contains(u, "f") {
		return Fahrenheit
	}
	return Celsius
}

func tempUnitCode(u TemperatureUnit) string {
	if u == Fahrenheit {
		return "F"
	}
	return "C"
}

func tempSymbol(u TemperatureUnit) string {
	if u == Fahrenheit {
		return "°F"
	}
	return "°C"
}

func convertTemp(value float64, from, to TemperatureUnit) float64 {
	if from == to {
		return value
	}
	if from == Celsius {
		return value*9.0/5.0 + 32
	}
	return (value - 32) * 5.0 / 9.0
}

func normalizeTimeUnit(unit string) string {
	u := strings.ToLower(unit)
	switch {
	case strings.HasPrefix(u, "h"):
		return "h"
	case strings.HasPrefix(u, "min"):
		return "min"
	case strings.HasPrefix(u, "sec"):
		return "s"
	default:
		return u
	}
}

// replaceInTextNodes applies a replacement function to text content in an HTML
// string, leaving anything inside < > tags untouched.
func replaceInTextNodes(html string, replacer func(string) string) string {
	var b strings.Builder
	b.Grow(len(html))
	i := 0
	for i < len(html) {
		if html[i] == '<' {
			j := strings.IndexByte(html[i:], '>')
			if j < 0 {
				b.WriteString(html[i:])
				break
			}
			b.WriteString(html[i : i+j+1])
			i += j + 1
		} else {
			j := strings.IndexByte(html[i:], '<')
			if j < 0 {
				b.WriteString(replacer(html[i:]))
				break
			}
			b.WriteString(replacer(html[i : i+j]))
			i += j
		}
	}
	return b.String()
}

func formatTagValue(v float64, rounding int) string {
	if rounding < 0 {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	p := math.Pow(10, float64(rounding))
	rounded := math.Round(v*p) / p
	s := strconv.FormatFloat(rounded, 'f', rounding, 64)
	if rounding > 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// convertTemperaturesInText replaces temperature patterns in plain text,
// converting values from one unit system to another. Temperatures already in
// the target unit are left unchanged.
func convertTemperaturesInText(text string, to TemperatureUnit, rounding int) string {
	return temperatureRe.ReplaceAllStringFunc(text, func(match string) string {
		groups := temperatureRe.FindStringSubmatch(match)
		if groups == nil {
			return match
		}
		value, err := strconv.ParseFloat(groups[1], 64)
		if err != nil {
			return match
		}
		fromUnit := parseTemperatureUnit(groups[2])
		if fromUnit == to {
			return match
		}
		converted := convertTemp(value, fromUnit, to)
		return formatTagValue(converted, rounding) + tempSymbol(to)
	})
}

// annotateTemperatures wraps detected temperature patterns in HTML text nodes
// with <span class="recipemd-temperature"> tags and data attributes.
func annotateTemperatures(html string, rounding int) string {
	return replaceInTextNodes(html, func(text string) string {
		return temperatureRe.ReplaceAllStringFunc(text, func(match string) string {
			groups := temperatureRe.FindStringSubmatch(match)
			if groups == nil {
				return match
			}
			value, err := strconv.ParseFloat(groups[1], 64)
			if err != nil {
				return match
			}
			fromUnit := parseTemperatureUnit(groups[2])
			return fmt.Sprintf(
				`<span class="recipemd-temperature" data-unit="%s" data-value="%s">%s</span>`,
				tempUnitCode(fromUnit),
				formatTagValue(value, rounding),
				match,
			)
		})
	})
}

// annotateTimes wraps detected time patterns in HTML text nodes with
// <span class="recipemd-time"> tags and data attributes.
func annotateTimes(html string) string {
	return replaceInTextNodes(html, func(text string) string {
		return timeRe.ReplaceAllStringFunc(text, func(match string) string {
			groups := timeRe.FindStringSubmatch(match)
			if groups == nil {
				return match
			}
			value, err := strconv.ParseFloat(groups[1], 64)
			if err != nil {
				return match
			}
			norm := normalizeTimeUnit(groups[2])
			return fmt.Sprintf(
				`<span class="recipemd-time" data-unit="%s" data-value="%s">%s</span>`,
				norm,
				strconv.FormatFloat(value, 'f', -1, 64),
				match,
			)
		})
	})
}
