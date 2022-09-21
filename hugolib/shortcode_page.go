package hugolib

import (
	"github.com/sunwei/hugo-playground/resources/page"
)

func newPageForRenderHook(p *pageState) page.Page {
	return &pageForRenderHooks{
		PageWithoutContent:      p,
		ContentProvider:         page.NopPage,
		TableOfContentsProvider: page.NopPage,
	}
}

// This is what is sent into the content render hooks (link, image).
type pageForRenderHooks struct {
	page.PageWithoutContent
	page.TableOfContentsProvider
	page.ContentProvider
}

func (p *pageForRenderHooks) page() page.Page {
	return p.PageWithoutContent.(page.Page)
}
