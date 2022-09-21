package markup_config

import (
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/markup/asciidocext/asciidocext_config"
	"github.com/sunwei/hugo-playground/markup/goldmark/goldmark_config"
	"github.com/sunwei/hugo-playground/markup/highlight"
	"github.com/sunwei/hugo-playground/markup/tableofcontents"
)

type Config struct {
	// Default markdown handler for md/markdown extensions.
	// Default is "goldmark".
	// Before Hugo 0.60 this was "blackfriday".
	DefaultMarkdownHandler string

	Highlight       highlight.Config
	TableOfContents tableofcontents.Config

	// Content renderers
	Goldmark    goldmark_config.Config
	AsciidocExt asciidocext_config.Config
}

func Decode(cfg config.Provider) (conf Config, err error) {
	return Default, nil
}

var Default = Config{
	DefaultMarkdownHandler: "goldmark",

	TableOfContents: tableofcontents.DefaultConfig,
	Highlight:       highlight.DefaultConfig,

	Goldmark:    goldmark_config.Default,
	AsciidocExt: asciidocext_config.Default,
}
