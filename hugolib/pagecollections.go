package hugolib

// PageCollections contains the page collections for a site.
type PageCollections struct {
	pageMap *pageMap
}

func newPageCollections(m *pageMap) *PageCollections {
	if m == nil {
		panic("must provide a pageMap")
	}

	c := &PageCollections{pageMap: m}

	return c
}
