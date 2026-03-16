package recipemd

import "fmt"

// ParseError represents a single parse error with source position information,
// suitable for use as an LSP diagnostic in the future.
type ParseError struct {
	Message string // human-readable error description
	Offset  int    // byte offset in source (0-based)
	Line    int    // 1-based line number
	Column  int    // 1-based column number
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, col %d: %s", e.Line, e.Column, e.Message)
}
