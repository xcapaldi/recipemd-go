package recipemd

import "encoding/json"

// RenderJSON serializes a Recipe as compact JSON.
func (p *Parser) RenderJSON(r *Recipe) ([]byte, error) {
	return json.Marshal(r)
}
