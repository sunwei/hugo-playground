package helpers

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// URLize is similar to MakePath, but with Unicode handling
// Example:
//     uri: Vim (text editor)
//     urlize: vim-text-editor
func (p *PathSpec) URLize(uri string) string {
	return p.URLEscape(p.MakePathSanitized(uri))
}

// MakePathSanitized creates a Unicode-sanitized string, with the spaces replaced
func (p *PathSpec) MakePathSanitized(s string) string {
	return strings.ToLower(p.MakePath(s))
}

// URLEscape escapes unicode letters.
func (p *PathSpec) URLEscape(uri string) string {
	// escape unicode letters
	parsedURI, err := url.Parse(uri)
	if err != nil {
		// if net/url can not parse URL it means Sanitize works incorrectly
		panic(err)
	}
	x := parsedURI.String()
	return x
}

// URLizeFilename creates an URL from a filename by escaping unicode letters
// and turn any filepath separator into forward slashes.
func (p *PathSpec) URLizeFilename(filename string) string {
	return p.URLEscape(filepath.ToSlash(filename))
}

// PrependBasePath prepends any baseURL sub-folder to the given resource
func (p *PathSpec) PrependBasePath(rel string, isAbs bool) string {
	basePath := p.GetBasePath(!isAbs)
	if basePath != "" {
		rel = filepath.ToSlash(rel)
		// Need to prepend any path from the baseURL
		hadSlash := strings.HasSuffix(rel, "/")
		rel = path.Join(basePath, rel)
		if hadSlash {
			rel += "/"
		}
	}
	return rel
}
