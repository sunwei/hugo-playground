package related

import (
	"fmt"
	"strings"
	"time"
)

// IndexConfig configures an index.
type IndexConfig struct {
	// The index name. This directly maps to a field or Param name.
	Name string

	// Contextual pattern used to convert the Param value into a string.
	// Currently only used for dates. Can be used to, say, bump posts in the same
	// time frame when searching for related documents.
	// For dates it follows Go's time.Format patterns, i.e.
	// "2006" for YYYY and "200601" for YYYYMM.
	Pattern string

	// This field's weight when doing multi-index searches. Higher is "better".
	Weight int

	// Will lower case all string values in and queries tothis index.
	// May get better accurate results, but at a slight performance cost.
	ToLower bool
}

// Keyword is the interface a keyword in the search index must implement.
type Keyword interface {
	String() string
}

/*
Config is the top level configuration element used to configure how to retrieve
related content in Hugo.

An example site config.toml:

	[related]
	threshold = 1
	[[related.indices]]
	name = "keywords"
	weight = 200
	[[related.indices]]
	name  = "tags"
	weight = 100
	[[related.indices]]
	name  = "date"
	weight = 1
	pattern = "2006"
*/
type Config struct {
	// Only include matches >= threshold, a normalized rank between 0 and 100.
	Threshold int

	// To get stable "See also" sections we, by default, exclude newer related pages.
	IncludeNewer bool

	// Will lower case all string values and queries to the indices.
	// May get better results, but at a slight performance cost.
	ToLower bool

	Indices IndexConfigs
}

// IndexConfigs holds a set of index configurations.
type IndexConfigs []IndexConfig

// InvertedIndex holds an inverted index, also sometimes named posting list, which
// lists, for every possible search term, the documents that contain that term.
type InvertedIndex struct {
	cfg   Config
	index map[string]map[Keyword][]Document

	minWeight int
	maxWeight int
}

// Document is the interface an indexable document in Hugo must fulfill.
type Document interface {
	// RelatedKeywords returns a list of keywords for the given index config.
	RelatedKeywords(cfg IndexConfig) ([]Keyword, error)

	// When this document was or will be published.
	PublishDate() time.Time

	// Name is used as an tiebreaker if both Weight and PublishDate are
	// the same.
	Name() string
}

// ToKeywords returns a Keyword slice of the given input.
func (cfg IndexConfig) ToKeywords(v any) ([]Keyword, error) {
	var (
		keywords []Keyword
		toLower  = cfg.ToLower
	)
	switch vv := v.(type) {
	case string:
		if toLower {
			vv = strings.ToLower(vv)
		}
		keywords = append(keywords, StringKeyword(vv))
	case []string:
		if toLower {
			vc := make([]string, len(vv))
			copy(vc, vv)
			for i := 0; i < len(vc); i++ {
				vc[i] = strings.ToLower(vc[i])
			}
			vv = vc
		}
		keywords = append(keywords, StringsToKeywords(vv...)...)
	case time.Time:
		layout := "2006"
		if cfg.Pattern != "" {
			layout = cfg.Pattern
		}
		keywords = append(keywords, StringKeyword(vv.Format(layout)))
	case nil:
		return keywords, nil
	default:
		return keywords, fmt.Errorf("indexing currently not supported for index %q and type %T", cfg.Name, vv)
	}

	return keywords, nil
}

// StringKeyword is a string search keyword.
type StringKeyword string

func (s StringKeyword) String() string {
	return string(s)
}

// StringsToKeywords converts the given slice of strings to a slice of Keyword.
func StringsToKeywords(s ...string) []Keyword {
	kw := make([]Keyword, len(s))

	for i := 0; i < len(s); i++ {
		kw[i] = StringKeyword(s[i])
	}

	return kw
}
