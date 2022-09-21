package page

import "strings"

const (
	KindPage = "page"

	// The rest are node types; home page, sections etc.

	KindHome    = "home"
	KindSection = "section"

	// Note tha before Hugo 0.73 these were confusingly named
	// taxonomy (now: term)
	// taxonomyTerm (now: taxonomy)
	KindTaxonomy = "taxonomy"
	KindTerm     = "term"
)

var kindMap = map[string]string{
	strings.ToLower(KindPage):     KindPage,
	strings.ToLower(KindHome):     KindHome,
	strings.ToLower(KindSection):  KindSection,
	strings.ToLower(KindTaxonomy): KindTaxonomy,
	strings.ToLower(KindTerm):     KindTerm,

	// Legacy, pre v0.53.0.
	"taxonomyterm": KindTaxonomy,
}

// GetKind gets the page kind given a string, empty if not found.
func GetKind(s string) string {
	return kindMap[strings.ToLower(s)]
}
