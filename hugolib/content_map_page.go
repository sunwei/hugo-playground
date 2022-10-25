package hugolib

import (
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
