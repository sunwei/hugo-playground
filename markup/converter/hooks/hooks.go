package hooks

import (
	"github.com/sunwei/hugo-playground/common/hugio"
	"github.com/sunwei/hugo-playground/common/text"
	"github.com/sunwei/hugo-playground/common/types/hstring"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/markup/internal/attributes"
	"io"
)

type RendererType int

const (
	LinkRendererType RendererType = iota + 1
	ImageRendererType
	HeadingRendererType
	CodeBlockRendererType
)

type GetRendererFunc func(t RendererType, id any) any

type CodeblockContext interface {
	AttributesProvider
	text.Positioner
	Options() map[string]any
	Type() string
	Inner() string
	Ordinal() int
	Page() any
}

type AttributesProvider interface {
	Attributes() map[string]any
}

type CodeBlockRenderer interface {
	RenderCodeblock(w hugio.FlexiWriter, ctx CodeblockContext) error
	identity.Provider
}

type IsDefaultCodeBlockRendererProvider interface {
	IsDefaultCodeBlockRenderer() bool
}

type AttributesOptionsSliceProvider interface {
	AttributesSlice() []attributes.Attribute
	OptionsSlice() []attributes.Attribute
}

type LinkRenderer interface {
	RenderLink(w io.Writer, ctx LinkContext) error
	identity.Provider
}

type LinkContext interface {
	Page() any
	Destination() string
	Title() string
	Text() hstring.RenderedString
	PlainText() string
}

// ElementPositionResolver provides a way to resolve the start Position
// of a markdown element in the original source document.
// This may be both slow and approximate, so should only be
// used for error logging.
type ElementPositionResolver interface {
	ResolvePosition(ctx any) text.Position
}
