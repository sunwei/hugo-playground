package markup

import (
	"github.com/sunwei/hugo-playground/markup/converter"
	"github.com/sunwei/hugo-playground/markup/goldmark"
	"github.com/sunwei/hugo-playground/markup/highlight"
	"github.com/sunwei/hugo-playground/markup/markup_config"
	"strings"
)

type ConverterProvider interface {
	Get(name string) converter.Provider
	GetMarkupConfig() markup_config.Config
	GetHighlighter() highlight.Highlighter
}

func NewConverterProvider() (ConverterProvider, error) {
	converters := make(map[string]converter.Provider)

	markupConfig, err := markup_config.Decode()
	if err != nil {
		return nil, err
	}

	cpc := converter.ProviderConfig{
		MarkupConfig: markupConfig,
		Highlighter:  highlight.New(markupConfig.Highlight),
	}

	defaultHandler := markupConfig.DefaultMarkdownHandler
	add := func(p converter.ProviderProvider, aliases ...string) error {
		c, err := p.New(cpc)
		if err != nil {
			return err
		}

		name := c.Name()

		aliases = append(aliases, name)

		if strings.EqualFold(name, defaultHandler) {
			aliases = append(aliases, "markdown")
		}

		addConverter(converters, c, aliases...)
		return nil
	}

	// default
	if err := add(goldmark.Provider); err != nil {
		return nil, err
	}

	return &converterRegistry{
		config:     cpc,
		converters: converters,
	}, nil
}

func addConverter(m map[string]converter.Provider, c converter.Provider, aliases ...string) {
	for _, alias := range aliases {
		m[alias] = c
	}
}

type converterRegistry struct {
	// Maps name (md, markdown, goldmark etc.) to a converter provider.
	// Note that this is also used for aliasing, so the same converter
	// may be registered multiple times.
	// All names are lower case.
	converters map[string]converter.Provider

	config converter.ProviderConfig
}

func (r *converterRegistry) Get(name string) converter.Provider {
	return r.converters[strings.ToLower(name)]
}

func (r *converterRegistry) GetHighlighter() highlight.Highlighter {
	return r.config.Highlighter
}

func (r *converterRegistry) GetMarkupConfig() markup_config.Config {
	return r.config.MarkupConfig
}
