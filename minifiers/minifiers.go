package minifiers

import (
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/media"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/transform"
	"github.com/tdewolff/minify/v2"
	"io"
	"regexp"
)

// Client wraps a minifier.
type Client struct {
	m *minify.M
}

// New creates a new Client with the provided MIME types as the mapping foundation.
// The HTML minifier is also registered for additional HTML types (AMP etc.) in the
// provided list of output formats.
func New(mediaTypes media.Types, outputFormats output.Formats, cfg config.Provider) (Client, error) {
	conf, err := decodeConfig(cfg)

	m := minify.New()
	if err != nil {
		return Client{}, err
	}

	// We use the Type definition of the media types defined in the site if found.
	addMinifier(m, mediaTypes, "css", getMinifier(conf, "css"))

	addMinifier(m, mediaTypes, "js", getMinifier(conf, "js"))
	m.AddRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), getMinifier(conf, "js"))

	addMinifier(m, mediaTypes, "json", getMinifier(conf, "json"))
	m.AddRegexp(regexp.MustCompile(`^(application|text)/(x-|(ld|manifest)\+)?json$`), getMinifier(conf, "json"))

	addMinifier(m, mediaTypes, "svg", getMinifier(conf, "svg"))

	addMinifier(m, mediaTypes, "xml", getMinifier(conf, "xml"))

	// HTML
	addMinifier(m, mediaTypes, "html", getMinifier(conf, "html"))
	for _, of := range outputFormats {
		if of.IsHTML {
			m.Add(of.MediaType.Type(), getMinifier(conf, "html"))
		}
	}

	return Client{m: m}, nil
}

func addMinifier(m *minify.M, mt media.Types, suffix string, min minify.Minifier) {
	types := mt.BySuffix(suffix)
	for _, t := range types {
		m.Add(t.Type(), min)
	}
}

// getMinifier returns the appropriate minify.MinifierFunc for the MIME
// type suffix s, given the config c.
func getMinifier(c minifyConfig, s string) minify.Minifier {
	switch {
	case s == "css" && !c.DisableCSS:
		return &c.Tdewolff.CSS
	case s == "js" && !c.DisableJS:
		return &c.Tdewolff.JS
	case s == "json" && !c.DisableJSON:
		return &c.Tdewolff.JSON
	case s == "svg" && !c.DisableSVG:
		return &c.Tdewolff.SVG
	case s == "xml" && !c.DisableXML:
		return &c.Tdewolff.XML
	case s == "html" && !c.DisableHTML:
		return &c.Tdewolff.HTML
	default:
		return noopMinifier{}
	}
}

// noopMinifier implements minify.Minifier [1], but doesn't minify content. This means
// that we can avoid missing minifiers for any MIME types in our minify.M, which
// causes minify to return errors, while still allowing minification to be
// disabled for specific types.
//
// [1]: https://pkg.go.dev/github.com/tdewolff/minify#Minifier
type noopMinifier struct{}

// Minify copies r into w without transformation.
func (m noopMinifier) Minify(_ *minify.M, w io.Writer, r io.Reader, _ map[string]string) error {
	_, err := io.Copy(w, r)
	return err
}

// Transformer returns a func that can be used in the transformer publishing chain.
// TODO(bep) minify config etc
func (m Client) Transformer(mediatype media.Type) transform.Transformer {
	_, params, min := m.m.Match(mediatype.Type())
	if min == nil {
		// No minifier for this MIME type
		return nil
	}

	return func(ft transform.FromTo) error {
		// Note that the source io.Reader will already be buffered, but it implements
		// the Bytes() method, which is recognized by the Minify library.
		return min.Minify(m.m, ft.To(), ft.From(), params)
	}
}
