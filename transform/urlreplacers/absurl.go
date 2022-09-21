package urlreplacers

import "github.com/sunwei/hugo-playground/transform"

var ar = newAbsURLReplacer()

// NewAbsURLTransformer replaces relative URLs with absolute ones
// in HTML files, using the baseURL setting.
func NewAbsURLTransformer(path string) transform.Transformer {
	return func(ft transform.FromTo) error {
		ar.replaceInHTML(path, ft)
		return nil
	}
}

// NewAbsURLInXMLTransformer replaces relative URLs with absolute ones
// in XML files, using the baseURL setting.
func NewAbsURLInXMLTransformer(path string) transform.Transformer {
	return func(ft transform.FromTo) error {
		ar.replaceInXML(path, ft)
		return nil
	}
}
