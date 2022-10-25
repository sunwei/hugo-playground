package hugolib

import (
	"fmt"
	"github.com/sunwei/hugo-playground/common/maps"
	"github.com/sunwei/hugo-playground/common/para"
	"github.com/sunwei/hugo-playground/resources/page"
	"sync"
)

type pageMap struct {
	s *Site
	*contentMap
}

type pageMaps struct {
	workers *para.Workers
	pmaps   []*pageMap
}

type pagesMapBucket struct {
	// Cascading front matter.
	cascade map[page.PageMatcher]maps.Params

	owner *pageState // The branch node

	*pagesMapBucketPages
}

type pagesMapBucketPages struct {
	pagesInit sync.Once
	pages     page.Pages

	pagesAndSectionsInit sync.Once
	pagesAndSections     page.Pages

	sectionsInit sync.Once
	sections     page.Pages
}

type viewName struct {
	singular string // e.g. "category"
	plural   string // e.g. "categories"
}

type pageMapQuery struct {
	Prefix string
	Filter contentTreeNodeCallback
}

func (m *pageMap) createListAllPages() page.Pages {
	pages := make(page.Pages, 0)
	m.contentMap.pageTrees.Walk(func(s string, n *contentNode) bool {
		if n.p == nil {
			panic(fmt.Sprintf("BUG: page not set for %q", s))
		}
		if contentTreeNoListAlwaysFilter(s, n) {
			return false
		}
		pages = append(pages, n.p)
		return false
	})

	page.SortByDefault(pages)
	return pages
}

func (v viewName) IsZero() bool {
	return v.singular == ""
}

func newPageMaps(h *HugoSites) *pageMaps {
	mps := make([]*pageMap, len(h.Sites))
	for i, s := range h.Sites {
		mps[i] = s.pageMap
	}
	return &pageMaps{
		workers: para.New(h.numWorkers),
		pmaps:   mps,
	}
}
