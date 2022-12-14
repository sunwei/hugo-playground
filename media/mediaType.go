package media

import (
	"fmt"
	"sort"
	"strings"
)

const (
	defaultDelimiter = "."
)

// Type (also known as MIME type and content type) is a two-part identifier for
// file formats and format contents transmitted on the Internet.
// For Hugo's use case, we use the top-level type name / subtype name + suffix.
// One example would be application/svg+xml
// If suffix is not provided, the sub type will be used.
// See // https://en.wikipedia.org/wiki/Media_type
type Type struct {
	MainType  string `json:"mainType"`  // i.e. text
	SubType   string `json:"subType"`   // i.e. html
	Delimiter string `json:"delimiter"` // e.g. "."

	// FirstSuffix holds the first suffix defined for this Type.
	FirstSuffix SuffixInfo `json:"firstSuffix"`

	// This is the optional suffix after the "+" in the MIME type,
	//  e.g. "xml" in "application/rss+xml".
	mimeSuffix string

	// E.g. "jpg,jpeg"
	// Stored as a string to make Type comparable.
	suffixesCSV string
}

// SuffixInfo holds information about a Type's suffix.
type SuffixInfo struct {
	Suffix     string `json:"suffix"`
	FullSuffix string `json:"fullSuffix"`
}

// Types is a slice of media types.
type Types []Type

func (t Types) Len() int           { return len(t) }
func (t Types) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t Types) Less(i, j int) bool { return t[i].Type() < t[j].Type() }

func (m *Type) init() {
	m.FirstSuffix.FullSuffix = ""
	m.FirstSuffix.Suffix = ""
	if suffixes := m.Suffixes(); suffixes != nil {
		m.FirstSuffix.Suffix = suffixes[0]
		m.FirstSuffix.FullSuffix = m.Delimiter + m.FirstSuffix.Suffix
	}
}

// Suffixes returns all valid file suffixes for this type.
func (m Type) Suffixes() []string {
	if m.suffixesCSV == "" {
		return nil
	}

	return strings.Split(m.suffixesCSV, ",")
}

// GetByType returns a media type for tp.
func (t Types) GetByType(tp string) (Type, bool) {
	for _, tt := range t {
		if strings.EqualFold(tt.Type(), tp) {
			return tt, true
		}
	}

	if !strings.Contains(tp, "+") {
		// Try with the main and sub type
		parts := strings.Split(tp, "/")
		if len(parts) == 2 {
			return t.GetByMainSubType(parts[0], parts[1])
		}
	}

	return Type{}, false
}

// GetByMainSubType gets a media type given a main and a sub type e.g. "text" and "plain".
// It will return false if no format could be found, or if the combination given
// is ambiguous.
// The lookup is case insensitive.
func (t Types) GetByMainSubType(mainType, subType string) (tp Type, found bool) {
	for _, tt := range t {
		if strings.EqualFold(mainType, tt.MainType) && strings.EqualFold(subType, tt.SubType) {
			if found {
				// ambiguous
				found = false
				return
			}

			tp = tt
			found = true
		}
	}
	return
}

// Type returns a string representing the main- and sub-type of a media type, e.g. "text/css".
// A suffix identifier will be appended after a "+" if set, e.g. "image/svg+xml".
// Hugo will register a set of default media types.
// These can be overridden by the user in the configuration,
// by defining a media type with the same Type.
func (m Type) Type() string {
	// Examples are
	// image/svg+xml
	// text/css
	if m.mimeSuffix != "" {
		return m.MainType + "/" + m.SubType + "+" + m.mimeSuffix
	}
	return m.MainType + "/" + m.SubType
}

// Definitions from https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types etc.
// Note that from Hugo 0.44 we only set Suffix if it is part of the MIME type.
var (
	HTMLType = newMediaType("text", "html", []string{"html"})

	JSONType = newMediaType("application", "json", []string{"json"})
	TOMLType = newMediaType("application", "toml", []string{"toml"})

	MarkdownType = newMediaType("text", "markdown", []string{"md", "markdown"})

	OctetType = newMediaType("application", "octet-stream", nil)

	TextType = newMediaType("text", "plain", []string{"txt"})
)

// DefaultTypes is the default media types supported by Hugo.
var DefaultTypes = Types{
	HTMLType,
	MarkdownType,
	TOMLType,
	TextType,
}

func newMediaType(main, sub string, suffixes []string) Type {
	t := Type{MainType: main, SubType: sub, suffixesCSV: strings.Join(suffixes, ","), Delimiter: defaultDelimiter}
	t.init()
	return t
}

// DecodeTypes takes a list of media type configurations and merges those,
// in the order given, with the Hugo defaults as the last resort.
func DecodeTypes(mms ...map[string]any) (Types, error) {
	var m Types

	// Maps type string to Type. Type string is the full application/svg+xml.
	mmm := make(map[string]Type)
	for _, dt := range DefaultTypes {
		mmm[dt.Type()] = dt
	}

	// no customized media type

	for _, v := range mmm {
		m = append(m, v)
	}
	sort.Sort(m)

	return m, nil
}

// FromString creates a new Type given a type string on the form MainType/SubType and
// an optional suffix, e.g. "text/html" or "text/html+html".
func fromString(t string) (Type, error) {
	t = strings.ToLower(t)
	parts := strings.Split(t, "/")
	if len(parts) != 2 {
		return Type{}, fmt.Errorf("cannot parse %q as a media type", t)
	}
	mainType := parts[0]
	subParts := strings.Split(parts[1], "+")

	subType := strings.Split(subParts[0], ";")[0]

	var suffix string

	if len(subParts) > 1 {
		suffix = subParts[1]
	}

	return Type{MainType: mainType, SubType: subType, mimeSuffix: suffix}, nil
}

// BySuffix will return all media types matching a suffix.
func (t Types) BySuffix(suffix string) []Type {
	suffix = strings.ToLower(suffix)
	var types []Type
	for _, tt := range t {
		if tt.hasSuffix(suffix) {
			types = append(types, tt)
		}
	}
	return types
}

func (m Type) hasSuffix(suffix string) bool {
	return strings.Contains(","+m.suffixesCSV+",", ","+suffix+",")
}

// IsZero reports whether this Type represents a zero value.
// For internal use.
func (m Type) IsZero() bool {
	return m.SubType == ""
}

// GetFirstBySuffix will return the first type matching the given suffix.
func (t Types) GetFirstBySuffix(suffix string) (Type, SuffixInfo, bool) {
	suffix = strings.ToLower(suffix)
	for _, tt := range t {
		if tt.hasSuffix(suffix) {
			return tt, SuffixInfo{
				FullSuffix: tt.Delimiter + suffix,
				Suffix:     suffix,
			}, true
		}
	}
	return Type{}, SuffixInfo{}, false
}

// FromStringAndExt creates a Type from a MIME string and a given extension.
func FromStringAndExt(t, ext string) (Type, error) {
	tp, err := fromString(t)
	if err != nil {
		return tp, err
	}
	tp.suffixesCSV = strings.TrimPrefix(ext, ".")
	tp.Delimiter = defaultDelimiter
	tp.init()
	return tp, nil
}
