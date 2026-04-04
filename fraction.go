package recipemd

import (
	"fmt"
	"math"
	"strings"
)

// fractionThreshold is the maximum absolute difference for snapping a
// decimal value to a common fraction.
const fractionThreshold = 0.04

// fractionEntry maps a fractional value to its Unicode vulgar fraction glyph.
type fractionEntry struct {
	value float64
	glyph string
}

// commonFractions lists cooking-friendly fractions ordered by value.
var commonFractions = []fractionEntry{
	{0.125, "⅛"},
	{0.25, "¼"},
	{1.0 / 3.0, "⅓"},
	{0.375, "⅜"},
	{0.5, "½"},
	{0.625, "⅝"},
	{2.0 / 3.0, "⅔"},
	{0.75, "¾"},
	{0.875, "⅞"},
}

// snapToFraction returns the closest common fraction glyph for frac (a value
// in [0, 1)) if one is within fractionThreshold. Returns "" if no match.
func snapToFraction(frac float64) string {
	best := ""
	bestDiff := fractionThreshold
	for _, f := range commonFractions {
		diff := math.Abs(frac - f.value)
		if diff < bestDiff {
			bestDiff = diff
			best = f.glyph
		}
	}
	return best
}

// FormatFactorFraction formats the numeric factor using vulgar fractions where
// possible. Examples: 1.5 -> "1½", 0.333 -> "⅓", 2.0 -> "2", 1.75 -> "1¾".
// When the fractional part doesn't match any common fraction, falls back to a
// two-decimal-place string.
func (a Amount) FormatFactorFraction() string {
	negative := a.Factor < 0
	f := math.Abs(a.Factor)

	whole := int(f)
	frac := f - float64(whole)

	var s string
	if frac < fractionThreshold {
		// Essentially a whole number.
		s = fmt.Sprintf("%d", whole)
	} else if glyph := snapToFraction(frac); glyph != "" {
		if whole == 0 {
			s = glyph
		} else {
			s = fmt.Sprintf("%d%s", whole, glyph)
		}
	} else {
		// Fallback to decimal.
		s = a.FormatFactor(2)
		if negative {
			// FormatFactor already handles the sign.
			return s
		}
	}

	if negative {
		return "-" + s
	}
	return s
}

// SerializeFraction formats the amount using fraction-friendly notation.
// When a unit is present it is appended after a space (e.g. "1½ cup").
func (a Amount) SerializeFraction() string {
	s := a.FormatFactorFraction()
	if a.Unit != nil {
		return s + " " + *a.Unit
	}
	return s
}

// SerializeCompound formats a slice of amounts as a human-readable compound
// string joined with " and " (e.g. "1 cup and 1 tbsp"). Each component uses
// fraction-friendly formatting.
func SerializeCompound(amounts []Amount) string {
	parts := make([]string, len(amounts))
	for i, a := range amounts {
		parts[i] = a.SerializeFraction()
	}
	return strings.Join(parts, " and ")
}
