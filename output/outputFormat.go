package output

import (
	"fmt"
	"github.com/sunwei/hugo-playground/media"
	"sort"
	"strings"
)

// Format represents an output representation, usually to a file on disk.
type Format struct {
	// The Name is used as an identifier. Internal output formats (i.e. HTML and RSS)
	// can be overridden by providing a new definition for those types.
	Name string `json:"name"`

	MediaType media.Type `json:"-"`

	// Must be set to a value when there are two or more conflicting mediatype for the same resource.
	Path string `json:"path"`

	// The base output file name used when not using "ugly URLs", defaults to "index".
	BaseName string `json:"baseName"`

	// The value to use for rel links
	//
	// See https://www.w3schools.com/tags/att_link_rel.asp
	//
	// AMP has a special requirement in this department, see:
	// https://www.ampproject.org/docs/guides/deploy/discovery
	// I.e.:
	// <link rel="amphtml" href="https://www.example.com/url/to/amp/document.html">
	Rel string `json:"rel"`

	// The protocol to use, i.e. "webcal://". Defaults to the protocol of the baseURL.
	Protocol string `json:"protocol"`

	// IsPlainText decides whether to use text/template or html/template
	// as template parser.
	IsPlainText bool `json:"isPlainText"`

	// IsHTML returns whether this format is int the HTML family. This includes
	// HTML, AMP etc. This is used to decide when to create alias redirects etc.
	IsHTML bool `json:"isHTML"`

	// Enable if it doesn't make sense to include this format in an alternative
	// format listing, CSS being one good example.
	// Note that we use the term "alternative" and not "alternate" here, as it
	// does not necessarily replace the other format, it is an alternative representation.
	NotAlternative bool `json:"notAlternative"`

	// Setting this will make this output format control the value of
	// .Permalink and .RelPermalink for a rendered Page.
	// If not set, these values will point to the main (first) output format
	// configured. That is probably the behaviour you want in most situations,
	// as you probably don't want to link back to the RSS version of a page, as an
	// example. AMP would, however, be a good example of an output format where this
	// behaviour is wanted.
	Permalinkable bool `json:"permalinkable"`

	// Setting this to a non-zero value will be used as the first sort criteria.
	Weight int `json:"weight"`
}

// Formats is a slice of Format.
type Formats []Format

func (formats Formats) Len() int      { return len(formats) }
func (formats Formats) Swap(i, j int) { formats[i], formats[j] = formats[j], formats[i] }
func (formats Formats) Less(i, j int) bool {
	fi, fj := formats[i], formats[j]
	if fi.Weight == fj.Weight {
		return fi.Name < fj.Name
	}

	if fj.Weight == 0 {
		return true
	}

	return fi.Weight > 0 && fi.Weight < fj.Weight
}

// GetByName gets a format by its identifier name.
func (formats Formats) GetByName(name string) (f Format, found bool) {
	for _, ff := range formats {
		if strings.EqualFold(name, ff.Name) {
			f = ff
			found = true
			return
		}
	}
	return
}

// An ordered list of built-in output formats.
var (
	HTMLFormat = Format{
		Name:          "HTML",
		MediaType:     media.HTMLType,
		BaseName:      "index",
		Rel:           "canonical",
		IsHTML:        true,
		Permalinkable: true,

		// Weight will be used as first sort criteria. HTML will, by default,
		// be rendered first, but set it to 10 so it's easy to put one above it.
		Weight: 10,
	}

	MarkdownFormat = Format{
		Name:        "MARKDOWN",
		MediaType:   media.MarkdownType,
		BaseName:    "index",
		Rel:         "alternate",
		IsPlainText: true,
	}

	JSONFormat = Format{
		Name:        "JSON",
		MediaType:   media.JSONType,
		BaseName:    "index",
		IsPlainText: true,
		Rel:         "alternate",
	}

	RobotsTxtFormat = Format{
		Name:        "ROBOTS",
		MediaType:   media.TextType,
		BaseName:    "robots",
		IsPlainText: true,
		Rel:         "alternate",
	}
)

// DefaultFormats contains the default output formats supported by Hugo.
var DefaultFormats = Formats{
	HTMLFormat,
	JSONFormat,
	MarkdownFormat,
}

// DecodeFormats takes a list of output format configurations and merges those,
// in the order given, with the Hugo defaults as the last resort.
func DecodeFormats(mediaTypes media.Types, maps ...map[string]any) (Formats, error) {
	f := make(Formats, len(DefaultFormats))
	copy(f, DefaultFormats)

	// no customized formats map

	sort.Sort(f)

	return f, nil
}

// FromFilename gets a Format given a filename.
func (formats Formats) FromFilename(filename string) (f Format, found bool) {
	// mytemplate.amp.html
	// mytemplate.html
	// mytemplate
	var ext, outFormat string

	parts := strings.Split(filename, ".")
	if len(parts) > 2 {
		outFormat = parts[1]
		ext = parts[2]
	} else if len(parts) > 1 {
		ext = parts[1]
	}

	if outFormat != "" {
		return formats.GetByName(outFormat)
	}

	if ext != "" {
		f, found = formats.GetBySuffix(ext)
		if !found && len(parts) == 2 {
			// For extensionless output formats (e.g. Netlify's _redirects)
			// we must fall back to using the extension as format lookup.
			f, found = formats.GetByName(ext)
		}
	}
	return
}

// GetBySuffix gets a output format given as suffix, e.g. "html".
// It will return false if no format could be found, or if the suffix given
// is ambiguous.
// The lookup is case insensitive.
func (formats Formats) GetBySuffix(suffix string) (f Format, found bool) {
	for _, ff := range formats {
		for _, suffix2 := range ff.MediaType.Suffixes() {
			if strings.EqualFold(suffix, suffix2) {
				if found {
					// ambiguous
					found = false
					return
				}
				f = ff
				found = true
			}
		}
	}
	return
}

// GetByNames gets a list of formats given a list of identifiers.
func (formats Formats) GetByNames(names ...string) (Formats, error) {
	var types []Format

	for _, name := range names {
		tpe, ok := formats.GetByName(name)
		if !ok {
			return types, fmt.Errorf("OutputFormat with key %q not found", name)
		}
		types = append(types, tpe)
	}
	return types, nil
}
