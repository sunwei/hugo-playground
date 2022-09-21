package minifiers

import (
	"github.com/sunwei/hugo-playground/config"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

type minifyConfig struct {
	// Whether to minify the published output (the HTML written to /public).
	MinifyOutput bool

	DisableHTML bool
	DisableCSS  bool
	DisableJS   bool
	DisableJSON bool
	DisableSVG  bool
	DisableXML  bool

	Tdewolff tdewolffConfig
}

type tdewolffConfig struct {
	HTML html.Minifier
	CSS  css.Minifier
	JS   js.Minifier
	JSON json.Minifier
	SVG  svg.Minifier
	XML  xml.Minifier
}

var defaultTdewolffConfig = tdewolffConfig{
	HTML: html.Minifier{
		KeepDocumentTags:        true,
		KeepConditionalComments: true,
		KeepEndTags:             true,
		KeepDefaultAttrVals:     true,
		KeepWhitespace:          false,
	},
	CSS: css.Minifier{
		Precision: 0,
		KeepCSS2:  true,
	},
	JS:   js.Minifier{},
	JSON: json.Minifier{},
	SVG: svg.Minifier{
		KeepComments: false,
		Precision:    0,
	},
	XML: xml.Minifier{
		KeepWhitespace: false,
	},
}

var defaultConfig = minifyConfig{
	Tdewolff: defaultTdewolffConfig,
}

func decodeConfig(cfg config.Provider) (conf minifyConfig, err error) {
	return defaultConfig, nil
}
