package recipemd

import "fmt"

// ParseError reports a single structural or value-level problem found while
// parsing a RecipeMD document. It implements the error interface.
//
// [Parser.Parse] may return multiple ParseErrors joined with [errors.Join];
// callers can inspect individual errors with [errors.As]:
//
//	var pe *recipemd.ParseError
//	if errors.As(err, &pe) {
//	    fmt.Printf("line %d: %s\n", pe.Line, pe.Message)
//	}
//
// The position fields (Offset, Line, Column) are intended to support LSP
// diagnostic reporting.
type ParseError struct {
	// Message is a human-readable description of the problem.
	Message string
	// Offset is the zero-based byte offset of the error in the source document.
	Offset int
	// Line is the one-based line number of the error.
	Line int
	// Column is the one-based column number (in bytes) of the error.
	Column int
}

// Error implements the error interface, returning a string of the form
// "line L, col C: message".
func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, col %d: %s", e.Line, e.Column, e.Message)
}
