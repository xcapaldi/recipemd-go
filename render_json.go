package recipemd

import "encoding/json"

// RenderJSON serialises r as compact JSON.
//
// Amount factors are encoded as quoted strings (via [Amount.MarshalJSON]) to
// preserve human-readable precision. All other fields use their standard JSON
// representations.
func (p *Parser) RenderJSON(r *Recipe) ([]byte, error) {
	return json.Marshal(r)
}
