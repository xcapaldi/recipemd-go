package recipemd

import "math"

// convertToBase converts a factor from the given unit to the base unit
// of that kind (grams for weight, milliliters for volume).
func convertToBase(factor float64, from *canonicalUnit) float64 {
	return factor * from.toBase
}

// convertFromBase converts a base-unit value to the given target unit.
func convertFromBase(base float64, to *canonicalUnit) float64 {
	return base / to.toBase
}

// bestUnit selects the most human-friendly unit for a base-unit value within
// the given kind and system. It picks the largest unit that yields a factor >= 1,
// falling back to the smallest unit if the value is very small.
func bestUnit(base float64, kind unitKind, system UnitSystem) *canonicalUnit {
	hierarchy := unitHierarchy(kind, system)
	if len(hierarchy) == 0 {
		return nil
	}
	// hierarchy is largest-first; find the largest unit yielding factor >= 1,
	// skipping units marked noAutoSelect (e.g. cl, dl).
	var smallest *canonicalUnit
	for _, u := range hierarchy {
		if u.noAutoSelect {
			continue
		}
		smallest = u
		if base/u.toBase >= 1 {
			return u
		}
	}
	// Value is very small; use the smallest selectable unit.
	if smallest != nil {
		return smallest
	}
	return hierarchy[len(hierarchy)-1]
}

// convertAmount converts a single Amount to the target system. Returns the
// converted Amount. If the unit is unknown or already in the target system,
// the original Amount is returned unchanged.
func convertAmount(a Amount, system UnitSystem) Amount {
	if a.Unit == nil {
		return a
	}
	cu := lookupUnit(*a.Unit)
	if cu == nil || cu.system == system {
		return a
	}
	base := convertToBase(a.Factor, cu)
	target := bestUnit(base, cu.kind, system)
	if target == nil {
		return a
	}
	display := target.display
	return Amount{
		Factor: convertFromBase(base, target),
		Unit:   &display,
	}
}

// decomposeAmount breaks an Amount into a human-friendly compound form using
// the target system's unit hierarchy. It returns at most two components.
//
// Examples:
//   - 17 tbsp -> [{1, "cup"}, {1, "tbsp"}]
//   - 3.3126 cups -> [{3.25, "cup"}, {1, "tbsp"}]
//
// If the unit is unknown or doesn't need decomposition, a single-element slice
// is returned.
func decomposeAmount(a Amount, system UnitSystem) []Amount {
	if a.Unit == nil {
		return []Amount{a}
	}

	cu := lookupUnit(*a.Unit)
	if cu == nil {
		return []Amount{a}
	}

	base := convertToBase(a.Factor, cu)

	hierarchy := unitHierarchy(cu.kind, system)
	if len(hierarchy) == 0 {
		return []Amount{a}
	}

	// Determine the starting unit: use bestUnit to find the appropriate level
	// so we don't decompose cups into pints, etc.
	startUnit := bestUnit(math.Abs(base), cu.kind, system)
	if startUnit == nil {
		return []Amount{a}
	}

	// Build the sub-hierarchy: startUnit and everything smaller.
	var subHierarchy []*canonicalUnit
	found := false
	for _, u := range hierarchy {
		if u.noAutoSelect {
			continue
		}
		if u.key == startUnit.key {
			found = true
		}
		if found {
			subHierarchy = append(subHierarchy, u)
		}
	}
	if len(subHierarchy) == 0 {
		converted := convertAmount(a, system)
		return []Amount{converted}
	}

	// Greedy decomposition: walk from the start unit down to smallest.
	var components []Amount
	remainder := math.Abs(base)
	negative := base < 0

	// Find the smallest selectable unit for remainder checks.
	smallestToBase := subHierarchy[len(subHierarchy)-1].toBase

	for i, u := range subHierarchy {
		if len(components) >= 2 {
			break
		}

		factor := remainder / u.toBase

		if len(components) == 0 {
			wholePart := math.Floor(factor)
			fracPart := factor - wholePart

			// Try snapping the fractional part to a common fraction.
			// If snapping leaves a negligible remainder, use the snapped value.
			// If snapping leaves a remainder that decomposes cleanly into
			// ~1 of a smaller unit, prefer that (e.g. 3¼ cup + 1 tbsp).
			// Otherwise, take just the whole part.
			snappedGlyph := snapToFraction(fracPart)
			if snappedGlyph != "" && wholePart >= 1 {
				snappedFrac := snapToFractionValue(fracPart)
				snappedFactor := wholePart + snappedFrac
				afterSnap := remainder - snappedFactor*u.toBase

				// Only accept the snap if it doesn't overshoot and the
				// remainder is negligible.
				if afterSnap >= -smallestToBase*0.1 && math.Abs(afterSnap) < smallestToBase*0.5 {
					// Remainder is negligible after snap; use snapped value.
					remainder = afterSnap
					display := u.display
					f := snappedFactor
					if negative {
						f = -f
						negative = false
					}
					components = append(components, Amount{Factor: f, Unit: &display})
					continue
				}
			}

			// Check if snapping to a lower fraction leaves a clean second
			// component. Try each fraction <= fracPart.
			bestCombo := tryFractionCombos(wholePart, fracPart, u, subHierarchy, smallestToBase)
			if bestCombo.factor > 0 {
				remainder -= bestCombo.factor * u.toBase
				display := u.display
				f := bestCombo.factor
				if negative {
					f = -f
					negative = false
				}
				components = append(components, Amount{Factor: f, Unit: &display})
				continue
			}

			// Take whole part only and let remainder flow down.
			if wholePart >= 1 {
				remainder -= wholePart * u.toBase
				display := u.display
				f := wholePart
				if negative {
					f = -f
					negative = false
				}
				components = append(components, Amount{Factor: f, Unit: &display})
			}
		} else {
			// Second component: snap to fraction or round to nearest whole.
			wholePart := math.Floor(factor)
			fracPart := factor - wholePart

			snappedGlyph := snapToFraction(fracPart)
			if snappedGlyph != "" {
				snapped := wholePart + snapToFractionValue(fracPart)
				if snapped >= 0.1 {
					display := u.display
					components = append(components, Amount{Factor: snapped, Unit: &display})
					break
				}
			}

			rounded := math.Round(factor)
			if rounded >= 1 {
				display := u.display
				f := rounded
				if negative {
					f = -f
					negative = false
				}
				components = append(components, Amount{Factor: f, Unit: &display})
				break
			}

			// Last unit: include fractional value if significant.
			if i == len(subHierarchy)-1 && factor >= 0.1 {
				display := u.display
				components = append(components, Amount{Factor: math.Round(factor*4)/4, Unit: &display})
			}
		}
	}

	// If remainder is negligible or decomposition produced nothing, fall back.
	if len(components) == 0 {
		converted := convertAmount(a, system)
		return []Amount{converted}
	}

	return components
}

// fractionComboResult holds the result of trying fraction combinations.
type fractionComboResult struct {
	factor float64 // the combined whole + fraction factor to use (0 = no good combo found)
}

// tryFractionCombos tries each common fraction <= fracPart and checks whether
// using (wholePart + fraction) leaves a remainder that decomposes cleanly into
// a small whole number of a smaller unit. Returns the best combination found.
func tryFractionCombos(wholePart, fracPart float64, u *canonicalUnit, subHierarchy []*canonicalUnit, smallestToBase float64) fractionComboResult {
	if wholePart < 1 {
		return fractionComboResult{}
	}

	type candidate struct {
		factor    float64 // wholePart + frac
		fracDist  float64 // distance from fracPart to chosen fraction
		cleanness float64 // how cleanly the remainder maps to a smaller unit
	}

	var best *candidate

	for _, f := range commonFractions {
		if f.value > fracPart {
			continue // fraction overshoots the original value
		}
		if f.value < fracPart-0.5 {
			continue // fraction too small (would leave too much remainder)
		}

		tryFactor := wholePart + f.value
		afterBase := tryFactor * u.toBase
		rem := (wholePart+fracPart)*u.toBase - afterBase // always >= 0 since f.value <= fracPart

		if rem < smallestToBase*0.3 {
			continue // remainder too small to be a meaningful second component
		}

		// Check if remainder maps cleanly to ~N of a smaller unit.
		for _, su := range subHierarchy[1:] { // skip current unit
			suFactor := rem / su.toBase
			rounded := math.Round(suFactor)
			if rounded >= 1 && math.Abs(suFactor-rounded) < 0.3 {
				fracDist := math.Abs(fracPart - f.value)
				cleanness := math.Abs(suFactor - rounded)
				// Prefer the fraction closest to the original value;
				// use cleanness of remainder as tiebreaker.
				if best == nil || fracDist < best.fracDist-0.01 ||
					(math.Abs(fracDist-best.fracDist) < 0.01 && cleanness < best.cleanness) {
					best = &candidate{factor: tryFactor, fracDist: fracDist, cleanness: cleanness}
				}
				break
			}
		}
	}

	if best != nil {
		return fractionComboResult{factor: best.factor}
	}
	return fractionComboResult{}
}

// snapToFractionValue returns the numeric value of the closest common fraction
// for frac, or frac itself if no match is within threshold.
func snapToFractionValue(frac float64) float64 {
	bestVal := frac
	bestDiff := fractionThreshold
	for _, f := range commonFractions {
		diff := math.Abs(frac - f.value)
		if diff < bestDiff {
			bestDiff = diff
			bestVal = f.value
		}
	}
	return bestVal
}

// ConvertUnits converts all ingredient amounts and yields in the recipe to the
// specified unit system. Unknown or unitless amounts are left unchanged.
//
// Ingredient amounts are decomposed into human-friendly compound forms where
// appropriate (e.g. "1 cup and 1 tbsp" instead of "17 tbsp"). Yields are
// converted but not decomposed.
//
// ConvertUnits is typically called after [Recipe.Scale]:
//
//	recipe.Scale(2)
//	recipe.ConvertUnits(recipemd.Imperial)
func (r *Recipe) ConvertUnits(system UnitSystem) {
	for i := range r.Yields {
		r.Yields[i] = convertAmount(r.Yields[i], system)
	}
	for j := range r.Ingredients {
		r.Ingredients[j].convertUnits(system)
	}
	for k := range r.IngredientGroups {
		r.IngredientGroups[k].convertUnits(system)
	}
}

func (ing *Ingredient) convertUnits(system UnitSystem) {
	if ing.Amount == nil {
		return
	}
	decomposed := decomposeAmount(*ing.Amount, system)
	if len(decomposed) == 1 {
		ing.Amount = &decomposed[0]
		ing.CompoundAmount = nil
	} else {
		// Set Amount to the simple converted value for programmatic use,
		// and CompoundAmount for display.
		converted := convertAmount(*ing.Amount, system)
		ing.Amount = &converted
		ing.CompoundAmount = decomposed
	}
}

func (g *IngredientGroup) convertUnits(system UnitSystem) {
	for i := range g.Ingredients {
		g.Ingredients[i].convertUnits(system)
	}
	for j := range g.IngredientGroups {
		g.IngredientGroups[j].convertUnits(system)
	}
}
