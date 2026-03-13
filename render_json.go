package recipemd

import "encoding/json"

// RenderJSON serializes a Recipe as indented JSON.
func (p *Parser) RenderJSON(r *Recipe) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
