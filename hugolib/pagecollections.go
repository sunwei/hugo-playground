package hugolib

import (
	"github.com/sunwei/hugo-playground/resources/page"
)

// PageCollections contains the page collections for a site.
type PageCollections struct {
	pageMap *pageMap
}

// Pages returns all pages.
// This is for the current language only.
func (c *PageCollections) Pages() page.Pages {
	panic("page collections Pages not implemented")
}

// RegularPages returns all the regular pages.
// This is for the current language only.
func (c *PageCollections) RegularPages() page.Pages {
	panic("page collections not implement RegularPages")
}

// AllPages returns all pages for all languages.
func (c *PageCollections) AllPages() page.Pages {
	panic("page collections not implement AllPages")
}

// AllRegularPages AllPages returns all regular pages for all languages.
func (c *PageCollections) AllRegularPages() page.Pages {
	panic("page collections not implement AllRegularPages")
}

func newPageCollections(m *pageMap) *PageCollections {
	if m == nil {
		panic("must provide a pageMap")
	}

	c := &PageCollections{pageMap: m}

	return c
}

func (*PageCollections) findPagesByKindIn(kind string, inPages page.Pages) page.Pages {
	var pages page.Pages
	for _, p := range inPages {
		if p.Kind() == kind {
			pages = append(pages, p)
		}
	}
	return pages
}
