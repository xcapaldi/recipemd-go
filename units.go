package recipemd

import "strings"

// UnitSystem represents a measurement system preference.
type UnitSystem int

const (
	// Metric selects the metric system (grams, kilograms, milliliters, liters).
	Metric UnitSystem = iota
	// Imperial selects the imperial/US customary system (ounces, pounds, cups, tablespoons, etc.).
	Imperial
)

// unitKind classifies what a unit measures.
type unitKind int

const (
	kindWeight      unitKind = iota
	kindVolume               // ml, L, cup, tbsp, tsp, fl oz, pint, quart, gallon
	kindTemperature          // reserved for future F <-> C support
)

// canonicalUnit holds the definition of one unit.
type canonicalUnit struct {
	key          string     // canonical lowercase key: "tbsp", "g", "ml"
	display      string     // preferred display form
	kind         unitKind   // weight, volume, or temperature
	system       UnitSystem // metric or imperial
	toBase       float64    // multiply factor by this to get base unit (g for weight, ml for volume)
	aliases      []string   // all accepted spellings (lowercase)
	noAutoSelect bool       // if true, bestUnit will not auto-select this unit (e.g. cl, dl, pint, fl oz)
}

// Unit tables. Base units: grams (weight), milliliters (volume).

var weightUnits = []canonicalUnit{
	// Metric
	{key: "mg", display: "mg", kind: kindWeight, system: Metric, toBase: 0.001, aliases: []string{"mg", "milligram", "milligrams"}},
	{key: "g", display: "g", kind: kindWeight, system: Metric, toBase: 1, aliases: []string{"g", "gram", "grams"}},
	{key: "kg", display: "kg", kind: kindWeight, system: Metric, toBase: 1000, aliases: []string{"kg", "kilogram", "kilograms", "kilo", "kilos"}},
	// Imperial
	{key: "oz", display: "oz", kind: kindWeight, system: Imperial, toBase: 28.3495, aliases: []string{"oz", "ounce", "ounces"}},
	{key: "lb", display: "lb", kind: kindWeight, system: Imperial, toBase: 453.592, aliases: []string{"lb", "lbs", "pound", "pounds"}},
}

var volumeUnits = []canonicalUnit{
	// Metric
	{key: "ml", display: "ml", kind: kindVolume, system: Metric, toBase: 1, aliases: []string{"ml", "milliliter", "milliliters", "millilitre", "millilitres"}},
	{key: "cl", display: "cl", kind: kindVolume, system: Metric, toBase: 10, aliases: []string{"cl", "centiliter", "centiliters", "centilitre", "centilitres"}, noAutoSelect: true},
	{key: "dl", display: "dl", kind: kindVolume, system: Metric, toBase: 100, aliases: []string{"dl", "deciliter", "deciliters", "decilitre", "decilitres"}, noAutoSelect: true},
	{key: "L", display: "L", kind: kindVolume, system: Metric, toBase: 1000, aliases: []string{"l", "liter", "liters", "litre", "litres"}},
	// Imperial
	{key: "tsp", display: "tsp", kind: kindVolume, system: Imperial, toBase: 4.92892, aliases: []string{"tsp", "teaspoon", "teaspoons"}},
	{key: "tbsp", display: "tbsp", kind: kindVolume, system: Imperial, toBase: 14.7868, aliases: []string{"tbsp", "tbs", "tablespoon", "tablespoons"}},
	{key: "fl oz", display: "fl oz", kind: kindVolume, system: Imperial, toBase: 29.5735, aliases: []string{"fl oz", "fluid ounce", "fluid ounces"}, noAutoSelect: true},
	{key: "cup", display: "cup", kind: kindVolume, system: Imperial, toBase: 236.588, aliases: []string{"cup", "cups"}},
	{key: "pint", display: "pint", kind: kindVolume, system: Imperial, toBase: 473.176, aliases: []string{"pint", "pints", "pt"}, noAutoSelect: true},
	{key: "quart", display: "quart", kind: kindVolume, system: Imperial, toBase: 946.353, aliases: []string{"quart", "quarts", "qt"}},
	{key: "gallon", display: "gallon", kind: kindVolume, system: Imperial, toBase: 3785.41, aliases: []string{"gallon", "gallons", "gal"}},
}

// aliasMap maps every known lowercase alias to its canonical unit definition.
var aliasMap map[string]*canonicalUnit

func init() {
	all := make([]canonicalUnit, 0, len(weightUnits)+len(volumeUnits))
	all = append(all, weightUnits...)
	all = append(all, volumeUnits...)

	aliasMap = make(map[string]*canonicalUnit, len(all)*3)
	for i := range all {
		for _, alias := range all[i].aliases {
			aliasMap[alias] = &all[i]
		}
	}
}

// lookupUnit returns the canonical unit for a raw unit string, or nil if unknown.
// Lookup is case-insensitive.
func lookupUnit(raw string) *canonicalUnit {
	return aliasMap[strings.ToLower(strings.TrimSpace(raw))]
}

// unitHierarchy returns the units for a given kind and system, ordered from
// largest to smallest by toBase value. Used for compound decomposition.
func unitHierarchy(kind unitKind, system UnitSystem) []*canonicalUnit {
	var source []canonicalUnit
	switch kind {
	case kindWeight:
		source = weightUnits
	case kindVolume:
		source = volumeUnits
	default:
		return nil
	}

	var result []*canonicalUnit
	for i := range source {
		if source[i].system == system {
			result = append(result, &source[i])
		}
	}

	// Sort descending by toBase (largest unit first).
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].toBase > result[j-1].toBase; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result
}
