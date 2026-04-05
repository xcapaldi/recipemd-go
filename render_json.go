package recipemd

import "encoding/json"

// RenderJSON serialises r as compact JSON.
//
// Amount factors are encoded as quoted strings (via [Amount.MarshalJSON]) to
// preserve human-readable precision. All other fields use their standard JSON
// representations.
func (p *Parser) RenderJSON(r *Recipe) ([]byte, error) {
	return p.RenderJSONWithOptions(r, RenderOptions{})
}

// RenderJSONWithOptions serialises r as compact JSON with configurable behavior.
//
// When [RenderOptions.ConvertTemperature] is set, all detected temperatures in
// the Description and Instructions fields are converted to the target unit
// before serialisation. A rounding of 2 decimal places is used for converted
// temperature values.
func (p *Parser) RenderJSONWithOptions(r *Recipe, opts RenderOptions) ([]byte, error) {
	if opts.ConvertTemperature != nil {
		r = recipeWithConvertedTemperatures(r, *opts.ConvertTemperature, 2)
	}
	return json.Marshal(r)
}
