package template

import (
	"fmt"
	"github.com/sunwei/hugo-playground/tpl/internal/go_templates/texttemplate/parse"
)

// Error describes a problem encountered during template Escaping.
type Error struct {
	// ErrorCode describes the kind of error.
	ErrorCode ErrorCode
	// Node is the node that caused the problem, if known.
	// If not nil, it overrides Name and Line.
	Node parse.Node
	// Name is the name of the template in which the error was encountered.
	Name string
	// Line is the line number of the error in the template source or 0.
	Line int
	// Description is a human-readable description of the problem.
	Description string
}

func (e *Error) Error() string {
	switch {
	case e.Node != nil:
		loc, _ := (*parse.Tree)(nil).ErrorContext(e.Node)
		return fmt.Sprintf("html/template:%s: %s", loc, e.Description)
	case e.Line != 0:
		return fmt.Sprintf("html/template:%s:%d: %s", e.Name, e.Line, e.Description)
	case e.Name != "":
		return fmt.Sprintf("html/template:%s: %s", e.Name, e.Description)
	}
	return "html/template: " + e.Description
}

// ErrorCode is a code for a kind of error.
type ErrorCode int

const (
	// OK indicates the lack of an error.
	OK ErrorCode = iota
	ErrNoSuchTemplate
	ErrPredefinedEscaper
	ErrAmbigContext
	ErrBranchEnd
	ErrBadHTML
	ErrOutputContext
	ErrEndContext
)

// errorf creates an error given a format string f and args.
// The template Name still needs to be supplied.
func errorf(k ErrorCode, node parse.Node, line int, f string, args ...any) *Error {
	return &Error{k, node, "", line, fmt.Sprintf(f, args...)}
}
