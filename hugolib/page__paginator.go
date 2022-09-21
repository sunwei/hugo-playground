package hugolib

import (
	"sync"

	"github.com/sunwei/hugo-playground/resources/page"
)

func newPagePaginator(source *pageState) *pagePaginator {
	return &pagePaginator{
		source:            source,
		pagePaginatorInit: &pagePaginatorInit{},
	}
}

type pagePaginator struct {
	*pagePaginatorInit
	source *pageState
}

type pagePaginatorInit struct {
	init    sync.Once
	current *page.Pager
}

func (p *pagePaginator) Paginate(seq any, options ...any) (*page.Pager, error) {
	var initErr error
	p.init.Do(func() {
		pagerSize, err := page.ResolvePagerSize(p.source.s.Cfg, options...)
		if err != nil {
			initErr = err
			return
		}

		pd := p.source.targetPathDescriptor
		pd.Type = p.source.outputFormat()
		paginator, err := page.Paginate(pd, seq, pagerSize)
		if err != nil {
			initErr = err
			return
		}

		p.current = paginator.Pagers()[0]
	})

	if initErr != nil {
		return nil, initErr
	}

	return p.current, nil
}

func (p *pagePaginator) Paginator(options ...any) (*page.Pager, error) {
	var initErr error
	p.init.Do(func() {
		pagerSize, err := page.ResolvePagerSize(p.source.s.Cfg, options...)
		if err != nil {
			initErr = err
			return
		}

		pd := p.source.targetPathDescriptor
		pd.Type = p.source.outputFormat()

		var pages page.Pages

		switch p.source.Kind() {
		case page.KindHome:
			// From Hugo 0.57 we made home.Pages() work like any other
			// section. To avoid the default paginators for the home page
			// changing in the wild, we make this a special case.
			pages = p.source.s.RegularPages()
		case page.KindTerm, page.KindTaxonomy:
			pages = p.source.Pages()
		default:
			pages = p.source.RegularPages()
		}

		paginator, err := page.Paginate(pd, pages, pagerSize)
		if err != nil {
			initErr = err
			return
		}

		p.current = paginator.Pagers()[0]
	})

	if initErr != nil {
		return nil, initErr
	}

	return p.current, nil
}

// reset resets the paginator to allow for a rebuild.
func (p *pagePaginator) reset() {
	p.pagePaginatorInit = &pagePaginatorInit{}
}
