package text

// Positioner represents a thing that knows its position in a text file or stream,
// typically an error.
type Positioner interface {
	Position() Position
}

// Position holds a source position in a text file or stream.
type Position struct {
	Filename     string // filename, if any
	Offset       int    // byte offset, starting at 0. It's set to -1 if not provided.
	LineNumber   int    // line number, starting at 1
	ColumnNumber int    // column number, starting at 1 (character count per line)
}

// IsValid returns true if line number is > 0.
func (pos Position) IsValid() bool {
	return pos.LineNumber > 0
}

func (pos Position) String() string {
	if pos.Filename == "" {
		pos.Filename = "<stream>"
	}
	return positionStringFormatfunc(pos)
}

var positionStringFormatfunc func(p Position) string
