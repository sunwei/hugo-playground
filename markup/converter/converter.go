package converter

import (
	"bytes"
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/markup/converter/hooks"
	"github.com/sunwei/hugo-playground/markup/highlight"
	"github.com/sunwei/hugo-playground/markup/markup_config"
	"github.com/sunwei/hugo-playground/markup/tableofcontents"
)

// Converter wraps the Convert method that converts some markup into
// another format, e.g. Markdown to HTML.
type Converter interface {
	Convert(ctx RenderContext) (Result, error)
	Supports(feature identity.Identity) bool
}

// RenderContext holds contextual information about the content to render.
type RenderContext struct {
	// Src is the content to render.
	Src []byte

	// Whether to render TableOfContents.
	RenderTOC bool

	// GerRenderer provides hook renderers on demand.
	GetRenderer hooks.GetRendererFunc
}

// Result represents the minimum returned from Convert.
type Result interface {
	Bytes() []byte
}

// Provider creates converters.
type Provider interface {
	New(ctx DocumentContext) (Converter, error)
	Name() string
}

// DocumentContext holds contextual information about the document to convert.
type DocumentContext struct {
	Document     any // May be nil. Usually a page.Page
	DocumentID   string
	DocumentName string
	Filename     string
}

// ProviderConfig configures a new Provider.
type ProviderConfig struct {
	MarkupConfig markup_config.Config

	Cfg config.Provider // Site config
	highlight.Highlighter
}

// ProviderProvider creates converter providers.
type ProviderProvider interface {
	New(cfg ProviderConfig) (Provider, error)
}

// AnchorNameSanitizer tells how a converter sanitizes anchor names.
type AnchorNameSanitizer interface {
	SanitizeAnchorName(s string) string
}

// NewProvider creates a new Provider with the given name.
func NewProvider(name string, create func(ctx DocumentContext) (Converter, error)) Provider {
	return newConverter{
		name:   name,
		create: create,
	}
}

type newConverter struct {
	name   string
	create func(ctx DocumentContext) (Converter, error)
}

func (n newConverter) New(ctx DocumentContext) (Converter, error) {
	return n.create(ctx)
}

func (n newConverter) Name() string {
	return n.name
}

var FeatureRenderHooks = identity.NewPathIdentity("markup", "renderingHooks")

var NopConverter = new(nopConverter)

type nopConverter int

func (nopConverter) Convert(ctx RenderContext) (Result, error) {
	return &bytes.Buffer{}, nil
}

func (nopConverter) Supports(feature identity.Identity) bool {
	return false
}

// TableOfContentsProvider provides the content as a ToC structure.
type TableOfContentsProvider interface {
	TableOfContents() tableofcontents.Root
}

// DocumentInfo holds additional information provided by some converters.
type DocumentInfo interface {
	AnchorSuffix() string
}
